# gtasks-mcp

`gtasks-mcp` is an MCP server for Google Tasks. It exposes a small set of task-management tools over a single MCP HTTP endpoint so an MCP client can list, read, search, create, update, delete, and clear tasks in a configured Google Tasks list.

The server is built in Go and uses OAuth against the Google Tasks API. It also exposes tasks as MCP resources using `gtasks:///TASK_ID` URIs.

## What It Does

- Connects to Google Tasks using a configurable OAuth credentials JSON file
- Persists the OAuth access token in `token.json` by default
- Targets one Google Tasks list, creating it if needed
- Exposes MCP tools for task CRUD operations
- Exposes MCP resources for listing and reading individual tasks
- Broadcasts task change notifications over MCP SSE streams

## Exposed MCP Tools

The server currently exposes these tools:

- `list`: list all tasks in the configured Google Tasks list
- `read`: read a task by task ID or resource URI
- `search`: search tasks by query string
- `create`: create a task with optional notes, due date, and recurrence
- `update`: update a task by ID or resource URI
- `delete`: delete a task by ID or resource URI
- `clear`: clear completed tasks from the configured list

## Resources

Tasks are also exposed as MCP resources.

- Resource URI scheme: `gtasks:///TASK_ID`
- `resources/list`: returns all tasks as resource entries
- `resources/read`: returns a task resource as JSON

## Transport and Endpoints

The server listens over HTTP and exposes:

- `/mcp`: the single MCP endpoint
- `/manifest`: lightweight manifest describing the server and tool schemas

The `/mcp` endpoint supports:

- `POST /mcp`: JSON-RPC requests
- `GET /mcp`: Server-Sent Events stream for MCP notifications
- `DELETE /mcp`: terminate an MCP session

The old split endpoints `/rpc` and `/events` are no longer used.

## Authentication and Setup

The server expects a Google OAuth client credentials JSON file. On first run, if the token file does not exist, it starts an interactive OAuth flow and writes the resulting token to disk.

Files used by the server:

- `gcp-oauth.keys.json`: Google OAuth client credentials
- `token.json`: stored OAuth token by default

Credential file resolution order:

1. `-credentials-file`
2. `GOOGLE_OAUTH_CREDENTIALS_FILE`
3. `/auth/gcp-oauth.keys.json`
4. `./gcp-oauth.keys.json`

To prepare Google API access:

1. Create a Google Cloud project and enable the Google Tasks API.
2. Create an OAuth client credential JSON file.
3. Save that file as `gcp-oauth.keys.json`.
4. Place it either at `/auth/gcp-oauth.keys.json` for container use or `./gcp-oauth.keys.json` for local repo-root use, or pass an explicit path.
5. Start the server and complete the interactive OAuth flow when prompted.

## Configuration

Startup options are available via flags, with environment-variable fallbacks for most values.

Flags:

- `-credentials-file`: OAuth client credentials JSON file path
- `-token-file`: OAuth token file path, default `token.json`
- `-tasklist`: Google Tasks list name
- `-log-level`: `debug`, `info`, `warn`, or `error`
- `-listen-addr`: full listen address such as `0.0.0.0:8080`
- `-port`: port number, default `8080`

Environment variables:

- `GOOGLE_OAUTH_CREDENTIALS_FILE`: default OAuth client credentials file path when `-credentials-file` is not passed
- `TASKLIST_NAME`: default task list name when `-tasklist` is not passed
- `LOG_LEVEL`: default log level when `-log-level` is not passed
- `LISTEN_ADDR`: default listen address when `-listen-addr` is not passed
- `PORT`: default port when `-port` is not passed

Defaults:

- task list: `My Tasks`
- log level: `info`
- port: `8080`

## Build Instructions

Build with the provided script:

```bash
./build.sh
```

That script runs `go mod tidy` and builds static Linux binaries for:

```bash
dist/gtasks-mcp-linux-amd64
dist/gtasks-mcp-linux-arm64
```

To build manually:

```bash
go build -o gtasks-mcp ./cmd/server
```

## Running the Server

Run the compiled binary:

```bash
./gtasks-mcp -tasklist "My Tasks" -port 8080
```

Or run directly with Go:

```bash
go run ./cmd/server -tasklist "My Tasks" -port 8080
```

Example using environment variables:

```bash
GOOGLE_OAUTH_CREDENTIALS_FILE="/auth/gcp-oauth.keys.json" TASKLIST_NAME="My Tasks" LOG_LEVEL=debug PORT=8080 ./gtasks-mcp
```

## Example MCP Usage

Once the server is running, MCP clients should connect to the `/mcp` endpoint.

Typical MCP transport flow:

1. `POST /mcp` with `initialize`
2. Read `MCP-Session-Id` from the response headers
3. Reuse that session ID on subsequent `POST /mcp` requests
4. Optionally open `GET /mcp` with the same `MCP-Session-Id` to receive SSE notifications
5. `DELETE /mcp` with that session ID to close the session when done

Example tool calls:

- create a task: `create { "title": "Pay rent", "due": "2026-03-25T00:00:00.000Z" }`
- search tasks: `search { "query": "rent" }`
- read a task by URI: `read { "uri": "gtasks:///TASK_ID" }`
- mark complete: `update { "id": "TASK_ID", "status": "completed" }`

`update.status` accepts only `needsAction` or `completed`.

## Notes

- The server manages one configured Google Tasks list at a time.
- If the named task list does not exist, the service creates it.
- Recurrence metadata is supported through the task service and exposed on tasks as `recurrence`.
