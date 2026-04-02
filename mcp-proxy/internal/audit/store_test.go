package audit

import (
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
