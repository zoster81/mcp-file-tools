# Migration from 1.8 to 2.0

This guide records intentional public API changes planned for `mcp-file-tools` 2.0. Development builds may contain these changes before the final `v2.0.0` release.

## Breaking-change table

| 1.8 behavior | 2.0 behavior | Migration |
|---|---|---|
| `directory_tree` was exposed as a deprecated JSON-tree tool. | `directory_tree` is removed. The server exposes 23 tools. | Use `tree`. Rename `excludePatterns` to `exclude`; consume its compact `tree`, `fileCount`, `dirCount`, and `truncated` fields instead of parsing JSON stored inside a string. |
| `detect_encoding` returned `has_bom`; `manage_bom` returned `hasBom`. | Both tools return `hasBOM`. | Update JSON consumers to read `hasBOM`. Other BOM fields remain camelCase. |
| Single-tool failures exposed only human-readable text. Batch read failures used a smaller code vocabulary. | Every MCP tool error includes `_meta.errorCode`; batch items use the same vocabulary. | Branch on the stable code and retain the text for diagnostics. |
| Ambiguous non-empty content could fall back silently to UTF-8 in text tools. | `detect_encoding` reports `ambiguous: true`; other text tools fail with `ENCODING_AMBIGUOUS` until `encoding` is supplied explicitly. | Call `detect_encoding`, inspect `ambiguous`, and pass an explicit registered encoding when needed. |
| Empty files were detected as UTF-8 without explaining that the result was conventional. | `detect_encoding` returns `encoding: "utf-8"`, `confidence: 0`, and `assumed: true`. Text tools treat empty input as UTF-8. | No override is required unless the caller intends to create content in another encoding. |
| `MCP_MEMORY_THRESHOLD` controlled several unrelated limits. | Separate hard limits control file input, decoded characters, lines, batches, matches, output, and future HTTP sessions. | Set the specific variables below. `MCP_MEMORY_THRESHOLD` remains a deprecated fallback for file and output byte limits during migration. |
| UTF-32 BOMs could be detected or added/removed but UTF-32 was not a registered text encoding. | UTF-32 remains BOM-management only. | Use `manage_bom` for UTF-32 signatures. Convert UTF-32 content externally before calling read, edit, grep, or conversion tools. |

## Stable error codes

Single-tool errors expose the code at `_meta.errorCode`. `read_multiple_files.results[].errorCode` uses the same values:

- `INVALID_INPUT`
- `INVALID_PATH`
- `ACCESS_DENIED`
- `SYMLINK_ESCAPE`
- `NOT_FOUND`
- `PERMISSION`
- `ENCODING`
- `ENCODING_AMBIGUOUS`
- `CONFLICT`
- `CANCELLED`
- `LIMIT`
- `IO_ERROR`
- `INTERNAL_ERROR`
- `OPERATION_FAILED`

`OPERATION_FAILED` is the fallback for errors that do not yet have a more specific domain category. Successful results do not include an error code.

## Configurable limits

All values must be positive decimal integers.

| Variable | Default | Scope |
|---|---:|---|
| `MCP_MAX_FILE_BYTES` | 67,108,864 | Full-document operations such as `edit_file`. |
| `MCP_MAX_DECODED_CHARACTERS` | 16,777,216 | Maximum returned decoded characters for `read_text_file`; a smaller request `maxCharacters` is allowed. |
| `MCP_MAX_LINE_BYTES` | 16,777,216 | Maximum bytes in one decoded UTF-8 line. |
| `MCP_MAX_BATCH_FILES` | 256 | Maximum paths accepted by `read_multiple_files`. |
| `MCP_MAX_MATCHES` | 10,000 | Server maximum for `grep_text_files.maxMatches`. The request default remains 1,000. |
| `MCP_MAX_OUTPUT_BYTES` | 67,108,864 | Aggregate read output, retained grep state, and inconsistent-line output. |
| `MCP_MAX_SESSIONS` | 128 | Reserved hard limit for native Streamable HTTP sessions when that transport is implemented. |
| `MCP_MEMORY_THRESHOLD` | — | Deprecated fallback for `MCP_MAX_FILE_BYTES` and `MCP_MAX_OUTPUT_BYTES`. Specific variables take precedence. |

## Schema review

Apart from the changes listed above, existing 1.8 input and output field names remain unchanged in R10. Optional fields continue to be omitted when they do not apply. The 2.0 schema tests reject snake_case output tags and verify that runtime registration matches the authoritative tool catalog.
