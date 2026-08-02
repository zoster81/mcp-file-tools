# MCP File Tools

<!-- mcp-name: io.github.zoster81/mcp-file-tools -->

[![Go Report Card](https://goreportcard.com/badge/github.com/zoster81/mcp-file-tools)](https://goreportcard.com/report/github.com/zoster81/mcp-file-tools)
[![Release](https://img.shields.io/github/v/release/zoster81/mcp-file-tools)](https://github.com/zoster81/mcp-file-tools/releases/latest)
[![License: GPL-3.0](https://img.shields.io/github/license/zoster81/mcp-file-tools)](LICENSE)
[![MCP Registry](https://img.shields.io/badge/MCP_Registry-io.github.zoster81%2Fmcp--file--tools-blue)](https://registry.modelcontextprotocol.io/?search=io.github.zoster81%2Fmcp-file-tools)

ChatGPT Web sees `Настройки` — not `????` or `Íàñòðîéêè`.

MCP server for file operations with non-UTF-8 encoding support. Auto-detects and converts 24 encodings (Cyrillic, Windows-125x, ISO-8859, KOI8, UTF-16 LE/BE, GBK/GB18030) and provides encoding-aware CRLF/LF detection and conversion for UTF-16 files.

**Perfect for:** exposing local Windows project files and controlled execution tools to ChatGPT Web through the OpenAI Secure MCP Tunnel, including legacy Delphi/Pascal projects, VB6 applications, old PHP/HTML sites, configuration and data files, and other text assets whose encoding cannot be inferred reliably from their filename or extension.

## Purpose of This Fork

This fork is maintained primarily to use a local MCP server from **ChatGPT Web through the OpenAI Secure MCP Tunnel**.

The server supports both **stdio** and native stateful **MCP Streamable HTTP**. ChatGPT Web through the currently validated OpenAI Secure MCP Tunnel deployment still uses stdio: the tunnel client launches the local process and bridges it to the remote MCP connector.

The currently validated deployment model is:

```text
ChatGPT Web
    -> OpenAI remote MCP connector
    -> OpenAI Secure MCP Tunnel
    -> local mcp-file-tools stdio process
    -> explicitly allowed Windows directories
```

The fork does not require Claude Code, Codex, or another local AI application. Version 2.0 removes the fork-owned Claude Code downloader plugin to avoid maintaining a second network installer and cache trust boundary; Claude users can still configure the released binary as an ordinary stdio MCP server.

Other MCP clients may launch the stdio process directly or connect to the native Streamable HTTP endpoint. The HTTP transport is bearer-authenticated, loopback-bound by default, stateful, and governed by the approved threat model and secure defaults in [docs/HTTP_SECURITY.md](docs/HTTP_SECURITY.md). Release history and remaining operator deployment work are tracked in [docs/ROADMAP.md](docs/ROADMAP.md).

### Process-wide directory and session model

Allowed directories are a **process-wide authorization boundary**. Every MCP connection or future HTTP session attached to one server process sees the same configured directory set and the same 23 tools, limits, execution flags, and error behavior. A session represents an independent protocol connection with its own requests, cancellation, and lifecycle; it is not a per-agent filesystem role or sandbox.

This deliberately supports deployments where several agents work on different projects under one allowed drive or workspace, read shared documentation or libraries, and follow prompt-level rules about where each agent may write. The server does not enforce those per-agent read/write conventions. When technical isolation is required, run separate server processes with narrower allowed directories and, for concurrent Git work, separate checkouts or worktrees.

Directories supplied when the process starts remain authoritative and cannot be changed by a session. For stdio compatibility only, a roots-capable client may provide dynamic MCP roots when the process starts with no directory arguments. Streamable HTTP disables client roots and every HTTP session shares the same process-wide configured directories.

The custom tunnel-oriented changes include authoritative process roots, Windows drive-root handling, optional local execution tools, a shared encoding/BOM-aware streaming text core, deterministic secure traversal, durable atomic mutations, transport-independent typed operation errors, bounded ordered concurrency and aggregate output budgets, shared process preparation, a transport-independent server builder, a fail-closed native Streamable HTTP transport, and an authoritative tool-metadata catalog. The upstream project remains the source of the original encoding-aware file-tool implementation.

## Current Release Status

Version `2.0.0` completes the planned 2.x API cleanup, bounded-memory text pipeline, transport-independent server architecture, fail-closed native Streamable HTTP transport, cross-platform CI, reproducible packaging, and migration documentation. The release exposes the same 23-tool catalog through stdio and native Streamable HTTP while preserving process-wide allowed-directory policy and disabled-by-default execution tools.

Release binaries and archives are produced for Windows, Linux, and macOS on amd64 and arm64. Deployment, credential rotation, service restart, and active rollback remain operator-controlled actions documented in [docs/ROADMAP.md](docs/ROADMAP.md) and [docs/PUBLISHING.md](docs/PUBLISHING.md).

Encoding detection is content-based. File extensions are not used to select or bias an encoding. Unicode BOMs and valid UTF-8 are authoritative. BOMless UTF-16 LE/BE is auto-detected only when structural and decoded-text evidence agree. Empty files are treated as assumed UTF-8; non-empty ambiguous input is reported explicitly and requires an `encoding` override in text operations.

The semantic-tag release workflow validates each release tag against a dated changelog entry before generating binaries, archives, checksums, and Registry metadata.

## What It Does

Provides 23 tools for file operations, encoding conversion, update checks, and optional local execution:
- [`read_text_file`](TOOLS.md#read_text_file) - Stream decoded text with bounded line and output memory
- [`read_multiple_files`](TOOLS.md#read_multiple_files) - Read files in deterministic order under one aggregate decoded-output budget
- [`write_file`](TOOLS.md#write_file) - Write through the shared encoder with explicit `auto`/`always`/`never`/`preserve` BOM policy
- [`edit_file`](TOOLS.md#edit_file) - Encoding/BOM-aware full-document edits with a hard configured size limit
- [`copy_file`](TOOLS.md#copy_file) - Copy a file to a new location
- [`delete_file`](TOOLS.md#delete_file) - Delete a file
- [`list_directory`](TOOLS.md#list_directory) - Browse directories with pattern filtering
- [`tree`](TOOLS.md#tree) - Compact deterministic tree through the shared secure walker (85% fewer tokens than JSON)
- [`search_files`](TOOLS.md#search_files) - Deterministic glob search that skips symlink, junction, and reparse-point escapes
- [`grep_text_files`](TOOLS.md#grep_text_files) - Deterministic streaming regex search with bounded context and aggregate retained state
- [`detect_encoding`](TOOLS.md#detect_encoding) - Auto-detect file encoding with confidence score
- [`convert_encoding`](TOOLS.md#convert_encoding) - Stream decoder-to-encoder conversion into durable staging with exact no-op suppression
- [`detect_line_endings`](TOOLS.md#detect_line_endings) - Stream CRLF/LF/mixed detection with bounded inconsistent-line output
- [`change_line_endings`](TOOLS.md#change_line_endings) - Stream LF/CRLF conversion while preserving encoding, BOM, and unrelated bytes
- [`manage_bom`](TOOLS.md#manage_bom) - Inspect a bounded prefix or stream BOM add/strip through durable staging
- [`list_encodings`](TOOLS.md#list_encodings) - Show all supported encodings
- [`get_file_info`](TOOLS.md#get_file_info) - Get file/directory metadata
- [`create_directory`](TOOLS.md#create_directory) - Create directories recursively (mkdir -p)
- [`move_file`](TOOLS.md#move_file) - Move or rename files and directories
- [`list_allowed_directories`](TOOLS.md#list_allowed_directories) - Show accessible directories
- [`run_script`](TOOLS.md#run_script) - Execute a supported script or executable inside an allowed directory when explicitly enabled
- [`shell`](TOOLS.md#shell) - Execute an unrestricted shell command when explicitly enabled
- [`check_for_updates`](TOOLS.md#check_for_updates) - Check the latest release of this fork with a cached GitHub request

**Supported encodings (24 total):**
- **Unicode:** UTF-8, UTF-16 LE, UTF-16 BE
- **Cyrillic:** Windows-1251, KOI8-R, KOI8-U, CP866, ISO-8859-5
- **Western European:** Windows-1252, ISO-8859-1, ISO-8859-15
- **Central European:** Windows-1250, ISO-8859-2
- **Greek:** Windows-1253, ISO-8859-7
- **Turkish:** Windows-1254, ISO-8859-9
- **Chinese:** GBK, GB18030
- **Other:** Hebrew (Windows-1255), Arabic (Windows-1256), Baltic (Windows-1257), Vietnamese (Windows-1258), Thai (Windows-874)

`manage_bom` additionally recognizes UTF-32 LE/BE BOM signatures, but UTF-32 is not one of the 24 registered read/write encodings.

See [TOOLS.md](TOOLS.md) for detailed parameters and examples.

**Security:** File operations and `run_script` paths are restricted to allowed directories. Recursive filesystem tools resolve every visited entry through a shared secure walker and skip symlinks, Windows junctions, and other reparse points that resolve outside those directories. Mutation handlers revalidate paths before commit and use optimistic snapshots plus atomic or no-replace platform operations. Before `run_script` starts, its script and working directory are revalidated and the script's metadata plus SHA-256 snapshot must still match; this reduces but cannot eliminate the final path-based TOCTOU window without handle-relative execution. The optional `shell` tool revalidates only its working directory; the command itself remains unrestricted and runs with the operating-system permissions of the MCP server process.

## Custom Fork Changes

This repository has evolved from its original upstream codebase. Compared with that baseline, the current source branch adds:

- optional `run_script` and `shell` MCP tools, disabled by default, with shared bounded process preparation but separate authorization policies;
- an authoritative embedded tool catalog consumed by runtime registration and Registry manifest generation, with drift tests for runtime metadata and documentation coverage;
- CLI-provided allowed directories as the authoritative fallback for tunnel clients that do not implement MCP roots requests;
- correct validation of descendants when a Windows drive root such as `D:\` is allowed;
- encoding-aware `detect_line_endings` and byte-preserving `change_line_endings` support for all 24 registered encodings, including UTF-16 LE/BE;
- real upstream encoding fixtures covering every registered encoding, including UTF-16 and GBK/GB18030 round-trip tests;
- conservative, extension-independent BOMless UTF-16 LE/BE detection with malformed-Unicode rejection, binary false-positive protection, deterministic mode semantics, and surrogate-pair handling across chunk boundaries;
- a shared document encoder used by edits, full writes, and encoding conversions, with public `auto`, `always`, `never`, and `preserve` BOM policies plus byte-identical conversion no-op suppression;
- a deterministic, cancellation-aware secure walker shared by `tree`, `search_files`, and `grep_text_files`, including native Windows junction/reparse-point resolution and protection for deeply nested missing paths behind escaping links;
- a shared atomic mutation layer for write, edit, conversion, line-ending, BOM, copy, move, and delete operations, with synced staging, transactional backups, no-replace destination commits, cleanup, and practical concurrent-modification detection;
- transport-independent typed operation errors for path validation, access control, encoding, decoding, output encoding, permissions, conflicts, cancellation, limits, and filesystem failures, with centralized MCP and batch mapping that preserves public messages and schemas;
- a shared bounded ordered worker coordinator used by `read_multiple_files` and `grep_text_files`, with deterministic commits, cancellation-aware dispatch, aggregate output/state budgets, and early stop for global match limits;
- a bounded-memory text pipeline with incremental decoding for all 24 encodings, 16 MiB decoded-line limits, SHA-256 read sessions, reader-based mutation staging, and hard configured limits for full-document editing;
- an explicit process configuration and shared server builder separated from transport startup, with a lifecycle-aware stdio runner, signal cancellation, explicit `stdio` transport selection, and equivalence tests across multiple connections to the same process-wide tool and root policy;
- native stateful Streamable HTTP with mandatory bearer authentication, exact Host/Origin validation, loopback defaults, bounded sessions and request resources, redacted access logging, and a second execution opt-in;
- release hardening with pinned cross-platform CI, reproducible GoReleaser archives, checksum-driven Registry publication, a non-root transport-neutral container, migration documentation, and sanitized public launch examples.

See [CHANGELOG.md](CHANGELOG.md) for the maintained list of fork-specific changes.

`server.template.json` contains only the fork-owned MCP Registry identity and release-neutral placeholders. On a fork release, the registry workflow downloads the published `checksums.txt`, generates a temporary `server.json` with the exact release URLs and SHA-256 values, and publishes only after every expected binary is represented.

## Installation

### ChatGPT Web through the OpenAI Secure MCP Tunnel

The currently validated deployment is Windows plus the OpenAI tunnel client. The tunnel launches this fork as a local stdio MCP process and bridges it to the remote connector used by ChatGPT Web.

Requirements:

- Windows PowerShell 5.1 or later;
- the official OpenAI [`tunnel-client`](https://github.com/openai/tunnel-client) executable;
- a Windows build of this fork;
- an OpenAI Runtime API key with the tunnel permissions required by your OpenAI configuration;
- a valid Tunnel ID;
- one explicit local directory to expose to the MCP server.

This project uses OpenAI's official Secure MCP Tunnel client, not a third-party tunnel implementation. See the [official OpenAI tunnel-client repository](https://github.com/openai/tunnel-client) and the [OpenAI Secure MCP Tunnel guide](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels) for tunnel installation, permissions, control-plane setup, and current product requirements.

The official client is the customer-run agent that connects a private or localhost MCP server to OpenAI-hosted products while keeping the MCP server off the public internet.

#### Build the fork locally

```powershell
git clone https://github.com/zoster81/mcp-file-tools.git
Set-Location .\mcp-file-tools
go test ./...
go build -o mcp-file-tools_windows_amd64.exe ./cmd/mcp-file-tools
```

The Go module is `github.com/zoster81/mcp-file-tools`, and all internal imports resolve through the fork namespace. Build from source for development commits; use only fork-owned release tags with matching assets for packaged installations.

#### Download a fork release

Published fork releases provide a directly downloadable Windows binary:

```powershell
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools" | Out-Null
Invoke-WebRequest `
    "https://github.com/zoster81/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" `
    -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools_windows_amd64.exe"
```

For unreleased development commits, build from source as shown above.

#### OpenAI Tunnel quick start

A sanitized English example is provided at [`examples/start-openai-tunnel.ps1`](examples/start-openai-tunnel.ps1). It is intentionally a single-transport stdio reference; an operator may combine stdio and native HTTP startup in a private launcher outside the repository.

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

The tunnel identifier must be `tunnel_` followed by exactly 32 lowercase hexadecimal characters. Never commit the edited script. The example selects stdio explicitly and keeps `run_script` and `shell` disabled by default.

To enable script execution for supported files located inside an allowed directory, change:

```powershell
$EnableRunScript = $true
```

To enable unrestricted shell commands, change:

```powershell
$EnableShell = $true
```

`run_script` validates the script path and working directory against the allowed roots, but the launched process is not sandboxed. `shell` validates only its working directory; the command itself can access anything permitted to the Windows identity running the tunnel. Enable these capabilities only for a trusted connector and after reviewing [TOOLS.md](TOOLS.md#execution-tools).

Run the test from Windows PowerShell with the complete one-line command:

```powershell
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$env:LOCALAPPDATA\OpenAI-Mcp-Tunnel\start-openai-tunnel.ps1"
```

From Command Prompt, use:

```bat
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\OpenAI-Mcp-Tunnel\start-openai-tunnel.ps1"
```

The script validates paths and placeholders, runs `tunnel-client doctor --explain`, then starts the tunnel with the local operator UI at `http://127.0.0.1:8080/ui`. This validated tunnel workflow continues to use the server's stdio transport even though the same binary also supports native Streamable HTTP.

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

The transport can be selected explicitly with `--transport=stdio` or `MCP_TRANSPORT=stdio`. A roots-capable stdio client may provide workspace directories dynamically only when the process starts without directory arguments. Once directories are configured at startup, they remain the authoritative process-wide set.

### Native Streamable HTTP

The native HTTP transport is stateful, bearer-authenticated, and bound to loopback by default. Every session shares the directory arguments supplied when the process starts; HTTP clients cannot add or change roots. The tracked HTTP launcher is a standalone reference even when a private deployment launcher starts both transports.

Create a private token file and start the endpoint from PowerShell:

```powershell
$tokenPath = Join-Path $env:TEMP "mcp-file-tools.token"
$tokenBytes = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try { $rng.GetBytes($tokenBytes) } finally { $rng.Dispose() }
[System.IO.File]::WriteAllText(
    $tokenPath,
    [Convert]::ToBase64String($tokenBytes),
    [System.Text.UTF8Encoding]::new($false)
)

$env:MCP_HTTP_TOKEN_FILE = $tokenPath
$env:MCP_HTTP_ADDR = "127.0.0.1:8765"
.\mcp-file-tools_windows_amd64.exe --transport=streamable-http D:\Projects
```

The MCP endpoint is `http://127.0.0.1:8765/mcp`. Clients must send the token as `Authorization: Bearer <token>` on every MCP `POST`, `GET`, and `DELETE` request. `/healthz` and `/readyz` expose only minimal liveness/readiness status. A complete sanitized Windows launcher with loopback defaults, optional TLS/proxy settings, environment restoration, and both execution gates disabled is available at [`examples/start-streamable-http.ps1`](examples/start-streamable-http.ps1).

`MCP_HTTP_TOKEN` and `MCP_HTTP_TOKEN_FILE` are cleared from the server process environment immediately after startup configuration is validated, preventing optional execution tools from inheriting the credential. The token itself remains fixed for the process lifetime; rotation requires a controlled restart.

Do not put tokens in command-line arguments, URLs, cookies, or query parameters. Browser CORS is disabled. Non-loopback listeners require explicit opt-in plus TLS or an explicitly trusted proxy boundary. See [`docs/HTTP_SECURITY.md`](docs/HTTP_SECURITY.md) for the complete deployment and threat model.

### Container image

The repository Dockerfile uses the Go version declared by `go.mod`, a version-pinned Alpine runtime, a statically linked binary, and an unprivileged runtime identity (`10001:10001`). The container working directory is `/data`; cache and temporary files use `/tmp/mcp-file-tools`. The image remains transport-neutral, so its entry point is the server binary and callers select stdio or Streamable HTTP explicitly.

Build a development image with an explicit embedded version:

```bash
docker build --build-arg VERSION=dev -t mcp-file-tools:dev .
```

A hardened stdio invocation mounts exactly one allowed root and keeps the rest of the container filesystem read-only:

```bash
docker run --rm -i \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  --mount type=bind,source=/absolute/project,target=/data \
  mcp-file-tools:dev --transport=stdio /data
```

The mounted directory must be accessible to UID/GID `10001`. For native HTTP, mount the workspace at `/data`, mount the bearer token and TLS files read-only under `/run/secrets`, publish port `8765`, and supply the fail-closed non-loopback/TLS settings documented above. A direct-TLS deployment can use an orchestration health check equivalent to:

```text
wget --no-check-certificate --spider -q https://127.0.0.1:8765/healthz
```

The Dockerfile intentionally does not bake in a health check because stdio has no HTTP endpoint. HTTP orchestrators should use `/healthz` for liveness and `/readyz` for readiness; stdio supervisors should monitor the process lifecycle. `SIGTERM` is the declared container stop signal and reaches the server's graceful-shutdown path.

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

### Project lineage

This project originated from the [original upstream repository](https://github.com/dimitar-grigorov/mcp-file-tools). The fork now owns its module path, release pipeline, update source, and MCP Registry namespace. Upstream integrations are separate from this distribution and are retained here only as historical lineage.

## How to Use

Once the connector is active, ask ChatGPT Web or the connected MCP client:
- "List all .pas files in the allowed project directory"
- "Read config.ini and detect its encoding"
- "Show all supported encodings"
- "Read MainForm.dfm using CP1251 encoding"
- "Detect this extensionless file's encoding and line endings"
- "Convert data.legacy from mixed endings to CRLF without changing its encoding or BOM"
- "Convert multilingual.data from UTF-8 to UTF-16 LE with `bom: auto` and create a backup"

**Security:** File tools access only explicitly allowed directories:
- **OpenAI Tunnel:** the directory arguments embedded in `MCP_COMMAND` are the authoritative process-wide set;
- **roots-capable stdio clients:** client-provided roots are accepted only when the process starts without configured directories;
- **multiple sessions:** every connection to one process shares the same allowed directories; prompt instructions may narrow an agent's intended write scope but are not server-enforced ACLs;
- **execution tools:** `run_script` validates its script and working-directory paths, while `shell` validates only its working directory and is otherwise unrestricted.

## Configuration

The server can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_TRANSPORT` | Process transport selection: `stdio` or `streamable-http`. The CLI `--transport` option takes precedence. | `stdio` |
| `MCP_HTTP_ADDR` | Native HTTP listen address. Only `localhost` or an IP literal is accepted; non-loopback requires explicit opt-in. | `127.0.0.1:8765` |
| `MCP_HTTP_PATH` | Clean absolute MCP endpoint path, distinct from `/healthz` and `/readyz`. | `/mcp` |
| `MCP_HTTP_TOKEN` | Bearer token supplied through the environment. Exactly one token source is required for HTTP. | unset |
| `MCP_HTTP_TOKEN_FILE` | Preferred bearer-token source; must reference a regular readable file. Mutually exclusive with `MCP_HTTP_TOKEN`. | unset |
| `MCP_HTTP_ALLOWED_HOSTS` | Additional comma-separated exact `Host` values. Wildcards and suffix matching are rejected. | listener-derived |
| `MCP_HTTP_ALLOWED_ORIGINS` | Comma-separated exact browser origins. Empty rejects every request carrying `Origin`; no CORS allow headers are emitted. | empty |
| `MCP_HTTP_ALLOW_NON_LOOPBACK` | Explicit opt-in required for a non-loopback listener. | disabled |
| `MCP_HTTP_TLS_CERT_FILE` | TLS certificate for direct HTTPS. Must be configured with `MCP_HTTP_TLS_KEY_FILE`. | unset |
| `MCP_HTTP_TLS_KEY_FILE` | TLS private key for direct HTTPS. Must be configured with `MCP_HTTP_TLS_CERT_FILE`. | unset |
| `MCP_HTTP_TRUSTED_PROXY_CIDRS` | Comma-separated proxy networks permitted to supply a bounded `X-Forwarded-For` chain. | empty |
| `MCP_HTTP_MAX_BODY_BYTES` | Maximum body size of one HTTP POST. | `16777216` |
| `MCP_HTTP_MAX_INFLIGHT_BODY_BYTES` | Aggregate reservation budget for concurrent HTTP POST bodies. | `67108864` |
| `MCP_HTTP_MAX_CONCURRENT_REQUESTS` | Maximum simultaneous non-SSE HTTP handlers. SSE streams remain bounded by `MCP_MAX_SESSIONS`. | `64` |
| `MCP_HTTP_SESSION_TIMEOUT` | Idle lifetime of a stateful HTTP session. | `15m` |
| `MCP_HTTP_ENABLE_EXECUTION` | Additional HTTP-only gate required before `run_script` or `shell` can use their existing authorization flags. | disabled |
| `MCP_DEFAULT_ENCODING` | Default encoding for newly created files when `write_file` is called without `encoding`. Existing files keep a confidently detected encoding. Legacy encodings such as `cp1251` remain available as explicit overrides. | `utf-8` |
| `MCP_MAX_FILE_BYTES` | Hard source-size limit for full-document operations such as `edit_file`. | `67108864` |
| `MCP_MAX_DECODED_CHARACTERS` | Maximum decoded characters returned by `read_text_file`. | `16777216` |
| `MCP_MAX_LINE_BYTES` | Maximum bytes in one decoded UTF-8 line. | `16777216` |
| `MCP_MAX_BATCH_FILES` | Maximum paths accepted by `read_multiple_files`. | `256` |
| `MCP_MAX_MATCHES` | Server maximum for `grep_text_files.maxMatches`. | `10000` |
| `MCP_MAX_OUTPUT_BYTES` | Aggregate read output, retained grep state, and inconsistent-line output budget. | `67108864` |
| `MCP_MAX_SESSIONS` | Maximum live native Streamable HTTP sessions. | `128` |
| `MCP_MEMORY_THRESHOLD` | Deprecated fallback for `MCP_MAX_FILE_BYTES` and `MCP_MAX_OUTPUT_BYTES`; specific variables take precedence. | unset |
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
- **Extensionless or custom-format text** (UTF-16, Windows code pages, ISO-8859, or UTF-8): detect from content and use an explicit encoding when evidence is ambiguous
- **Visual Basic 6** (Windows-1252): Forms and config files with Western European characters
- **Legacy PHP/HTML** (CP1251, ISO-8859-1): Web apps with localized content
- **Old config files** (Various): INI, properties, registry files with legacy encodings

**How it works:**
```
User: Read config.ini and change the title to "Настройки"
Assistant: [read_text_file with cp1251] → [modify UTF-8] → [write_file with cp1251]
```

The original encoding can be preserved while the public `bom` policy controls BOM output explicitly. The default `auto` policy writes UTF-8 and legacy encodings without BOM and UTF-16 LE/BE with their canonical BOM; use `preserve` when BOM presence must match an existing file.

## Contributing

Contributor workflow is documented in [`CONTRIBUTING.md`](CONTRIBUTING.md). The intentional 1.8-to-2.0 API changes are listed in [`docs/MIGRATION_2.0.md`](docs/MIGRATION_2.0.md). Coding agents should read the root [`AGENTS.md`](AGENTS.md) and the nearest scoped `AGENTS.md` before editing a subtree. Public planning and verification gates remain in [`docs/ROADMAP.md`](docs/ROADMAP.md) and [`docs/DEVELOPMENT_CHECKLIST.md`](docs/DEVELOPMENT_CHECKLIST.md).

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
# Specify transport and allowed directory
go run ./cmd/mcp-file-tools --transport=stdio /path/to/project
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
