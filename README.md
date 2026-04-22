# opengrok-go-mcp

MCP server for searching and reading OpenGrok code from OpenCode.

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

For Basic auth, use only the base64 token value:

```jsonc
"OPENGROK_MCP_BASIC_AUTH_TOKEN": "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
```

Do not include the `Basic ` prefix. Set exactly one of
`OPENGROK_MCP_API_TOKEN` or `OPENGROK_MCP_BASIC_AUTH_TOKEN`.

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

Common optional settings:

- `OPENGROK_MCP_API_TOKEN`: sends `Authorization: Bearer <token>`.
- `OPENGROK_MCP_BASIC_AUTH_TOKEN`: sends `Authorization: Basic <token>`.
- `OPENGROK_MCP_PROJECTS`: comma-separated known OpenGrok projects.
- `OPENGROK_MCP_DEFAULT_PROJECT`: project used when tool calls omit `project`.
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
- `get_file_context`
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
