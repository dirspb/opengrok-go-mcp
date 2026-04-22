# opengrok-go-mcp

MCP server for searching and reading code on OpenGrok.

## OpenCode

Add this to `opencode.json`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "opengrok": {
      "type": "local",
      "command": [
        "go",
        "run",
        "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@latest"
      ],
      "enabled": true,
      "environment": {
        "OPENGROK_MCP_BASE_URL": "https://grok.example.com/source/api/v1",
        "OPENGROK_MCP_WEB_BASE_URL": "https://grok.example.com/source",
        "OPENGROK_MCP_PROJECTS": "platform",
        "OPENGROK_MCP_DEFAULT_PROJECT": "platform",
        "OPENGROK_MCP_BASIC_AUTH_TOKEN": "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
      }
    }
  }
}
```

For Basic auth use only the base64 token value, without the `Basic ` prefix. Set exactly one of
`OPENGROK_MCP_API_TOKEN` or `OPENGROK_MCP_BASIC_AUTH_TOKEN`.

## Claude Code

Add to `~/.claude.json` under `mcpServers`, or run `claude mcp add`:

```json
{
  "mcpServers": {
    "opengrok": {
      "command": "go",
      "args": ["run", "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@latest"],
      "env": {
        "OPENGROK_MCP_BASE_URL": "https://grok.example.com/source/api/v1",
        "OPENGROK_MCP_WEB_BASE_URL": "https://grok.example.com/source",
        "OPENGROK_MCP_DEFAULT_PROJECT": "platform",
        "OPENGROK_MCP_BASIC_AUTH_TOKEN": "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
      }
    }
  }
}
```

## Codex

Add to `.codex` in the project root or `~/.codex/config.toml` globally:

```toml
[[mcp_servers]]
name = "opengrok"
command = ["go", "run", "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@latest"]

[mcp_servers.env]
OPENGROK_MCP_BASE_URL = "https://grok.example.com/source/api/v1"
OPENGROK_MCP_WEB_BASE_URL = "https://grok.example.com/source"
OPENGROK_MCP_DEFAULT_PROJECT = "platform"
OPENGROK_MCP_BASIC_AUTH_TOKEN = "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
```

`OPENGROK_MCP_PROJECTS` is optional but recommended when `/projects/indexed` is
not accessible. When configured, explicit project arguments must match this
list. Agents should normally omit `project` and let the server use
`OPENGROK_MCP_DEFAULT_PROJECT`.

## HTTP Mode

OpenCode should use local command mode. For manual HTTP use:

```bash
OPENGROK_MCP_TRANSPORT=http \
OPENGROK_MCP_BASE_URL=https://grok.example.com/source/api/v1 \
OPENGROK_MCP_WEB_BASE_URL=https://grok.example.com/source \
go run ./cmd/opengrok-go-mcp
```

The HTTP MCP endpoint is available at:

```text
http://127.0.0.1:8765/mcp
```

## Environment

Required:

- `OPENGROK_MCP_BASE_URL`: OpenGrok API base URL ending in `/api/v1`.
- `OPENGROK_MCP_WEB_BASE_URL`: OpenGrok web UI base URL, used for citations and raw file fallback.
- `OPENGROK_MCP_DEFAULT_PROJECT`: project used when tool calls omit `project`.

Common optional settings:

- `OPENGROK_MCP_API_TOKEN`: sends `Authorization: Bearer <token>`.
- `OPENGROK_MCP_BASIC_AUTH_TOKEN`: sends `Authorization: Basic <token>`.
- `OPENGROK_MCP_PROJECTS`: comma-separated known OpenGrok projects.
- `DEBUG=1`: log OpenGrok API and web requests to stderr.
- `OPENGROK_MCP_TRANSPORT=http`: enable Streamable HTTP mode.
- `OPENGROK_MCP_LISTEN`: HTTP listen address, default `127.0.0.1:8765`.

Less common:

- `OPENGROK_MCP_PROJECT_REQUIRED`: default `true`.
- `OPENGROK_MCP_PROBE_FILE`: optional `project/path/to/file` probe for file-read capability.
- `OPENGROK_MCP_LOG_LEVEL`: reserved logging level setting.

## Tools

At startup, the server probes OpenGrok and exposes only working tools:

- `search_code`
- `search_symbol_definitions`
- `search_symbol_references`
- `read_file`
- `get_file_context` (same as `read_file` with optional line window)
- `list_projects`

File reads try `/api/v1/file/content` first, then fall back to authenticated
`/raw/{project}/{path}` under `OPENGROK_MCP_WEB_BASE_URL`.

Search and file outputs include `citation.url`. Agents should include it when
answering about a specific class or file.

## Resources

Resources are exposed only when the matching capability is enabled:

- `opengrok://projects`
- `opengrok://project/{project}`
- `opengrok://project/{project}/files/{+path}`

## Security

`opengrok-go-mcp` binds to `127.0.0.1` by default in HTTP mode. Do not expose it
externally without authentication and network controls.

Avoid passing secrets as CLI flags. Use environment variables for OpenGrok auth tokens.

## Known Limitations

- **Multi-project result attribution is heuristic.** When searching across multiple
  projects, result paths are matched against the queried project names by prefix.
  If OpenGrok returns a path that doesn't match any queried project, the server falls
  back to the default project. This can misattribute hits when project names overlap
  or when OpenGrok returns unexpected path formats.

- **`list_projects` pagination is not implemented.** The `cursor` field in
  `list_projects` input is accepted but ignored. All projects are returned in a single
  response. This is fine for typical OpenGrok instances but will need addressing for
  deployments with very large project counts.

- **No retry or backoff on transient OpenGrok errors.** A flaky upstream will surface
  errors directly to the agent on every failed call. Consider fronting the server with
  a reverse proxy that handles retries if your OpenGrok instance is unstable.

- **MCP Go SDK is pre-1.0.** Breaking changes may occur on SDK upgrades. The pinned
  version is noted in `go.mod`; review release notes before upgrading.
