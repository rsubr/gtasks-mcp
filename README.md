# gtasks-mcp

`gtasks-mcp` is an MCP server for Google Tasks. It exposes a small set of task-management tools over HTTP JSON-RPC and Server-Sent Events so an MCP client can list, read, search, create, update, delete, and clear tasks in a configured Google Tasks list.

The server is built in Go and uses OAuth against the Google Tasks API. It also exposes tasks as MCP resources using `gtasks:///TASK_ID` URIs.

## What It Does

- Connects to Google Tasks using OAuth credentials from `gcp-oauth.keys.json`
- Persists the OAuth access token in `token.json` by default
- Targets one Google Tasks list, creating it if needed
- Exposes MCP tools for task CRUD operations
- Exposes MCP resources for listing and reading individual tasks
- Broadcasts simple task change events over SSE

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

The server listens over HTTP and registers three endpoints:

- `/rpc`: MCP JSON-RPC endpoint
- `/events`: SSE endpoint for simple task change notifications
- `/manifest`: lightweight manifest describing the server and tool schemas

## Authentication and Setup

The server expects a Google OAuth client credentials file named `gcp-oauth.keys.json` in the project root. On first run, if the token file does not exist, it starts an interactive OAuth flow and writes the resulting token to disk.

Files used by the server:

- `gcp-oauth.keys.json`: Google OAuth client credentials
- `token.json`: stored OAuth token by default

To prepare Google API access:

1. Create a Google Cloud project and enable the Google Tasks API.
2. Create an OAuth client credential JSON file.
3. Save that file as `gcp-oauth.keys.json` in the repository root.
4. Start the server and complete the interactive OAuth flow when prompted.

## Configuration

Startup options are available via flags, with environment-variable fallbacks for most values.

Flags:

- `-token-file`: OAuth token file path, default `token.json`
- `-tasklist`: Google Tasks list name
- `-log-level`: `debug`, `info`, `warn`, or `error`
- `-listen-addr`: full listen address such as `0.0.0.0:8080`
- `-port`: port number, default `8080`

Environment variables:

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

That script runs:

```bash
go mod tidy
go build -o gtasks-mcp -ldflags="-s -w" ./cmd/server
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
TASKLIST_NAME="My Tasks" LOG_LEVEL=debug PORT=8080 ./gtasks-mcp
```

## Example MCP Usage

Once the server is running, MCP clients should initialize against the `/rpc` endpoint.

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
