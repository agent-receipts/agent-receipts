package proxy

import (
	"testing"
)

func TestParseMessage(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"/tmp/test"}}}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Method != "tools/call" {
		t.Errorf("expected tools/call, got %s", msg.Method)
	}
	if !msg.IsRequest() {
		t.Error("expected request")
	}
	if !msg.IsToolCall() {
		t.Error("expected tool call")
	}

	params, err := msg.ParseToolCallParams()
	if err != nil {
		t.Fatal(err)
	}
	if params.Name != "read_file" {
		t.Errorf("expected read_file, got %s", params.Name)
	}
}

func TestParseMessageResponse(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":"hello"}}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if !msg.IsResponse() {
		t.Error("expected response")
	}
	if msg.IsRequest() {
		t.Error("should not be request")
	}
}

func TestParseMessageInvalid(t *testing.T) {
	if msg := ParseMessage([]byte("not json")); msg != nil {
		t.Error("expected nil for invalid JSON")
	}
	if msg := ParseMessage([]byte(`{"jsonrpc":"1.0"}`)); msg != nil {
		t.Error("expected nil for wrong jsonrpc version")
	}
}

func TestParseToolCallParamsNil(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	params, err := msg.ParseToolCallParams()
	if err != nil {
		t.Fatal(err)
	}
	if params == nil {
		t.Fatal("expected non-nil params for nil Params field")
	}
	if params.Name != "" {
		t.Errorf("expected empty name, got %s", params.Name)
	}
}

func TestParseMessageNotification(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil")
	}
	if !msg.IsNotification() {
		t.Error("expected notification")
	}
}

func TestIDStringWithStringID(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","id":"abc","method":"tools/call","params":{"name":"test"}}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if got := msg.IDString(); got != "abc" {
		t.Errorf("expected IDString() = %q, got %q", "abc", got)
	}
}

func TestParseMessageNullID(t *testing.T) {
	line := []byte(`{"jsonrpc":"2.0","id":null,"result":{}}`)
	msg := ParseMessage(line)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	// id is present in JSON (as null), so RawMessage is non-nil (contains "null").
	if msg.ID == nil {
		t.Error("expected ID to be non-nil (JSON null)")
	}
}

func TestParseMessageBatchReturnsNil(t *testing.T) {
	line := []byte(`[{"jsonrpc":"2.0","id":1,"method":"test"}]`)
	msg := ParseMessage(line)
	if msg != nil {
		t.Error("expected nil for batch (JSON array) message")
	}
}
