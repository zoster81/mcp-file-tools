# Persistent Backup Lifecycle Design

## Status

**Approved R17 design. The ten lifecycle decisions were accepted on 2026-08-04, and phased R18 implementation is now authorized.**

This document is the authoritative security boundary, storage format, lifecycle, restore contract, garbage-collection model, limits, failure semantics, and verification gate for the persistent backup subsystem. Implementation must remain phased so each durability and recovery boundary can be reviewed and verified independently.

R18 phases 1–5 are implemented in source. Phase 1 provides disabled-by-default configuration, strict non-overlapping store-path validation, owner-only permissions, a platform-native lifetime writer lock, an immutable versioned descriptor, and denial of the internal root to ordinary filesystem tools. Phase 2 adds internal exact-byte object capture, strict checksummed manifests, conservative quota reservations, a rebuildable derived index, bounded startup recovery, and quick/full read-only audit primitives. Phase 3 exposes only the bounded read-only `backup_store` status/list/inspect/audit surface. Phase 4 binds preview-only `backupPolicy: "required"` into `edit_file`. Phase 5 adds exact manifest-level package policy, side-effect-free aggregate preflight, atomic conservative all-target reservation, and durable capture of every changed package pre-state before the first commit. Restore, garbage collection, mutable pinning, and automatic rollback remain unavailable.

The existing transactional `.bak` behavior of `convert_encoding` remains unchanged. `edit_file` and `patch_package` create persistent backups only when their approved preview/manifest explicitly binds `backupPolicy: "required"`; omitted policy, direct editing, and logical no-ops continue to create none.

## Goals

The persistent backup subsystem is designed to:

- durably capture exact pre-mutation bytes before an approved mutation can commit;
- deduplicate identical bytes without weakening integrity verification;
- bound total storage, object size, manifests, versions per target, pinned records, diagnostics, and operation time;
- let clients list and inspect bounded backup metadata without exposing store paths or file contents;
- restore exact bytes through one-shot preview/apply with current-state validation;
- preserve the current target before a destructive restore whenever the target exists;
- garbage-collect only through an explicit dry-run/apply plan;
- recover deterministically from interrupted object, manifest, index, restore, and garbage-collection operations;
- preserve stdio and Streamable HTTP equivalence and the process-wide trust model;
- remain disabled unless an operator configures a dedicated store.

## Non-goals

The initial implementation must not provide:

- transparent backup of every write or mutation;
- automatic rollback of a partially committed multi-file package;
- a distributed or network-shared backup service;
- concurrent writers from several server processes;
- per-session or per-agent backup ACLs inside one server process;
- arbitrary restore destinations;
- background or scheduled garbage collection;
- guaranteed secure deletion on SSD, copy-on-write, journaled, snapshotted, or remote filesystems;
- application-managed encryption keys or an encryption-at-rest format;
- direct browsing or mutation of store files through ordinary filesystem tools;
- migration of existing adjacent `.bak` files into the store automatically.

Operators requiring encryption at rest should place the store on an operating-system or volume-level encrypted filesystem. Application-managed encryption requires a separate key-management and rotation design.

## Security boundary

### Dedicated internal storage root

The approved store is an explicit process-wide operator authority separate from public allowed directories.

- `MCP_BACKUP_STORE_DIR` is unset by default; when unset, persistent backup features are unavailable.
- The value must be an absolute path supplied at process startup, never by an MCP request.
- The store path and every existing ancestor are normalized, resolved, and checked for symlinks, junctions, reparse points, and path aliases.
- The store must not equal, contain, or be contained by any public allowed directory after lexical and resolved comparison.
- Ordinary filesystem tools must reject the store and all descendants even if a later roots update would otherwise overlap it.
- The store path is never returned in MCP results or written to ordinary logs.
- If a configured store cannot be validated or exclusively locked, startup fails rather than silently disabling required backup policy.

This deliberate internal filesystem authority was approved as a security-boundary change. R18 phase 1 enforces the path-separation, protected-root, permission, descriptor, and lifetime-lock requirements before later data operations are added.

### Permissions and local trust

- New directories use owner-only permissions where the platform supports them.
- Store metadata, manifests, objects, locks, staging files, and trash entries use owner-only permissions.
- Entries must be regular files or real directories. Links, reparse points, sockets, devices, and other special files fail closed.
- Immutable object installation uses no-replace creation.
- Store-root identity is retained for the process lifetime, operation boundaries revalidate the root and internal layout, and regular store files must have exactly one hard link where the platform exposes reliable primitives.
- A local actor with the same operating-system identity as the server remains able to tamper with the store. Content hashes detect accidental corruption but are not a defense against a fully compromised process identity.

### Process ownership

The initial format supports one writer process at a time.

- Startup acquires one platform-native exclusive store lock.
- A second process using the same store fails startup.
- The lock is held for the process lifetime and released on graceful or operating-system process termination.
- In-process operations are serialized at the store transaction boundary, while expensive source hashing may occur outside the critical section under an explicit reservation.

Multi-process writers, shared network filesystems, and distributed locking are deferred.

## Store format

The approved layout is versioned and contains no target bytes outside content-addressed objects:

```text
<store>/
  store.json
  store.lock
  objects/
    sha256/
      ab/
        abcdef...                 immutable exact bytes
  manifests/
    <backup-id>.json             immutable backup record
  index/
    index-v1.json                derived, replaceable cache
  staging/                       synced temporary files only
  trash/                         GC-renamed manifests and objects
```

### Store descriptor

`store.json` is created once and then treated as immutable except through an explicit future format migration. It contains:

- `formatVersion: "backup-store-v1"`;
- a cryptographically random store identifier;
- creation timestamp in UTC;
- `objectAlgorithm: "sha256"`;
- canonical manifest and index versions.

Unknown fields are rejected. A missing, malformed, unsupported, or replaced descriptor makes the configured store unavailable.

### Objects

Objects contain exact original file bytes.

- Object identity is lowercase SHA-256 of the complete bytes.
- The object path is derived solely from the validated digest.
- Objects are staged, hashed while written, synced, closed, and installed with no-replace semantics.
- If an object already exists, size and SHA-256 are verified before it is referenced.
- Object files are immutable after installation.
- One object may be referenced by many manifests.
- Object size is bounded before capture by the configured maximum and by the source operation's existing file limit.

Digest filenames provide integrity and deduplication, not authorization.

### Backup manifests

A manifest is the immutable source of truth for one captured target state. The implemented internal `backup-manifest-v1` record contains:

- 256-bit random `backupId`;
- store format and manifest version;
- UTC creation time;
- normalized original absolute target path;
- source operation category such as edit, patch package, or restore;
- object algorithm, digest, and byte size;
- original `content-v1` fingerprint;
- original regular-file mode and modification time where meaningful;
- optional bounded user label;
- optional encoding, BOM, and line-ending observations remain deferred until a public review surface needs them;
- immutable pinned state for the first version, or a separately journaled pin record if mutable pinning is approved;
- a canonical manifest checksum.

Restore must revalidate object bytes and current path authorization. It must never trust recorded encoding or metadata without inspecting the object and current target.

### Derived index

The index accelerates bounded listing and quota calculations but is never authoritative.

- Manifests and objects remain the durable source of truth.
- The in-memory index is generated from a deterministic manifest and object scan.
- The persisted `index-v1.json` remains compact: it stores only the generation digest and aggregate counts, never every path or manifest row.
- Index replacement uses synced staging and atomic replacement.
- A missing, corrupt, stale, or tampered index is rebuilt under explicit limits.
- An interrupted index update cannot invalidate a committed manifest.

This avoids a transaction that depends on atomically updating both a manifest and one global mutable database file.

## Configuration and limits

The approved defaults are:

| Setting | Default | Purpose |
|---|---:|---|
| `MCP_BACKUP_STORE_DIR` | unset | Enables the dedicated internal store. |
| `MCP_BACKUP_MAX_TOTAL_BYTES` | `1073741824` | Maximum retained unique object bytes. |
| `MCP_BACKUP_MAX_OBJECT_BYTES` | `67108864` | Maximum bytes in one object. |
| `MCP_BACKUP_MAX_MANIFESTS` | `10000` | Maximum live manifests. |
| `MCP_BACKUP_MAX_VERSIONS_PER_TARGET` | `32` | Maximum unpinned versions retained per target. |
| `MCP_BACKUP_MAX_PINNED` | `256` | Maximum pinned manifests. |
| `MCP_BACKUP_RETENTION_DAYS` | `30` | Age threshold used by GC planning, not automatic deletion. |
| `MCP_BACKUP_PLAN_TTL_SECONDS` | `900` | Lifetime of restore and GC preview capabilities. |

All values must be positive and overflow-safe. Configuration loading enforces hard maxima of 1 TiB total bytes, 1 GiB per object, 1,000,000 manifests, 10,000 versions per target, 100,000 pinned manifests, 3,650 retention days, and 86,400 seconds for plan lifetime; environment values above those maxima fall back to the documented defaults, while invalid direct internal store options fail closed. `MCP_MAX_OUTPUT_BYTES` continues to bound future MCP output, and `MCP_MAX_BATCH_FILES` will bound targets in one backup-integrated package operation.

Phase 2 consumes total-byte, object-size, manifest-count, per-target-version, and immutable-pin limits through conservative process-local reservations. Configuring `MCP_BACKUP_STORE_DIR` still does not automatically create backups because no public mutation path calls the internal capture primitive. Retention and plan-lifetime values remain inactive until GC and restore phases. No quota failure triggers implicit garbage collection.

## Capture transaction

The internal phase-2 capture primitive implements the durable portion of this transaction. Phase 4 connects it only to approval-bound `edit_file` apply; the caller supplies an already normalized and authorized target path, and the backup is committed before the associated target mutation begins.

1. Validate the requested target through current allowed-root policy.
2. Capture a bounded digest-bearing snapshot and stable identity.
3. Compute the object digest while reading exactly the approved source size.
4. Acquire or confirm a quota reservation for worst-case new unique bytes and one manifest.
5. Revalidate the target path, identity, metadata, size, and digest.
6. Install or verify the immutable object.
7. Write, sync, and atomically install the immutable manifest.
8. Sync affected store directories.
9. Update or invalidate the derived index.
10. Release the reservation and return the backup identifier to the prepared mutation.
11. Only then allow the target mutation to stage or commit according to its existing contract.

If object or manifest persistence fails, the target mutation does not begin. A committed backup remains valid even when the later target mutation fails; manifests describe captured state and do not claim that the associated mutation succeeded.

### Reservations and quotas

- Reservations are process-local, bounded, and associated with one active operation.
- Total committed unique object bytes plus live reservations must remain within quota.
- Deduplication may reduce committed bytes, but admission reserves the conservative full object size until an existing object is verified.
- Cancellation or failure releases the reservation.
- Individual and package-wide reservations are implemented. Package admission reserves every changed source at its full byte size plus all manifest, pinned, and per-target version capacity atomically before the first capture; verified deduplication may reduce only the committed object bytes.
- Quota exhaustion is a preflight failure, not a trigger for implicit garbage collection.

## Mutation integration

Persistent backup behavior must be explicit and approval-bound.

### Edit preview/apply

R18 phase 4 implements the additive preview field `backupPolicy: "required"`. The value is accepted only in that exact form and retained inside the one-shot preview capability; apply accepts only `previewId` and therefore cannot weaken, remove, or replace it.

- Omitted policy preserves the no-persistent-backup behavior.
- Required policy is valid only for preview and requires a configured store.
- Preview validates availability and returns the retained policy without creating a backup.
- Apply consumes the capability, revalidates path, identity, and fingerprints, preflights worst-case response size, then captures the exact approved pre-state before permission changes or target replacement.
- Apply returns `backupId` only after durable manifest commit. A later mutation failure remains an error but preserves that identifier in structured output.
- Capture failure, quota exhaustion, cancellation before capture, or manifest mismatch prevents mutation.
- Logical no-ops create no backup because no target mutation follows.
- Direct editing remains unchanged and rejects `backupPolicy`.

### Patch packages

R18 phase 5 implements exact manifest-level `backupPolicy: "required"` for patch packages.

- Inspect validates the exact policy value without requiring a configured store.
- Dry run requires package capture authority, prepares the exact changed set, and performs a side-effect-free conservative aggregate quota preflight without creating objects or manifests.
- The retained one-shot capability binds the policy; apply still accepts only `previewId`.
- Apply stages every changed output, atomically reserves the complete package backup budget, and captures changed pre-states in manifest order.
- Every returned manifest is verified against its expected backup ID shape, normalized target path, `patch_package` source operation, and approved pre-state fingerprint.
- All package targets are revalidated after capture. The first target commit cannot begin until every changed target has a verified durable manifest.
- If capture is incomplete, no server target mutation begins. Any already durable prefix remains valid and its per-target `backupId` is returned.
- A derived-index-only error does not invalidate complete authoritative manifests; apply may continue after all results are verified.
- Package commit remains per-file and may still return `PARTIAL_COMMIT`; all durable IDs remain in structured output, but backups do not make multi-file replacement atomic.
- Logical no-op targets receive no backup. Automatic rollback remains out of scope.

### Existing `.bak` conversion behavior

`convert_encoding.backup=true` continues to use its existing adjacent transactional `.bak` contract. It is not silently redirected into the persistent store. Any future migration must be a separate public API decision with compatibility tests.

## Public management surface

R18 phase 3 implements one always-registered read-only `backup_store` tool, bringing the unreleased source catalog to 27 tools. The initial strict action union contains:

- `status`: no additional fields; when disabled it returns `enabled: false`, and when configured it returns redacted version, health, generation, quota, counts, residue, and bounded path-free issues;
- `list`: optional `cursor`, `limit`, `targetPath`, and `pinned`; pages are newest-first, limited to 100 records, filtered through current root authorization, and use an authenticated keyset cursor bound to filters, the allowed/protected-root policy snapshot, and store generation; target visibility is revalidated on every page;
- `inspect`: required `backupId`; it validates the manifest, verifies current target authorization, and fully hashes the referenced object before returning metadata without bytes;
- `audit`: optional `auditMode=quick|full`, `maxObjects`, and `maxBytes`; requested bounds cannot exceed configured store limits, and the operation never repairs or deletes data.

The tool never accepts a store path, object path, executable, shell command, restore destination, mutation policy, or deletion instruction. Unknown fields and cross-action parameters are rejected. Every response remains within `MCP_MAX_OUTPUT_BYTES`.

Restore and garbage-collection actions remain deferred. Later phases may extend the same tool with `restorePreview`, `restoreApply`, `gcDryRun`, and `gcApply` only after their separate one-shot schemas and safety tests are complete.

## Listing and review

- Results are ordered newest-first by creation time and backup ID.
- Pagination uses an authenticated opaque cursor rather than unbounded client-controlled offsets.
- The cursor is bound to the exact target/pinned filters, current allowed/protected-root policy snapshot, and store generation; target visibility is revalidated on every page, tampering or filter changes fail with `INVALID_INPUT`, and generation changes fail with `CONFLICT`.
- Filters include exact target path and pinned state after current normal path validation.
- Results whose original target is no longer authorized are omitted; inspect fails with `ACCESS_DENIED` for such targets.
- Returned metadata never includes internal store paths or store identifiers.
- File bytes are never returned by list or inspect.
- Diff generation belongs to restore preview and remains subject to encoding, line, file, and output limits.

## Restore preview/apply

Restore is one-shot and state-bound, following the R16 capability model.

### Restore preview

`restorePreview` should:

1. claim no mutation authority yet;
2. validate the backup ID and manifest;
3. verify the referenced object size and SHA-256;
4. require the original target path to be authorized by current roots;
5. capture the current target state, including a stable identity when it exists;
6. prepare exact target bytes and a result fingerprint;
7. return current/result fingerprints, file metadata, bounded diff when safely decodable, and a 256-bit expiring preview ID;
8. retain exact object identity and current target precondition in a bounded process-local cache.

The initial restore destination is the manifest's original target only. Alternate destinations are deferred.

### Restore apply

`restoreApply` accepts only the preview ID.

- The preview is atomically consumed before validation; every outcome is terminal.
- The object, manifest, target path, target identity, and current fingerprint are revalidated.
- If the target exists, its exact current state is captured as a new durable backup before replacement. This safety backup is mandatory and cannot be disabled initially.
- If that safety backup cannot be admitted or persisted, restore does not mutate the target.
- Restored bytes are staged through the durable mutation layer.
- Existing targets use optimistic replacement; missing targets use no-replace creation.
- The final target fingerprint is verified before success.
- A post-replacement sync or verification failure reports actual target state and the safety-backup ID; it does not claim rollback.

A restore never deletes or consumes the source backup.

## Garbage collection

Garbage collection is always explicit preview/apply.

### Policy

A candidate manifest becomes eligible only when all conditions hold:

- it is not pinned;
- it exceeds the configured age threshold or per-target version limit;
- deleting it preserves a configured minimum recent-version floor;
- it is not referenced by an active restore or package reservation;
- the store generation matches the plan's observed generation.

Objects become eligible only when no live manifest references them.

### GC dry run

`gcDryRun` computes a deterministic plan containing bounded manifest IDs, reclaimable unique bytes, object reference changes, and reasons. It stores the exact plan behind a 256-bit expiring capability. It changes nothing on disk.

### GC apply

`gcApply` consumes only the plan ID and revalidates the store generation, pin state, active reservations, and reference counts.

- Manifests are atomically renamed into `trash` before removal.
- Only after manifest references are gone may unreferenced objects be renamed into `trash`.
- Directory sync occurs after each namespace phase.
- Deletion from trash is best effort but cleanup failures are surfaced.
- A crash may leave trash or orphan objects, never a live manifest whose object was intentionally deleted first.
- A stale plan fails with `CONFLICT`; the client creates a new dry run.

## Pinning

Pinning changes retention semantics and therefore requires a crash-consistent design.

The preferred initial choice is immutable pin state at backup creation. Mutable pin/unpin can be added later through append-only pin records or replacement manifests, but must not rewrite the original backup record in place. The final decision requires explicit approval because mutable pinning affects quotas, GC planning, and recovery.

## Startup recovery and degraded state

After acquiring the exclusive lock, initialization now performs a bounded structural scan:

- validate `store.json` and directory types;
- reject links, reparse points, hard-linked regular files, unexpected special files, and path escapes;
- validate manifest filenames, sizes, schemas, checksums, unique IDs, owner-only permissions, and single-link state;
- validate referenced object paths and sizes without necessarily hashing every object;
- rebuild the derived index when missing or stale;
- identify staging files, trash entries, orphan objects, missing objects, and duplicate manifests.

Capture and audit revalidate the retained store-root identity. Capture additionally revalidates the internal layout before staging and again before durable object or manifest installation. Audit reports structural entries without deleting or repairing them.

Full object hashing is performed by internal full audit, public `backup_store.inspect`, and every capture dedup verification. Restore verification remains a later phase.

Structural corruption must not be repaired automatically. The approved fail-closed behavior is:

- if the store is configured and structurally corrupt, server startup fails;
- no ordinary mutation silently proceeds under a configured required-backup policy;
- a future offline repair utility, if added, requires a separate design and never deletes uncertain data by default.

This favors safety over availability. Maintainers may revise this to an online degraded read-only mode only after defining clear operator recovery behavior.

## Integrity audit

The internal audit primitive has two implemented modes:

- `quick`: validate structure, manifest checksums, object presence, size, index consistency, references, staging, trash, and orphan counts;
- `full`: additionally stream and hash every referenced object under explicit object, byte, time, and output limits.

Audit is read-only. It never repairs, deletes, or quarantines data. Phase 3 exposes the same bounded results through `backup_store.audit` without store paths or bytes.

## Failure and error semantics

The implementation should reuse the existing stable error vocabulary unless a later reviewed phase proves that a new public code is essential.

- malformed schemas or unsupported versions: `INVALID_INPUT`;
- path overlap or invalid store/target paths: `INVALID_PATH`, `ACCESS_DENIED`, or `SYMLINK_ESCAPE`;
- stale preview, changed target, changed store generation, or lost lock: `CONFLICT`;
- quota, count, object-size, plan-size, output, or audit bounds: `LIMIT`;
- cancellation: `CANCELLED`;
- permissions: `PERMISSION`;
- object, manifest, sync, corruption, or filesystem failures: `IO_ERROR`;
- target replacement followed by uncertain durability or verification: structured state evidence and, where multi-target state is involved, `PARTIAL_COMMIT`.

Human-readable errors must not expose store paths, object filenames, temporary files, or file contents through MCP or HTTP logs.

## Crash-consistency invariants

The implementation must preserve these invariants at every injected failure point:

1. A live manifest never intentionally references an object that was not durably installed first.
2. A target mutation requiring backup never begins before its manifest is durable.
3. The index is disposable and rebuildable.
4. GC removes manifest references before objects.
5. Restore captures current existing target state before replacement.
6. No operation relies on cleanup succeeding to preserve the last known-good bytes.
7. Staging and trash may accumulate after crashes, but they are bounded, detectable, and never treated as live backups without validation.
8. No failure is reported as atomic rollback unless actual target and backup fingerprints prove it.

## Complexity

- Capture: `O(file bytes)` time and bounded streaming memory.
- Restore preview/apply: `O(current bytes + object bytes)` time and bounded streaming memory, excluding explicitly bounded diff construction.
- List: `O(manifests scanned)` worst-case under the configured manifest bound, with `O(page size)` retained output and keyset pagination; bounded index rebuild is `O(manifests)`.
- Quick audit: `O(manifests + objects)` metadata work.
- Full audit: `O(total referenced object bytes)`.
- GC planning: `O(manifests + objects)` with bounded retained candidates.
- Store memory: bounded by one operation's buffers, reservations, page or plan limits, and index limits.

No API may retain all file contents, all diffs, or an unbounded manifest set in memory.

## Required tests

### Configuration and path security

- disabled-by-default behavior;
- missing, relative, overlapping, aliased, symlinked, junction-backed, reparse, and special-file store paths;
- public-tool denial for every store descendant;
- owner-only permissions and permission failures;
- exclusive lock acquisition, stale process termination, and second-process rejection;
- Windows long paths, drive roots, case folding, short paths, and Unix/macOS aliases.

### Store format and capture

- initialization and immutable descriptor validation;
- unknown fields and unsupported versions;
- exact object hashing, no-replace install, deduplication, object collision simulation, and existing-object corruption;
- manifest canonicalization, checksum, duplicate IDs, oversized manifests, and index rebuild;
- quota reservations, cancellation, saturation, overflow, object-size, manifest-count, per-target, and pinned limits;
- source changes during hashing and between capture and target mutation;
- failures at every write, sync, close, rename, index, and cleanup position.

### Mutation integration

- omitted policy preserves current behavior;
- required backup bound into edit and package previews;
- backup failure prevents target mutation;
- every package backup is durable before first commit;
- package partial commit retains all captured backups and reports actual states;
- CP1251, UTF-16 BOM/CRLF, read-only, no-op, and ambiguous-encoding behavior;
- replay, expiry, eviction, restart invalidation, and concurrent apply attempts.

### Restore

- exact restore, missing-target no-replace restore, stale target, same-content replacement, read-only target, and object corruption;
- mandatory safety backup of current state;
- quota failure before restore mutation;
- failure at safety capture, staging, replacement, directory sync, final verification, and cleanup;
- source backup remains live after success and failure;
- encoding/BOM/line-ending preservation and bounded diff fallback;
- direct/HTTP equivalence and redacted logging.

### Garbage collection and recovery

- deterministic candidate ordering and reasons;
- retention floor, per-target maximum, immutable pin state, active reservations, and deduplicated reference counts;
- stale plans and concurrent new manifests;
- crashes before and after manifest/object trash renames and deletions;
- orphan, staging, trash, missing-object, corrupt-manifest, and stale-index startup cases;
- quick/full audit bounds, cancellation, corruption reporting, and no mutation;
- repeated recovery, race detector, and multi-platform failure injection.

### Repository and release gates

- schema and tool-catalog drift;
- stdio/HTTP metadata and representative result equivalence;
- fuzzing of descriptor, manifest, cursor, and plan decoders;
- full Go, race, static-analysis, vulnerability, Gitleaks, documentation, six-target build, and native runtime smoke gates;
- explicit verification that no store path, object bytes, backup IDs used as capabilities, or private operator state enters logs or tracked fixtures.

## Devil's advocate findings

### Risk: the store becomes a hidden root escape

A store under or adjacent to a public workspace could be read, overwritten, fingerprinted, or deleted through existing tools. The approved mitigation is a non-overlapping internal root configured only at startup, denied to ordinary tools, and resolved with the same or stricter path-security primitives. R18 phase 1 implements and tests this boundary; later phases must preserve it.

### Risk: backup creation causes the mutation it is meant to protect to fail

Quota exhaustion, disk-full, permission, sync, or lock failures can block mutations. This is intentional for `required` policy. Backup remains opt-in, quota admission occurs before mutation, and no implicit GC is attempted. Operators choose store capacity and retention explicitly.

### Risk: a mutable global index becomes a single corruption point

The index is derived and replaceable. Immutable manifests and content-addressed objects remain authoritative. Startup and audit can rebuild the index under bounds.

### Risk: deduplication trusts attacker-controlled object names

Every existing object is verified by regular-file type, stable path, size, and complete SHA-256 before reference. Digest filenames alone are never trusted.

### Risk: restore destroys the current state

Restore of an existing target requires a durable safety backup of current exact bytes before replacement. Quota or capture failure prevents restore. The system still does not claim automatic rollback after an uncertain post-replacement failure.

### Risk: garbage collection deletes shared data

GC is dry-run/apply, generation-bound, manifest-first, reference-counted, pin-aware, and stale-plan rejecting. Objects are never deleted before all live manifest references are removed.

### Risk: backups retain secrets indefinitely

Backups are persistent copies. The design uses restrictive permissions, bounded retention, explicit GC, no content in logs or metadata responses, and clear secure-deletion limitations. Encryption and cryptographic erasure remain a separate design.

### Risk: cross-process races corrupt quotas and references

The first implementation supports one writer process per store through an exclusive lifetime lock. Shared and distributed writers are deferred.

## Approval record

Maintainers explicitly accepted all ten decisions on 2026-08-04:

1. a dedicated non-overlapping internal store root is a new process-wide authority;
2. one writer process per store is the initial concurrency model;
3. backup remains disabled by default and opt-in through approval-bound mutation policy;
4. immutable objects and manifests are authoritative, with a rebuildable derived index;
5. the documented quota and retention defaults are accepted, together with mandatory status reporting and conservative preflight estimates in the phases that consume them;
6. restore is initially limited to the original target and always safety-backs up an existing target;
7. GC remains explicit dry-run/apply with no background deletion;
8. encryption at rest and secure deletion guarantees are deferred;
9. existing adjacent `.bak` conversion behavior remains separate;
10. no automatic patch-package rollback is introduced.

Approval authorizes phased implementation, not a single monolithic change. Each R18 phase must preserve the complete boundary above, add focused failure-injection tests, pass the relevant regression and cross-platform gates, and avoid exposing a partially implemented public promise. R18 phases 1 and 2 added no public backup tool or automatic backup behavior. Phase 3 added only the read-only management surface. Phase 4 connected capture solely to approval-bound `edit_file`. Phase 5 connects package capture solely to a strict `patch-package-v1` manifest carrying `backupPolicy: "required"`; omitted-policy mutation paths retain their prior behavior.
