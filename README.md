# gtasks-mcp

An MCP server for Google Tasks. Connects AI assistants to a personal Google Tasks account via OAuth, exposing tools to list, create, update, search, and delete tasks.

Docker image: `rsubr/gtasks-mcp:latest`

## Prerequisites

1. Create a Google Cloud project and enable the **Google Tasks API**.
2. Create an **OAuth 2.0 client credential** (Desktop app type) and download the JSON file.
3. Save it as `gcp-oauth.keys.json` and place it in `./auth/`.

## Generating token.json (first time only)

The OAuth flow requires interactive input, so the token must be created before running the server as a daemon.

Run the container interactively:

```bash
docker run -it --rm \
  -v ./auth:/auth \
  rsubr/gtasks-mcp:latest
```

Or with docker compose:

```bash
docker compose run --rm gtasks-mcp
```

The server prints an authorization URL:

```
Open URL: https://accounts.google.com/o/oauth2/auth?...
Paste the full redirect URL or just the authorization code:
```

1. Open the URL in your browser and sign in.
2. After approving, Google redirects to a `localhost` URL that fails to load — that is expected.
3. Copy the **full redirect URL** from your browser's address bar and paste it into the terminal.

The server saves `token.json` to `./auth/` and starts. Press `Ctrl+C` to stop. All subsequent runs use the saved token automatically.

## Running with Docker Compose

Create `./auth/` with your credentials and token, then:

```bash
docker compose up -d
```

The included `docker-compose.yaml` mounts `./auth` into the container and exposes port `8080`.

## Configuration

| Flag | Env var | Default | Description |
|---|---|---|---|
| `-credentials-file` | `GOOGLE_OAUTH_CREDENTIALS_FILE` | `/auth/gcp-oauth.keys.json` | OAuth credentials JSON |
| `-token-file` | — | `token.json` | OAuth token file |
| `-tasklist` | `TASKLIST_NAME` | `My Tasks` | Google Tasks list name |
| `-log-level` | `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `-listen-addr` | `LISTEN_ADDR` | `:8080` | Listen address |
| `-port` | `PORT` | `8080` | Port (ignored if `-listen-addr` is set) |

## Integrating with AI Agents

The MCP endpoint is `https://gtasks-mcp.rsubr.in/mcp` (or your own hosted address).

### Claude Code

```bash
claude mcp add --transport http gtasks https://gtasks-mcp.rsubr.in/mcp
```

Or in `.mcp.json` / `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "gtasks": {
      "type": "http",
      "url": "https://gtasks-mcp.rsubr.in/mcp"
    }
  }
}
```

### Codex

```bash
codex mcp add gtasks --url https://gtasks-mcp.rsubr.in/mcp
```

Or in `~/.codex/config.toml`:

```toml
[mcp_servers.gtasks]
url = "https://gtasks-mcp.rsubr.in/mcp"
```

### Gemini CLI

```bash
gemini mcp add --transport http gtasks https://gtasks-mcp.rsubr.in/mcp
```

Or in `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "gtasks": {
      "httpUrl": "https://gtasks-mcp.rsubr.in/mcp",
      "timeout": 30000
    }
  }
}
```

### Verify

```bash
claude mcp list
codex mcp list
gemini mcp list
```
