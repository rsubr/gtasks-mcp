package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gtasks-mcp/internal/logging"
	"gtasks-mcp/internal/tasks"
)

const protocolVersion = "2025-06-18"

type Server struct {
	tasks *tasks.Service
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
}

type callToolResult struct {
	Content          []map[string]any `json:"content"`
	StructuredResult map[string]any   `json:"structuredContent,omitempty"`
	IsError          bool             `json:"isError,omitempty"`
}

type rpcResponseRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func NewServer(tasks *tasks.Service) *Server {
	return &Server{tasks: tasks}
}

func (s *Server) Start(addr string) error {
	logging.Info("starting http server", "listen_addr", addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)
	mux.HandleFunc("/events", s.handleSSE)
	mux.HandleFunc("/manifest", s.handleManifest)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	logging.Debug("sse client connected", "remote_addr", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		logging.Error("sse streaming unsupported", "remote_addr", r.RemoteAddr)
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 8)
	clients.Lock()
	clients.m[ch] = true
	clients.Unlock()
	defer func() {
		clients.Lock()
		delete(clients.m, ch)
		clients.Unlock()
		logging.Debug("sse client disconnected", "remote_addr", r.RemoteAddr)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			logging.Debug("sending sse event", "remote_addr", r.RemoteAddr)
			_, _ = w.Write([]byte("event: message\n"))
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	logging.Debug("received rpc request", "remote_addr", r.RemoteAddr, "method", r.Method, "path", r.URL.Path)
	if r.Method != http.MethodPost {
		s.writeError(w, nil, -32600, "invalid request", map[string]any{"details": "POST required", "source": "internal"})
		return
	}

	requests, isBatch, err := decodeRequests(r)
	if err != nil {
		s.writeError(w, nil, -32700, "parse error", map[string]any{"details": "malformed JSON request body", "source": "internal"})
		return
	}
	if isBatch && len(requests) == 0 {
		s.writeError(w, nil, -32600, "invalid request", map[string]any{"details": "batch request must not be empty", "source": "internal"})
		return
	}

	if !isBatch {
		resp := s.dispatchRequest(requests[0])
		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.writeJSONValue(w, resp)
		return
	}

	responses := make([]Response, 0, len(requests))
	for _, req := range requests {
		if resp := s.dispatchRequest(req); resp != nil {
			responses = append(responses, *resp)
		}
	}

	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.writeJSONValue(w, responses)
}

func decodeRequests(r *http.Request) ([]*Request, bool, error) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false, err
	}

	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, false, fmt.Errorf("request body is empty")
	}

	if body[0] != '[' {
		var req Request
		if err := decodeJSONStrict(body, &req); err != nil {
			return nil, false, err
		}
		return []*Request{&req}, false, nil
	}

	var rawItems []json.RawMessage
	if err := decodeJSONStrict(body, &rawItems); err != nil {
		return nil, true, err
	}

	requests := make([]*Request, 0, len(rawItems))
	for _, raw := range rawItems {
		var req Request
		if err := decodeJSONStrict(raw, &req); err != nil {
			requests = append(requests, nil)
			continue
		}
		reqCopy := req
		requests = append(requests, &reqCopy)
	}
	return requests, true, nil
}

func decodeJSONStrict(data []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing data")
		}
		return err
	}
	return nil
}

func (s *Server) dispatchRequest(req *Request) *Response {
	if req == nil {
		resp := newErrorResponse(nil, -32600, "invalid request", map[string]any{"details": "request object is invalid", "source": "internal"})
		return &resp
	}

	if err := validateRequest(req); err != nil {
		resp := newErrorResponse(req.ID, -32600, "invalid request", map[string]any{"details": err.Error(), "source": "internal"})
		return &resp
	}

	logging.Info("dispatching rpc method", "rpc_method", req.Method, "request_id", rawIDForLog(req.ID))

	if strings.HasPrefix(req.Method, "notifications/") {
		return nil
	}

	recorder := newRPCResponseRecorder()
	switch req.Method {
	case "initialize":
		s.handleInitialize(recorder, req)
	case "ping":
		s.writeResult(recorder, req.ID, map[string]any{})
	case "tools/list":
		s.writeResult(recorder, req.ID, map[string]any{"tools": ToolSchemas()})
	case "tools/call":
		s.handleToolCall(recorder, req)
	case "resources/list":
		s.handleResourcesList(recorder, req)
	case "resources/read":
		s.handleResourcesRead(recorder, req)
	default:
		s.writeError(recorder, req.ID, -32601, "method not found", map[string]any{"details": fmt.Sprintf("unsupported method %q", req.Method), "source": "internal"})
	}

	resp, err := recorder.response()
	if err != nil {
		fallback := newErrorResponse(req.ID, -32000, "internal server error", map[string]any{"details": err.Error(), "source": "internal"})
		return &fallback
	}
	return resp
}

func validateRequest(req *Request) error {
	if req == nil {
		return fmt.Errorf("request missing")
	}
	if req.JSONRPC != "2.0" {
		return fmt.Errorf(`jsonrpc must be "2.0"`)
	}
	if strings.TrimSpace(req.Method) == "" {
		return fmt.Errorf("method is required")
	}
	return nil
}

func (s *Server) handleNotification(w http.ResponseWriter, req *Request) {
	logging.Debug("handling notification", "rpc_method", req.Method)
	w.WriteHeader(http.StatusNoContent)
}

func newRPCResponseRecorder() *rpcResponseRecorder {
	return &rpcResponseRecorder{header: make(http.Header)}
}

func (r *rpcResponseRecorder) Header() http.Header {
	return r.header
}

func (r *rpcResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *rpcResponseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *rpcResponseRecorder) response() (*Response, error) {
	if r.body.Len() == 0 {
		return nil, nil
	}

	var resp Response
	if err := json.Unmarshal(r.body.Bytes(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func newErrorResponse(id json.RawMessage, code int, message string, data map[string]any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func newResultResponse(id json.RawMessage, result any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req *Request) {
	var params initializeParams
	if len(bytes.TrimSpace(req.Params)) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "initialize params are invalid", "source": "internal"})
			return
		}
	}

	if strings.TrimSpace(params.ProtocolVersion) == "" {
		s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "protocolVersion is required", "source": "internal"})
		return
	}

	logging.Info("client initialized session", "client_name", params.ClientInfo.Name, "client_version", params.ClientInfo.Version, "client_protocol", params.ProtocolVersion)
	s.writeResult(w, req.ID, map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "gtasks-mcp",
			"version": "0.1.0",
		},
		"instructions": "Use the available tools to manage tasks in the configured Google Tasks list.",
	})
}

func (s *Server) handleToolCall(w http.ResponseWriter, req *Request) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "tools/call params are invalid", "source": "internal"})
		return
	}

	if strings.TrimSpace(params.Name) == "" {
		s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "tool name is required", "source": "internal"})
		return
	}

	logging.Info("handling tool call", "tool", params.Name, "request_id", rawIDForLog(req.ID))

	args := params.Arguments
	if len(bytes.TrimSpace(args)) == 0 {
		args = params.Input
	}
	if len(bytes.TrimSpace(args)) == 0 {
		args = []byte("{}")
	}

	switch params.Name {
	case "list":
		res, err := s.tasks.List()
		if err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to list tasks", err))
			return
		}
		s.writeResult(w, req.ID, toolSuccessResult("Listed tasks.", map[string]any{"tasks": res}))
	case "read":
		var input struct {
			ID  string `json:"id"`
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "read arguments are invalid", "source": "internal"})
			return
		}

		taskID, err := resolveTaskID(input.ID, input.URI)
		if err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": err.Error(), "source": "internal"})
			return
		}

		res, err := s.tasks.Get(taskID)
		if err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to read task", err))
			return
		}
		s.writeResult(w, req.ID, toolSuccessResult("Read task.", map[string]any{"task": res}))
	case "search":
		var input struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "search arguments are invalid", "source": "internal"})
			return
		}
		if strings.TrimSpace(input.Query) == "" {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "query is required", "source": "internal"})
			return
		}

		res, err := s.tasks.Search(input.Query)
		if err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to search tasks", err))
			return
		}
		s.writeResult(w, req.ID, toolSuccessResult("Search completed.", map[string]any{"tasks": res}))
	case "create":
		var input struct {
			Title      string `json:"title"`
			Notes      string `json:"notes"`
			Due        string `json:"due"`
			Recurrence string `json:"recurrence"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "create arguments are invalid", "source": "internal"})
			return
		}
		if strings.TrimSpace(input.Title) == "" {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "title is required", "source": "internal"})
			return
		}

		res, err := s.tasks.Create(input.Title, input.Notes, input.Due, input.Recurrence)
		if err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to create task", err))
			return
		}
		broadcastResourcesListChanged()
		broadcastResourceUpdated(res.URI)
		s.writeResult(w, req.ID, toolSuccessResult("Task created.", map[string]any{"task": res}))
	case "update":
		var input struct {
			ID         string  `json:"id"`
			URI        string  `json:"uri"`
			Title      *string `json:"title"`
			Notes      *string `json:"notes"`
			Status     *string `json:"status"`
			Due        *string `json:"due"`
			Recurrence *string `json:"recurrence"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "update arguments are invalid", "source": "internal"})
			return
		}

		taskID, err := resolveTaskID(input.ID, input.URI)
		if err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": err.Error(), "source": "internal"})
			return
		}

		if input.Status != nil && *input.Status != "" && *input.Status != "needsAction" && *input.Status != "completed" {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "status must be needsAction or completed", "source": "internal"})
			return
		}

		res, err := s.tasks.Update(taskID, input.Title, input.Notes, input.Status, input.Due, input.Recurrence)
		if err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to update task", err))
			return
		}
		broadcastResourcesListChanged()
		broadcastResourceUpdated(res.URI)
		s.writeResult(w, req.ID, toolSuccessResult("Task updated.", map[string]any{"task": res}))
	case "delete":
		var input struct {
			ID  string `json:"id"`
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "delete arguments are invalid", "source": "internal"})
			return
		}

		taskID, err := resolveTaskID(input.ID, input.URI)
		if err != nil {
			s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": err.Error(), "source": "internal"})
			return
		}

		if err := s.tasks.Delete(taskID); err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to delete task", err))
			return
		}
		broadcastResourcesListChanged()
		s.writeResult(w, req.ID, toolSuccessResult("Task deleted.", map[string]any{"id": taskID, "uri": tasks.ResourceURI(taskID)}))
	case "clear":
		if err := s.tasks.Clear(); err != nil {
			s.writeResult(w, req.ID, toolErrorResult("failed to clear completed tasks", err))
			return
		}
		broadcastResourcesListChanged()
		s.writeResult(w, req.ID, toolSuccessResult("Completed tasks cleared.", map[string]any{"taskListId": s.tasks.TaskListID()}))
	default:
		s.writeError(w, req.ID, -32601, "method not found", map[string]any{"details": fmt.Sprintf("unknown tool %q", params.Name), "source": "internal"})
	}
}

func (s *Server) handleResourcesList(w http.ResponseWriter, req *Request) {
	logging.Info("listing resources", "request_id", rawIDForLog(req.ID))
	taskItems, err := s.tasks.List()
	if err != nil {
		s.writeError(w, req.ID, -32001, "google api error", map[string]any{"details": err.Error(), "source": "google"})
		return
	}

	resources := make([]map[string]any, 0, len(taskItems))
	for _, task := range taskItems {
		resources = append(resources, map[string]any{
			"uri":         task.URI,
			"name":        task.ID,
			"title":       task.Title,
			"description": "Google Tasks task resource",
			"mimeType":    "application/json",
		})
	}

	s.writeResult(w, req.ID, map[string]any{"resources": resources})
}

func (s *Server) handleResourcesRead(w http.ResponseWriter, req *Request) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": "resources/read params are invalid", "source": "internal"})
		return
	}

	taskID, err := tasks.ParseResourceURI(params.URI)
	if err != nil {
		s.writeError(w, req.ID, -32602, "invalid params", map[string]any{"details": err.Error(), "source": "internal"})
		return
	}

	task, err := s.tasks.Get(taskID)
	if err != nil {
		s.writeError(w, req.ID, -32001, "google api error", map[string]any{"details": err.Error(), "source": "google"})
		return
	}

	logging.Info("reading resource", "request_id", rawIDForLog(req.ID), "uri", params.URI)
	b, err := json.Marshal(task)
	if err != nil {
		s.writeError(w, req.ID, -32000, "internal server error", map[string]any{"details": err.Error(), "source": "internal"})
		return
	}

	s.writeResult(w, req.ID, map[string]any{
		"contents": []map[string]any{
			{
				"uri":      task.URI,
				"mimeType": "application/json",
				"text":     string(b),
			},
		},
	})
}

func resolveTaskID(id, uri string) (string, error) {
	if strings.TrimSpace(id) != "" {
		return id, nil
	}
	if strings.TrimSpace(uri) != "" {
		return tasks.ParseResourceURI(uri)
	}
	return "", fmt.Errorf("id or uri is required")
}

func toolSuccessResult(message string, payload map[string]any) callToolResult {
	return callToolResult{
		Content: []map[string]any{
			{
				"type": "text",
				"text": message,
			},
		},
		StructuredResult: payload,
	}
}

func toolErrorResult(message string, err error) callToolResult {
	return callToolResult{
		Content: []map[string]any{
			{
				"type": "text",
				"text": fmt.Sprintf("%s: %v", message, err),
			},
		},
		IsError: true,
	}
}

func (s *Server) writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	logging.Debug("sending rpc result", "request_id", rawIDForLog(id))
	s.writeJSON(w, newResultResponse(id, result))
}

func (s *Server) writeError(w http.ResponseWriter, id json.RawMessage, code int, message string, data map[string]any) {
	logging.Warn("sending rpc error", "request_id", rawIDForLog(id), "code", code, "message", message)
	s.writeJSON(w, newErrorResponse(id, code, message, data))
}

func (s *Server) writeJSONValue(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) writeJSON(w http.ResponseWriter, resp Response) {
	s.writeJSONValue(w, resp)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	logging.Debug("serving manifest", "remote_addr", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":  "Google Tasks MCP",
		"tools": ToolSchemas(),
	})
}

func rawIDForLog(id json.RawMessage) string {
	if len(bytes.TrimSpace(id)) == 0 {
		return "<notification>"
	}
	return string(id)
}
