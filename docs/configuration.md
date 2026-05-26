# Configuration Reference

Every environment variable and default. This is the canonical list — the
README shows only required and common optional vars, with a link here.

## Required

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_BASE_URL` | *(required)* | OpenGrok API base URL ending in `/api/v1` |
| `OPENGROK_MCP_DEFAULT_PROJECT` | *(required)* | Project used when tool calls omit `project`. Optional only when `OPENGROK_MCP_PROJECTS` contains exactly one project |

## Transport And Surface

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_TRANSPORT` | `stdio` | Set to `http` to enable Streamable HTTP mode |
| `OPENGROK_MCP_LISTEN` | `127.0.0.1:8765` | HTTP listen address |
| `OPENGROK_MCP_TOOL_SURFACE` | `full` | `full` (fine-grained tools), `compact` (wrapper tools), or `gateway` (experimental) |
| `OPENGROK_MCP_MEMORY_ENABLED` | `true` | Process-scoped memory tools. Disabled over HTTP regardless of this setting |

## Authentication And URLs

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_API_TOKEN` | *(none)* | Sends `Authorization: Bearer <token>` |
| `OPENGROK_MCP_BASIC_AUTH_TOKEN` | *(none)* | Sends `Authorization: Basic <token>`. Only one of API_TOKEN or BASIC_AUTH_TOKEN may be set |
| `OPENGROK_MCP_WEB_BASE_URL` | derived from `BASE_URL` | OpenGrok web UI base URL, used for citations and raw file fallback. Derived from `OPENGROK_MCP_BASE_URL` by stripping `/api/v1` |
| `OPENGROK_MCP_PROJECTS` | *(none)* | Comma-separated known OpenGrok projects |
| `OPENGROK_MCP_PROJECT_REQUIRED` | `true` | Require `project` parameter in tool calls |
| `OPENGROK_MCP_PROBE_FILE` | *(none)* | Optional `project/path/to/file` probe for file-read capability verification |

## Logging And Debug

| Variable | Default | Description |
|---|---|---|
| `DEBUG` | `false` | Set to `1` to log OpenGrok API and web requests to stderr |
| `OPENGROK_MCP_LOG_LEVEL` | `info` | Reserved logging level setting |

## Security

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_INSECURE_SKIP_TLS_VERIFY` | `false` | Disable TLS certificate verification. Use only against internal instances with broken certificates |
| `OPENGROK_MCP_CURSOR_SECRET` | *(none)* | HMAC secret for cursor signing. Set in shared or remote deployments |

## Context Expansion

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_AUTO_EXPAND_CONTEXT` | `true` | Set to `false` to disable automatic context expansion |
| `OPENGROK_MCP_CONTEXT_BEFORE` | `5` | Lines before a match to include |
| `OPENGROK_MCP_CONTEXT_AFTER` | `10` | Lines after a match to include |
| `OPENGROK_MCP_MAX_EXPANDED_RESULTS` | `10` | Max search results to expand context for |
| `OPENGROK_MCP_MAX_EXPANDED_FILES` | `5` | Max unique files to fetch context from |
| `OPENGROK_MCP_CONTEXT_FETCH_CONCURRENCY` | `3` | Concurrent file fetches during context expansion |

## Retry

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_RETRY_MAX_ATTEMPTS` | `2` | Max retry attempts for transient OpenGrok errors (transport failures, HTTP 429, HTTP 5xx) |
| `OPENGROK_MCP_RETRY_BASE_DELAY` | `200ms` | Base delay for exponential backoff between retries |

## Cache

| Variable | Default | Description |
|---|---|---|
| `OPENGROK_MCP_CACHE_ENABLED` | `false` | Enable in-process response cache |
| `OPENGROK_MCP_CACHE_TTL` | `5m` | Cache entry lifetime |
| `OPENGROK_MCP_CACHE_MAX_SIZE` | `1000` | Maximum cache entries |

## Context Budgets

Each tier (`MINIMAL`, `DEFAULT`, `MAXIMAL`) has four overridable values.
The table shows defaults for each tier.

| Tier | `_BEFORE` | `_AFTER` | `_RESULTS` | `_FILES` |
|---|---|---|---|---|
| `MINIMAL` | 2 | 3 | 3 | 2 |
| `DEFAULT` | 5 | 10 | 10 | 5 |
| `MAXIMAL` | 15 | 30 | 25 | 10 |

Override examples:

```
OPENGROK_MCP_BUDGET_MINIMAL_BEFORE=1
OPENGROK_MCP_BUDGET_DEFAULT_RESULTS=15
OPENGROK_MCP_BUDGET_MAXIMAL_AFTER=50
```
