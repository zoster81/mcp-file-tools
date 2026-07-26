# Filesystem Subsystem Agent Guide

This guide applies to `internal/filesystem/`. Follow the repository root [`AGENTS.md`](../../AGENTS.md) first.

## Safety and durability invariants

- Treat every path and filesystem result as untrusted.
- Use `internal/security` for path resolution and allowed-root enforcement; do not add weaker local prefix checks.
- Recursive traversal must remain deterministic, cancellation-aware, and fail closed for symlinks, Windows junctions, and other reparse points that escape allowed roots.
- Mutations stage data in the destination directory, sync staged content before commit, and use the platform-specific atomic or no-replace primitive.
- Existing targets use optimistic snapshots. Initially missing destinations must not overwrite a path created concurrently.
- Preserve source permissions and timestamps where the public operation promises them.
- Backup replacement must remain transactional. On target failure, restore the previous backup or preserve an explicit recovery artifact.
- Cleanup errors must not silently hide the primary operation error; use joined errors where appropriate.
- Keep platform-specific namespace and sync behavior in the existing OS-specific files.

## Design guidance

Prefer small reusable primitives over handler-specific mutation paths. A new mutation API must define:

- expected-state semantics;
- overwrite/no-replace behavior;
- staging location and permissions;
- file and directory sync policy;
- cancellation boundary;
- rollback and cleanup behavior;
- concurrent source or destination changes;
- Windows, Linux, and macOS behavior.

Do not claim complete atomicity across operations that the operating system cannot provide. Preserve explicit TOCTOU limitations.

## Tests

Use temporary directories and injected `mutationOps` where failure simulation is required. Cover:

- staging, sync, close, commit, and cleanup failures;
- pre-existing and concurrently created destinations;
- source changes during copy or preparation;
- backup creation, replacement, rollback, and rollback failure;
- permissions, modification times, and regular-file checks;
- Unix directory sync and Windows no-replace/write-through behavior;
- deterministic traversal, cancellation, max depth, exclusion, and escaping links.

Never weaken a security or durability assertion to accommodate platform setup. Fix the implementation or isolate the platform-specific prerequisite.

## Verification

```bash
go test ./internal/filesystem -count=1
go test ./filetoolsserver/handler -count=1
go test ./... -count=1
git diff --check
```

Run the race detector and cross-platform builds when concurrency or OS-specific mutation code changes.
