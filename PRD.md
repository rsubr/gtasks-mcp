# Product Requirements Document (PRD)

## Product Name

Google Tasks MCP Server (Go)

---

## 1. Overview

### Objective

Build a Model Context Protocol (MCP) server in Go that integrates with Google Tasks and exposes task operations via:

* HTTP + Server-Sent Events (SSE)
* Strict JSON-RPC 2.0 compliance

The server enables AI agents (Claude, ChatGPT, etc.) to:

* List, search, read, create, update, delete, and clear tasks
* Access tasks as MCP resources

---

## 2. Core Requirements

### Protocol

* Transport: HTTP + SSE
* Protocol: JSON-RPC 2.0 (strict compliance)
* Must interoperate with Claude Code, ChatGPT, and other MCP clients

### Language & Dependencies

* Language: Go
* Use Go standard library wherever possible
* Use official Google SDK modules as needed for Google Tasks integration:

  * google.golang.org/api/tasks/v1
  * google.golang.org/api/option
  * other official `google.golang.org/api/...` support modules when required by the SDK
* Allowed external dependency for OAuth flow:

  * golang.org/x/oauth2

---

## 3. Architecture

### High-Level Components

* HTTP Server (net/http)
* SSE Stream Handler
* JSON-RPC Engine
* OAuth2 Authentication Manager
* Google Tasks Service Wrapper
* Tool Handlers
* Resource Handlers
* Logger

---

## 4. Authentication Design

### Flow

1. Load OAuth credentials from `gcp-oauth.keys.json`
2. Load token from file (CLI arg)
3. If token missing:

   * Generate auth URL
   * Print URL to stdout
   * Accept authorization code via stdin
4. Exchange code for token
5. Persist token to file

### Scope

[https://www.googleapis.com/auth/tasks](https://www.googleapis.com/auth/tasks)

### Token Storage

* Path provided via CLI:
  --token-file
* Stored as JSON

---

## 5. Task List Strategy

### Configuration

* Task list name provided via:

  * ENV: TASKLIST_NAME
  * or CLI: --tasklist

### Behavior

* On startup:

  1. Fetch all task lists
  2. Find matching name
  3. If not found, create it
  4. Store taskListId

* All operations are restricted to this task list

---

## 6. MCP API

### Endpoint

* POST /rpc

### SSE Endpoint

* GET /events

### MCP JSON-RPC Methods

The server must implement standard MCP JSON-RPC methods rather than
inventing one JSON-RPC method per tool.

Required methods:

* `initialize`
* `tools/list`
* `tools/call`
* `resources/list`
* `resources/read`

Example initialize request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {
      "name": "example-client",
      "version": "1.0.0"
    }
  }
}
```

Example tool call request:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search",
    "arguments": {
      "query": "inbox"
    }
  }
}
```

---

## 7. Tools

### search

* Fetch all tasks
* Perform local filtering (title + notes)

Input:
{
"query": "string"
}

---

### list

* Return all tasks
* No pagination required

---

### create

Input:
{
"title": "string",
"notes": "string (optional)",
"due": "string (optional)"
}

---

### update

Input:
{
"id": "string",
"uri": "string",
"title": "string (optional)",
"notes": "string (optional)",
"status": "needsAction|completed",
"due": "string (optional)"
}

---

### delete

Input:
{
"id": "string",
"uri": "string"
}

Behavior:

* Deletes from the configured task list only
* Does not accept `taskListId` because the server is already scoped to one configured list

---

### clear

Input:
{
}

Behavior:

* Clears completed tasks from the configured task list only
* Does not accept `taskListId` because the server is already scoped to one configured list

---

## 8. Resource Model

### URI

* gtasks:///<task_id>

### Behavior

* Maps to tasks.get
* Returns full task details

---

## 9. Data Model

```go
type Task struct {
    ID      string
    Title   string
    Notes   string
    Status  string
    Due     string
    Updated string
}
```

---

## 10. Error Handling (AI-Agent Compatible)

### Strategy

All errors MUST strictly follow JSON-RPC 2.0 specification.

### Format

```
{
  "jsonrpc": "2.0",
  "id": <request_id>,
  "error": {
    "code": <int>,
    "message": <string>,
    "data": {
      "details": "optional",
      "source": "google|internal"
    }
  }
}
```

### Rules

* NEVER return raw HTTP errors
* ALWAYS wrap errors in JSON-RPC format
* ALWAYS include request id
* Errors must be deterministic and machine-readable

### Error Codes

-32600 → Invalid Request
-32601 → Method Not Found
-32602 → Invalid Params
-32000 → Internal Server Error
-32001 → Google API Error

### Google API Mapping

* 404 → -32001 ("Task not found")
* 401 → -32001 ("Unauthorized")
* 403 → -32001 ("Forbidden")
* 429 → -32001 ("Rate limited")

### Debug Behavior

* Controlled via LOG_LEVEL=debug
* Adds extra fields in error.data:

  * raw_response
  * request_payload

---

## 11. Logging

### Levels

* debug
* info
* warn
* error

### Behavior

* debug: full request/response tracing
* info: normal operations
* warn: recoverable issues
* error: failures

---

## 12. Go Project Structure

/cmd/server/main.go

/internal/
auth/
mcp/
tasks/
logging/

---

## 13. Docker Deployment

### Base Image

* gcr.io/distroless/static:nonroot

### Build

* Multi-stage build
* CGO disabled

---

## 14. Non-Functional Requirements

### Performance

* Target <100ms per request (excluding Google API latency)

### Reliability

* Must gracefully handle Google API failures

### Security

* Tokens stored securely in file
* No secrets in logs

---

## 15. Acceptance Criteria

* Works with Claude MCP
* Works with ChatGPT MCP
* OAuth flow works without browser
* All tools functional
* JSON-RPC compliant
* Dockerized
