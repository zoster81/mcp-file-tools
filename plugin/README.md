# mcp-file-tools (Claude Code plugin)

Installs the [`mcp-file-tools`](https://github.com/dimitar-grigorov/mcp-file-tools)
MCP server into Claude Code via `/plugin install`.

The server provides filesystem operations with non-UTF-8 encoding support
(CP1251, CP1252, KOI8-R, ISO-8859, ...) plus auto-detection and UTF-8 conversion.

## Install

```
/plugin marketplace add dimitar-grigorov/mcp-file-tools
/plugin install mcp-file-tools
```

## How it works

`plugin.json` declares one MCP server (`file-tools`) launched as
`bash ${CLAUDE_PLUGIN_ROOT}/bin/run.sh`. On first launch the script downloads the
pinned release binary for your OS/arch, verifies its SHA-256 against the release
`checksums.txt`, caches it, then runs it. Later launches reuse the cache.

No directory configuration is needed: Claude Code sends the workspace folder via the
MCP roots protocol and the server allows it automatically.

## Requirements

- macOS / Linux: `bash` and `curl` or `wget` (already present).
- Windows: Git Bash on `PATH` (ships with Git for Windows), since the server is
  launched via `bash`. A PowerShell launcher (`bin/run.ps1`) is included if you prefer
  to avoid the Git Bash dependency.

## Alternative without the plugin

Install the binary with `go install
github.com/dimitar-grigorov/mcp-file-tools/cmd/mcp-file-tools@latest` (or download it
from Releases), then add to `.mcp.json`:

```json
{
  "mcpServers": {
    "file-tools": { "command": "mcp-file-tools" }
  }
}
```
