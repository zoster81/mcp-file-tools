# Changelog

This file records changes maintained in the `zoster81/mcp-file-tools` fork relative to the upstream `dimitar-grigorov/mcp-file-tools` project.

The upstream baseline for the first fork-specific changes is commit `52665aa080b24f6427e3fc485df76cc0a8ce1238`.

## Unreleased

### Added

- Added `internal/execution`, a shared process-preparation primitive for absolute working-directory validation, bounded timeout/output handling, cancellation, process-tree termination, and caller-supplied pre-launch revalidation.
- Added an embedded authoritative tool catalog consumed by MCP runtime registration and Registry manifest generation, with tests enforcing runtime metadata and README/TOOLS coverage.
- Added streaming SHA-256 filesystem snapshots for optimistic pre-execution verification without loading complete scripts into memory.

### Changed

- Changed the default encoding for newly created files from legacy `cp1251` to standard UTF-8; existing files still preserve a confidently detected encoding, and `MCP_DEFAULT_ENCODING` or an explicit `encoding` can select legacy formats.
- Replaced the historical active handoff with an authoritative R7-R14 roadmap, a reusable development checklist, and a separate R1-R6 history document; internal commit builds continue without intermediate public releases until `2.0.0`.
- Generalized encoding documentation, runtime instructions, tool metadata, and acceptance fixtures so detection is explicitly content-based and independent of file extensions; MQL remains only an ordinary possible input domain.
- Refactored `run_script` and `shell` to share only process-level mechanics while retaining separate authorization policies and independent feature flags.
- Made both execution tools revalidate their working directory immediately before launch; `run_script` also verifies script metadata and SHA-256 content before execution.
- Made `server.template.json` release-neutral for tool metadata; `scripts/generate-server-json.js` now injects the Registry projection from the authoritative catalog.

### Fixed

- Rejected script replacement or in-place content changes between `run_script` preparation and launch, including same-size and restored-timestamp changes that metadata-only checks can miss.

## 1.8.0 - 2026-07-25

### Added

- Added `examples/start-openai-tunnel.ps1`, a sanitized English Windows PowerShell 5.1 quick start for ChatGPT Web through the official OpenAI Secure MCP Tunnel.
- Added real upstream encoding fixtures and byte-identical line-ending round-trip tests for all 24 registered encodings, including UTF-16 LE/BE and GBK/GB18030.
- Added optional `hasBOM` and `bomType` metadata to single-file and batch read results.
- Added transport-independent typed operation errors for validation, access control, encoding, decoding, output encoding, conflicts, cancellation, limits, permissions, and filesystem failures.
- Added a generic bounded ordered concurrency coordinator with deterministic serial commits, cancellation modes, early stop, and run statistics.

### Changed

- Updated fork installation, download, update, plugin, and release commands to target `zoster81/mcp-file-tools`.
- Linked the official `openai/tunnel-client` repository and OpenAI Secure MCP Tunnel guide.
- Added complete PowerShell and Command Prompt launch commands plus explicit instructions for enabling `run_script` and `shell`.
- Limited original-project references to historical attribution and upstream synchronization documentation.
- Configured GoReleaser and the plugin launcher to download and publish fork releases.
- Migrated the Go module path, all internal imports, linker flags, manual test harness, and operational metadata to `github.com/zoster81/mcp-file-tools`.
- Replaced the inherited registry manifest with a fork-owned release template and checksum-driven generator, and restricted OIDC publication to `zoster81/mcp-file-tools`.
- Pinned actionlint, ShellCheck, Staticcheck, govulncheck, GoReleaser, and MCP Publisher versions in CI, with SHA-256 verification for downloaded workflow and registry tools.
- Documented the fork-specific execution tools, environment flags, limits, result fields, and security boundaries.
- Added an explicit summary of differences from the upstream project to `README.md`.
- Added the previously missing `check_for_updates` reference and corrected its exposed cache interval from two hours to the implemented 30 minutes.
- Redirected update checks and release links from the upstream project to `zoster81/mcp-file-tools`.
- Made update notifications client-neutral for OpenAI Tunnel and other MCP connector transports instead of referring specifically to Claude Code.
- Added the ChatGPT Web/OpenAI Secure MCP Tunnel deployment purpose to `README.md`, explicitly documenting that the current server transport is stdio and requires a compatible bridge.
- Recorded native HTTP/JSON or Streamable HTTP transport as a future compatibility direction, not as an implemented capability.
- Invalidated cached release data when it belongs to a different repository source.
- Updated the fork documentation and runtime tool descriptions to list all 24 encodings and document MetaTrader 4/5 MQL sources (`.mq4`, `.mq5`, `.mqh`) commonly stored as UTF-16 LE with BOM and CRLF endings.
- Refactored `read_text_file`, `read_multiple_files`, and `edit_file` to use one shared encoding/BOM-aware document pipeline and consistent batch error classification.
- Added a shared document encoder with internal BOM-preserve policy for edits.
- Migrated `grep_text_files` to the shared encoding/BOM-aware document decoder with bounded per-file scanning and deterministic ordered aggregation.
- Migrated `write_file` and `convert_encoding` to the shared document encoder and added the public `auto`, `always`, `never`, and `preserve` BOM policies.
- Migrated `detect_line_endings` and `change_line_endings` to shared text-document path, encoding, BOM, mode, and commit validation while retaining byte-exact newline conversion.
- Consolidated recursive traversal for `search_files`, `grep_text_files`, `tree`, and `directory_tree` into one deterministic, cancellation-aware filesystem walker while preserving each tool's public filtering, limit, ordering, and error behavior.
- Added a shared durable mutation layer for write, edit, encoding conversion, line-ending conversion, BOM changes, copy, move, and delete operations, with exclusive same-directory staging, file sync, platform-specific atomic/no-replace commits, directory sync where supported, cleanup, and optimistic concurrent-modification snapshots.
- Made encoding-conversion backups transactional: the original is staged and synced before target commit, an existing backup is preserved until success, and target failures restore the previous backup or remove a newly created backup.
- Aligned `README.md`, `TOOLS.md`, publishing notes, plugin and marketplace metadata, Smithery metadata, runtime tool descriptions, and the project roadmap with the completed R1-R5 capabilities and fork-owned release pipeline.
- Centralized conversion of operation failures into MCP error results and `read_multiple_files` per-file error codes, removing duplicated string-based classification while preserving public messages and schemas.
- Replaced the duplicated `read_multiple_files` and `grep_text_files` worker pools with the shared bounded ordered coordinator while preserving input order, partial failures, cancellation behavior, exact global match limits, and bounded pending results.
- Kept `tree`, `directory_tree`, and `search_files` on the serial secure walker because traversal-time pruning, deterministic lexical order, and early limits are part of their public behavior.
- Added a fail-closed release-version check that requires the Git tag, Claude plugin metadata, and marketplace metadata to use the same semantic version before GoReleaser runs.

### Fixed

- Fixed Windows 8.3 short-path roots so canonical long paths remain authorized across repeated validation, backup creation, grep processing, secure traversal callbacks, and junction resolution tests.
- Fixed the Unix backup-permission regression test so it performs a byte-changing conversion instead of entering the intentional byte-identical no-op path with no backup.
- Fixed `detect_line_endings` so it decodes the selected or auto-detected encoding before analyzing CRLF/LF sequences, including UTF-16 LE/BE.
- Fixed `change_line_endings` so it preserves encoding, BOM state, and every non-line-ending byte across all 24 registered encodings.
- Fixed four Staticcheck `ST1005` diagnostics in execution-tool error messages.
- Fixed UTF-8 and UTF-16 transport BOMs leaking into `read_text_file` and `read_multiple_files` content while preserving a meaningful leading `U+FEFF` code point.
- Added deterministic rejection of BOM/encoding conflicts across the shared text-document pipeline, including `detect_line_endings`, instead of decoding with the wrong byte order.
- Fixed `read_text_file` partial reads so CRLF separators no longer leak a trailing `\r` into paginated lines.
- Fixed `edit_file` on UTF-16 LE/BE so normal edits preserve the original BOM and consistent CRLF/LF style instead of converting CRLF to LF.
- Made logical no-op edits byte-identical across all 24 registered encodings.
- Made edit encoding failures occur before filesystem mutation, leaving the original file unchanged.
- Fixed UTF-16 LE/BE grep so BOM-bearing files are auto-detected and BOMless files can be searched with an explicit encoding instead of being rejected by raw NUL-byte binary checks.
- Fixed parallel grep ordering and global `maxMatches` enforcement, including accurate `truncated` reporting when the limit is reached exactly.
- Revalidated every traversed grep file before reading so file symlinks or junctions resolving outside allowed directories are skipped.
- Added native Windows final-path resolution for junctions and other reparse points, closing workspace escapes that `filepath.EvalSymlinks` does not resolve on Windows.
- Rejected existing and deeply nested missing paths whose nearest existing ancestor resolves outside the allowed directories, while retaining support for legitimate missing destinations under safe symlinks or junctions.
- Made all recursive filesystem tools skip entries resolving outside allowed directories and report deterministic lexical traversal order; `tree` also revalidates immediately before encoding detection.
- Prevented initially missing write/copy/move destinations from overwriting files created concurrently by committing with native no-replace operations.
- Rejected practical concurrent changes between read/prepare and commit for document replacements, BOM changes, deletes, and other migrated mutations.
- Fixed backup replacement so a stale `.bak` is replaced only after the new backup is durably staged, and rollback preserves the prior backup on target-commit failure.
- Fixed mutation cleanup so staging files are removed and cleanup failures are surfaced instead of silently ignored.
- Fixed UTF-16 writes and conversions so `auto` emits exactly one canonical BOM, `never` remains BOMless, and UTF-8/legacy `auto` output remains BOM-free.
- Preserved CRLF, LF, CR, and mixed line-ending sequences exactly during encoding conversion, and rejected invalid, impossible, or unrepresentable output before filesystem mutation.
- Skipped byte-identical encoding conversions without rewriting the file or creating a requested backup.

### Removed

- Removed the duplicated handler-local `atomicWriteFile`, `atomicWriteWithBackup`, and temporary-path implementation after all consumers migrated to the shared filesystem mutation layer.
- Removed source backup files that were not part of the runtime implementation.

## 2026-07-23

### Added

- Added the optional `run_script` MCP tool for executing supported script and executable files inside an allowed directory.
- Added the optional `shell` MCP tool for unrestricted shell commands with an allowed working directory.
- Added independent `MCP_ENABLE_RUN_SCRIPT` and `MCP_ENABLE_SHELL` feature flags, plus the combined `MCP_ENABLE_EXECUTION` flag.
- Added bounded stdout and stderr capture, execution timeouts, cancellation reporting, and process-tree termination attempts.

### Changed

- CLI-provided allowed directories remain authoritative when an MCP client does not support server-initiated roots requests.
- MCP roots updates augment rather than replace the CLI directory baseline.

### Fixed

- Fixed Windows drive-root validation so an allowed root such as `D:\` also permits its descendants while continuing to reject paths on other drives.

### Commits

- `e0ef0d8026c615ba055918d04c0b498d3692aa5a` — execution tools and tunnel-compatible roots handling.
- `db2360e2041b6fc1065d3e89743ab016a8b6f748` — Windows drive-root path validation.
