package audit

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"
)

func TestCreateAndEndSession(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionID := "sess-roundtrip"
	if err := store.CreateSession(sessionID, "test-server", "test-cmd"); err != nil {
		t.Fatal(err)
	}

	if err := store.EndSession(sessionID); err != nil {
		t.Fatal(err)
	}

	// Verify ended_at is populated.
	var endedAt *string
	err = store.db.QueryRow("SELECT ended_at FROM sessions WHERE id = ?", sessionID).Scan(&endedAt)
	if err != nil {
		t.Fatal(err)
	}
	if endedAt == nil {
		t.Error("expected ended_at to be non-nil after EndSession")
	}
}

func TestLogMessageReturnsValidID(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionID := "sess-logmsg"
	if err := store.CreateSession(sessionID, "test-server", "test-cmd"); err != nil {
		t.Fatal(err)
	}

	msgID, err := store.LogMessage(sessionID, "client_to_server", "1", "tools/call", `{"test":"data"}`)
	if err != nil {
		t.Fatal(err)
	}
	if msgID <= 0 {
		t.Errorf("expected positive message ID, got %d", msgID)
	}
}

func TestInsertToolCallAllFields(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionID := "sess-toolcall"
	if err := store.CreateSession(sessionID, "test-server", "test-cmd"); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	tcID, err := store.InsertToolCall(ToolCallRecord{
		SessionID:     sessionID,
		RequestMsgID:  1,
		ResponseMsgID: 2,
		ToolName:      "read_file",
		Arguments:     `{"path":"/tmp/test"}`,
		Result:        `{"content":"hello"}`,
		Error:         "",
		OperationType: "read",
		RiskScore:     10,
		RiskReasons:   []string{"file_access"},
		PolicyAction:  "pass",
		ApprovedBy:    "http",
		RequestedAt:   now,
		RespondedAt:   now.Add(100 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	if tcID <= 0 {
		t.Errorf("expected positive tool call ID, got %d", tcID)
	}

	// Verify approved_by was stored.
	var approvedBy *string
	err = store.db.QueryRow("SELECT approved_by FROM tool_calls WHERE id = ?", tcID).Scan(&approvedBy)
	if err != nil {
		t.Fatal(err)
	}
	if approvedBy == nil || *approvedBy != "http" {
		t.Errorf("expected approved_by = 'http', got %v", approvedBy)
	}
}

func TestEncryptionSalt(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// First call generates a 16-byte salt.
	salt1, err := store.EncryptionSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt1) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(salt1))
	}

	// Second call returns the same salt.
	salt2, err := store.EncryptionSalt()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(salt1, salt2) {
		t.Errorf("expected same salt on second call, got %x vs %x", salt1, salt2)
	}

	// Salt is persisted in the metadata table.
	var encoded string
	err = store.db.QueryRow("SELECT value FROM metadata WHERE key = 'encryption_salt'").Scan(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(salt1, persisted) {
		t.Errorf("persisted salt doesn't match: %x vs %x", salt1, persisted)
	}
}

func TestEncryptionSaltRejectsCorrupted(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Insert a corrupted salt (wrong length).
	if _, err := store.db.Exec("INSERT INTO metadata (key, value) VALUES ('encryption_salt', 'abcd')"); err != nil {
		t.Fatal(err)
	}

	_, err = store.EncryptionSalt()
	if err == nil {
		t.Error("expected error for corrupted salt")
	}
}
