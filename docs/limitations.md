# Known Limitations

This document describes current operational limitations of `opengrok-go-mcp`.

## Search And Discovery

- **Multi-project result attribution is heuristic.** OpenGrok search results
  provide paths rather than a distinct project field. The server attributes
  results using the longest matching requested project prefix; when none
  matches, it marks the hit with `attribution_uncertain=true` and emits a
  warning. Verify flagged results before relying on citations or links.

- **Project traversal is capped at 5,000 entries.** The OpenGrok `/list`
  response is capped before `list_files` pagination and
  `get_project_overview` aggregation. These operations report
  `truncated=true` and a warning, but cannot page beyond that cap. Narrow the
  requested `path` for large projects.

- **`list_symbols` kind filtering is page-local.** OpenGrok supplies a page of
  definition matches and the MCP filters that page by ctags `kind`.
  `filtered_total_hits` describes the returned page, not a complete
  kind-filtered inventory. Continue with `next_cursor` or narrow
  `path_prefix`.

- **`search_implementations` is best-effort.** OpenGrok does not expose
  language-semantic implementation relationships. This operation returns
  candidate symbol-reference matches, not guaranteed implementations.

- **Search sorting is page-local.** `sort=path` sorts only results fetched for
  the current page. `sort=date` preserves OpenGrok order and returns a
  warning; there is no global date-sorted or path-sorted result set.

## Capability And Response Boundaries

- **File-read capability detection can be optimistic.** Without
  `OPENGROK_MCP_PROBE_FILE`, a configured web fallback URL is sufficient to
  expose file-read and compound-read operations. If access is unavailable,
  failure appears on the first real read. Configure a readable
  `project/path/to/file` probe to verify access during startup.

- **OpenGrok response bodies are limited to 32 MiB.** API and raw fallback
  responses exceeding this bound fail explicitly. Narrow searches or
  directory listings, and avoid requesting very large files through the raw
  fallback.

## State And Transport

- **Memory is process-scoped and ephemeral.** Memory tools use one in-process
  store for the lifetime of a stdio server. Data is not durable or separated
  by client session, and memory tools are intentionally not exposed in HTTP
  mode.

- **HTTP mode has no inbound client authentication.** The MCP HTTP handler
  relies on its loopback default bind address or external authentication and
  network controls. Do not expose it directly to untrusted networks.

- **Stdio mode does not install signal-aware graceful shutdown.** HTTP mode
  handles `SIGINT` and `SIGTERM` through a shutdown context; stdio mode runs
  the MCP transport using a background context. Clients should close stdio
  transport streams when ending a session.
