# MCP File Tools

[![Go Report Card](https://goreportcard.com/badge/github.com/zoster81/mcp-file-tools)](https://goreportcard.com/report/github.com/zoster81/mcp-file-tools)
[![Release](https://img.shields.io/github/v/release/zoster81/mcp-file-tools)](https://github.com/zoster81/mcp-file-tools/releases/latest)
[![License: GPL-3.0](https://img.shields.io/github/license/zoster81/mcp-file-tools)](LICENSE)
[![Upstream MCP Registry](https://img.shields.io/badge/Upstream-MCP_Registry-blue)](https://registry.modelcontextprotocol.io/?search=mcp-file-tools)

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

### ChatGPT Web through the OpenAI Secure MCP Tunnel

The currently validated deployment is Windows plus the OpenAI tunnel client. The tunnel launches this fork as a local stdio MCP process and bridges it to the remote connector used by ChatGPT Web.

Requirements:

- Windows PowerShell 5.1 or later;
- `tunnel-client.exe` from the OpenAI Secure MCP Tunnel setup;
- a Windows build of this fork;
- an OpenAI Runtime API key with the tunnel permissions required by your OpenAI configuration;
- a valid Tunnel ID;
- one explicit local directory to expose to the MCP server.

#### Build the fork locally

```powershell
git clone https://github.com/zoster81/mcp-file-tools.git
Set-Location .\mcp-file-tools
go test ./...
go build -o mcp-file-tools_windows_amd64.exe ./cmd/mcp-file-tools
```

The Go module currently retains the upstream module path for source compatibility. For that reason, use clone-and-build for the fork; `go install github.com/dimitar-grigorov/...` installs the upstream project and does not include the fork-specific tunnel and execution changes.

#### Download a fork release

After this fork publishes its first GitHub Release, the Windows binary can be downloaded with:

```powershell
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools" | Out-Null
Invoke-WebRequest `
    "https://github.com/zoster81/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" `
    -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools_windows_amd64.exe"
```

Until a fork release exists, build from source as shown above.

#### OpenAI Tunnel quick start

A sanitized English example is provided at [`examples/start-openai-tunnel.ps1`](examples/start-openai-tunnel.ps1).

Place these files in the same private working directory:

```text
tunnel-client.exe
mcp-file-tools_windows_amd64.exe
start-openai-tunnel.ps1
```

Copy the example outside the Git checkout before entering credentials:

```powershell
$runDirectory = "$env:LOCALAPPDATA\OpenAI-Mcp-Tunnel"
New-Item -ItemType Directory -Force $runDirectory | Out-Null
Copy-Item .\examples\start-openai-tunnel.ps1 $runDirectory
Copy-Item .\mcp-file-tools_windows_amd64.exe $runDirectory
# Copy tunnel-client.exe from your OpenAI tunnel installation into the same directory.
notepad "$runDirectory\start-openai-tunnel.ps1"
```

Replace only the placeholders:

```powershell
$RuntimeApiKey = "REPLACE_WITH_RUNTIME_API_KEY"
$TunnelId = "tunnel_REPLACE_WITH_ID"
$AllowedDirectory = "C:\Path\To\AllowedProject"
```

Never commit the edited script. The example keeps `run_script` and `shell` disabled by default. Enable either capability only after reviewing the security boundaries in [TOOLS.md](TOOLS.md#execution-tools).

Run the test from Windows PowerShell:

```powershell
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass `
    -File "$env:LOCALAPPDATA\OpenAI-Mcp-Tunnel\start-openai-tunnel.ps1"
```

The script validates paths and placeholders, runs `tunnel-client doctor --explain`, then starts the tunnel with the local operator UI at `http://127.0.0.1:8080/ui`. The MCP server itself remains stdio-only; the tunnel is the bridge to ChatGPT Web.

### Other stdio MCP clients

The same binary can be used directly by clients that launch local stdio MCP servers. Supply every allowed directory as a command-line argument.

```json
{
  "mcpServers": {
    "file-tools": {
      "type": "stdio",
      "command": "C:\\Tools\\mcp-file-tools_windows_amd64.exe",
      "args": ["D:\\Projects", "C:\\Users\\YOUR_NAME\\Documents"]
    }
  }
}
```

A roots-capable client may also provide workspace directories dynamically. CLI-provided directories remain the authoritative baseline in this fork.

### Updating the fork

The update checker is notification-only and checks releases from `zoster81/mcp-file-tools`. It never downloads or replaces a binary.

To update a manual Windows installation:

1. stop the OpenAI tunnel or other MCP client using the binary;
2. download the latest fork release;
3. replace the executable;
4. restart the tunnel and run its diagnostics.

```powershell
Invoke-WebRequest `
    "https://github.com/zoster81/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" `
    -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools_windows_amd64.exe"
```

Set `MCP_NO_UPDATE_CHECK=1` before starting the server to disable release checks.

### Upstream integrations and registry

This fork originates from [`dimitar-grigorov/mcp-file-tools`](https://github.com/dimitar-grigorov/mcp-file-tools). The existing MCP Registry entry and the original Claude Code marketplace integration belong to the upstream project and do not represent this fork or its additional tools.

The fork retains upstream plugin files for compatibility work, but they require a matching fork release before they can download fork binaries successfully. The upstream plugin can still be installed separately with:

```text
/plugin marketplace add dimitar-grigorov/mcp-file-tools
/plugin install mcp-file-tools
```

That command installs the upstream implementation, not this fork.

## How to Use

Once the connector is active, ask ChatGPT Web or the connected MCP client:
- "List all .pas files in the allowed project directory"
- "Read config.ini and detect its encoding"
- "Show all supported encodings"
- "Read MainForm.dfm using CP1251 encoding"

**Security:** File tools access only explicitly allowed directories:
- **OpenAI Tunnel:** the directory arguments embedded in `MCP_COMMAND` are the authoritative baseline;
- **roots-capable stdio clients:** client-provided roots may augment that baseline;
- **execution tools:** `run_script` validates its script and working-directory paths, while `shell` validates only its working directory and is otherwise unrestricted.

## Configuration

The server can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_DEFAULT_ENCODING` | Default encoding for `write_file` when none specified | `cp1251` |
| `MCP_MEMORY_THRESHOLD` | Memory threshold in bytes. Files smaller are loaded into memory for faster I/O; larger files use streaming. Also affects encoding detection mode. | `67108864` (64MB) |
| `MCP_ENABLE_RUN_SCRIPT` | Enables only the `run_script` tool. Accepted true values: `1`, `true`, `yes`, `on`, `enabled`. | disabled |
| `MCP_ENABLE_SHELL` | Enables only the unrestricted `shell` tool. Accepted true values: `1`, `true`, `yes`, `on`, `enabled`. | disabled |
| `MCP_ENABLE_EXECUTION` | Enables both `run_script` and `shell`; use only in a trusted environment. | disabled |

To override, set environment variables in the tunnel launcher or another stdio client configuration:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "C:\\Tools\\mcp-file-tools_windows_amd64.exe",
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
