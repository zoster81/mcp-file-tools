# Handler Subsystem Agent Guide

This guide applies to `filetoolsserver/handler/`. Follow the repository root [`AGENTS.md`](../../AGENTS.md) first.

## Responsibilities

Handlers are MCP adapters. Keep encoding, filesystem, security, concurrency, execution, and typed-error logic in the corresponding `internal` package whenever it is reusable or policy-bearing.

Preserve the flow:

1. validate and normalize input;
2. validate the requested path against allowed roots;
3. prepare through shared domain primitives;
4. honor context cancellation before mutation or process start;
5. revalidate path and expected state at the commit boundary;
6. map typed errors and return stable MCP output.

## Public behavior

- Treat input/output structs and JSON field names as public API.
- Preserve existing messages and error codes unless an explicit compatibility milestone changes them.
- Keep text operations on the shared `textDocument` pipeline.
- Preserve encoding, BOM, and line-ending semantics promised by each tool.
- Do not add filename- or extension-based encoding behavior.
- Keep `run_script` and `shell` authorization separate even when they share process mechanics.
- Bound batches, matches, output, worker coordination, and memory according to the current operation contract.

## Tool metadata

`internal/toolcatalog/catalog.json` is authoritative for MCP names, titles, descriptions, and annotations. When adding or changing a tool:

- update catalog metadata;
- update registration and schemas;
- add focused handler tests;
- update README and `TOOLS.md` links/sections;
- run catalog and server drift tests;
- update release projection tests when applicable.

Do not register handler-local descriptions that diverge from the catalog.

## Tests

Prefer table-driven tests and `t.TempDir()`. Cover successful behavior plus invalid input, path denial, missing files, permissions, encoding failures, BOM conflicts, cancellation, concurrent modification, cleanup, and platform-specific behavior.

For mutation handlers, verify both returned metadata and bytes on disk. For read/search handlers, verify deterministic ordering, truncation, limits, and stable partial errors.

Keep fixtures generic and content-based. A test filename must not imply special encoding semantics.

## Verification

```bash
go test ./filetoolsserver/handler -count=1
go test ./filetoolsserver ./internal/toolcatalog -count=1
go run test_server.go
go test ./... -count=1
git diff --check
```

Run affected internal package tests first when a handler delegates to changed domain logic.
