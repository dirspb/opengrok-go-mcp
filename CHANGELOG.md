# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
`docs/release-process.md` for changelog and versioning rules.

## [Unreleased]

### Added
- Spec Kit setup: constitution, templates, and contribution-workflow policy.

## [0.3.0-beta.2] - 2026-05-26

### Added
- `tokenized` and `path_exclude` search parameters (multiple space-separated
  exclude terms supported).
- Auto-quoting of bare multi-word code queries, with a warning when applied.
- Warnings for `date:` misuse and large tokenized result sets.
- Pagination fields (`page`, `total_pages`, `has_more`) on search results.
- `ReadOnlyHint` annotations on all tools.

### Changed
- Default tool surface is now `full`.
- Unified `list_files` pagination (`total_files` reported as `total_hits`).
- Replaced `filtered_total_hits` with pagination plus an honest kind-filter
  warning.
- Upgraded go-sdk from v1.2.0 to v1.4.0.

### Fixed
- `date:` warning now matches the field token instead of a substring.
- Coerce string-encoded scalar tool arguments.
- Accept object payloads in compact wrapper tools.

## [0.3.0-beta.1] - 2026-04-25

### Added
- `list_symbols` tool with ctags kind filtering and a pagination cost warning,
  gated behind the `ListSymbols` capability.
- Automatic search-result context expansion (`expand_context`, with
  `AutoExpandContext`, `ContextBefore`, `ContextAfter` config).
- `Result.Kind` wired from the ctags tag rather than search mode.

### Fixed
- Added a mutex for concurrent file-content fetches.

## [0.2.0] - 2026-04-22

### Added
- Cursor pagination for `list_projects` and `read_file` (500-line page cap).
- Pagination and warning fields on MCP types.
- Search warning when `total_hits` exceeds 500.
- `OPENGROK_MCP_INSECURE_SKIP_TLS_VERIFY` for legacy TLS certificates.

### Fixed
- Guard against empty-file panic in `pagedFileContext`.
- Guard against out-of-bounds offset in `list_projects` pagination.

## [0.1.0] - 2026-04-22

### Added
- Initial release: OpenGrok-backed MCP server with code search, file read, and
  symbol search; required default project; 32 MiB response-body cap.
