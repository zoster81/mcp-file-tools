# Security Subsystem Agent Guide

This guide applies to `internal/security/`. Follow the repository root [`AGENTS.md`](../../AGENTS.md) first.

## Fail-closed invariants

- Normalize absolute paths before comparison and compare real path components, not textual prefixes.
- Windows comparisons are case-insensitive and must handle drive roots, UNC paths, long-path prefixes, junctions, and other reparse points.
- Existing symlinks or reparse points that cannot be resolved are errors, not missing paths.
- For a missing target, resolve the nearest existing ancestor and project the missing suffix onto that resolved path.
- A path is allowed only when both its requested location and resolved location remain inside an allowed root.
- Prefix lookalikes such as `project2` or `project-backup` must not match `project`.
- Empty, relative, NUL-containing, malformed, or unresolvable paths must fail safely.
- Keep security errors compatible with `internal/operation` categories and handler mappings.

## Change requirements

Any change to normalization, path equality, prefix handling, resolution, or allowed-root semantics requires explicit negative tests. Include Windows-specific tests when behavior differs, even if implementation occurs on another platform.

Do not introduce path exceptions based on filenames, caller identity, tool name, or deployment environment. Do not bypass final revalidation in handlers or filesystem consumers.

Document residual TOCTOU windows honestly; path-based validation cannot be described as handle-relative sandboxing.

## Tests

Cover at least the relevant subset:

- exact roots and descendants;
- sibling and textual-prefix attacks;
- `.` and `..`, mixed separators, quotes, whitespace, and NULs;
- drive roots, drive-letter casing, UNC and long-path forms;
- safe and escaping symlinks, junctions, and reparse points;
- missing descendants behind safe and escaping ancestors;
- existing but unresolvable links;
- multiple allowed roots and empty root sets;
- fuzz invariants for normalization and containment.

Use generic synthetic paths. Never embed private workstation directories in tests.

## Verification

```bash
go test ./internal/security -count=1
go test ./internal/filesystem ./filetoolsserver/handler -count=1
go test ./... -count=1
git diff --check
```

Run Windows-specific tests or at least Windows test compilation when changing platform path behavior.
