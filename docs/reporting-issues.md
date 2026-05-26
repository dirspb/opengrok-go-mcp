# Reporting Issues

`opengrok-go-mcp` is pre-1.0. Some tools, schemas, warnings, configuration options, and transport behavior may be incomplete, unstable, or subject to change before a stable release.

Please create an issue when something is confusing, unreliable, incorrect, or
hard for an AI agent to use. Reports from both humans and agents are useful.

You can file through the GitHub issue forms (**Bug Report** or **Agent
Confusion Report**), which mirror the structure below. This page is the prose
guide those forms link to.

## What To Report

- a tool is missing, unavailable, or exposed when it does not work
- a tool description or schema led an agent to the wrong call
- pagination, cursor, warning, citation, or response shape is confusing
- OpenGrok search behavior differs from the tool description
- errors are unclear or not actionable
- HTTP, stdio, authentication assumptions, TLS verification, raw-file fallback, or memory behavior is surprising
- token usage, automatic file fetching, or response size is larger than expected
- a cold agent could not discover the right workflow from the tool surface
- a tool result lacks enough context for an agent to decide whether to paginate, narrow the query, or verify locally

## Before You File

- Remove secrets, tokens, internal hostnames, and private source content.
- Keep enough structure to reproduce the issue: tool name, inputs, mode, and
  relevant response fields.
- If reporting an agent experience, include what the agent saw, what it tried,
  and where it became confused.

## Issue Template

Copy the template below into a new GitHub issue and fill in the details that
apply. Leave sections blank only when they are genuinely not relevant.

```markdown
## Summary

What went wrong, or what was confusing?

## Reporter

- [ ] Human user
- [ ] AI agent
- [ ] Human reporting an AI agent experience

## Environment

- opengrok-go-mcp version or commit:
- MCP client:
- Transport: stdio / HTTP
- Tool surface: full / compact / gateway
- OpenGrok version or deployment notes, if known:
- Relevant configuration, with secrets removed:

## Tool Or Workflow

- Tool or operation:
- Input payload, with secrets and private code removed:
- Expected behavior:
- Actual behavior:

## Agent UX Notes

If an AI agent was involved:

- What task was the agent trying to complete?
- What tool description/schema/warning did it rely on?
- What call did it make first?
- What confused it or caused the wrong behavior?
- What wording, field, warning, or example would have helped?

## Evidence

Paste sanitized logs, warnings, response snippets, citations, or screenshots.

## Impact

- [ ] Blocks use
- [ ] Produces wrong answer
- [ ] Causes excessive tool calls or token use
- [ ] Confusing but has a workaround
- [ ] Documentation issue

## Possible Fix

Optional: describe any change you think would help.
```

## Notes For AI Agents

When creating an issue, be explicit about uncertainty. If you inferred behavior
from a tool description, say so. If a response contained `warning`,
`best_effort`, `truncated`, or `attribution_uncertain`, include those fields in
the report.
