# ROADMAP

<!--
AI agent note: This roadmap uses a structured format with explicit metadata blocks per item.
Each entry has an ID, priority, status, and a clear decision-record section.
When implementing, check the `status` field first — only items marked `planned`
or `approved` are actionable for implementation.
-->

---

## Item: 001 — Mandatory Cursor Signing in HTTP Mode

- **Status:** planned
- **Priority:** medium
- **Effort:** small
- **Dependencies:** none
- **Labels:** security, http, cursor

### Problem

Cursor signing via `OPENGROK_MCP_CURSOR_SECRET` is optional and disabled by
default. The server logs `"WARNING: cursor signing disabled; set
OPENGROK_MCP_CURSOR_SECRET for integrity"` at startup when no secret is
configured.

For local stdio transport this is acceptable — the cursor value never leaves the
process boundary, and tampering requires access to the same machine/process.

In HTTP mode, cursors are serialized into JSON-RPC response payloads and
transmitted over the network. An unsigned cursor is a base64-encoded JSON blob
that any caller with access to the HTTP endpoint can decode, modify (e.g. bump
`offset` to skip results, alter `query` to mismatch state), and re-encode. The
server does call `Validate()` to check that the decoded query context matches
the expected context for the tool call, which catches many tampering attempts.
However, validation is not comprehensive — a determined caller could craft
cursor state that bypasses the checks, or simply reuse a cursor from a
different tool invocation in ways that confuse pagination.

### Proposed Solution

Two options, to be decided:

**Option A (recommended) — auto-generate a process-local secret.**

When `OPENGROK_MCP_CURSOR_SECRET` is not set **and** the transport is HTTP,
generate a random 32-byte HMAC key at process startup and assign it to
`cursor.Secret`. This makes cursors tamper-proof by default in HTTP mode
without any user configuration. The existing `signPayload` / `verifyPayload`
pair works unchanged — they just get a real key instead of the empty string.
The startup log line changes to not emit a warning in this case.

**Option B — strongly document the requirement.**

Keep the current behavior but add a prominent warning in the README and HTTP
mode docs that `OPENGROK_MCP_CURSOR_SECRET` should always be set when using
HTTP transport. Accept the risk that users may skip it.

### Decision Record

- **2026-05-23:** Added as planned item after review of cursor integrity
  posture. Option A preferred but needs implementation work.

### Implementation Notes

- Affected file: `cmd/opengrok-go-mcp/main.go` — conditionally generate secret
  when `cfg.CursorSecret == "" && transport == http`.
- Affected file: `internal/cursor/cursor.go` — no changes needed; already
  supports empty-secret graceful fallback via `signPayload`/`verifyPayload`.
- Affected test: `internal/cursor/cursor_test.go` — verify that a
  process-local secret produces signed cursors that `verifyPayload` accepts
  and that unsigned cursors are rejected by `verifyPayload` when a secret is
  set.
- Use `crypto/rand` for key generation, not `math/rand`.
- The secret must be set once before any cursor is encoded or decoded. A
  `sync.Once` guard in `cursor.go` is optional but defensive.

---

## Template (for future items)

Copy-paste this block for new roadmap entries:

```markdown
---

## Item: XXX — Short Title

- **Status:** planned | approved | in-progress | completed | deferred | rejected
- **Priority:** low | medium | high | critical
- **Effort:** tiny | small | medium | large | xlarge
- **Dependencies:** none | item-XXX | item-XXX
- **Labels:** comma-separated tags

### Problem

Describe the issue or opportunity in 2–4 sentences. What doesn't work well or
what new capability is needed?

### Proposed Solution

1–3 approaches with trade-offs. If there is a clear recommendation, mark it.

### Decision Record

- **YYYY-MM-DD:** Added / updated / marked completed. One-liner per entry.

### Implementation Notes

- Bullet list of affected files, modules, test changes, or architectural
  concerns that an implementing agent should know.
```
