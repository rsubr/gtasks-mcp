package mcp

import (
	"encoding/json"
	"sync"

	"gtasks-mcp/internal/logging"
)

var clients = struct {
	sync.Mutex
	m map[chan []byte]bool
}{m: make(map[chan []byte]bool)}

func broadcast(msg any) {
	payload, err := json.Marshal(msg)
	if err != nil {
		logging.Warn("failed to marshal sse message", "error", err)
		return
	}

	clients.Lock()
	defer clients.Unlock()
	for ch := range clients.m {
		select {
		case ch <- payload:
		default:
		}
	}
}

func broadcastNotification(method string, params map[string]any) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		msg["params"] = params
	}
	broadcast(msg)
}

func broadcastResourcesListChanged() {
	broadcastNotification("notifications/resources/list_changed", nil)
}

func broadcastResourceUpdated(uri string) {
	broadcastNotification("notifications/resources/updated", map[string]any{"uri": uri})
}
