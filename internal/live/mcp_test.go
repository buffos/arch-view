package live

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestMCPServeReturnsProtocolMessagesWithoutStdoutNoise(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_snapshot_status","arguments":{"max_items":10,"max_bytes":4096}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}
`)
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("MCP output lines = %d, output=%q", len(lines), output.String())
	}
	var initialize map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initialize); err != nil {
		t.Fatal(err)
	}
	if initialize["jsonrpc"] != "2.0" || initialize["id"].(float64) != 1 || initialize["result"].(map[string]any)["serverInfo"].(map[string]any)["name"] != MCPServerName {
		t.Fatalf("initialize response = %#v", initialize)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &status); err != nil {
		t.Fatal(err)
	}
	structured := status["result"].(map[string]any)["structuredContent"].(map[string]any)
	if structured["schema_version"] != QuerySchemaVersion || structured["session_id"] != "session:mcp" || structured["revision"].(float64) != 1 {
		t.Fatalf("status tool envelope = %#v", structured)
	}
	var unknown map[string]any
	if err := json.Unmarshal([]byte(lines[3]), &unknown); err != nil {
		t.Fatal(err)
	}
	if unknown["result"].(map[string]any)["isError"] != true {
		t.Fatalf("unknown tool response = %#v", unknown)
	}
}

func TestMCPResourcesAndMalformedRequestsAreStructured(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-resources")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader("not-json\n{" + `"jsonrpc":"2.0","id":1,"method":"resources/list"` + "}\n" + `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"archview://session/session:mcp-resources/status"}}` + "\n")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("resource output lines = %d, output=%q", len(lines), output.String())
	}
	var parseError map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &parseError); err != nil {
		t.Fatal(err)
	}
	if parseError["error"].(map[string]any)["code"].(float64) != -32700 {
		t.Fatalf("parse error = %#v", parseError)
	}
	var resources map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &resources); err != nil {
		t.Fatal(err)
	}
	if len(resources["result"].(map[string]any)["contents"].([]any)) != 1 {
		t.Fatalf("resource response = %#v", resources)
	}
}

func TestMCPPositiveIntegerParsingIsStrict(t *testing.T) {
	if value, err := parsePositiveInt("12"); err != nil || value != 12 {
		t.Fatalf("valid integer = %d, %v", value, err)
	}
	if _, err := parsePositiveInt("12junk"); err == nil {
		t.Fatal("integer parser accepted trailing text")
	}
}

func TestMCPOversizedMessageReturnsStructuredError(t *testing.T) {
	server := NewMCPServer(NewLocalQueryAdapter(newTransportTestSession(t, "session:mcp-large")))
	server.MaxMessageBytes = 16
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(strings.Repeat("x", 64)+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("oversized response = %q: %v", output.String(), err)
	}
	if response["error"].(map[string]any)["code"].(float64) != -32600 {
		t.Fatalf("oversized response = %#v", response)
	}
}

func TestMCPRejectsUnknownToolArgumentsAndAdvertisesClosedSchemas(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-schema")
	server := NewMCPServer(NewLocalQueryAdapter(session))
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"find_files","arguments":{"max_itmes":10}}}` + "\n")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	var listed map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &listed); err != nil {
		t.Fatal(err)
	}
	tools := listed["result"].(map[string]any)["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("MCP tool catalog is empty")
	}
	schema := tools[0].(map[string]any)["inputSchema"].(map[string]any)
	if schema["additionalProperties"] != false || schema["properties"] == nil {
		t.Fatalf("MCP tool schema = %#v", schema)
	}
	var rejected map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &rejected); err != nil {
		t.Fatal(err)
	}
	result := rejected["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(result["structuredContent"].(map[string]any)["message"].(string), "invalid") {
		t.Fatalf("unknown tool argument response = %#v", rejected)
	}
}

func TestMCPServeClosesBlockingInputOnCancellation(t *testing.T) {
	session := newTransportTestSession(t, "session:mcp-cancel")
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- NewMCPServer(NewLocalQueryAdapter(session)).Serve(ctx, reader, io.Discard)
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("MCP Serve remained blocked after cancellation")
	}
}
