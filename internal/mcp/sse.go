package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"gtasks-mcp/internal/logging"
)

type sessionState struct {
	ID              string
	ProtocolVersion string
	clients         map[chan []byte]struct{}
}

var sessions = struct {
	sync.Mutex
	byID map[string]*sessionState
}{
	byID: make(map[string]*sessionState),
}

func createSession(protocolVersion string) (*sessionState, error) {
	id, err := randomSessionID()
	if err != nil {
		return nil, err
	}

	session := &sessionState{
		ID:              id,
		ProtocolVersion: protocolVersion,
		clients:         make(map[chan []byte]struct{}),
	}

	sessions.Lock()
	sessions.byID[id] = session
	sessions.Unlock()

	return session, nil
}

func getSession(id string) (*sessionState, bool) {
	sessions.Lock()
	defer sessions.Unlock()

	session, ok := sessions.byID[id]
	return session, ok
}

func deleteSession(id string) bool {
	sessions.Lock()
	session, ok := sessions.byID[id]
	if ok {
		delete(sessions.byID, id)
	}
	sessions.Unlock()
	if !ok {
		return false
	}

	for ch := range session.clients {
		close(ch)
	}
	return true
}

func addSessionClient(sessionID string, ch chan []byte) bool {
	sessions.Lock()
	defer sessions.Unlock()

	session, ok := sessions.byID[sessionID]
	if !ok {
		return false
	}
	session.clients[ch] = struct{}{}
	return true
}

func removeSessionClient(sessionID string, ch chan []byte) {
	sessions.Lock()
	defer sessions.Unlock()

	session, ok := sessions.byID[sessionID]
	if !ok {
		return
	}
	delete(session.clients, ch)
}

func broadcast(sessionID string, msg any) {
	payload, err := json.Marshal(msg)
	if err != nil {
		logging.Warn("failed to marshal sse message", "error", err)
		return
	}

	sessions.Lock()
	session, ok := sessions.byID[sessionID]
	if !ok {
		sessions.Unlock()
		return
	}

	targets := make([]chan []byte, 0, len(session.clients))
	for ch := range session.clients {
		targets = append(targets, ch)
	}
	sessions.Unlock()

	for _, ch := range targets {
		select {
		case ch <- payload:
		default:
		}
	}
}

func broadcastNotification(sessionID, method string, params map[string]any) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		msg["params"] = params
	}
	broadcast(sessionID, msg)
}

func broadcastResourcesListChanged(sessionID string) {
	broadcastNotification(sessionID, "notifications/resources/list_changed", nil)
}

func broadcastResourceUpdated(sessionID, uri string) {
	broadcastNotification(sessionID, "notifications/resources/updated", map[string]any{"uri": uri})
}

func randomSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
