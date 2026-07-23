# Changelog

This file records changes maintained in the `zoster81/mcp-file-tools` fork relative to the upstream `dimitar-grigorov/mcp-file-tools` project.

The upstream baseline for the first fork-specific changes is commit `52665aa080b24f6427e3fc485df76cc0a8ce1238`.

## Unreleased

### Added

- Added `examples/start-openai-tunnel.ps1`, a sanitized English Windows PowerShell 5.1 quick start for ChatGPT Web through the official OpenAI Secure MCP Tunnel.
- Added real upstream encoding fixtures and byte-identical line-ending round-trip tests for all 24 registered encodings, including UTF-16 LE/BE and GBK/GB18030.

### Changed

- Updated fork installation, download, update, plugin, and release commands to target `zoster81/mcp-file-tools`.
- Linked the official `openai/tunnel-client` repository and OpenAI Secure MCP Tunnel guide.
- Added complete PowerShell and Command Prompt launch commands plus explicit instructions for enabling `run_script` and `shell`.
- Kept upstream references only where they represent attribution, the retained Go module path, or the existing upstream MCP Registry entry.
- Configured GoReleaser and the plugin launcher to download and publish fork releases.
- Guarded the upstream MCP Registry workflow so it cannot publish upstream metadata from this fork.
- Documented the fork-specific execution tools, environment flags, limits, result fields, and security boundaries.
- Added an explicit summary of differences from the upstream project to `README.md`.
- Added the previously missing `check_for_updates` reference and corrected its exposed cache interval from two hours to the implemented 30 minutes.
- Redirected update checks and release links from the upstream project to `zoster81/mcp-file-tools`.
- Made update notifications client-neutral for OpenAI Tunnel and other MCP connector transports instead of referring specifically to Claude Code.
- Added the ChatGPT Web/OpenAI Secure MCP Tunnel deployment purpose to `README.md`, explicitly documenting that the current server transport is stdio and requires a compatible bridge.
- Recorded native HTTP/JSON or Streamable HTTP transport as a future compatibility direction, not as an implemented capability.
- Invalidated cached release data when it belongs to a different repository source.
- Updated the fork documentation and runtime tool descriptions to list all 24 encodings and document MetaTrader 4/5 MQL sources (`.mq4`, `.mq5`, `.mqh`) commonly stored as UTF-16 LE with BOM and CRLF endings.

### Fixed

- Fixed `detect_line_endings` so it decodes the selected or auto-detected encoding before analyzing CRLF/LF sequences, including UTF-16 LE/BE.
- Fixed `change_line_endings` so it preserves encoding, BOM state, and every non-line-ending byte across all 24 registered encodings.
- Fixed four Staticcheck `ST1005` diagnostics in execution-tool error messages.

### Removed

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
