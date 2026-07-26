# mcp-file-tools fork (Claude Code plugin)

Installs the [`zoster81/mcp-file-tools`](https://github.com/zoster81/mcp-file-tools)
fork into Claude Code via `/plugin install`.

The server provides 24 MCP tools for encoding-aware filesystem operations, BOM-preserving text handling, deterministic secure traversal, durable atomic mutations with rollback-aware backups, typed operation errors with centralized MCP/batch mapping, bounded ordered batch concurrency, fork-specific stdio tunnel compatibility, and optional execution tools documented in the main [README](../README.md) and [TOOLS reference](../TOOLS.md). Encoding behavior is content-based and does not depend on file extensions.

> This Claude Code integration is optional and is not used by the OpenAI Secure MCP Tunnel deployment. It remains frozen during internal development and will be reviewed at the final `2.0.0` release gate. If retained, the plugin version must match the release binaries and `checksums.txt` asset.

## Install

```text
/plugin marketplace add zoster81/mcp-file-tools
/plugin install mcp-file-tools
```

## How it works

`.mcp.json` declares one MCP server (`file-tools`) launched as
`node ${CLAUDE_PLUGIN_ROOT}/bin/run.js`. On first launch, the script downloads the
pinned fork release binary for the current OS and architecture, verifies its
SHA-256 against `checksums.txt`, caches it, and hands stdio directly to the server.
Later launches reuse the cached binary.

Claude Code can provide the workspace through the MCP roots protocol. CLI directory
arguments remain authoritative when the server is launched by a transport such as
the OpenAI Secure MCP Tunnel.

## Requirements

The plugin requires Claude Code and its bundled Node runtime. The downloaded Go
server binary has no runtime dependency.

## Alternative without the plugin

Clone and build the fork directly:

```bash
git clone https://github.com/zoster81/mcp-file-tools.git
cd mcp-file-tools
go test ./...
go build -o mcp-file-tools ./cmd/mcp-file-tools
```

Then reference the built executable from a stdio MCP client configuration or use
the OpenAI Tunnel example in
[`examples/start-openai-tunnel.ps1`](../examples/start-openai-tunnel.ps1).

Project lineage and upstream synchronization policy are documented in the main [README](../README.md) and [publishing notes](../docs/PUBLISHING.md).
