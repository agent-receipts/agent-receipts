// mock_server is a minimal MCP server for e2e testing.
// It reads JSON-RPC 2.0 messages from stdin (one per line) and responds on stdout.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var msg message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.JSONRPC != "2.0" {
			continue
		}

		var resp any
		switch msg.Method {
		case "initialize":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"serverInfo":      map[string]any{"name": "mock-server", "version": "0.1.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			}
		case "tools/list":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "read_file", "description": "Read a file"},
						{"name": "write_file", "description": "Write a file"},
					},
				},
			}
		case "tools/call":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "ok"},
					},
				},
			}
		case "notifications/initialized":
			continue // Notification, no response.
		default:
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result":  map[string]any{},
			}
		}

		b, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		fmt.Fprintf(os.Stdout, "%s\n", b)
	}
}
