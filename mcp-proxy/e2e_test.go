//go:build e2e

package e2e_test

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/agent-receipts/ar/sdk/go/receipt"
	receiptStore "github.com/agent-receipts/ar/sdk/go/store"
)

// buildBinary compiles a Go package into the given directory and returns the binary path.
func buildBinary(t *testing.T, pkg, dir, name string) string {
	t.Helper()
	out := filepath.Join(dir, name)
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build %s: %v", pkg, err)
	}
	return out
}

// sendJSON writes a JSON-RPC message followed by a newline.
func sendJSON(t *testing.T, w *os.File, msg any) {
	t.Helper()
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// readResponse reads one JSON-RPC response line and returns it parsed.
func readResponse(t *testing.T, scanner *bufio.Scanner) map[string]any {
	t.Helper()
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan: %v", err)
		}
		t.Fatal("unexpected EOF from proxy")
	}
	var resp map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", scanner.Text(), err)
	}
	return resp
}

func TestE2EProxyToolCallFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Build the mock server and proxy binaries.
	mockBin := buildBinary(t, "./testdata", tmpDir, "mock-server")
	proxyBin := buildBinary(t, "./cmd/mcp-proxy", tmpDir, "mcp-proxy")

	// Generate a keypair and write the private key to a temp file.
	kp, err := receipt.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(keyPath, []byte(kp.PrivateKey), 0600); err != nil {
		t.Fatal(err)
	}

	auditDBPath := filepath.Join(tmpDir, "audit.db")
	receiptDBPath := filepath.Join(tmpDir, "receipts.db")
	chainID := "e2e-test-chain"

	// Start the proxy with the mock server as the wrapped command.
	cmd := exec.Command(proxyBin,
		"--db", auditDBPath,
		"--receipt-db", receiptDBPath,
		"--key", keyPath,
		"--chain", chainID,
		"--http", "127.0.0.1:0", // Use port 0 to avoid conflicts.
		"--", mockBin,
	)
	cmd.Stderr = os.Stderr

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() {
		stdinPipe.Close()
		cmd.Wait()
	})

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Give the proxy a moment to start the child process.
	time.Sleep(500 * time.Millisecond)

	// Cast stdinPipe to *os.File for sendJSON — it's actually an io.WriteCloser.
	// We need to write through the pipe directly.
	stdinWriter := stdinPipe

	// Send tools/call for read_file.
	req1, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": "read_file", "arguments": map[string]any{"path": "/tmp/test"}},
	})
	if _, err := stdinWriter.Write(append(req1, '\n')); err != nil {
		t.Fatalf("write req1: %v", err)
	}

	resp1 := readResponse(t, scanner)
	if resp1["error"] != nil {
		t.Fatalf("expected success for read_file, got error: %v", resp1["error"])
	}

	// Send tools/call for write_file.
	req2, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  map[string]any{"name": "write_file", "arguments": map[string]any{"path": "/tmp/out", "content": "hello"}},
	})
	if _, err := stdinWriter.Write(append(req2, '\n')); err != nil {
		t.Fatalf("write req2: %v", err)
	}

	resp2 := readResponse(t, scanner)
	if resp2["error"] != nil {
		t.Fatalf("expected success for write_file, got error: %v", resp2["error"])
	}

	// Close stdin to signal EOF; proxy should exit.
	stdinPipe.Close()
	cmd.Wait()

	// Open the receipt store and verify the chain.
	rStore, err := receiptStore.Open(receiptDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rStore.Close()

	chain, err := rStore.GetChain(chainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 2 {
		t.Fatalf("expected 2 receipts, got %d", len(chain))
	}

	// Verify chain integrity.
	result := receipt.VerifyChain(chain, kp.PublicKey)
	if !result.Valid {
		t.Fatalf("chain verification failed: broken at %d, error: %s", result.BrokenAt, result.Error)
	}

	// Verify sequence numbers.
	if chain[0].CredentialSubject.Chain.Sequence != 1 {
		t.Errorf("receipt 0: expected sequence 1, got %d", chain[0].CredentialSubject.Chain.Sequence)
	}
	if chain[1].CredentialSubject.Chain.Sequence != 2 {
		t.Errorf("receipt 1: expected sequence 2, got %d", chain[1].CredentialSubject.Chain.Sequence)
	}

	// First receipt should have no previous hash.
	if chain[0].CredentialSubject.Chain.PreviousReceiptHash != nil {
		t.Error("receipt 0: expected nil PreviousReceiptHash")
	}
	// Second receipt should have a previous hash.
	if chain[1].CredentialSubject.Chain.PreviousReceiptHash == nil {
		t.Error("receipt 1: expected non-nil PreviousReceiptHash")
	}
}

func TestE2EProxyBlockedCall(t *testing.T) {
	tmpDir := t.TempDir()

	mockBin := buildBinary(t, "./testdata", tmpDir, "mock-server")
	proxyBin := buildBinary(t, "./cmd/mcp-proxy", tmpDir, "mcp-proxy")

	kp, err := receipt.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(keyPath, []byte(kp.PrivateKey), 0600); err != nil {
		t.Fatal(err)
	}

	auditDBPath := filepath.Join(tmpDir, "audit.db")
	receiptDBPath := filepath.Join(tmpDir, "receipts.db")
	chainID := "e2e-test-blocked"

	cmd := exec.Command(proxyBin,
		"--db", auditDBPath,
		"--receipt-db", receiptDBPath,
		"--key", keyPath,
		"--chain", chainID,
		"--http", "127.0.0.1:0",
		"--", mockBin,
	)
	cmd.Stderr = os.Stderr

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() {
		stdinPipe.Close()
		cmd.Wait()
	})

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	time.Sleep(500 * time.Millisecond)

	// Send a tool call that should be blocked: delete_secrets has risk >= 70.
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": "delete_secrets", "arguments": map[string]any{}},
	})
	if _, err := stdinPipe.Write(append(req, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}

	resp := readResponse(t, scanner)

	// Should be a JSON-RPC error.
	errObj, ok := resp["error"]
	if !ok || errObj == nil {
		t.Fatalf("expected error response for blocked call, got: %v", resp)
	}
	errMap, ok := errObj.(map[string]any)
	if !ok {
		t.Fatalf("expected error to be an object, got: %T", errObj)
	}
	code, _ := errMap["code"].(float64)
	if int(code) != -32001 {
		t.Errorf("expected error code -32001, got %v", errMap["code"])
	}

	// Close stdin and wait.
	stdinPipe.Close()
	cmd.Wait()

	// Verify no receipts were created for the blocked call.
	rStore, err := receiptStore.Open(receiptDBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rStore.Close()

	chain, err := rStore.GetChain(chainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 0 {
		t.Errorf("expected 0 receipts for blocked call, got %d", len(chain))
	}
}
