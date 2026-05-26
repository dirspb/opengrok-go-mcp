# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.24 or NEEDS CLARIFICATION

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk/mcp`, OpenGrok HTTP API, or NEEDS CLARIFICATION

**Storage**: In-memory process state only, configuration via environment variables, or N/A

**Testing**: `go test ./...`; targeted package tests before full-suite verification

**Target Platform**: Local stdio MCP server and loopback HTTP transport, or NEEDS CLARIFICATION

**Project Type**: Go CLI/MCP server

**Performance Goals**: Agent-friendly latency and bounded response sizes; specify page sizes, caps, and concurrency where changed

**Constraints**: Preserve MCP schema compatibility, capability gating, citation URLs, warning semantics, and secure environment-based auth

**Scale/Scope**: OpenGrok projects and result sets covered by the affected tools; include pagination and truncation behavior

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **MCP Contract**: Identify every changed tool, resource, schema field,
  warning, cursor, citation, capability gate, and tool-surface variant.
- **OpenGrok Semantics**: State whether behavior is full-text, path,
  definition, reference, heuristic, page-local, truncated, or fallback-based.
- **Test Evidence**: List the tests that fail against old behavior or
  otherwise prove the new behavior, plus the targeted verification command for
  each behavioral slice. For non-trivial behavior changes, state whether tasks
  are ordered test-first or why another sequence is clearer.
- **Agent UX Validation**: Define a realistic first-use task for a fresh
  lightweight or mid-tier subagent where available, or fresh-session simulation
  otherwise. Keep upfront context minimal so the report can reveal whether tool
  names, descriptions, schemas, warnings, defaults, and examples work for
  first-use discovery.
- **Security**: Confirm secrets stay in environment variables, HTTP remains
  loopback-first, and risky options are explicit and documented.
- **Compatibility and Docs**: Note any public behavior or default changes,
  migration impact, and required README or `docs/` updates.
- **Experimental Surface**: Identify any experimental tools, operations,
  config names, and docs labels, or state "None".
- **Resource Bounds**: Define explicit limits, defaults, and warnings for any
  feature that can increase response size, tool-call count, or automatic file
  fetching.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
cmd/opengrok-go-mcp/
internal/cache/
internal/config/
internal/cursor/
internal/links/
internal/mcpserver/
internal/opengrok/
docs/
README.md
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
