package audit

import (
	"testing"
	"time"
)

func TestIntentTrackerGroupsWithinThreshold(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionID := "test-session-intent-1"
	if err := store.CreateSession(sessionID, "test-server", "test-cmd"); err != nil {
		t.Fatal(err)
	}

	gap := 1 * time.Second
	tracker := NewIntentTracker(store, sessionID, gap)

	// Insert two dummy tool calls.
	now := time.Now()
	tc1, err := store.InsertToolCall(ToolCallRecord{
		SessionID:     sessionID,
		ToolName:      "tool_a",
		OperationType: "read",
		PolicyAction:  "pass",
		RequestedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	tc2, err := store.InsertToolCall(ToolCallRecord{
		SessionID:     sessionID,
		ToolName:      "tool_b",
		OperationType: "read",
		PolicyAction:  "pass",
		RequestedAt:   now.Add(100 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Track both within gap threshold.
	if err := tracker.Track(tc1, now); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Track(tc2, now.Add(100*time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	// Verify both are linked to the same intent.
	var count int
	err = store.db.QueryRow("SELECT COUNT(DISTINCT intent_id) FROM intent_tool_calls").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 intent, got %d", count)
	}
}

func TestIntentTrackerSplitsOutsideThreshold(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionID := "test-session-intent-2"
	if err := store.CreateSession(sessionID, "test-server", "test-cmd"); err != nil {
		t.Fatal(err)
	}

	gap := 100 * time.Millisecond
	tracker := NewIntentTracker(store, sessionID, gap)

	now := time.Now()
	tc1, err := store.InsertToolCall(ToolCallRecord{
		SessionID:     sessionID,
		ToolName:      "tool_a",
		OperationType: "read",
		PolicyAction:  "pass",
		RequestedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	tc2, err := store.InsertToolCall(ToolCallRecord{
		SessionID:     sessionID,
		ToolName:      "tool_b",
		OperationType: "read",
		PolicyAction:  "pass",
		RequestedAt:   now.Add(500 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Track with timestamps outside the gap.
	if err := tracker.Track(tc1, now); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Track(tc2, now.Add(500*time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	// Verify they are in separate intents.
	var count int
	err = store.db.QueryRow("SELECT COUNT(DISTINCT intent_id) FROM intent_tool_calls").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 intents, got %d", count)
	}
}
