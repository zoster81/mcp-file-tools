# Backup Store Subsystem Agent Guide

This guide applies to `internal/backupstore/`. Follow the repository root [`AGENTS.md`](../../AGENTS.md) first.

## Security and durability invariants

- The store is a dedicated internal authority configured only at process startup and must never overlap a public allowed directory in either requested or resolved form.
- Reject relative paths, symlinks, junctions, reparse points, special files, path aliases, and uncertain component resolution.
- Hold one platform-native exclusive writer lock for the complete store lifetime.
- Treat `store.json`, objects, and manifests as immutable. Derived indexes may be rebuilt but are never authoritative.
- Create directories and files with owner-only permissions where supported and never expose internal paths or stored bytes through ordinary logs or MCP results.
- Preserve the ordering invariant that durable objects precede manifests and manifest removal precedes object removal.
- Never add background deletion, implicit quota-triggered garbage collection, automatic rollback, or multi-process writers without a separately approved design.

## Implementation guidance

Keep path validation, locking, descriptor handling, object storage, manifests, indexing, restore, and garbage collection in separate small components. Use bounded streaming reads and explicit size limits. Revalidate file type and stable identity around every no-replace install or immutable read.

Platform-specific locking, reparse detection, and directory synchronization belong in build-tagged files. Failure injection must cover write, sync, close, rename, cleanup, lock, corruption, and crash-recovery boundaries.

## Verification

```bash
go test ./internal/backupstore -count=1
go test ./internal/security ./internal/config ./filetoolsserver/handler ./cmd/mcp-file-tools -count=1
go test ./... -count=1
git diff --check
```

Run the race detector and six-target builds when locking, concurrency, or platform-specific files change.
