# Verified Change Workflows

This document is the approved design baseline for R16. It defines the intended behavior, security boundaries, sequencing, and completion gates for deterministic fingerprints, edit preview/apply, declared patch packages, and structured verification. Phases 1 and 2 are implemented in source: deterministic fingerprints and bounded one-shot `edit_file` preview/apply are complete, while patch packages and structured verification remain pending.

Persistent backup storage and user-managed change review are approved in principle but remain outside the initial R16 implementation until a separate retention and lifecycle design is approved.

## Goals

R16 should let an MCP client:

- identify the exact state of files or directory trees;
- preview a mutation once and later apply exactly that prepared mutation;
- declare, preflight, apply, and verify a bounded set of coordinated file edits;
- run selected repository checks through typed inputs and outputs instead of arbitrary shell strings;
- preserve the existing encoding, BOM, line-ending, path-security, mutation-durability, and transport-equivalence guarantees.

## Non-goals

The initial R16 scope does not include:

- a persistent backup repository;
- retention, quota, pinning, garbage collection, or user-managed restore;
- automatic rollback of a partially committed multi-file package;
- arbitrary command execution or a generic build/test orchestrator;
- create, delete, move, or rename operations inside patch packages;
- per-session filesystem ACLs or HTTP client mutation of process roots;
- a claim of atomic multi-file commit where the operating system cannot provide it.

## Shared security and compatibility contract

All R16 operations must:

- use the existing allowed-root, symlink, junction, reparse-point, and missing-ancestor validation;
- preserve all existing tools; the explicitly reviewed R16 schema change adds the read-only `fingerprint_paths` tool as the 24th source-catalog entry;
- expose identical schemas and behavior through stdio and stateful Streamable HTTP;
- use stable typed error codes and bounded diagnostic output;
- reject oversized input before expensive parsing, hashing, diffing, or staging;
- honor cancellation during reads, hashing, preparation, staging, verification, and cleanup;
- keep protocol output free of secrets, file contents not required by the result, capability tokens, and private operational state;
- preserve current encoding, BOM, and line-ending behavior for every prepared or applied edit.

## 1. Deterministic fingerprints

### Purpose

A fingerprint proves what state was inspected or approved. Preview/apply and patch packages will use it as an optimistic precondition and as post-operation evidence.

### Proposed behavior

The initial public operation should accept one or more explicit files or directory roots and return:

- one aggregate SHA-256 fingerprint;
- the selected fingerprint mode and options;
- total regular files, directories, and bytes inspected;
- optional bounded per-entry fingerprints when explicitly requested;
- a stable typed error when any required entry changes during inspection.

The default content fingerprint should hash canonical records in deterministic lexical order. A record should include the Unicode-NFC normalized slash-separated relative path, entry type, byte length, and file-content SHA-256. Modification times, ownership, and platform-specific permission bits must be excluded by default because they are not stable across copies and operating systems. Metadata-sensitive modes may include explicitly selected fields and must identify those fields in the result.

Directory traversal should reuse the secure deterministic walker. `.gitignore` behavior must be explicit; the initial mode is default-on with an opt-out, matching R15 recursive tools. VCS-internal `.git` directories remain excluded. `content-v1` records real directories and regular files only: in-root symlinks, junctions, and other reparse-point entries are neither followed nor included, while entries resolving outside allowed roots fail closed.

### Complexity and limits

- Time: `O(total bytes read + entries)`.
- Retained memory: bounded by traversal depth, hashing buffers, and explicitly requested bounded entry output.
- Every file must be streamed rather than loaded in full.
- Path count, entry count, total result bytes, and optional entry details must have explicit limits.

### Required tests

Cover identical content under different absolute roots, deterministic ordering, content changes, metadata-only changes, optional metadata modes, nested `.gitignore`, opt-out, in-root link exclusion without traversal, escaping links, files changed during hashing, cancellation, unreadable files, very large files, empty trees, cross-platform path separators, output truncation, and repeated-run stability.

## 2. Edit preview/apply

### Compatibility approach

The preferred minimal design is to extend `edit_file` with an explicit action while preserving the current direct-edit behavior when the new action is omitted. Exact replacement, whitespace-flexible replacement, unified patch input, fuzzy matching, encoding selection, BOM preservation, line-ending preservation, and current limits must continue to use one preparation pipeline.

### Preview

A preview request supplies the current edit or patch inputs and returns:

- an unguessable `previewId`;
- creation and expiration timestamps;
- target path and target fingerprint;
- proposed-result fingerprint;
- the exact bounded unified diff shown for approval;
- encoding, BOM, and line-ending metadata relevant to the prepared result;
- whether the operation would be a logical no-op.

The prepared operation must be stored in a process-local bounded preview cache. The identifier should contain at least 256 bits of cryptographic randomness, must never be listed, and must not be written to ordinary logs. A process restart invalidates all previews.

The cache has independent bounds for entry count, dynamic retained bytes, and lifetime: `MCP_MAX_EDIT_PREVIEWS=128`, `MCP_MAX_EDIT_PREVIEW_BYTES=67108864`, and `MCP_EDIT_PREVIEW_TTL_SECONDS=900` by default. Expired entries are removed lazily before deterministic FIFO capacity eviction. Each live preview retains one platform-appropriate open-file identity reference; eviction, expiry, response-limit failure, apply claim, and process exit release it.

### Apply

Apply accepts the `previewId` rather than a second copy of the edit instructions. The server must atomically claim the preview to prevent concurrent replay, then verify:

- the preview exists and is not expired;
- it has not already been consumed or claimed;
- the target still resolves to the same allowed path;
- the current target fingerprint matches the preview precondition;
- the prepared result still matches the recorded result fingerprint.

The existing durable mutation layer performs the commit. Apply atomically removes the capability before validation, preserves an open file identity until the last pre-commit check, verifies current and prepared `content-v1` fingerprints, then closes the identity reference immediately before durable replacement. Any conflict, cancellation, or failed apply makes the preview terminal; the client creates a new preview rather than retrying an uncertain capability. Successful application returns the actual post-commit fingerprint and diff metadata without re-emitting the consumed token.

A preview token is a narrowly scoped process-local capability. Because all connections already share one process-wide root and authorization policy, the initial design does not add per-session ownership. Possession of the unguessable token plus normal server access authorizes only its exact prepared mutation.

### Required tests

Cover exact preview/apply, no-op preview, expired and evicted previews, malformed and guessed identifiers, replay, two concurrent apply attempts, file changes between preview and apply, same-content path replacement, encoding/BOM/line-ending preservation, fuzzy ambiguity, patch mismatch, cancellation, write failures, count/byte/output cache limits, restart invalidation, redacted logs, and cross-session direct/HTTP application.

## 3. Declared patch packages

### Purpose

A patch package coordinates preconditions and verification for several existing regular files in one bounded request. It reduces repeated tool calls and detects package-wide conflicts before the first mutation.

### Initial package format

Use a versioned JSON manifest with:

- package format version;
- an optional caller label that is not trusted as an identifier;
- a bounded ordered list of target operations;
- one unique normalized target path per operation;
- the expected pre-edit fingerprint for every target;
- exactly one supported existing-file edit form per target;
- optional expected post-edit fingerprints.

The initial version must reject file creation, deletion, movement, renaming, `/dev/null` patches, duplicate resolved targets, overlapping aliases, external links, unbounded embedded data, and unknown manifest fields where ambiguity would be unsafe.

### Actions

- `inspect`: validate package structure, limits, paths, duplicate targets, and declared algorithms without preparing mutations.
- `dryRun`: resolve and fingerprint every target, prepare every result, and return bounded per-file diffs plus aggregate pre/post fingerprints. No source file is changed.
- `apply`: consume the exact dry-run package preview, revalidate every precondition, stage all outputs, then commit in deterministic manifest order.
- `verify`: compare current target fingerprints with the package's expected post-state and return per-file plus aggregate results.

Package apply must not reaccept a modified manifest. It should use the same one-shot preview capability model as `edit_file`.

### Partial-commit contract

All targets must pass parsing, path validation, fingerprint validation, preparation, and staging before the first commit. The initial R16 package does not promise automatic rollback because persistent backup and recovery policy remains a separate design gate.

If a commit fails after earlier files were committed, the operation must stop and return a stable `PARTIAL_COMMIT` result containing bounded, machine-readable lists of:

- files confirmed committed;
- files confirmed unchanged;
- the file whose commit failed;
- files whose final state could not be conclusively classified;
- actual post-operation fingerprints where available.

Documentation must describe this limitation prominently. The implementation must not call the operation atomic or transactional across files.

### Required tests

Cover malformed manifests, unsupported versions, duplicate and aliased targets, path escapes, external links, stale fingerprints, preparation and staging failures before commit, deterministic commit order, injected failure on each commit position, accurate partial results, cancellation at every phase, exact verification, bounded diagnostics, large packages, cross-directory targets, direct/HTTP equivalence, and preservation of unrelated files.

## 4. Structured verification

### Purpose

Structured verification replaces fragile shell command strings with allowlisted operations, typed arguments, fixed executable invocation where necessary, bounded diagnostics, and stable results.

### Initial approved checks

The first implementation tranche should remain filesystem- and repository-adjacent:

- JSON parsing for explicit files;
- text-format checks for explicit files, including selected encoding, BOM, line-ending, and trailing-whitespace expectations;
- `git diff --check` for an explicit repository root and optional explicit paths, invoked directly without a shell;
- fingerprint comparison through the shared fingerprint primitive rather than a duplicate hashing implementation.

The exact public shape may be one bounded batch-verification tool or a small number of narrowly scoped tools. The final choice must minimize catalog growth while keeping schemas clear and independently permissionable.

Arbitrary executables, user-supplied command strings, compound commands, package-manager operations, generic test runners, and language-specific build orchestration are outside the initial R16 scope.

### Execution rules

- Never invoke a shell for a structured check.
- Resolve only the fixed executable required by the selected check.
- Construct arguments from validated fields, never by concatenating a command string.
- Set an explicit working directory inside an allowed root.
- Use bounded stdout/stderr capture, explicit timeout, cancellation, and stable exit classification.
- Treat a missing optional executable, non-repository directory, malformed file, and failed check as distinct results.
- Do not inherit unrelated credentials or execution-enabling environment variables where they are not required.

### Required tests

Cover valid and malformed JSON, unsupported encodings, BOM and mixed-line-ending cases, trailing whitespace, Git repositories and non-repositories, paths containing spaces and metacharacters, command-injection attempts, missing Git, timeout, cancellation, oversized diagnostics, path escapes, symlink aliases, deterministic result ordering, and direct/HTTP equivalence.

## 5. Deferred persistent backup and change-review design gate

Persistent backups are approved in principle but must not be implemented as incidental `.bak` files or an unbounded side effect of edit/apply.

A separate design review must decide at least:

- dedicated backup-store location and allowed-root relationship;
- content-addressed deduplication and integrity verification;
- global byte quota, per-target version limit, maximum object and manifest counts;
- age-based retention and manually pinned backups;
- manifest/index format and crash-consistent updates;
- dry-run and apply phases for garbage collection;
- safe restore through its own preview/apply and current-state fingerprint checks;
- secret handling, restrictive permissions, logging rules, and deletion expectations;
- orphan-object recovery, corrupted-index behavior, and interrupted cleanup;
- behavior when quota is exhausted before a mutation;
- whether backups are enabled by default or require explicit opt-in.

Until that design is approved:

- R16 preview/apply creates no persistent backup;
- patch packages provide no retained rollback point;
- change review is limited to the approved preview diff and returned pre/post fingerprints;
- existing tool-specific backup behavior remains unchanged and must not be generalized accidentally.

## Implementation sequence

1. Finalize fingerprint schema and implement the shared streaming primitive.
2. Add bounded preview storage and one-shot preview/apply to existing-file edits.
3. Define the versioned patch-package manifest and implement inspect plus dry-run.
4. Implement package apply and verify with the explicit partial-commit contract.
5. Add the initial structured verification checks.
6. Run the complete R16 compatibility, race, security, catalog, transport-equivalence, documentation, and repository verification gates.
7. Hold a separate backup lifecycle brainstorming and design review before adding persistent backup storage, restore, garbage collection, or package rollback.

## R16 completion gate

R16 may be marked complete only when:

- fingerprints are deterministic, streamed, secure-walker-backed, and stable across repeated runs;
- preview/apply guarantees that the applied mutation is exactly the approved prepared mutation and rejects replay or stale targets;
- patch packages preflight every target before commit, report partial commit precisely, and make no false atomicity or rollback claim;
- structured checks use typed inputs and no arbitrary shell command;
- all new caches, manifests, inputs, outputs, diagnostics, and execution time are bounded;
- schemas, errors, documentation, catalog metadata, and behavior are equivalent over stdio and Streamable HTTP;
- focused, adversarial, full regression, race, static-analysis, vulnerability, manual MCP, and repository consistency checks pass;
- no persistent backup store or retention policy is introduced without the separate approved design.
