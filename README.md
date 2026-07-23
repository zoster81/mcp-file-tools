# MCP File Tools

[![Go Report Card](https://goreportcard.com/badge/github.com/dimitar-grigorov/mcp-file-tools)](https://goreportcard.com/report/github.com/dimitar-grigorov/mcp-file-tools)
[![Release](https://img.shields.io/github/v/release/zoster81/mcp-file-tools)](https://github.com/zoster81/mcp-file-tools/releases/latest)
[![License: GPL-3.0](https://img.shields.io/github/license/dimitar-grigorov/mcp-file-tools)](LICENSE)
[![MCP Registry](https://img.shields.io/badge/MCP-Registry-blue)](https://registry.modelcontextprotocol.io/?search=mcp-file-tools)

ChatGPT Web sees `Настройки` — not `????` or `Íàñòðîéêè`.

MCP server for file operations with non-UTF-8 encoding support. Auto-detects and converts 24 encodings (Cyrillic, Windows-125x, ISO-8859, KOI8, UTF-16, GBK/GB18030) so AI assistants can read and write legacy files without corrupting data.

**Perfect for:** exposing local Windows project files and controlled execution tools to ChatGPT Web through the OpenAI Secure MCP Tunnel, including legacy Delphi/Pascal projects, VB6 applications, old PHP/HTML sites, and non-UTF-8 configuration files.

## Purpose of This Fork

This fork is maintained primarily to use a local MCP server from **ChatGPT Web through the OpenAI Secure MCP Tunnel**.

The server currently exposes **stdio transport only**. ChatGPT Web does not connect directly to this process: the OpenAI tunnel client launches the local stdio server and bridges it to the remote MCP connector.

The currently validated deployment model is:

```text
ChatGPT Web
    -> OpenAI remote MCP connector
    -> OpenAI Secure MCP Tunnel
    -> local mcp-file-tools stdio process
    -> explicitly allowed Windows directories
```

The fork does not require Claude Code, Codex, or another local AI application. The upstream integrations for those clients may still work and are retained as reference, but they are not the primary deployment target of this fork.

Another browser-hosted LLM could use the current server only if its MCP connector infrastructure provides an equivalent gateway capable of launching and bridging a local stdio MCP process. Native HTTP/JSON or Streamable HTTP transport is **not implemented yet**.

A future compatibility phase may add an optional native HTTP/JSON transport while preserving stdio support. That work requires a separate security, authentication, binding, and deployment design before implementation.

The custom tunnel-oriented changes include authoritative CLI roots, Windows drive-root handling, and optional local execution tools. The upstream project remains the source of the original encoding-aware file-tool implementation.

## What It Does

Provides 24 tools for file operations, encoding conversion, update checks, and optional local execution:
- [`read_text_file`](TOOLS.md#read_text_file) - Read files with encoding auto-detection and conversion
- [`read_multiple_files`](TOOLS.md#read_multiple_files) - Read multiple files concurrently with encoding support
- [`write_file`](TOOLS.md#write_file) - Write files in specific encodings
- [`edit_file`](TOOLS.md#edit_file) - Line-based edits with diff preview and whitespace-flexible matching
- [`copy_file`](TOOLS.md#copy_file) - Copy a file to a new location
- [`delete_file`](TOOLS.md#delete_file) - Delete a file
- [`list_directory`](TOOLS.md#list_directory) - Browse directories with pattern filtering
- [`tree`](TOOLS.md#tree) - Compact indented tree view (85% fewer tokens than JSON)
- [`directory_tree`](TOOLS.md#directory_tree-deprecated) - Get recursive tree view as JSON (deprecated, use `tree`)
- [`search_files`](TOOLS.md#search_files) - Recursively search for files matching glob patterns
- [`grep_text_files`](TOOLS.md#grep_text_files) - Regex search in file contents with encoding support
- [`detect_encoding`](TOOLS.md#detect_encoding) - Auto-detect file encoding with confidence score
- [`convert_encoding`](TOOLS.md#convert_encoding) - Convert file between encodings
- [`detect_line_endings`](TOOLS.md#detect_line_endings) - Detect line ending style (CRLF/LF/mixed)
- [`change_line_endings`](TOOLS.md#change_line_endings) - Convert line endings to LF or CRLF
- [`manage_bom`](TOOLS.md#manage_bom) - Detect, strip, or add Unicode BOM
- [`list_encodings`](TOOLS.md#list_encodings) - Show all supported encodings
- [`get_file_info`](TOOLS.md#get_file_info) - Get file/directory metadata
- [`create_directory`](TOOLS.md#create_directory) - Create directories recursively (mkdir -p)
- [`move_file`](TOOLS.md#move_file) - Move or rename files and directories
- [`list_allowed_directories`](TOOLS.md#list_allowed_directories) - Show accessible directories
- [`run_script`](TOOLS.md#run_script) - Execute a supported script or executable inside an allowed directory when explicitly enabled
- [`shell`](TOOLS.md#shell) - Execute an unrestricted shell command when explicitly enabled
- [`check_for_updates`](TOOLS.md#check_for_updates) - Check the latest release of this fork with a cached GitHub request

**Supported encodings (22 total):**
- **Unicode:** UTF-8, UTF-16 LE, UTF-16 BE (with BOM detection for UTF-16 and UTF-32)
- **Cyrillic:** Windows-1251, KOI8-R, KOI8-U, CP866, ISO-8859-5
- **Western European:** Windows-1252, ISO-8859-1, ISO-8859-15
- **Central European:** Windows-1250, ISO-8859-2
- **Greek:** Windows-1253, ISO-8859-7
- **Turkish:** Windows-1254, ISO-8859-9
- **Other:** Hebrew (1255), Arabic (1256), Baltic (1257), Vietnamese (1258), Thai (874)

See [TOOLS.md](TOOLS.md) for detailed parameters and examples.

**Security:** File operations and `run_script` paths are restricted to allowed directories. The optional `shell` tool validates only its working directory; the command itself is unrestricted and runs with the operating-system permissions of the MCP server process.

## Custom Fork Changes

This repository is a custom fork of [`dimitar-grigorov/mcp-file-tools`](https://github.com/dimitar-grigorov/mcp-file-tools). Compared with the upstream project, this fork currently adds:

- optional `run_script` and `shell` MCP tools, disabled by default;
- CLI-provided allowed directories as the authoritative fallback for tunnel clients that do not implement MCP roots requests;
- correct validation of descendants when a Windows drive root such as `D:\` is allowed.

See [CHANGELOG.md](CHANGELOG.md) for the maintained list of fork-specific changes.

## Installation

> **Fork deployment note:** the custom tunnel and execution changes are not present in upstream release binaries. Until this fork publishes its own GitHub releases, build the fork locally and launch that binary through the OpenAI tunnel client.

### Upstream Claude Code plugin (reference)

The upstream project can be installed in Claude Code as follows:

```
/plugin marketplace add dimitar-grigorov/mcp-file-tools
/plugin install mcp-file-tools
```

On first launch the plugin downloads the right binary for your OS, verifies its
SHA-256, caches it, and keeps it pinned to a known version. The server is
automatically scoped to the folder you have open (via the MCP roots protocol), so
there is nothing to configure. It needs nothing beyond Claude Code itself; the
launcher runs on Node, which Claude Code already uses, so it works the same on
Windows, macOS, and Linux.

The plugin only accesses your current workspace. To grant access to directories
outside it, use a manual install (below).

**Already added the server the manual way?** Remove the old `claude mcp add` entry so
you are not running two copies:

```
claude mcp remove file-tools
```

### Updating the plugin

```
claude plugin marketplace update mcp-file-tools
claude plugin update mcp-file-tools@mcp-file-tools
```

Use the full `plugin@marketplace` id, not the bare name. Or enable auto-update in
`/plugin` → **Marketplaces**.

### MCP Registry

This server is listed in the [Official MCP Registry](https://registry.modelcontextprotocol.io/?search=mcp-file-tools) for discovery by any MCP client.

### Manual install (other MCP clients, or access outside your workspace)

Download the binary for your platform, then register it with the directories it may access.

| Platform | Release asset | Suggested path |
|----------|---------------|----------------|
| Windows x64 | `mcp-file-tools_windows_amd64.exe` | `%LOCALAPPDATA%\Programs\mcp-file-tools\mcp-file-tools.exe` |
| Linux x64 | `mcp-file-tools_linux_amd64` | `~/.local/bin/mcp-file-tools` |
| macOS ARM64 | `mcp-file-tools_darwin_arm64` | `~/.local/bin/mcp-file-tools` |

Windows (PowerShell, not CMD):

```powershell
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
claude mcp add --scope user file-tools -- "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe" "D:\Projects"
```

Linux / macOS (swap the asset name from the table for your platform):

```bash
mkdir -p ~/.local/bin
curl -L "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_linux_amd64" -o ~/.local/bin/mcp-file-tools
chmod +x ~/.local/bin/mcp-file-tools
claude mcp add --scope user file-tools -- ~/.local/bin/mcp-file-tools ~/Projects
```

### Go install (all platforms)

```bash
# Requires Go 1.26+
go install github.com/dimitar-grigorov/mcp-file-tools/cmd/mcp-file-tools@latest
# Linux / macOS
claude mcp add --scope user file-tools -- $(go env GOPATH)/bin/mcp-file-tools ~/Projects
```

```powershell
# Windows PowerShell
claude mcp add --scope user file-tools -- "$(go env GOPATH)\bin\mcp-file-tools.exe" "D:\Projects"
```

### Other Clients

For Claude Desktop, VSCode, or Cursor, use the downloaded binary path in your config:

**Claude Desktop** (`%APPDATA%\Claude\claude_desktop_config.json` on Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

Windows:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects", "C:\\Users\\YOUR_NAME\\Documents"]
    }
  }
}
```

macOS / Linux:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/Users/YOUR_NAME/.local/bin/mcp-file-tools",
      "args": ["/Users/YOUR_NAME/Projects", "/Users/YOUR_NAME/Documents"]
    }
  }
}
```

The `args` array specifies allowed directories the server can access. Add as many directories as you need.

**VSCode / Cursor (Claude Code extension)**

If you already ran `claude mcp add --scope user` from the installation steps above, the server is already available in VSCode — no extra config needed.

To configure separately for VSCode only:
```powershell
claude mcp add --scope user file-tools -- "%LOCALAPPDATA%\Programs\mcp-file-tools\mcp-file-tools.exe" "D:\Projects"
```

Alternatively, create a **per-project config** by adding `.mcp.json` to your project root:
```json
{
  "mcpServers": {
    "file-tools": {
      "type": "stdio",
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects", "D:\\Other\\Directory"]
    }
  }
}
```

**Note:** The `type: "stdio"` field is required. The `args` array specifies allowed directories — the VSCode extension does not automatically add the workspace directory, so you must list all directories you want to access. To add more directories later, re-run the `claude mcp add` command with all directories listed (it overwrites the previous config).

**OpenAI Codex CLI**

Codex does not have an `mcp add` command -- you need to edit `~/.codex/config.toml` manually.

Windows (PowerShell):
```powershell
# Download
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
```

Then add to `~/.codex/config.toml`:
```toml
[mcp_servers.file-tools]
command = "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe"
args = ["D:\\Projects"]
```

### Auto-approve All Tools (Claude Code)

To skip permission prompts for all file-tools commands, create `.claude/settings.local.json` in your project root:

```json
{
  "permissions": {
    "allow": [
      "Bash(ls *)",
      "Bash(grep *)",
      "Bash(sort *)",
      "Bash(wc *)",
      "Bash(find *)",
      "Bash(echo *)",
      "Grep",
      "Glob",
      "WebSearch",
      "mcp__file-tools__read_text_file",
      "mcp__file-tools__read_multiple_files",
      "mcp__file-tools__write_file",
      "mcp__file-tools__edit_file",
      "mcp__file-tools__copy_file",
      "mcp__file-tools__list_directory",
      "mcp__file-tools__tree",
      "mcp__file-tools__directory_tree",
      "mcp__file-tools__search_files",
      "mcp__file-tools__grep_text_files",
      "mcp__file-tools__detect_encoding",
      "mcp__file-tools__convert_encoding",
      "mcp__file-tools__detect_line_endings",
      "mcp__file-tools__change_line_endings",
      "mcp__file-tools__manage_bom",
      "mcp__file-tools__list_encodings",
      "mcp__file-tools__get_file_info",
      "mcp__file-tools__create_directory",
      "mcp__file-tools__list_allowed_directories",
      "mcp__file-tools__check_for_updates"
    ]
  }
}
```

This auto-approves safe read-only and editing file-tools operations plus common shell commands and web search. Destructive operations (`delete_file`, `move_file`) and `WebFetch` are intentionally excluded — Claude will ask before using them. Adjust to your needs.

### Update

The server checks for updates automatically and notifies you through tool responses when a newer version is available. To update:

1. Close all Claude Code sessions (the binary is locked while running)
2. Re-download the binary:

```powershell
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" `
    -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
```

To disable update checks, set the environment variable `MCP_NO_UPDATE_CHECK=1`.

### Verify & Uninstall

```bash
# Check which file-tools server is connected (plugin or manual)
claude mcp list

# Remove a manual install
claude mcp remove file-tools

# Remove the plugin
claude plugin uninstall mcp-file-tools
```

## How to Use

Once installed, just ask Claude:
- "List all .pas files in this directory"
- "Read config.ini and detect its encoding"
- "Show all supported encodings"
- "Read MainForm.dfm using CP1251 encoding"

**Security:** The server only accesses directories you explicitly allow:
- **Automatic:** Claude Desktop/Code provide workspace directories automatically
- **Manual:** Specify directories in config `args: ["/path/to/project"]`

## Configuration

The server can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_DEFAULT_ENCODING` | Default encoding for `write_file` when none specified | `cp1251` |
| `MCP_MEMORY_THRESHOLD` | Memory threshold in bytes. Files smaller are loaded into memory for faster I/O; larger files use streaming. Also affects encoding detection mode. | `67108864` (64MB) |
| `MCP_ENABLE_RUN_SCRIPT` | Enables only the `run_script` tool. Accepted true values: `1`, `true`, `yes`, `on`, `enabled`. | disabled |
| `MCP_ENABLE_SHELL` | Enables only the unrestricted `shell` tool. Accepted true values: `1`, `true`, `yes`, `on`, `enabled`. | disabled |
| `MCP_ENABLE_EXECUTION` | Enables both `run_script` and `shell`; use only in a trusted environment. | disabled |

To override, set environment variables in your config (Claude Desktop example):
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects"],
      "env": {
        "MCP_DEFAULT_ENCODING": "utf-8"
      }
    }
  }
}
```

## Use Cases

### Legacy Codebases

Many legacy projects use non-UTF-8 encodings that AI assistants can't handle natively:

- **Delphi/Pascal** (Windows-1251): Source files with Cyrillic UI text
- **Visual Basic 6** (Windows-1252): Forms and config files with Western European characters
- **Legacy PHP/HTML** (CP1251, ISO-8859-1): Web apps with localized content
- **Old config files** (Various): INI, properties, registry files with legacy encodings

**How it works:**
```
User: Read config.ini and change the title to "Настройки"
Assistant: [read_text_file with cp1251] → [modify UTF-8] → [write_file with cp1251]
```

The original encoding is preserved - files remain compatible with legacy tools.

## Development

**Prerequisites:** Go 1.26+

```bash
# Run tests
go test ./...

# Build
go build -o mcp-file-tools ./cmd/mcp-file-tools
```

### Debugging with MCP Inspector

[MCP Inspector](https://github.com/modelcontextprotocol/inspector) provides a web UI for testing MCP servers.

**Prerequisites:** Node.js v18+

```bash
# Run with allowed directory (required)
npx @modelcontextprotocol/inspector go run ./cmd/mcp-file-tools -- /path/to/allowed/dir

# Or with built binary
npx @modelcontextprotocol/inspector ./mcp-file-tools.exe C:\Projects
```

Opens a browser where you can view tools, call them with custom arguments, and inspect responses.

### Manual Debugging

Run the server with an allowed directory and send JSON-RPC commands via stdin:

```bash
# Specify allowed directory
go run ./cmd/mcp-file-tools /path/to/project
```

Example commands (paste into terminal):

```json
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_directory","arguments":{"path":"/path/to/project","pattern":"*.go"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_text_file","arguments":{"path":"/path/to/project/main.pas","encoding":"cp1251"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"detect_encoding","arguments":{"path":"/path/to/project/file.txt"}}}
```

## License

GPL-3.0 - see [LICENSE](LICENSE)
