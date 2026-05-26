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
        "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@v0.3.0-beta.2"
      ],
      "enabled": true,
      "environment": {
        "OPENGROK_MCP_BASE_URL": "https://grok.example.com/source/api/v1",
        "OPENGROK_MCP_DEFAULT_PROJECT": "platform",
        "OPENGROK_MCP_BASIC_AUTH_TOKEN": "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
      }
    }
  }
}
```

> **Developing on a branch?** Replace `@v0.3.0-beta.2` with the path to a local
> clone: `["go", "run", "/path/to/opengrok-go-mcp/cmd/opengrok-go-mcp"]`.

For Basic auth use only the base64 token value, without the `Basic ` prefix. Set exactly one of
`OPENGROK_MCP_API_TOKEN` or `OPENGROK_MCP_BASIC_AUTH_TOKEN`.

`OPENGROK_MCP_WEB_BASE_URL` may be omitted when `OPENGROK_MCP_BASE_URL` ends in
`/api/v1`; the server derives it by trimming that suffix.

## Claude Code

Add to `~/.claude.json` under `mcpServers`, or run `claude mcp add`:

```json
{
  "mcpServers": {
    "opengrok": {
      "command": "go",
      "args": ["run", "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@v0.3.0-beta.2"],
      "env": {
        "OPENGROK_MCP_BASE_URL": "https://grok.example.com/source/api/v1",
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
command = ["go", "run", "github.com/rokasklive/opengrok-go-mcp/cmd/opengrok-go-mcp@v0.3.0-beta.2"]

[mcp_servers.env]
OPENGROK_MCP_BASE_URL = "https://grok.example.com/source/api/v1"
OPENGROK_MCP_DEFAULT_PROJECT = "platform"
OPENGROK_MCP_BASIC_AUTH_TOKEN = "Ik5ldmVyIGdvbm5hIGdpdmUgeW91IHVwIjoiTmV2ZXIgZ29ubmEgbGV0IHlvdSBkb3duIg=="
```

`OPENGROK_MCP_PROJECTS` is optional but recommended when `/projects/indexed` is
not accessible. If it contains exactly one project, `OPENGROK_MCP_DEFAULT_PROJECT`
may be omitted and the server uses that project as the default. When multiple
projects are configured, set `OPENGROK_MCP_DEFAULT_PROJECT`. Explicit project
arguments must match the configured list.

## HTTP Mode

OpenCode should use local command mode. For manual HTTP use:

```bash
OPENGROK_MCP_TRANSPORT=http \
OPENGROK_MCP_BASE_URL=https://grok.example.com/source/api/v1 \
OPENGROK_MCP_DEFAULT_PROJECT=platform \
go run ./cmd/opengrok-go-mcp
```

The HTTP MCP endpoint is available at:

```text
http://127.0.0.1:8765/mcp
```

## Environment

Required:

- `OPENGROK_MCP_BASE_URL`: OpenGrok API base URL ending in `/api/v1`.
- `OPENGROK_MCP_DEFAULT_PROJECT`: project used when tool calls omit `project`. Optional only when `OPENGROK_MCP_PROJECTS` contains exactly one project.

Common optional settings:

- `OPENGROK_MCP_API_TOKEN`: sends `Authorization: Bearer <token>`.
- `OPENGROK_MCP_BASIC_AUTH_TOKEN`: sends `Authorization: Basic <token>`.
- `OPENGROK_MCP_WEB_BASE_URL`: OpenGrok web UI base URL, used for citations and raw file fallback. Derived from `OPENGROK_MCP_BASE_URL` when omitted and the API URL ends in `/api/v1`.
- `OPENGROK_MCP_PROJECTS`: comma-separated known OpenGrok projects.
- `DEBUG=1`: log OpenGrok API and web requests to stderr.
- `OPENGROK_MCP_TRANSPORT=http`: enable Streamable HTTP mode.
- `OPENGROK_MCP_LISTEN`: HTTP listen address, default `127.0.0.1:8765`.
- `OPENGROK_MCP_TOOL_SURFACE`: tool registration surface, default `full`.
  Set to `compact` to expose fewer wrapper tools (e.g. `opengrok_search`, `opengrok_read`).
  Set to `gateway` to expose only `opengrok_discover` and `opengrok_call`
  (experimental).
- `OPENGROK_MCP_MEMORY_ENABLED`: default `true`. Set to `false` to disable
  process-scoped memory tools. Memory tools are never exposed over HTTP.

Less common:

- `OPENGROK_MCP_PROJECT_REQUIRED`: default `true`.
- `OPENGROK_MCP_PROBE_FILE`: optional `project/path/to/file` probe for file-read capability.
- `OPENGROK_MCP_LOG_LEVEL`: reserved logging level setting.
- `OPENGROK_MCP_INSECURE_SKIP_TLS_VERIFY=true`: disable TLS certificate verification. Use only against internal OpenGrok instances with invalid or mismatched certificates (e.g. expired corporate certs). Never use against public or untrusted hosts.
- `OPENGROK_MCP_AUTO_EXPAND_CONTEXT`: default `true`. Set to `false` to disable automatic context expansion in search results.
- `OPENGROK_MCP_CONTEXT_BEFORE`: default `5`. Lines before a match to include in auto-expanded context.
- `OPENGROK_MCP_CONTEXT_AFTER`: default `10`. Lines after a match to include in auto-expanded context.
- `OPENGROK_MCP_MAX_EXPANDED_RESULTS`: default `10`. Maximum number of search results to expand context for.
- `OPENGROK_MCP_MAX_EXPANDED_FILES`: default `5`. Maximum number of unique files to fetch context from.
- `OPENGROK_MCP_CONTEXT_FETCH_CONCURRENCY`: default `3`. Number of concurrent file fetches during context expansion.
- `OPENGROK_MCP_RETRY_MAX_ATTEMPTS`: default `2`. Maximum retry attempts for transient OpenGrok errors (transport failures, HTTP 429, HTTP 5xx).
- `OPENGROK_MCP_RETRY_BASE_DELAY`: default `200ms`. Base delay for exponential backoff between retries.

## Tools

At startup, the server probes OpenGrok and exposes only working tools:

- `search_code` — full-text, path, history, definition, or reference search. Returns up to the configured page size per call; pass `next_cursor` for subsequent pages. `total_hits` is always present. When `total_hits > 500`, a `warning` field advises narrowing the query. Accepts `expand_context` (bool, default `true`) to auto-fetch surrounding lines for each result. Each result includes a `kind` field containing the ctags kind (`class`, `function`, `method`, `interface`, etc.) when OpenGrok returns it.
- `search_symbol_definitions` — search for symbol definitions. Accepts `expand_context` (bool, default `true`) to auto-fetch surrounding lines for each result.
- `search_symbol_references` — search for symbol references. Accepts `expand_context` (bool, default `true`) to auto-fetch surrounding lines for each result.
- `list_symbols` — list symbol definitions filtered by ctags kind (`class`, `interface`, `function`, `method`, etc.) and optionally scoped to a path prefix. Designed for architect-oriented structural queries: "what classes exist in this package?", "find all interfaces under `src/api/`". Returns lean `SymbolItem` results; use `read_file` or `get_file_context` to drill in. Set `include_snippets=false` for broad sweeps to reduce token cost. When `total_hits > 100`, a `warning` field includes a remaining-call estimate. Enabled automatically when `search_symbol_definitions` is available.
- `read_file` — read full file content. Returns up to 500 lines per call; `truncated` and `next_cursor` indicate more content, `total_lines` is always returned.
- `get_file_context` — read a line window around a specific `line_number` from search results.
- `list_projects` — list indexed projects, paginated at 50 per page; `total_projects` is always returned.
- `list_files` — list files within a project path, paginated. Returns `FileItem` entries with `project`, `path`, `name`, `is_directory`, `num_lines`, `loc`, `size`, `description`, and `resource_uri`. Includes `next_cursor` for pagination. If the OpenGrok listing exceeds the safety cap, `truncated=true` and `warning` state that totals and remaining pages are incomplete. Gated by `ListFiles` capability (probed at startup).
- `get_project_overview` — return project metadata including total files, total directories, and top-level file and directory listings. Returns `truncated=true` with a warning when derived from a capped file listing. Gated by `ListFiles` capability.
- `search_implementations` — search for class and interface implementations by delegating to symbol-reference search (`mode=reference`). Results are best-effort candidate references, not exhaustive. Accepts `expand_context`. Gated by `SearchSymbolReferences` capability.
- `search_cross_project_references` — search for symbol references across all configured projects. Returns grouped results by project with `attribution_uncertain` warnings. Gated by `SearchSymbolReferences` capability.
- `search_and_read` and `find_symbol_and_references` — compound operations that return file content; exposed only when their search capabilities and `GetFileContext` are enabled.
- `memory_set`, `memory_get`, `memory_list`, `memory_delete`, `memory_clear` — process-scoped investigation memory; exposed only for stdio servers with the `Memory` capability enabled. These tools are not registered for HTTP transport because memory is not isolated by client session.

With `OPENGROK_MCP_TOOL_SURFACE=compact`, the server exposes
fewer wrapper tools instead of the fine-grained tools listed above, only when
their backing capabilities are enabled:

- `opengrok_projects` — list indexed projects.
- `opengrok_search` — dispatch `operation=code`, `operation=definitions`, or
  `operation=references` with a `payload` matching the corresponding full tool
  input.
- `opengrok_symbols` — dispatch `operation=list`, `operation=implementations`, or
  `operation=cross_project_references` with a `payload` matching the
  corresponding full tool input. `implementations` and `cross_project_references`
  are gated by `SearchSymbolReferences` capability.
- `opengrok_read` — dispatch `operation=file` or `operation=context` with a
  `payload` matching `read_file`/`get_file_context` input.
- `opengrok_compound` — dispatch compound read operations only when
  `GetFileContext` and the relevant search capabilities are available.
- `opengrok_memory` — process-scoped memory, available only for stdio servers
  with the `Memory` capability enabled.

In compact mode the fine-grained tools, such as `search_code`, are not
registered. Resources remain available in both full and compact modes when their
backing capabilities are enabled.

### Gateway mode (experimental)

`OPENGROK_MCP_TOOL_SURFACE=gateway` exposes only two tools:

- `opengrok_discover` — returns the list of enabled operations and their
  descriptions. Use this to learn what the server can do.
- `opengrok_call` — dispatch any operation by name with a JSON payload. The
  operation name must match an entry from `opengrok_discover`.

Gateway mode is useful when the agent benefits from a single-call discovery
contract instead of a large static tool list. It is experimental and may change
in future releases. Gateway operations use the same capability rules as the full
and compact surfaces; process-scoped memory operations are not registered over
HTTP.

File reads try `/api/v1/file/content` first, then fall back to authenticated
`/raw/{project}/{path}` under `OPENGROK_MCP_WEB_BASE_URL`.

Search and file outputs include `citation.url`. Agents should include it when
answering about a specific class or file.

## Resources

Resources are exposed only when the matching capability is enabled:

- `opengrok://projects`
- `opengrok://project/{project}`
- `opengrok://project/{project}/files/{+path}`

## Recommended Workflows

### Structural queries (subclasses, implementers, call graphs)

OpenGrok indexes full text plus ctags **definitions** — it knows where a class or
method is *defined*, but not relationships like `extends`, `implements`, or call
edges. For structural questions, combine two tools:

1. Use `search_symbol_definitions` or `list_symbols` (with `path_prefix` / `kind`)
   to find the relevant package(s) and narrow scope — OpenGrok answers *"where is it?"* well.
2. Run a local AST tool (e.g. [ast-grep](https://ast-grep.github.io)) scoped to those
   paths for precise structural matching — *"what are the relationships?"*

A full-text search for `extends BaseController` returns every textual hit (fields,
parameters, comments), not just subclasses. The two-step workflow above replaces the
"search → truncate → grep → repeat" loop.

### Reading pagination

Every paginated tool (`search_*`, `list_symbols`, `list_files`) returns top-level
pagination fields:

- `has_more` — `true` if more pages exist; fetch them by passing `next_cursor`.
- `page` / `total_pages` — your position, e.g. page 1 of 3.
- `total_hits` — the global, unfiltered match count from OpenGrok.

For `list_symbols` with a `kind` filter, `total_hits` counts definitions *before*
the kind filter (OpenGrok cannot filter by ctags kind server-side). A `warning`
makes this explicit; narrow with `path_prefix` to enumerate a kind fully.

## Recommended Configuration

Choose the setup that matches your usage:

- **Full mode (default).** The server exposes every fine-grained tool
  individually (`search_code`, `read_file`, `search_symbol_definitions`, …),
  each with detailed, per-argument descriptions. This is where the richest
  guidance reaches the agent. No additional config needed.

- **Local stdio + compact mode.** Set `OPENGROK_MCP_TOOL_SURFACE=compact` to
  expose fewer, higher-level wrapper tools (`opengrok_search`, `opengrok_read`,
  `opengrok_symbols`, `opengrok_projects`). A cleaner tool list for agents that
  prefer dispatching through a single wrapper; note that per-argument schemas
  are not surfaced for the nested `payload` in this mode.

- **HTTP mode (controlled local/internal setups only).** Set
  `OPENGROK_MCP_TRANSPORT=http`. The server binds to `127.0.0.1:8765` by
  default. Do not expose it externally without authentication and network
  controls. Prefer local stdio when possible; HTTP is useful for agents that
  only support remote MCP endpoints, or when running the server as a sidecar
  process.

## Security

Avoid passing secrets as CLI flags. Use environment variables for OpenGrok auth tokens.

## Development Workflow

This project uses GitHub Spec Kit for non-trivial feature planning.

For meaningful behavior changes, new MCP tools, schema changes, configuration
changes, or changes that affect agent-facing behavior, contributors should
start from the project constitution:

- `.specify/memory/constitution.md`

Feature work should generally produce:

- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/tasks.md`

Small bug fixes, documentation edits, dependency bumps, and mechanical
refactors do not require a full Spec Kit workflow unless they affect the public
MCP contract.

All changes must preserve the MCP contract, OpenGrok semantics, security
posture, compatibility expectations, and documentation requirements described
in the constitution.

## Known Limitations

- Large project traversal is bounded, and some search and discovery
  operations are best-effort rather than language-semantic.
- HTTP transport is intended for controlled local or internal setups and does
  not add inbound client authentication.

See [docs/limitations.md](docs/limitations.md) for the detailed current list,
behavioral impact, and mitigations.

## License

`opengrok-go-mcp` is licensed under the Apache License 2.0 (`Apache-2.0`) for
new releases starting with `v0.3.0-beta.2`.

Earlier published releases up to and including `v0.3.0-beta.1` were released
under `CC0-1.0` and remain available under those terms.
