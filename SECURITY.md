# Security Policy

## Supported Versions

This project is pre-1.0. Security fixes target the latest release and `main`.
Older beta releases are not maintained and will not receive backports.

## Reporting a Vulnerability

Use **GitHub's private vulnerability reporting**: go to the repository Security
tab and click "Report a vulnerability". Do **not** open a public issue for
security vulnerabilities. No email contact is published for security reports.

## Scope

**In scope:**

- The MCP server itself and its handling of credentials (env vars, secrets)
- Network exposure of the HTTP transport
- Raw-file access behavior

**Out of scope:**

- Vulnerabilities in OpenGrok itself
- Vulnerabilities in the network or infrastructure the server is deployed into

## Security Model At A Glance

This table points to the canonical documentation for each security topic. It
does not restate that documentation.

| Topic | Where it's documented |
|---|---|
| Secrets in env, not flags/logs | constitution Principle IV; `AGENTS.md` "Security Rules" |
| HTTP mode has no inbound auth | `docs/limitations.md` "State And Transport" |
| TLS verification bypass is opt-in | `docs/limitations.md` "Capability And Response Boundaries"; `AGENTS.md` "Security Rules" |
| Raw-file fallback risk | `docs/limitations.md` "Capability And Response Boundaries" |
| Memory is ephemeral, disabled over HTTP | `docs/limitations.md` "State And Transport"; `AGENTS.md` "Security Rules" |
