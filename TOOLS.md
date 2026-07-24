# Tools Reference

## File Operations

### read_text_file

Read file contents with automatic encoding detection and optional partial reading. UTF-8 files pass through unchanged; other encodings convert to UTF-8. A Unicode transport BOM is removed from returned content and reported separately through `hasBOM` and `bomType`.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): Encoding name (auto-detects if omitted)
- `offset` (optional): Start reading from this line number (1-indexed)
- `limit` (optional): Maximum number of lines to read
- `maxCharacters` (optional): Truncate content at this character count to prevent token overflow

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "offset": 100,
  "limit": 50
}
```

**Response:**
```json
{
  "content": "line 100\nline 101\n...",
  "totalLines": 500,
  "fileSizeBytes": 15234,
  "startLine": 100,
  "endLine": 149,
  "truncated": false,
  "detectedEncoding": "utf-16-le",
  "encodingConfidence": 100,
  "hasBOM": true,
  "bomType": "utf-16-le"
}
```

### read_multiple_files

Read multiple files concurrently through the same encoding/BOM-aware document pipeline used by `read_text_file`. Individual file failures don't stop the operation.

**Parameters:**
- `paths` (required): Array of file paths to read
- `encoding` (optional): Encoding for all files (auto-detected per file if omitted)

**Example:**
```json
{
  "paths": ["/path/to/file1.pas", "/path/to/file2.pas"],
  "encoding": "cp1251"
}
```

**Response:**
```json
{
  "results": [
    {
      "path": "/path/to/file1.pas",
      "content": "program Hello;...",
      "detectedEncoding": "utf-16-le",
      "encodingConfidence": 100,
      "hasBOM": true,
      "bomType": "utf-16-le"
    },
    {
      "path": "/path/to/file2.pas",
      "content": "unit Utils;..."
    }
  ],
  "successCount": 2,
  "errorCount": 0
}
```

### write_file

Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8.

**Parameters:**
- `path` (required): Path to the file
- `content` (required): Content to write
- `encoding` (optional): Target encoding (default: cp1251)

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "content": "program Hello;\nbegin\n  writeln('Zdravei');\nend.",
  "encoding": "cp1251"
}
```

**Response:**
```json
{
  "message": "Successfully wrote 48 bytes to /path/to/file.pas"
}
```

### edit_file

Make line-based edits to a text file through the shared encoding/BOM-aware document pipeline. Supports exact matching and whitespace-flexible matching. Returns a git-style unified diff showing changes.

**Parameters:**
- `path` (required): Path to the file to edit
- `edits` (required): Array of edit operations, each with `oldText` and `newText`
- `dryRun` (optional): If true, returns diff without writing changes (default: false)
- `encoding` (optional): File encoding (auto-detected if not specified)
- `forceWritable` (optional): If true, clears read-only flag before editing (default: false — fails on read-only files)

**Features:**
- Exact text matching (first occurrence)
- Whitespace-flexible matching (ignores leading whitespace differences)
- Preserves original indentation
- Preserves UTF-8/UTF-16 BOM state explicitly
- Preserves CRLF or LF line endings for consistently formatted files
- Skips writes for logical no-op edits, preserving the original bytes across all 24 encodings
- Rejects unrepresentable replacement text before touching the file
- Atomic write (temp file + rename)
- Fails on read-only files by default (set `forceWritable: true` only when user explicitly requests it)

**Example:**
```json
{
  "path": "/path/to/file.go",
  "edits": [
    {
      "oldText": "func oldName()",
      "newText": "func newName()"
    }
  ],
  "dryRun": false
}
```

**Response:**
```json
{
  "diff": "--- /path/to/file.go\n+++ /path/to/file.go\n@@ -1,3 +1,3 @@\n-func oldName()\n+func newName()\n",
  "readOnlyCleared": true
}
```

The `readOnlyCleared` field indicates if the read-only flag was removed (only present when true).

## Directory Operations

### list_directory

List files and directories with optional pattern filtering.

**Parameters:**
- `path` (required): Path to directory
- `pattern` (optional): Glob pattern like `*.pas` or `*.dfm` (default: `*`)

**Example:**
```json
{
  "path": "/path/to/project",
  "pattern": "*.pas"
}
```

**Response:**
```json
{
  "files": ["main.pas", "utils.pas", "forms.pas"]
}
```

### tree

Compact indented tree view optimized for AI/LLM consumption. Uses ~85% fewer tokens than `directory_tree`.

**Parameters:**
- `path` (required): Root directory
- `maxDepth` (optional): Maximum recursion depth (0 = unlimited)
- `maxFiles` (optional): Maximum entries to return (default: 1000)
- `dirsOnly` (optional): Only show directories, not files
- `exclude` (optional): Array of patterns to exclude
- `showEncoding` (optional): Detect and display encoding per file (useful for auditing legacy codebases)

**Example:**
```json
{
  "path": "/path/to/project",
  "maxDepth": 3,
  "exclude": ["node_modules", ".git"]
}
```

**Example with encoding:**
```json
{
  "path": "/path/to/legacy-project",
  "showEncoding": true,
  "exclude": [".git"]
}
```

**Response (with showEncoding):**
```json
{
  "tree": "src/\n  main.pas  [windows-1251]\n  utils.pas  [windows-1251]\nREADME.md  [utf-8]",
  "fileCount": 3,
  "dirCount": 1,
  "truncated": false
}
```

**Response:**
```json
{
  "tree": "src/\n  handler/\n    read.go\n    write.go\n  server.go\nREADME.md",
  "fileCount": 4,
  "dirCount": 2,
  "truncated": false
}
```

### directory_tree (deprecated)

Get a recursive tree view as JSON. **Use `tree` instead for 85% fewer tokens.**

**Parameters:**
- `path` (required): Root directory
- `excludePatterns` (optional): Array of glob patterns to exclude

**Response:**
```json
{
  "tree": "{\"name\":\"project\",\"type\":\"directory\",\"children\":[...]}"
}

### get_file_info

Get metadata about a file or directory (size, timestamps, permissions).

**Parameters:**
- `path` (required): Path to file or directory

### create_directory

Create a directory recursively (like `mkdir -p`). Succeeds if already exists.

**Parameters:**
- `path` (required): Path to directory to create

### move_file

Move or rename files and directories. Fails if destination exists.

**Parameters:**
- `source` (required): Path to move
- `destination` (required): Destination path

### copy_file

Copy a file. Fails if destination exists. Does not copy directories.

**Parameters:**
- `source` (required): Source file path
- `destination` (required): Destination path

### delete_file

Delete a file. Does not delete directories.

**Parameters:**
- `path` (required): Path to delete

### search_files

Recursively search for files and directories matching a glob pattern.

**Parameters:**
- `path` (required): Root directory to search from
- `pattern` (required): Glob pattern (`*.txt` for current dir, `**/*.txt` for recursive)
- `excludePatterns` (optional): Array of patterns to exclude
- `maxResults` (optional): Maximum number of results to return (default: 10000)

**Example:**
```json
{
  "path": "/path/to/project",
  "pattern": "**/*.go",
  "excludePatterns": ["vendor", "node_modules"]
}
```

**Response:**
```json
{
  "files": [
    "/path/to/project/main.go",
    "/path/to/project/src/utils.go"
  ]
}
```

### grep_text_files

Search file contents using regex patterns with encoding support. Supports context lines and concurrent searching.

**Parameters:**
- `pattern` (required): Regular expression pattern to search for
- `paths` (required): Array of file or directory paths to search
- `caseSensitive` (optional): Case-sensitive matching (default: true)
- `contextBefore` (optional): Number of lines to show before each match
- `contextAfter` (optional): Number of lines to show after each match
- `maxMatches` (optional): Maximum total matches to return (default: 1000)
- `include` (optional): Glob pattern to include files (e.g., `*.go`)
- `exclude` (optional): Glob pattern to exclude files (e.g., `*_test.go`)
- `encoding` (optional): File encoding (auto-detected if omitted)

**Example:**
```json
{
  "pattern": "func\\s+\\w+",
  "paths": ["/path/to/project"],
  "include": "*.go",
  "contextBefore": 1,
  "contextAfter": 2,
  "maxMatches": 100
}
```

**Response:**
```json
{
  "matches": [
    {
      "path": "/path/to/project/main.go",
      "line": 15,
      "column": 1,
      "text": "func main() {",
      "before": ["package main"],
      "after": ["    fmt.Println(\"Hello\")", "}"],
      "encoding": "utf-8"
    }
  ],
  "totalMatches": 1,
  "filesSearched": 5,
  "filesMatched": 1,
  "truncated": false
}
```

## Encoding Tools

### detect_encoding

Detect the encoding of a file with confidence percentage. Useful for diagnosing encoding issues (garbled text, � characters).

**Parameters:**
- `path` (required): Path to the file
- `mode` (optional): Detection mode
  - `sample` (default): Read begin/middle/end samples - fast, good for most files
  - `chunked`: Read all chunks with weighted averaging - thorough but slower
  - `full`: Read entire file - most accurate but uses more memory

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "mode": "chunked"
}
```

**Response:**
```json
{
  "encoding": "windows-1251",
  "confidence": 95,
  "has_bom": false
}
```

### convert_encoding

Convert a file from one encoding to another. Reads in source encoding, writes in target encoding.

**Parameters:**
- `path` (required): Path to the file to convert
- `from` (optional): Source encoding (auto-detected if omitted)
- `to` (required): Target encoding
- `backup` (optional): Create a `.bak` backup file before converting (default: false)

**Example:**
```json
{
  "path": "/path/to/file.pas",
  "from": "cp1251",
  "to": "utf-8",
  "backup": true
}
```

**Response:**
```json
{
  "message": "Converted /path/to/file.pas from windows-1251 to utf-8",
  "sourceEncoding": "windows-1251",
  "targetEncoding": "utf-8",
  "backupPath": "/path/to/file.pas.bak"
}
```

### detect_line_endings

Detect line ending style (CRLF/LF/mixed) after decoding the file with encoding support, and find lines with inconsistent endings. This works across all 24 registered encodings, including UTF-16 LE/BE source files.

**Parameters:**
- `path` (required): Path to the file to analyze
- `encoding` (optional): File encoding; auto-detected if omitted. Use an explicit value for ambiguous legacy encodings.

**Example:**
```json
{
  "path": "/path/to/file.mq5",
  "encoding": "utf-16-le"
}
```

**Response:**
```json
{
  "style": "mixed",
  "totalLines": 150,
  "inconsistentLines": [45, 78, 123]
}
```

**Style values:**
- `crlf`: All lines use Windows line endings (\\r\\n)
- `lf`: All lines use Unix line endings (\\n)
- `mixed`: File has both CRLF and LF endings - `inconsistentLines` lists lines with minority style
- `none`: File has no line endings (single line or empty)

**MetaTrader/MQL note:** `.mq4`, `.mq5`, and `.mqh` files are commonly stored as UTF-16 LE with BOM and CRLF line endings. Auto-detection handles BOM-bearing files; use `"encoding": "utf-16-le"` for deterministic handling when the BOM is absent or detection is ambiguous.

### change_line_endings

Convert line endings in a file to LF or CRLF while preserving the original encoding, BOM state, and every byte not belonging to a line-ending sequence. The implementation handles UTF-16 LE/BE code units separately and applies byte-preserving CR/LF replacement to the other registered encodings. Use after `detect_line_endings` to fix mixed or wrong line endings. No-op if the file already uses the target style.

**Parameters:**
- `path` (required): Path to the file
- `style` (required): Target line ending style (`"lf"` or `"crlf"`)
- `encoding` (optional): File encoding; auto-detected if omitted. Use an explicit value for ambiguous legacy encodings.

**Example:**
```json
{
  "path": "/path/to/file.mq5",
  "style": "lf",
  "encoding": "utf-16-le"
}
```

**Response:**
```json
{
  "message": "Converted /path/to/file.mq5 from crlf to lf (3 lines changed)",
  "originalStyle": "crlf",
  "newStyle": "lf",
  "linesChanged": 3
}
```

### manage_bom

Detect, strip, or add Unicode BOM (Byte Order Mark). UTF-8 BOM breaks PHP/shell scripts. UTF-16 files need BOMs for proper detection.

**Parameters:**
- `path` (required): Path to the file
- `action` (required): `"detect"`, `"strip"`, or `"add"`
- `encoding` (required for "add"): BOM encoding — `utf-8`, `utf-16-le`, `utf-16-be`, `utf-32-le`, `utf-32-be`

**Example (detect):**
```json
{
  "path": "/path/to/file.php",
  "action": "detect"
}
```

**Response:**
```json
{
  "message": "BOM detected: utf-8 (3 bytes)",
  "hasBom": true,
  "bomType": "utf-8",
  "bomBytes": 3,
  "changed": false
}
```

**Example (strip):**
```json
{
  "path": "/path/to/file.php",
  "action": "strip"
}
```

**Response:**
```json
{
  "message": "Stripped utf-8 BOM (3 bytes) from /path/to/file.php",
  "hasBom": false,
  "bomType": "utf-8",
  "bomBytes": 3,
  "changed": true
}
```

**Example (add):**
```json
{
  "path": "/path/to/file.txt",
  "action": "add",
  "encoding": "utf-16-le"
}
```

**Response:**
```json
{
  "message": "Added utf-16-le BOM (2 bytes) to /path/to/file.txt",
  "hasBom": true,
  "bomType": "utf-16-le",
  "bomBytes": 2,
  "changed": true
}
```

### list_encodings

Returns all 24 supported encodings with name, aliases, and description.

### list_allowed_directories

Returns directories the server is allowed to access. If empty, add paths as args in config.

### check_for_updates

Checks the latest GitHub release of the `zoster81/mcp-file-tools` fork and returns the current version, latest version, and an update message when applicable.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `force` | boolean | no | When `true`, bypasses the cached result and performs a fresh request |

Without `force`, the result is cached for 30 minutes to avoid repeated GitHub API calls. The cache records the configured release source, so cache data from the upstream repository is ignored. A background update check also runs once when the MCP server initializes.

The checker is notification-only: it never downloads, replaces, installs, or restarts the MCP server. It requires at least one published GitHub Release in the fork; if the fork has no release, the GitHub endpoint returns no latest version and the checker remains silent.

## Execution Tools

The execution tools are fork-specific and disabled by default. Enable only the capability that is required:

| Variable | Effect |
|----------|--------|
| `MCP_ENABLE_RUN_SCRIPT=1` | Enables `run_script` only |
| `MCP_ENABLE_SHELL=1` | Enables `shell` only |
| `MCP_ENABLE_EXECUTION=1` | Enables both tools |

Accepted true values are `1`, `true`, `yes`, `on`, and `enabled`, matched case-insensitively.

Both tools run as child processes of the MCP server, inherit its environment and operating-system permissions, receive closed standard input, and capture stdout and stderr separately. The default timeout is 60 seconds, the maximum is 600 seconds, and each output stream is limited to 256 KiB. On timeout or cancellation, the implementation attempts to terminate the process tree; on Windows it uses `taskkill /T /F`.

### run_script

Executes a script or executable whose path is inside an allowed directory. The optional working directory is also validated. When `cwd` is omitted, the script's parent directory is used.

**Security boundary:** validating the script path does not sandbox the script. Once launched, it runs with the full permissions and environment of the MCP server process and may access resources that the operating system allows.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Script or executable path inside an allowed directory |
| `args` | string[] | no | Arguments passed without shell interpolation |
| `cwd` | string | no | Working directory inside an allowed directory; defaults to the script directory |
| `timeoutSeconds` | integer | no | Timeout from 1 to 600 seconds; defaults to 60 |

**Supported file types and interpreter selection:**

| Extension | Execution behavior |
|-----------|--------------------|
| `.ps1` | `pwsh` when available, otherwise Windows PowerShell; uses `-NoProfile -NonInteractive -ExecutionPolicy Bypass -File` |
| `.bat`, `.cmd` | `cmd.exe /d /s /c` on Windows |
| `.py` | `py -3` when available, otherwise `python`/`python3` |
| `.js`, `.mjs`, `.cjs` | `node` |
| `.sh` | `bash` |
| `.exe`, `.com` | Executed directly |

**Example:**

```json
{
  "path": "D:\\Dev\\project\\verify.ps1",
  "args": ["-Mode", "Fast"],
  "cwd": "D:\\Dev\\project",
  "timeoutSeconds": 120
}
```

### shell

Executes an arbitrary command through a selected shell.

**Critical security warning:** only `cwd` is checked against the allowed directories. The command text is intentionally unrestricted and can read, modify, execute, or access anything permitted to the MCP server's Windows or Unix identity, including paths outside the allowed directories and network resources. Do not enable this tool for untrusted clients or prompts.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | string | yes | Non-empty command interpreted by the selected shell |
| `cwd` | string | no | Working directory inside an allowed directory; defaults to the first allowed directory |
| `shell` | string | no | Shell selector described below |
| `timeoutSeconds` | integer | no | Timeout from 1 to 600 seconds; defaults to 60 |

**Shell selectors:**

| Platform | Default | Accepted values |
|----------|---------|-----------------|
| Windows | Windows PowerShell | `powershell`, `windows-powershell`, `pwsh`, `powershell-core`, `cmd` |
| Other platforms | `sh` | `sh`, `bash`, `pwsh`, `powershell` |

**Example:**

```json
{
  "command": "git status --short",
  "cwd": "D:\\Dev\\project",
  "shell": "powershell",
  "timeoutSeconds": 60
}
```

### Execution result

Both tools return the same result shape:

```json
{
  "workingDirectory": "D:\\Dev\\project",
  "exitCode": 0,
  "stdout": "...",
  "stderr": "...",
  "timedOut": false,
  "outputTruncated": false,
  "durationMillis": 125,
  "executionCancelled": false
}
```

A non-zero exit code, timeout, or cancellation marks the MCP tool result as an error while preserving the structured execution output.

## Supported Encodings

| Name | Aliases | Description |
|------|---------|-------------|
| utf-8 | utf8, ascii | Unicode, no conversion |
| utf-16-le | utf16le, utf-16le | Unicode UTF-16 Little Endian |
| utf-16-be | utf16be, utf-16be | Unicode UTF-16 Big Endian |
| windows-1251 | cp1251 | Windows Cyrillic |
| koi8-r | koi8r | Russian Cyrillic (Unix/Linux) |
| koi8-u | koi8u | Ukrainian Cyrillic (Unix/Linux) |
| ibm866 | cp866, dos-866 | DOS Cyrillic |
| iso-8859-5 | iso88595, cyrillic | ISO Cyrillic |
| windows-1252 | cp1252 | Windows Western European |
| iso-8859-1 | iso88591, latin1 | Latin-1 Western European |
| iso-8859-15 | iso885915, latin9 | Latin-9 Western European (Euro) |
| windows-1250 | cp1250 | Windows Central European |
| iso-8859-2 | iso88592, latin2 | Latin-2 Central European |
| windows-1253 | cp1253 | Windows Greek |
| iso-8859-7 | iso88597, greek | ISO Greek |
| windows-1254 | cp1254 | Windows Turkish |
| iso-8859-9 | iso88599, latin5 | Latin-5 Turkish |
| windows-1255 | cp1255 | Windows Hebrew |
| windows-1256 | cp1256 | Windows Arabic |
| windows-1257 | cp1257 | Windows Baltic |
| windows-1258 | cp1258 | Windows Vietnamese |
| windows-874 | cp874, tis-620 | Windows Thai |
