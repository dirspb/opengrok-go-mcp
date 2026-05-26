## 1. Goals

This MCP should do four things well:

* let the agent search code with **project context first**
* expose **clean pagination** instead of raw OpenGrok offsets
* return **clickable URLs** to OpenGrok views for human convenience
* keep names and schemas predictable so an agent does not get confused

OpenGrok already supports project-aware search and offset-based pagination, and its UI includes an xref view for rendered source browsing. ([GitHub][1])

---

## 2. Design stance

**Blunt recommendation:** treat OpenGrok as the **backend index**, and make the MCP the **curation layer**.

Do **not** mirror OpenGrok 1:1.
Do **not** expose raw Lucene-ish knobs unless there is a strong reason.
Do **not** force the agent to remember whether OpenGrok wants `projects=foo&projects=bar` or `start/maxresults`.

Instead:

* use a **small number of tools**
* use **project-scoped defaults**
* normalize pagination to **cursor**
* always return **web links**
* return both **display snippets** and **stable identifiers**

That fits MCP better: tools for actions/search and resources for stable, readable context. MCP resources natively support `cursor` and `nextCursor`, so it is reasonable to mirror that style in tool outputs too. ([Model Context Protocol][2])

---

## 3. Naming convention

Use **lower_snake_case** for tool names and response field names.

Examples:

* `search_code`
* `get_file_context`
* `list_projects`
* `resolve_symbol`
* `project`
* `file_path`
* `next_cursor`
* `display_url`

Why:

* readable
* stable across languages
* easy for agents to predict
* avoids mixed casing drift between OpenGrok and your MCP

For internal mappings to OpenGrok, keep them hidden:

* `query.full_text` → OpenGrok `full`
* `query.definition` → OpenGrok `def`
* `query.reference` → OpenGrok `ref` or refs-style mapping depending on version handling
* `pagination.offset` → OpenGrok `start`
* `pagination.limit` → OpenGrok `maxresults`

OpenGrok documents field-style parameters such as `full`, `def`, `path`, `hist`, `projects`, `maxresults`, and `start`, and its Java constants show dedicated params for full search, definitions, refs, path, project, and “all projects” behavior. ([GitHub][1])

---

## 4. Server capabilities

### Tools

Use tools for search/navigation operations:

* `list_projects`
* `search_code`
* `search_symbol_definitions`
* `search_symbol_references`
* `get_file_context`
* `get_project_overview`

### Resources

Use resources for durable context surfaces:

* `opengrok://projects`
* `opengrok://project/{project}`
* `opengrok://project/{project}/files/{path}`
* `opengrok://project/{project}/search-help`

This lines up with MCP’s distinction: tools are model-invoked actions; resources are readable context identified by URIs. ([Model Context Protocol][2])

---

## 5. Project-first behavior

This is the important part you asked for.

Every search tool should support:

* `project` — a single project
* `projects` — optional array for multi-project search
* `default_project` — server/session default
* `allow_all_projects` — explicit escape hatch

### Rule

If the caller provides no project:

1. use `default_project` if configured
2. otherwise fail softly with a message like:
   `"No project selected. Call list_projects first or pass project explicitly."`

### Why

Agents perform much better when the search space is smaller. OpenGrok itself supports explicit project selection and also has an “all projects” search parameter for large instances. ([GitHub][1])

### Suggested precedence

```text
explicit project(s) > session default_project > server default_project > error
```

### Optional session helper

You may also expose:

* `set_default_project`

But I would only add it if your client preserves session state well. Otherwise keep project explicit.

---

## 6. Tool spec

## `list_projects`

Lists indexed projects the agent can target.

### Input

```json
{
  "cursor": "optional"
}
```

### Output

```json
{
  "projects": [
    {
      "project": "platform",
      "title": "platform",
      "description": "Indexed OpenGrok project",
      "project_url": "https://grok.example.com/source/search?project=platform",
      "resource_uri": "opengrok://project/platform"
    }
  ],
  "next_cursor": null
}
```

### Backing OpenGrok call

* `GET /api/v1/projects`
* optionally `GET /api/v1/projects/indexed`

OpenGrok documents `/projects` and `/projects/indexed` for listing projects. ([GitHub][1])

---

## `search_code`

Primary full-text/path/history search.

### Input

```json
{
  "project": "platform",
  "query": "simulateDepletion",
  "mode": "full_text",
  "path_prefix": "src/services/",
  "file_type": "swift",
  "page_size": 20,
  "cursor": "optional",
  "include_links": true
}
```

### Attributes

* `project`: preferred single project
* `query`: human search string
* `mode`: enum

  * `full_text`
  * `definition`
  * `reference`
  * `path`
  * `history`
* `path_prefix`: directory restriction
* `file_type`: file-type restriction
* `page_size`: requested page size
* `cursor`: opaque pagination cursor
* `include_links`: whether to generate browser URLs

### Output

```json
{
  "project": "platform",
  "mode": "full_text",
  "query": "simulateDepletion",
  "total_hits": 137,
  "results": [
    {
      "result_id": "platform:src/services/depletion/Engine.swift:42",
      "project": "platform",
      "file_path": "src/services/depletion/Engine.swift",
      "line_number": 42,
      "snippet": "func simulateDepletion(days: Int) -> SimulationResult {",
      "display_title": "Engine.swift:42",
      "display_url": "https://grok.example.com/source/xref/platform/src/services/depletion/Engine.swift#42",
      "raw_url": "https://grok.example.com/source/raw/platform/src/services/depletion/Engine.swift",
      "resource_uri": "opengrok://project/platform/files/src/services/depletion/Engine.swift#line=42"
    }
  ],
  "page_size": 20,
  "next_cursor": "opaque",
  "diagnostics": {
    "offset_used": 0,
    "opengrok_start": 0,
    "opengrok_maxresults": 20
  }
}
```

### Mapping to OpenGrok

* `mode=full_text` → `?full=...`
* `mode=definition` → `?def=...`
* `mode=path` → `?path=...`
* `mode=history` → `?hist=...`
* `project` → `?projects=platform`
* `page_size` → `?maxresults=20`
* cursor-decoded offset → `?start=0`

OpenGrok documents those search parameters directly, including `projects`, `maxresults`, and `start`. ([GitHub][1])

### Notes

* Keep `mode` singular and obvious.
* Do not expose raw OpenGrok field names to the model.
* Return `total_hits` from OpenGrok `resultCount`/hit count equivalent.
* Always return `display_url` when possible.

---

## `search_symbol_definitions`

Dedicated definition search, separate from generic search to reduce ambiguity.

### Input

```json
{
  "project": "platform",
  "symbol": "simulateDepletion",
  "page_size": 20,
  "cursor": "optional"
}
```

### Output

```json
{
  "project": "platform",
  "symbol": "simulateDepletion",
  "results": [
    {
      "symbol": "simulateDepletion",
      "file_path": "src/services/depletion/Engine.swift",
      "line_number": 42,
      "kind": "definition",
      "snippet": "func simulateDepletion(days: Int) -> SimulationResult {",
      "display_url": "https://grok.example.com/source/xref/platform/src/services/depletion/Engine.swift#42"
    }
  ],
  "next_cursor": null
}
```

### Why separate tool

Agents are better with purpose-built tools than one giant Swiss-army schema.

OpenGrok’s API explicitly supports definition-oriented search fields, and its query parameter constants distinguish definitions and references. ([GitHub][1])

---

## `search_symbol_references`

Same idea, but reference-focused.

### Input

```json
{
  "project": "platform",
  "symbol": "simulateDepletion",
  "page_size": 20,
  "cursor": "optional"
}
```

### Output

```json
{
  "project": "platform",
  "symbol": "simulateDepletion",
  "results": [
    {
      "symbol": "simulateDepletion",
      "file_path": "src/viewmodels/InventoryViewModel.swift",
      "line_number": 118,
      "kind": "reference",
      "snippet": "let result = engine.simulateDepletion(days: 7)",
      "display_url": "https://grok.example.com/source/xref/platform/src/viewmodels/InventoryViewModel.swift#118"
    }
  ],
  "next_cursor": "opaque"
}
```

---

## `get_file_context`

Returns file-oriented context for a path, optionally around a line.

### Input

```json
{
  "project": "platform",
  "file_path": "src/services/depletion/Engine.swift",
  "line_number": 42,
  "before": 30,
  "after": 60,
  "include_annotations": true,
  "include_links": true
}
```

### Output

```json
{
  "project": "platform",
  "file_path": "src/services/depletion/Engine.swift",
  "line_number": 42,
  "start_line": 12,
  "end_line": 102,
  "content": "…",
  "display_url": "https://grok.example.com/source/xref/platform/src/services/depletion/Engine.swift#42",
  "raw_url": "https://grok.example.com/source/raw/platform/src/services/depletion/Engine.swift",
  "annotations_available": true,
  "resource_uri": "opengrok://project/platform/files/src/services/depletion/Engine.swift#line=42"
}
```

### Backing source

Use xref-oriented retrieval where possible. OpenGrok’s UI explicitly has an xref view, and its query parameters include annotation-related controls. ([GitHub][3])

### Recommendation

Even if the raw OpenGrok API is awkward here, normalize it in MCP so the agent always gets:

* file
* line
* surrounding text
* one clickable URL

---

## `get_project_overview`

Gives the model a head start.

### Input

```json
{
  "project": "platform"
}
```

### Output

```json
{
  "project": "platform",
  "project_url": "https://grok.example.com/source/search?project=platform",
  "resource_uri": "opengrok://project/platform",
  "notes": [
    "Searches default to this project unless overridden."
  ],
  "hints": {
    "preferred_search_scope": "project_only",
    "search_examples": [
      {
        "label": "Find InventoryViewModel",
        "tool_call": {
          "tool": "search_code",
          "arguments": {
            "project": "platform",
            "query": "InventoryViewModel",
            "mode": "path"
          }
        }
      }
    ]
  }
}
```

### Purpose

This is not from OpenGrok directly. It is an MCP convenience layer that tells the agent:

* what project it is in
* how searches should default
* how to stay scoped

That directly addresses your “agent comes in with project in mind” requirement.

---

## 7. Pagination spec

OpenGrok search is offset-based via `start` and `maxresults`. MCP resources support cursor-based pagination using `cursor` and `nextCursor`. Your MCP should expose **cursor pagination** and hide offsets. ([GitHub][1])

### Public MCP contract

All list/search tools return:

* `page_size`
* `next_cursor`
* optional `total_hits`

All list/search inputs accept:

* `cursor`
* `page_size`

### Cursor shape

Opaque to the model, but internally it can encode:

```json
{
  "project": "platform",
  "query": "simulateDepletion",
  "mode": "full_text",
  "offset": 20,
  "page_size": 20,
  "path_prefix": "src/services/"
}
```

Base64url-encode it and sign it if you want tamper resistance.

### Rules

* if `cursor` is present, ignore fresh paging params except safety checks
* cursor must be tied to the original query context
* return `next_cursor = null` when exhausted
* cap `page_size`, for example at 100

### Why not expose `start`

Because it leaks OpenGrok internals and invites the model to make brittle assumptions.

---

## 8. Clickable links spec

This is worth doing.

Every result that points to a file or search should include:

* `display_url` — human-friendly browser URL
* `raw_url` — direct/raw file URL if available
* `title` or `display_title`
* `anchor_line` where applicable

### Minimum per hit

```json
{
  "display_title": "Engine.swift:42",
  "display_url": "https://grok.example.com/source/xref/platform/src/services/depletion/Engine.swift#42"
}
```

### Link conventions

Use:

* search page links for search summaries
* xref page links for file/line navigation
* raw page links for plain file content

OpenGrok’s UI documentation confirms xref as a core rendered source view, and public issue examples show commonly used `/xref/...` and `/raw/...` URL patterns in deployed instances. The latter is practical evidence rather than a formal API guarantee, so I would document these as **configurable URL templates**, not hardcoded protocol truth. ([GitHub][3])

### Recommended URL template config

```json
{
  "web_base_url": "https://grok.example.com/source",
  "url_templates": {
    "search": "{web_base_url}/search?project={project}&full={query}",
    "xref": "{web_base_url}/xref/{project}/{file_path}#{line_number}",
    "raw": "{web_base_url}/raw/{project}/{file_path}"
  }
}
```

### Important

Do not derive links from API paths. Treat browser URLs and API URLs as separate config.

---

## 9. Resource URI convention

Use predictable custom URIs.

### Resource URIs

* `opengrok://projects`
* `opengrok://project/{project}`
* `opengrok://project/{project}/files/{path}`
* `opengrok://project/{project}/files/{path}#L{line}`
* `opengrok://project/{project}/search-help`

### Why

MCP resources are URI-addressed by design. ([Model Context Protocol][2])

### Suggested resource payloads

`opengrok://project/{project}`
contains:

* project name
* search defaults
* repository types if available
* example searches
* top-level path hints if you can precompute them

`opengrok://project/{project}/files/{path}`
contains:

* file content
* mime type
* optional last indexed metadata
* display_url/raw_url

---

## 10. Result object schema

Use one normalized hit schema across search tools.

```json
{
  "result_id": "string",
  "project": "string",
  "file_path": "string",
  "line_number": 0,
  "column_number": null,
  "kind": "full_text | definition | reference | path | history",
  "symbol": "string | null",
  "snippet": "string",
  "display_title": "string",
  "display_url": "string",
  "raw_url": "string | null",
  "resource_uri": "string",
  "score": null,
  "metadata": {}
}
```

### Rules

* `result_id` should be stable enough for dedupe
* `file_path` should be project-relative
* `kind` should come from MCP semantics, not OpenGrok naming quirks
* `metadata` is for source-specific extras, not core fields

---

## 11. Error model

Use explicit, boring errors.

### Examples

#### Missing project

```json
{
  "error": {
    "code": "PROJECT_REQUIRED",
    "message": "No project selected. Pass project or call list_projects first."
  }
}
```

#### Invalid cursor

```json
{
  "error": {
    "code": "INVALID_CURSOR",
    "message": "Cursor is invalid or does not match the current query."
  }
}
```

#### OpenGrok upstream unavailable

```json
{
  "error": {
    "code": "UPSTREAM_UNAVAILABLE",
    "message": "OpenGrok API did not respond."
  }
}
```

#### No results

Return success with empty `results`, not an error.

---

## 12. Search-mode mapping table

This is the main implementation bridge.

| MCP mode     | OpenGrok input                                                  |
| ------------ | --------------------------------------------------------------- |
| `full_text`  | `full`                                                          |
| `definition` | `def`                                                           |
| `reference`  | refs-style mapping / query-param constant-backed implementation |
| `path`       | `path`                                                          |
| `history`    | `hist`                                                          |

OpenGrok’s documented REST search params include `full`, `def`, `path`, `hist`, and project selection, while its Java query constants also expose dedicated full, defs, refs, path, and project search params. ([GitHub][1])

---

## 13. Suggested defaults

These defaults will make agents noticeably less stupid:

* `project_required = true`
* `page_size_default = 20`
* `page_size_max = 100`
* `include_links_default = true`
* `mode_default = full_text`
* `path_prefix_default = null`
* `allow_all_projects_default = false`

And for ranking:

* prefer same-project hits
* prefer exact path hits when `mode=path`
* prefer exact symbol hits for symbol tools

---

## 14. Minimal implementation config

```json
{
  "opengrok_api_base_url": "https://grok.example.com/source/api/v1",
  "opengrok_web_base_url": "https://grok.example.com/source",
  "default_project": "platform",
  "project_required": true,
  "page_size_default": 20,
  "page_size_max": 100,
  "include_links_default": true,
  "enable_raw_links": true,
  "url_templates": {
    "xref": "{web_base_url}/xref/{project}/{file_path}#{line_number}",
    "raw": "{web_base_url}/raw/{project}/{file_path}",
    "search": "{web_base_url}/search?project={project}&full={query}"
  }
}
```

---

## 15. Recommended first version

If you want this to stay lean, ship only:

* `list_projects`
* `search_code`
* `search_symbol_definitions`
* `search_symbol_references`
* `get_file_context`

And resources:

* `opengrok://projects`
* `opengrok://project/{project}`
* `opengrok://project/{project}/files/{path}`

That is enough to make the MCP useful without turning it into a protocol zoo.

---

## 17. Transport and Local Execution (Addendum)

### 17.1 Transport Choice

The MCP server must use HTTP transport, not stdio.

Rationale:
- better compatibility with office tooling such as OpenCode
- easier local process management
- simpler debugging and inspection
- avoids stdio transport quirks in editor/agent integrations

The implementation should use the official MCP Go SDK:
- modelcontextprotocol/go-sdk

The transport target is Streamable HTTP.

### 17.2 Local-Only by Default

The server must run as a local service by default.

Default bind:
- 127.0.0.1
Default port:
- 8765

Example:
- http://127.0.0.1:8765/mcp

The server must not listen on external interfaces unless explicitly configured.

### 17.3 Command-Line Distribution Model

The project should be distributed as a normal Go command.

Command name:
- opengrok-go-mcp

Recommended command path:
- github.com/your-org/opengrok-go-mcp/cmd/opengrok-go-mcp

One-off execution:
- go run github.com/your-org/opengrok-go-mcp/cmd/opengrok-go-mcp@latest

Installed local command:
- go install github.com/your-org/opengrok-go-mcp/cmd/opengrok-go-mcp@latest

After install:
- opengrok-go-mcp

This is the Go equivalent of a lightweight “npx-style” developer experience, while still producing a normal native binary.

### 17.4 CLI Requirements

The executable should support:

- --listen
  Example: 127.0.0.1:8765

- --base-url
  OpenGrok API base URL

- --web-base-url
  OpenGrok web UI base URL for clickable links

- --default-project
  Optional default project scope

- --project-required
  Default: true

- --read-timeout
- --write-timeout
- --log-level

Environment variable equivalents are allowed.

### 17.5 Dependency Policy

Implementation must use:
- Go standard library
- modelcontextprotocol/go-sdk

Strongly preferred:
- net/http for HTTP serving and client calls
- encoding/json for JSON serialization

Avoid additional third-party dependencies unless there is a strong, documented justification.

### 17.6 Operational Model

The service is a thin local adapter:

Agent / OpenCode
    -> HTTP MCP endpoint on localhost
    -> OpenGrok MCP server
    -> OpenGrok REST API

No database, no background workers, no external message bus.

### 17.7 Security Posture

Default mode must be local-only.

If remote exposure is ever enabled later, it must be an explicit opt-in and treated as a separate deployment mode with its own authentication and network controls.

### 17.8 Recommended Project Layout

/opengrok-mcp
  /cmd/opengrok-mcp
  /internal/mcpserver
  /internal/opengrok
  /internal/config
  /internal/links

The cmd package should only bootstrap configuration and server startup.
Core logic should remain in internal packages.
