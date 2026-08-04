# Verified Change Workflows

This document is the approved and completed design baseline for R16. It defines the implemented behavior, security boundaries, sequencing, and completion gates for deterministic fingerprints, edit preview/apply, declared patch packages, and structured verification. Phases 1–5 are implemented and verified in source: deterministic fingerprints, bounded one-shot `edit_file` preview/apply, complete strict `patch-package-v1` inspect/dry-run/apply/verify, and typed `verify_state` checks.

Persistent backup storage and user-managed change review remain outside R16 itself. Their separate lifecycle design was approved as R17 on 2026-08-04, and phased implementation is tracked independently in R18.

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
- preserve all existing tools; the reviewed R16 additions expose `fingerprint_paths`, `patch_package`, and `verify_state`, bringing the unreleased source catalog to 26 tools while the published 2.0.0 baseline remains 23;
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

The implemented `patch-package-v1` JSON manifest contains:

- `formatVersion: "patch-package-v1"`;
- optional `label`, bounded to 256 bytes and never trusted as an identifier;
- `fingerprintAlgorithm: "sha256"` and `fingerprintMode: "content-v1"`;
- a bounded ordered `targets` array;
- one unique Unicode-NFC slash-normalized declared path per target;
- required `expectedFingerprint` and optional `expectedResultFingerprint`;
- exactly one of `edits` or one strict single-file unified `patch`;
- optional `encoding` and `forceWritable` with `edit_file` semantics.

The input and every nested manifest object reject unknown JSON fields. The current implementation rejects creation, deletion, movement, renaming, `/dev/null` patches, duplicate spellings, duplicate resolved targets, symlink/junction aliases, hard-link aliases, external links, unsupported versions or algorithms, unbounded edit arrays, and oversized embedded data.

### Actions

- `inspect` **implemented**: validate structure, semantic input size, target count, paths, existing regular-file types, duplicate or aliased targets, edit shapes, fuzzy thresholds, strict patch structure, and declared algorithms without reading target contents.
- `dryRun` **implemented**: retain one stable open-file identity per bounded target, obtain a coherent package-wide pre-state, verify every declared fingerprint, prepare each exact result through the shared edit pipeline, enforce aggregate prepared/output limits, perform final identity and package-wide state verification, return ordered diffs plus `patch-package-aggregate-v1` evidence, and store the exact package behind a bounded expiring 256-bit capability. No source file is changed.
- `apply` **implemented**: atomically consume the exact dry-run capability, reject a resubmitted manifest, revalidate paths, identities, bounded current snapshots, prepared bytes, and read-only authorization, durably stage every changed output before the first commit, commit in deterministic manifest order, then require a coherent two-pass final fingerprint verification before success. Every apply attempt is terminal.
- `verify` **implemented**: require `expectedResultFingerprint` for every target, compare current `content-v1` fingerprints, and return per-file plus expected/actual aggregate results. Mismatch returns structured `CONFLICT`.

The package cache is process-local and independent from edit previews. Its defaults are `MCP_MAX_PATCH_PACKAGE_PREVIEWS=16`, `MCP_MAX_PATCH_PACKAGE_PREVIEW_BYTES=134217728`, and `MCP_PATCH_PACKAGE_PREVIEW_TTL_SECONDS=900`. Expired entries are removed before deterministic FIFO eviction. Capability removal, response-limit failure, claim, and process exit release all retained identities.

Current limits also include `MCP_MAX_BATCH_FILES` for target count, `MCP_MAX_FILE_BYTES` per source/patch/result, `MCP_MAX_PATCH_PACKAGE_BYTES=16777216` for semantic manifest input, `MCP_MAX_PATCH_PACKAGE_PREPARED_BYTES=67108864` for per-request preparation state, and `MCP_MAX_OUTPUT_BYTES` for combined structured/text output including worst-case partial-state diagnostics. The aggregate binds manifest order, canonical declared paths, and ordered per-target `content-v1` fingerprints; it is evidence, not an atomicity claim.

### Partial-commit contract

All targets must pass parsing, path validation, fingerprint validation, preparation, and staging before the first commit. The initial R16 package does not promise automatic rollback because persistent backup and recovery policy remains a separate design gate.

If a failure occurs after a target may have been committed, the operation stops and performs a bounded best-effort fingerprint classification. Targets are reported as `committed` when current bytes equal the prepared result, `unchanged` when current bytes equal the approved pre-state, or `unknown` otherwise. A result containing any committed or unknown target uses stable `PARTIAL_COMMIT` metadata and includes the failed index/path, underlying failure code/message, counts, and actual fingerprints where available.

This also handles filesystem APIs that can report an error after replacement, such as a directory-sync failure: classification relies on actual fingerprints rather than assuming that an error means no mutation. The implementation does not call the operation atomic or transactional and does not attempt automatic rollback.

### Required tests

Implemented tests cover malformed and unknown-field manifests, unsupported versions/algorithms/modes, duplicate paths, hard-link aliases, path escapes, stale fingerprints, same-content path replacement, strict patches, UTF-16 BOM/CRLF and CP1251 preservation, force-writable approval, no-op metadata preservation, count/manifest/preparation/cache/output limits, bounded snapshot hashing, expiry, eviction, restart invalidation, redacted tokens, cancellation before, during, and after staging, cleanup failures, concurrent claim, all-target staging, deterministic commit order, injected failure at every commit position, post-replacement error classification, external changes during final verification, exact and mismatched verification, cross-session HTTP/direct apply/replay/verify, and byte-identical preservation before commit.

## 4. Structured verification

### Purpose

Structured verification replaces fragile shell command strings with allowlisted operations, typed arguments, fixed executable invocation where necessary, bounded diagnostics, and stable results.

### Initial approved checks

The implemented `verify_state` tool remains filesystem- and repository-adjacent and supports one ordered bounded batch of:

- JSON parsing for explicit decoded files;
- text-format checks for explicit files, including selected encoding, BOM, line-ending, and trailing-whitespace expectations;
- fixed direct `git diff --check` for an explicit repository root and optional literal relative paths, invoked without a shell;
- fingerprint comparison through the shared fingerprint primitive rather than a duplicate hashing implementation.

A failed expectation is returned as `passed=false`; an operational problem uses a stable per-check `errorCode`. The single batch tool minimizes catalog growth while preserving a strict discriminated union and deterministic result order.

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

Implemented tests cover valid and malformed JSON, unsupported encodings, UTF-16 BOM/CRLF, trailing whitespace and bounded diagnostics, fingerprint matches and mismatches, strict unknown fields and invalid unions, non-repositories, relative path escapes, escaping symlink aliases, literal Git paths beginning with option-like text and containing spaces/metacharacters, missing Git, timeout, cancellation, output limits, filtered environment variables, deterministic result ordering, shared fingerprint behavior, direct/HTTP equivalence, manual MCP smoke, and the complete repository verification gates.

## 5. Approved persistent backup follow-on design

The follow-on lifecycle is maintained in [PERSISTENT_BACKUP_LIFECYCLE.md](PERSISTENT_BACKUP_LIFECYCLE.md). Maintainers approved its ten decisions as R17 on 2026-08-04, and R18 now implements that contract in separate reviewable phases rather than extending R16 incidentally.

R18 phase 1 adds only the disabled-by-default internal store foundation: strict path separation, protected-root denial, owner-only permissions, one platform-native lifetime writer lock, an immutable versioned descriptor, and an empty layout. It does not capture bytes, create manifests, expose a public backup tool, or change any R16 mutation schema.

Until the later R18 mutation-integration phases are implemented:

- R16 preview/apply creates no persistent backup;
- patch packages provide no retained rollback point;
- change review is limited to the approved preview diff and returned pre/post fingerprints;
- existing tool-specific backup behavior remains unchanged and must not be generalized accidentally.

Every later phase must preserve the approved dedicated store, content-addressed objects, immutable manifests, derived index, quotas, explicit retention and pinning, restore safety backup, dry-run/apply GC, secret-handling, crash-recovery, disabled-by-default policy, separate `.bak` behavior, and no-automatic-rollback decisions.

## Implementation sequence

1. Finalize fingerprint schema and implement the shared streaming primitive.
2. Add bounded preview storage and one-shot preview/apply to existing-file edits.
3. Define the versioned patch-package manifest and implement inspect plus dry-run.
4. Implement package apply and verify with the explicit partial-commit contract.
5. Add the initial structured verification checks. **Implemented.**
6. Run the complete R16 compatibility, race, security, catalog, transport-equivalence, documentation, and repository verification gates.
7. Hold and approve a separate backup lifecycle design before adding persistent backup storage, restore, garbage collection, or package rollback. **Completed as R17; implementation is tracked in R18.**

## R16 completion gate

R16 may be marked complete only when:

- fingerprints are deterministic, streamed, secure-walker-backed, and stable across repeated runs;
- preview/apply guarantees that the applied mutation is exactly the approved prepared mutation and rejects replay or stale targets;
- patch packages preflight every target before commit, report partial commit precisely, and make no false atomicity or rollback claim;
- structured checks use typed inputs and no arbitrary shell command;
- all new caches, manifests, inputs, outputs, diagnostics, and execution time are bounded;
- schemas, errors, documentation, catalog metadata, and behavior are equivalent over stdio and Streamable HTTP;
- focused, adversarial, full regression, race, static-analysis, vulnerability, manual MCP, and repository consistency checks pass;
- R16 introduces no persistent backup store or retention policy; later work is governed by the separately approved R17 design and R18 roadmap.
