# Tools Reference

The authoritative 23-tool catalog and 3 guided prompts are transport-independent. Stdio and native stateful Streamable HTTP expose the same schemas, annotations, process-wide allowed directories, limits, execution policy, typed errors, and prompt workflows. Transport setup and security differ, but tool behavior does not; see [README.md](README.md), [docs/PROJECT_DIRECTION.md](docs/PROJECT_DIRECTION.md), and [docs/HTTP_SECURITY.md](docs/HTTP_SECURITY.md).

## Guided Prompts

- `audit_encodings(path)`: read-only project encoding, BOM, ambiguity, and line-ending audit.
- `fix_mojibake(path)`: evidence-driven diagnosis and approval-gated repair of garbled legacy text.
- `migrate_to_utf8(path, pattern?)`: search, batch dry-run, approval, backup-enabled conversion, and final verification workflow.

These prompt concepts and the implementation approaches reviewed are credited to the [original project](docs/PROJECT_DIRECTION.md#reciprocal-feature-exchange); the resulting code is reworked for this fork's tool names, mutation guarantees, limits, and dual-transport server rather than mechanically synchronized. Prompt instructions guide clients; they do not add per-agent filesystem ACLs or bypass tool authorization.

## Error Handling

Reusable domain failures carry transport-independent typed categories for invalid input or paths, access denial, symlink escapes, missing files, permissions, encoding, conflicts, cancellation, limits, and filesystem failures. Every failed MCP tool call preserves human-readable text and adds a stable machine-readable code at `_meta.errorCode`. `read_multiple_files.results[].errorCode` uses the same vocabulary.

Stable codes are `INVALID_INPUT`, `INVALID_PATH`, `ACCESS_DENIED`, `SYMLINK_ESCAPE`, `NOT_FOUND`, `PERMISSION`, `ENCODING`, `ENCODING_AMBIGUOUS`, `CONFLICT`, `CANCELLED`, `LIMIT`, `IO_ERROR`, `INTERNAL_ERROR`, and the fallback `OPERATION_FAILED`. Successful results omit error codes. See [docs/MIGRATION_2.0.md](docs/MIGRATION_2.0.md).

## File Operations

Mutating file tools share a durable filesystem layer. Replacement data is staged in the destination directory, synced before commit, and installed with platform-specific atomic operations. Existing-file snapshots detect practical concurrent modifications; initially missing destinations use no-replace commits. On Unix, containing directories are synced after namespace changes; on Windows, replacement and no-replace moves use write-through flags. These protections reduce but do not eliminate every path-based TOCTOU window.

### read_text_file

Read file contents through the shared incremental decoder with automatic encoding detection and optional partial reading. UTF-8 files pass through unchanged; other registered encodings convert to UTF-8. Empty files are treated as assumed UTF-8; non-empty ambiguous input fails with `ENCODING_AMBIGUOUS` until `encoding` is supplied. A Unicode transport BOM is removed from returned content and reported separately through `hasBOM` and `bomType`. `MCP_MAX_LINE_BYTES`, `MCP_MAX_DECODED_CHARACTERS`, and `MCP_MAX_OUTPUT_BYTES` bound the result.

**Parameters:**
- `path` (required): Path to the file
- `encoding` (optional): Encoding name (auto-detects if omitted)
- `offset` (optional): Start reading from this line number (1-indexed)
- `limit` (optional): Maximum number of lines to read
- `maxCharacters` (optional): Truncate content at this character count to prevent token overflow
- `lineNumbers` (optional): Prefix each returned line with its absolute 1-based line number (default: false)

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

Read multiple files through the same incremental encoding/BOM-aware pipeline used by `read_text_file`. `MCP_MAX_BATCH_FILES` limits input count and `MCP_MAX_OUTPUT_BYTES` bounds aggregate decoded output. The ordered coordinator preserves input order, using parallelism only when the aggregate worst-case output fits the budget. Individual file failures do not stop the operation, and cancellation still produces one stable result for every requested path.

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

**Per-file failure example:**
```json
{
  "path": "/path/to/missing.pas",
  "error": "file not found: /path/to/missing.pas",
  "errorCode": "NOT_FOUND"
}
```

### write_file

Write UTF-8 input text using the selected target encoding through the shared document encoder. The supplied line endings are written exactly as provided. Encoding failures and invalid BOM policies are rejected before filesystem mutation. The result is staged and synced before an atomic commit; existing targets are checked for concurrent changes, and new targets use a no-replace commit so a concurrently created file is not overwritten.

**Parameters:**
- `path` (required): Path to the file
- `content` (required): UTF-8 content to write
- `encoding` (optional): Target encoding. New files use the configured default (`utf-8` by default); existing files preserve a confidently detected encoding. Set `MCP_DEFAULT_ENCODING` or pass `encoding` explicitly for legacy formats such as `cp1251`.
- `bom` (optional): BOM policy — `auto` (default), `always`, `never`, or `preserve`

**BOM policy:**
- `auto`: UTF-8 and legacy encodings have no BOM; UTF-16 LE/BE receive their canonical BOM
- `always`: Require the target encoding's canonical BOM; fails for encodings without BOM support
- `never`: Write no BOM
- `preserve`: Preserve the existing file's BOM presence, using the canonical BOM for the selected target encoding; a new file has no BOM

**Example:**
```json
{
  "path": "/path/to/multilingual.data",
  "content": "title = \"città\"\r\n",
  "encoding": "utf-16-le",
  "bom": "auto"
}
```

**Response:**
```json
{
  "message": "Successfully wrote 36 bytes to /path/to/multilingual.data (encoding: utf-16-le, BOM: auto)",
  "encoding": "utf-16-le",
  "bomPolicy": "auto",
  "hasBOM": true,
  "bomType": "utf-16-le"
}
```

### edit_file

Edit one text file through the shared encoding/BOM-aware document pipeline. Editing and diff generation inherently require full-document state, so the source and any supplied patch are rejected when they exceed `MCP_MAX_FILE_BYTES`. Supply either `edits` or one strict single-file unified `patch`, never both. Edit operations retain exact and whitespace-flexible matching and may opt into bounded fuzzy matching with an explicit similarity threshold; fuzzy edits require one unique best match and fail safely on ambiguity. Returns a git-style unified diff. Non-dry-run edits use the shared synced atomic replacement layer and reject a file that changed after decoding.

**Parameters:**
- `path` (required): Path to the file to edit
- `edits` (conditionally required): Array of operations with `oldText`, `newText`, and optional `similarity` from `0.50` to `1.0`
- `patch` (conditionally required): One strict unified diff for the target file; multi-file patches, creation/deletion, unordered/overlapping hunks, and no-newline markers are rejected
- `dryRun` (optional): If true, returns diff without writing changes (default: false)
- `encoding` (optional): File encoding (auto-detected if not specified)
- `forceWritable` (optional): If true, clears read-only flag before editing (default: false — fails on read-only files)

**Features:**
- Exact text matching (first occurrence)
- Whitespace-flexible matching (ignores leading whitespace differences)
- Opt-in bounded fuzzy matching with deterministic comparison budgets and unique-best-match enforcement
- Strict single-file unified patch application with exact context/deletion validation and bounded hunk count
- Preserves original indentation
- Preserves UTF-8/UTF-16 BOM state explicitly
- Preserves CRLF or LF line endings for consistently formatted files
- Skips writes for logical no-op edits, preserving the original bytes across all 24 encodings
- Rejects unrepresentable replacement text before touching the file
- Durable atomic replacement with same-directory exclusive staging, file sync, path revalidation, and concurrent-modification detection
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

The recursive tools `tree`, `search_files`, and `grep_text_files` use one deterministic, cancellation-aware secure walker. Every traversed entry is resolved before it is exposed to the tool; symlinks, Windows junctions, and other reparse points that resolve outside the allowed directories are skipped. Directory links encountered below the requested root are not followed. Nested `.gitignore` files are respected by default; each must be a bounded regular file inside an allowed root, and callers may opt out explicitly with `respectGitignore: false`.

### list_directory

List files and directories with optional pattern filtering.

**Parameters:**
- `path` (required): Path to directory
- `pattern` (optional): Glob pattern like `*.pas` or `*.dfm` (default: `*`)
- `sortBy` (optional): `name` (default), `mtime`, or `size`
- `reverse` (optional): Reverse the selected deterministic order (default: false)

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

Compact indented tree view optimized for AI/LLM consumption. It returns entries in deterministic lexical traversal order and skips links or reparse points that resolve outside the allowed directories.

**Parameters:**
- `path` (required): Root directory
- `maxDepth` (optional): Maximum recursion depth (0 = unlimited)
- `maxFiles` (optional): Maximum entries to return (default: 1000)
- `dirsOnly` (optional): Only show directories, not files
- `exclude` (optional): Array of patterns to exclude
- `showEncoding` (optional): Detect and display encoding per file (useful for auditing legacy codebases)
- `respectGitignore` (optional): Apply nested `.gitignore` rules (default: true)

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

### get_file_info

Get metadata about a file or directory (size, timestamps, permissions).

**Parameters:**
- `path` (required): Path to file or directory

### create_directory

Create a directory recursively (like `mkdir -p`). Succeeds if already exists.

**Parameters:**
- `path` (required): Path to directory to create

### move_file

Move or rename files and directories with a platform-native no-replace operation. A destination created concurrently is not overwritten. Namespace changes are synced where the platform provides a directory-sync mechanism.

**Parameters:**
- `source` (required): Path to move
- `destination` (required): Destination path

### copy_file

Copy a regular file through exclusive same-directory staging, preserving source permissions and modification time where the platform supports them. The staged data is synced and installed atomically without replacing an existing or concurrently created destination. Does not copy directories.

**Parameters:**
- `source` (required): Source file path
- `destination` (required): Destination path

### delete_file

Delete a file after path revalidation and an optimistic metadata snapshot check, then sync the containing directory where supported. Does not delete directories.

**Parameters:**
- `path` (required): Path to delete

### search_files

Recursively search for files and directories matching a glob pattern through the secure walker. Results are selected with bounded top-K retention, so `maxResults` bounds memory even when sorting globally by modification time or size. Entries resolving outside allowed directories are skipped, and nested `.gitignore` files are respected by default.

**Parameters:**
- `path` (required): Root directory to search from
- `pattern` (required): Glob pattern (`*.txt` for current dir, `**/*.txt` for recursive)
- `excludePatterns` (optional): Array of patterns to exclude
- `respectGitignore` (optional): Apply nested `.gitignore` rules (default: true)
- `maxResults` (optional): Maximum number of retained results (default: 10000)
- `sortBy` (optional): `name`, `mtime`, or `size`; when omitted, preserve the historical deterministic traversal order
- `reverse` (optional): Reverse the selected deterministic order; with omitted `sortBy`, this selects reverse name order

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

Search decoded text incrementally using one regex `pattern` or a `patterns` array combined with OR semantics. UTF-8 and structurally clear UTF-16 LE/BE are auto-detected; ambiguous non-empty input requires an explicit `encoding`. Directory inputs use the `.gitignore`-aware secure walker. Content mode preserves deterministic traversal order and bounded context queues; path/count modes scan each selected file without letting an early high-match file hide later files. `offset + maxMatches` is bounded by `MCP_MAX_MATCHES`, and retained output remains within `MCP_MAX_OUTPUT_BYTES`.

**Parameters:**
- `pattern` (conditionally required): One regular expression
- `patterns` (conditionally required): Array of regular expressions combined with OR semantics; may be used with `pattern`
- `paths` (required): Array of file or directory paths to search
- `caseSensitive` (optional): Case-sensitive matching (default: true)
- `contextBefore` (optional): Number of lines to show before each match
- `contextAfter` (optional): Number of lines to show after each match
- `maxMatches` (optional): Maximum total matches to return (default: 1000)
- `include` / `exclude` (optional): Backward-compatible single glob filters
- `includes` / `excludes` (optional): Arrays of glob filters
- `encoding` (optional): File encoding (auto-detected if omitted)
- `outputMode` (optional): `content` (default), `files_with_matches`, or `count`
- `matchesOnly` (optional): In content mode return only the regex substring in `text`
- `offset` (optional): Zero-based result offset for paging
- `respectGitignore` (optional): Apply nested `.gitignore` rules for directory inputs (default: true)

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

Detect the encoding of a file with confidence percentage. Detection is based on BOMs and content, never on filename or extension. Unicode BOMs and valid UTF-8 are authoritative. Empty files return assumed UTF-8 with confidence 0 and `assumed: true`. Non-empty input without sufficient text evidence returns `ambiguous: true` instead of a forced encoding. UTF-32 BOM signatures may be reported, but UTF-32 remains BOM-management only.

**Parameters:**
- `path` (required): Path to the file
- `mode` (optional): Detection mode
  - `sample` (default): Read begin/middle/end samples - fast, good for most files
  - `chunked`: Stream all chunks, preserving UTF-16 code-unit state and aggregating legacy evidence - thorough but slower
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
  "hasBOM": false
}
```

### convert_encoding

Convert one file or a bounded batch through the selected decoder and target encoder. `dryRun` performs a complete preflight without writing and reports unrepresentable runes with Unicode code point plus 1-based line/column locations. Batch results preserve input order, report per-file success or stable error codes, and may succeed partially. Actual changed output uses synced same-directory staging, byte-identical no-op suppression, optional transactional backup, path revalidation, and concurrent-source-change rejection.

**Parameters:**
- `path` (conditionally required): One file to convert
- `paths` (conditionally required): Bounded array of files; mutually exclusive with `path`
- `from` (optional): Source encoding (auto-detected if omitted)
- `to` (required): Target encoding
- `backup` (optional): Transactionally create or replace a `.bak` backup before committing the conversion (default: false). The backup is staged and synced first; if target commit fails, a previous backup is restored or a newly created backup is removed. If restoration itself fails, the previous backup remains in a recovery staging file whose path is included in the error.
- `bom` (optional): BOM policy — `auto` (default), `always`, `never`, or `preserve`
- `dryRun` (optional): Preview per-file changes and unsupported-character locations without mutation

**BOM policy:**
- `auto`: UTF-8 and legacy targets have no BOM; UTF-16 LE/BE targets receive their canonical BOM
- `always`: Require the target encoding's canonical BOM; fails before mutation for encodings without BOM support
- `never`: Write no BOM
- `preserve`: Preserve source BOM presence using the canonical BOM of the target encoding

**Example:**
```json
{
  "path": "/path/to/multilingual.data",
  "from": "utf-16-le",
  "to": "utf-8",
  "backup": true,
  "bom": "auto"
}
```

**Response:**
```json
{
  "message": "Successfully converted /path/to/multilingual.data from utf-16-le to utf-8 (BOM: auto) (backup: /path/to/multilingual.data.bak)",
  "sourceEncoding": "utf-16-le",
  "targetEncoding": "utf-8",
  "backupPath": "/path/to/multilingual.data.bak",
  "bomPolicy": "auto",
  "hasBOM": false,
  "changed": true
}
```

### detect_line_endings

Detect line ending style (CRLF/LF/mixed) through the shared incremental decoder, and find lines with inconsistent endings. This works across all 24 registered encodings. Uniform files require one pass; mixed files use a second digest-verified pass that retains only minority line numbers. `MCP_MAX_LINE_BYTES` bounds each decoded line and `MCP_MAX_OUTPUT_BYTES` bounds the returned list. Ambiguous non-empty input requires an explicit encoding.

**Parameters:**
- `path` (required): Path to the file to analyze
- `encoding` (optional): File encoding; auto-detected if omitted. Use an explicit value for ambiguous legacy encodings.

**Example:**
```json
{
  "path": "/path/to/extensionless-file",
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

**Detection note:** encoding is inferred from bytes and decoded-content evidence, not from file extensions. UTF-16 LE/BE is auto-detected from a BOM or from conservative structural validation; use an explicit encoding when BOMless evidence is short, malformed, or endian-ambiguous.

### change_line_endings

Stream line-ending conversion to LF or CRLF while preserving the original encoding, BOM state, and every byte not belonging to a line-ending sequence. Shared path, encoding, BOM, and mode validation runs before the specialized conversion; an explicit encoding that conflicts with a Unicode BOM fails before mutation. The bounded transformer preserves CRLF pairs and standalone CR across chunk boundaries, handles UTF-16 LE/BE code units separately, and stages output directly on disk. Changed output uses synced atomic replacement with concurrent-modification detection. Use after `detect_line_endings` to fix mixed or wrong line endings. No-op if the file already uses the target style and preserves the file modification time.

**Parameters:**
- `path` (required): Path to the file
- `style` (required): Target line ending style (`"lf"` or `"crlf"`)
- `encoding` (optional): File encoding; auto-detected if omitted. Use an explicit value for ambiguous legacy encodings.

**Example:**
```json
{
  "path": "/path/to/extensionless-file",
  "style": "lf",
  "encoding": "utf-16-le"
}
```

**Response:**
```json
{
  "message": "Converted /path/to/extensionless-file from crlf to lf (3 lines changed)",
  "originalStyle": "crlf",
  "newStyle": "lf",
  "linesChanged": 3
}
```

### manage_bom

Detect, strip, or add Unicode BOM (Byte Order Mark). Detection reads at most four prefix bytes. Strip and add stream the unchanged body into synced same-directory staging instead of loading the file. UTF-8 BOM breaks PHP/shell scripts. UTF-16 BOMs remain the authoritative and most interoperable encoding signal, although structurally clear BOMless UTF-16 may also be detected. Snapshot verification and path revalidation ensure cancellation or detected concurrent changes leave the original file unchanged.

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
  "hasBOM": true,
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
  "hasBOM": false,
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
  "hasBOM": true,
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

On native Streamable HTTP, the corresponding flag above is necessary but not sufficient: `MCP_HTTP_ENABLE_EXECUTION=1` is also required. This dual opt-in prevents execution enabled for a trusted stdio deployment from being exposed automatically on the network transport.

Both tools use one internal process-preparation primitive for absolute working-directory validation, timeout bounds, closed standard input, bounded stdout/stderr capture, cancellation, and process-tree termination. The primitive does not authorize commands or paths: `run_script` and `shell` retain separate handler policies. Immediately before launch, both revalidate the working directory against the current allowed roots. `run_script` also verifies that the authorized script still matches its prepared metadata and SHA-256 snapshot. The default timeout is 60 seconds, the maximum is 600 seconds, and each output stream is limited to 256 KiB. On timeout or cancellation, Windows termination uses `taskkill /T /F` before the direct process kill.

### run_script

Executes a regular script or executable whose path is inside an allowed directory. The optional working directory is also validated. When `cwd` is omitted, the script's parent directory is used. Script arguments are passed directly to the selected interpreter or executable without shell interpolation. The path, working directory, metadata, and SHA-256 content snapshot are checked again immediately before launch.

**Security boundary:** pre-launch path and digest revalidation reduces replacement races but cannot eliminate the final check-to-exec window without a handle-relative launch primitive. The script is not sandboxed: once launched, it runs with the operating-system permissions and normal environment of the MCP server process and may access resources that the operating system allows. In Streamable HTTP mode, the bearer-token environment variables are removed immediately after startup configuration is snapshotted, so child processes do not inherit those credentials.

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

**Critical security warning:** only `cwd` is checked against the allowed directories. The command text is intentionally unrestricted and can read, modify, execute, or access anything permitted to the MCP server's Windows or Unix identity, including paths outside the allowed directories and network resources. Do not enable this tool for untrusted clients or prompts. Streamable HTTP additionally requires `MCP_HTTP_ENABLE_EXECUTION=1`, and the HTTP bearer-token environment variables are removed before tool execution can start.

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
