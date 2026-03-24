package mcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gtasks-mcp/internal/tasks"
)

func TestToolsListWithUnavailableBackend(t *testing.T) {
	server := NewServer(tasks.NewUnavailable("My Tasks", errors.New("missing credentials")))

	initReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`))
	initRec := httptest.NewRecorder()
	server.handleMCP(initRec, initReq)

	if initRec.Code != http.StatusOK {
		t.Fatalf("initialize status = %d, want %d", initRec.Code, http.StatusOK)
	}

	sessionID := initRec.Header().Get("MCP-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize did not return MCP-Session-Id")
	}
	protocolHeader := initRec.Header().Get("MCP-Protocol-Version")
	if protocolHeader == "" {
		t.Fatal("initialize did not return MCP-Protocol-Version")
	}
	t.Cleanup(func() {
		deleteSession(sessionID)
	})

	toolsReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	toolsReq.Header.Set("MCP-Session-Id", sessionID)
	toolsReq.Header.Set("MCP-Protocol-Version", protocolHeader)
	toolsRec := httptest.NewRecorder()
	server.handleMCP(toolsRec, toolsReq)

	if toolsRec.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, want %d", toolsRec.Code, http.StatusOK)
	}

	var resp struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(toolsRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}
	if len(resp.Result.Tools) == 0 {
		t.Fatal("tools/list returned no tools")
	}
}

func TestToolCallReportsUnavailableBackend(t *testing.T) {
	server := NewServer(tasks.NewUnavailable("My Tasks", errors.New("missing credentials")))

	initReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`))
	initRec := httptest.NewRecorder()
	server.handleMCP(initRec, initReq)

	sessionID := initRec.Header().Get("MCP-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize did not return MCP-Session-Id")
	}
	protocolHeader := initRec.Header().Get("MCP-Protocol-Version")
	if protocolHeader == "" {
		t.Fatal("initialize did not return MCP-Protocol-Version")
	}
	t.Cleanup(func() {
		deleteSession(sessionID)
	})

	callReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list","arguments":{}}}`))
	callReq.Header.Set("MCP-Session-Id", sessionID)
	callReq.Header.Set("MCP-Protocol-Version", protocolHeader)
	callRec := httptest.NewRecorder()
	server.handleMCP(callRec, callReq)

	if callRec.Code != http.StatusOK {
		t.Fatalf("tools/call status = %d, want %d", callRec.Code, http.StatusOK)
	}

	var resp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(callRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode tools/call response: %v", err)
	}
	if !resp.Result.IsError {
		t.Fatal("tools/call did not report backend error")
	}
	if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "unavailable") {
		t.Fatalf("unexpected tools/call error content: %+v", resp.Result.Content)
	}
}
