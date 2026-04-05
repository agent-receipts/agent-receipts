//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/agent-receipts/ar/sdk/go/receipt"
)

type testVectors struct {
	Keys             vectorKeys             `json:"keys"`
	Canonicalization vectorCanonicalization `json:"canonicalization"`
	Hashing          vectorHashing          `json:"hashing"`
	Signing          vectorSigning          `json:"signing"`
}

type vectorKeys struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type vectorCanonicalization struct {
	SimpleInput     any    `json:"simpleInput"`
	SimpleExpected  string `json:"simpleExpected"`
	ReceiptInput    any    `json:"receiptInput"`
	ReceiptExpected string `json:"receiptExpected"`
}

type vectorHashing struct {
	SimpleInput     string `json:"simpleInput"`
	SimpleExpected  string `json:"simpleExpected"`
	ReceiptExpected string `json:"receiptExpected"`
}

type vectorSigning struct {
	Unsigned           json.RawMessage `json:"unsigned"`
	Signed             json.RawMessage `json:"signed"`
	VerificationMethod string          `json:"verificationMethod"`
}

func loadVectors(t *testing.T, path string) testVectors {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var v testVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	return v
}

// TestCrossLanguageTSCanonicalization verifies the Go SDK produces the same
// canonical JSON as the TypeScript SDK.
func TestCrossLanguageTSCanonicalization(t *testing.T) {
	v := loadVectors(t, "../../sdk/py/tests/fixtures/ts_vectors.json")

	t.Run("simple_object", func(t *testing.T) {
		got, err := receipt.Canonicalize(v.Canonicalization.SimpleInput)
		if err != nil {
			t.Fatal(err)
		}
		if got != v.Canonicalization.SimpleExpected {
			t.Errorf("got  %s\nwant %s", got, v.Canonicalization.SimpleExpected)
		}
	})

	t.Run("receipt", func(t *testing.T) {
		got, err := receipt.Canonicalize(v.Canonicalization.ReceiptInput)
		if err != nil {
			t.Fatal(err)
		}
		if got != v.Canonicalization.ReceiptExpected {
			t.Errorf("got  %s\nwant %s", got, v.Canonicalization.ReceiptExpected)
		}
	})
}

// TestCrossLanguageTSHashing verifies the Go SDK produces the same SHA-256
// hashes as the TypeScript SDK.
func TestCrossLanguageTSHashing(t *testing.T) {
	v := loadVectors(t, "../../sdk/py/tests/fixtures/ts_vectors.json")

	t.Run("simple_string", func(t *testing.T) {
		got := receipt.SHA256Hash(v.Hashing.SimpleInput)
		if got != v.Hashing.SimpleExpected {
			t.Errorf("got %s, want %s", got, v.Hashing.SimpleExpected)
		}
	})

	t.Run("receipt_hash", func(t *testing.T) {
		var signed receipt.AgentReceipt
		if err := json.Unmarshal(v.Signing.Signed, &signed); err != nil {
			t.Fatal(err)
		}
		got, err := receipt.HashReceipt(signed)
		if err != nil {
			t.Fatal(err)
		}
		if got != v.Hashing.ReceiptExpected {
			t.Errorf("got %s, want %s", got, v.Hashing.ReceiptExpected)
		}
	})
}

// TestCrossLanguageTSSignatureVerifiesInGo verifies that a receipt signed by
// the TypeScript SDK (or regenerated with the shared key) can be verified by Go.
func TestCrossLanguageTSSignatureVerifiesInGo(t *testing.T) {
	v := loadVectors(t, "../../sdk/py/tests/fixtures/ts_vectors.json")

	var signed receipt.AgentReceipt
	if err := json.Unmarshal(v.Signing.Signed, &signed); err != nil {
		t.Fatal(err)
	}

	valid, err := receipt.Verify(signed, v.Keys.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Error("TS-signed receipt did not verify in Go")
	}
}

// TestCrossLanguageTSSignatureFailsWithWrongKey verifies that a wrong key
// correctly rejects the signature.
func TestCrossLanguageTSSignatureFailsWithWrongKey(t *testing.T) {
	v := loadVectors(t, "../../sdk/py/tests/fixtures/ts_vectors.json")

	var signed receipt.AgentReceipt
	if err := json.Unmarshal(v.Signing.Signed, &signed); err != nil {
		t.Fatal(err)
	}

	otherKP, err := receipt.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	valid, err := receipt.Verify(signed, otherKP.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Error("signature should not verify with wrong key")
	}
}

// TestCrossLanguageTSSignatureFailsWhenTampered verifies that tampering
// with the receipt invalidates the signature.
func TestCrossLanguageTSSignatureFailsWhenTampered(t *testing.T) {
	v := loadVectors(t, "../../sdk/py/tests/fixtures/ts_vectors.json")

	var signed receipt.AgentReceipt
	if err := json.Unmarshal(v.Signing.Signed, &signed); err != nil {
		t.Fatal(err)
	}

	signed.CredentialSubject.Action.Type = "filesystem.file.delete"

	valid, err := receipt.Verify(signed, v.Keys.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Error("tampered receipt should not verify")
	}
}
