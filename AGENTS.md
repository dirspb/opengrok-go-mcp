# Agent Guide

## Project Purpose

`opengrok-go-mcp` exposes OpenGrok through MCP so agents can search indexed
code, read source files, follow symbols, and answer with source citations.

## Repository Map

- `cmd/opengrok-go-mcp/`: server entrypoint and startup capability probing
- `internal/config/`: environment, flags, defaults, validation
- `internal/mcpserver/`: MCP tools, schemas, pagination, warnings, memory
- `internal/opengrok/`: OpenGrok API and web/raw-file client
- `docs/`: limitations, agent usage patterns, design notes
- `.specify/`: Spec Kit constitution, templates, scripts, extension hooks

## Core MCP Contract Rules

- Treat tool names, input fields, output fields, warnings, pagination, cursor
  semantics, citations, resources, and environment variables as public contract.
- Write tool names, descriptions, schemas, warnings, defaults, and examples for
  a cold agent seeing the server for the first time.
- Keep behavior consistent across full, compact, and gateway surfaces unless a
  spec explicitly says otherwise.
- Preserve `citation.url` for source-backed answers.
- New or changed response-size, tool-call, or auto-fetch behavior needs limits,
  defaults, and warnings.
- For the detailed field-by-field contract (inputs, outputs, errors, warnings,
  pagination, citations, truncation, capability gates, compatibility,
  experimental fields), see `docs/tool-contracts.md`.

## OpenGrok Semantics And Limitations

- OpenGrok is full-text search plus ctags definitions, not an AST/call-graph
  engine.
- `search_implementations`, cross-project attribution, kind filtering, sorting,
  and project overview data can be best-effort, heuristic, or page-local.
- Surface uncertainty with warnings; do not claim exhaustive semantic knowledge
  unless the implementation proves it.
- For structural certainty, use OpenGrok to find candidate files and then verify
  with language-aware tooling.

## Tool Surfaces

- Default: `full` surface with fine-grained tools such as `search_code`,
  `read_file`, `get_file_context`, `list_symbols`, and symbol search tools.
- `compact` surface groups operations behind `opengrok_search`,
  `opengrok_symbols`, `opengrok_read`, and related wrappers.
- `gateway` surface is experimental; use `opengrok_discover` before
  `opengrok_call`.
- Tools are capability-gated at startup. If a tool is missing, assume the server
  could not verify the backing OpenGrok capability.

## Compatibility Rules

- Keep existing public defaults stable unless a spec and migration note justify
  a change.
- Experimental features must be labeled in tool descriptions, docs, and config
  names.
- Do not silently alter stable tool behavior from an experimental path.

## Security Rules

- Secrets belong in environment variables, not CLI flags or logs.
- HTTP mode has no built-in inbound client auth; keep loopback/default trusted
  network assumptions unless a spec changes that.
- `OPENGROK_MCP_INSECURE_SKIP_TLS_VERIFY` is only for controlled internal hosts.
- Memory is process-scoped, ephemeral, and disabled over HTTP.

## Spec Kit Workflow

Use GitHub Spec Kit for meaningful behavior changes: new MCP tools, schema
changes, config/env changes, transport/security changes, OpenGrok query
behavior, compatibility changes, or anything affecting agent reliability/token
usage.

Start from `.specify/memory/constitution.md`. Feature work generally produces:

- `specs/<feature>/spec.md`
- `specs/<feature>/plan.md`
- `specs/<feature>/tasks.md`

Small docs fixes, typo fixes, formatting-only changes, dependency metadata
updates, and trivial test-only cleanups may skip the full workflow.

## Testing Commands

- Format Go changes: `gofmt -w <files>`
- Targeted package test: `go test ./internal/<package>/`
- Full verification: `go test ./...`
- Documentation whitespace check: `git diff --check`
- For non-trivial agent-facing changes, dispatch a fresh lightweight or mid-tier
  subagent where available, or use a fresh-session simulation otherwise. Give it
  a realistic task and minimal context, then capture first-use findings on
  descriptions, schemas, warnings, defaults, and examples.

## Documentation Update Rules

- Update `README.md` for human-facing setup, configuration, or behavior changes.
- Update `docs/limitations.md` when behavior is best-effort, heuristic,
  truncated, page-local, or security-sensitive.
- Update `docs/agent-usage-patterns.md` when agent workflow guidance changes.
- Keep examples concise and avoid duplicating long tool schemas in multiple
  places.

## Common Mistakes To Avoid

- Do not infer an OpenGrok project from the local repository name.
- Do not paginate broad result sets blindly; narrow with path, kind, file type,
  or query.
- Do not claim implementation/call-graph certainty from text matches alone.
- Do not bypass warnings in responses.
- Do not introduce new public fields without tests and docs.
- Do not assume future agents know project context that is absent from tool
  descriptions or schemas.

## Review Checklist

- MCP contract impact identified and tested
- Fresh-subagent usability findings captured for agent-facing changes
- OpenGrok semantic limits documented
- Pagination, cursor, warning, citation behavior preserved
- Security and transport assumptions unchanged or specified
- Compatibility/default changes justified
- README/docs updated where user-facing behavior changed
- `go test ./...` or justified narrower verification run
