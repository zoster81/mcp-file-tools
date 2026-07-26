# Roadmap History — R1–R6

> Archived on 2026-07-26 after R6 was committed, pushed, deployed, and verified.
> This document preserves the previous detailed handoff and its technical evidence.
> It is historical and must not be used to select new work.
>
> Active planning now lives in [ROADMAP.md](ROADMAP.md), the reusable engineering gate in [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md), and the current operational handoff in `D:\OpenAI-Tunnel\todo.md`.

## START HERE — Mandatory handoff

Repository:

- GitHub: `https://github.com/zoster81/mcp-file-tools`
- Local checkout: `D:\OpenAI-Tunnel\mcp-file-tools-src`
- Local connector: `@Ryzen9_0`
- Authorized workspace: `D:\OpenAI-Tunnel` and descendants only

Current verified baseline:

- Branch: `main`, aligned with local tracking `origin/main` after an explicit `git fetch origin --prune`.
- Source commit and `origin/main`: `279808acb2ceb8d9052de5c4d558234c1e44401d` (`Consolidate execution preparation and tool metadata`).
- Repository working tree: clean.
- Previous release binary retained: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64.exe`, version `1.8.0`, SHA-256 `0463DED458AE173146DC432D4A158263C776F00603F9DE2583168A5F4ABA3315`.
- Active R6 binary: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_279808a.exe`, version `1.8.0-2-g279808a`, SHA-256 `4B5F6DBCEAA49400A3AF73A9FD60D471941AA51BA8E1A27769FFF9424D548CE1`.
- `D:\OpenAI-Tunnel\start-tunnel.ps1` references the active R6 binary and remains UTF-8 with BOM and LF endings.
- Active MCP process: PID `25720`, parent tunnel PID `27732`, running `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_279808a.exe`.
- Tunnel client PID `27732` owns listener `127.0.0.1:8080`; the MCP process parent relationship is verified.
- The workspace contains the previous official `1.8.0` binary and the active R6 binary; no obsolete development binaries remain.

Current initiative:

> Refactor all handlers to reuse shared domain primitives, remove duplicated filesystem/encoding/concurrency/error-handling code, and keep MCP handlers thin.

Current milestone:

> **COMPLETED, PUSHED, AND DEPLOYED — R6 commit `279808a` is on `origin/main`, the verified Windows binary is active, and the restarted tunnel is serving it on `127.0.0.1:8080`.**

R1-R6 source refactoring is complete and active in the connector runtime. No next refactoring milestone is selected.

### Mandatory read order before editing

Read these files in this order before changing or publishing the prepared release:

1. `D:\OpenAI-Tunnel\todo.md`
2. `D:\OpenAI-Tunnel\mcp-file-tools-src\docs\PUBLISHING.md`
3. `D:\OpenAI-Tunnel\mcp-file-tools-src\.github\workflows\release.yml`
4. `D:\OpenAI-Tunnel\mcp-file-tools-src\.github\workflows\publish-registry.yml`
5. `D:\OpenAI-Tunnel\mcp-file-tools-src\.goreleaser.yml`
6. `D:\OpenAI-Tunnel\mcp-file-tools-src\scripts\verify-release-version.js`
7. `D:\OpenAI-Tunnel\mcp-file-tools-src\scripts\verify-release-version.test.js`
8. `D:\OpenAI-Tunnel\mcp-file-tools-src\scripts\generate-server-json.js`
9. `D:\OpenAI-Tunnel\mcp-file-tools-src\plugin\.claude-plugin\plugin.json`
10. `D:\OpenAI-Tunnel\mcp-file-tools-src\.claude-plugin\marketplace.json`
11. `D:\OpenAI-Tunnel\mcp-file-tools-src\plugin\bin\run.js`
12. `D:\OpenAI-Tunnel\mcp-file-tools-src\CHANGELOG.md`

Before any edit, also verify branch, `HEAD`, `origin/main`, working tree, and active MCP process.
Preserve all existing user changes. Do not commit or push without explicit authorization.

## Current repository and deployment state

Committed baseline:

- `2511fdb40aab05d1864d96b0a765d197e9a7a0da` — `Migrate line ending handlers to shared text core`.
- `97cd91fe41d9cca33b461df3b4e8ad6342a3c636` — `Consolidate secure traversal and project documentation`.
- `79a9ada5330a43465bfa73bd47178bb224abb8be` — `Add durable atomic mutation layer`.
- `920eb398139c44ea40997dccf2a8d1dd1f142362` — `Add typed operation error model`.
- `daf83eb5b4b8e9c85566fa786ea5987d70b5639e` — `Migrate project identity to fork namespace`.
- `d5513a6129b2727c3d2eabdf1d3612fb78cccdb6` — `Consolidate bounded ordered concurrency`.
- `edcffd2326ad25a901cf41847b8d6fc5350d06b7` — `Prepare v1.8.0 release`.
- `0ceb55e61cd667d4d7cd296e1c42ddfd221aefc4` — `Fix Unix backup permission test`.
- `4941a6909bdf16e4317f0c1ee126e8530d4ec58f` — `Fix Windows short-path root handling`.
- `2aebd71` — `Fix Windows path identity assertions`.
- Local `main` points to `2aebd71`; local tracking `origin/main` points to `4941a69`.
- Working tree is clean.

R2 repository state:

- Secure shared traversal, Windows junction/reparse-point resolution, missing-path ancestor validation, consumer migrations, tests, documentation, plugin metadata, publishing notes, Smithery metadata, and runtime descriptions are committed in `97cd91f`.
- Public MCP schemas remain unchanged.
- R2 push status: not pushed.

R3 repository state:

- Durable mutation primitives, handler migrations, failure-injection tests, documentation, plugin metadata, publishing notes, Smithery metadata, and runtime descriptions are committed in `79a9ada`.
- Public MCP schemas remain unchanged.
- R3 push status: not pushed.

Deployment state:

- Prepared binary: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_d5513a6.exe`.
- Prepared version: `v1.7.3-20-gd5513a6`.
- Prepared size: `10404352` bytes.
- Prepared SHA-256: `CC8AFA22446D5D0E23B9B5EB298FC2D1F8217C79920E1B5ABABEAF60EB25045A`.
- `start-tunnel.ps1` points to `d5513a6`; its 6108-byte UTF-8-BOM/LF representation was preserved and PowerShell parsing reports zero errors.
- R5 commit status: committed locally as `d5513a6`.
- Push status: R5, release preparation, Unix CI fix `0ceb55e`, and Windows runtime fix `4941a69` are pushed; final Windows assertion fix `2aebd71` is not pushed, so `main` is one commit ahead of local tracking `origin/main`.
- Restart status: completed and verified.
- Active process PID `28308` runs `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_d5513a6.exe`, version `v1.7.3-20-gd5513a6`.

## Refactoring rules

- Handlers are MCP adapters, not reusable implementation layers.
- A handler must not call another handler to reuse behavior.
- Shared behavior belongs in small internal domain primitives.
- Preserve public MCP schemas and response behavior unless a change is explicitly approved.
- Preserve encoding, BOM, line endings, permissions, and byte identity where the operation is a no-op.
- Validate paths again immediately before mutation where the current architecture permits it.
- Prefer focused TDD and minimal diffs.
- Never mix unrelated roadmap milestones in one change.

# R1 — Shared text-document core

## Completed

- [x] R1.1 Introduce shared `textDocument` metadata and read/decode pipeline.
- [x] R1.2 Make `read_text_file` and `read_multiple_files` use the shared pipeline.
- [x] R1.3 Migrate `edit_file` to shared decode/encode behavior.
- [x] R1.4 Migrate `grep_text_files` to the shared decoder.
- [x] R1.5 Migrate `write_file` and `convert_encoding` to the shared encoder.
- [x] Strip transport BOMs from returned text while preserving meaningful leading `U+FEFF`.
- [x] Reject BOM/explicit-encoding conflicts in the shared reader.
- [x] Preserve BOM and consistent CRLF/LF during edits.
- [x] Suppress byte-identical edit and conversion writes.
- [x] Add public `auto`, `always`, `never`, and `preserve` BOM policies.
- [x] Preserve exact CRLF/LF/CR/mixed sequences during encoding conversion.
- [x] Make invalid policies and unrepresentable output fail before mutation.

Relevant completed commits:

- `d642cbf` — shared text-document handling
- `1faf96a` — grep migration
- `cca7053` — write/convert migration and public BOM policies

## COMPLETED — R1.6 Migrate line-ending handlers

### Problem

`HandleDetectLineEndings` and `HandleChangeLineEndings` still duplicate parts of the file-read, encoding-resolution, BOM-validation, and mutation flow. This creates inconsistent behavior: the shared reader rejects BOM/encoding conflicts, while `detect_line_endings` currently does not.

### Target design

- `detect_line_endings` must obtain decoded content and metadata from the shared text-document pipeline.
- `change_line_endings` must reuse shared path/encoding/BOM validation and shared mutation primitives.
- Specialized byte-level newline conversion may remain as a low-level primitive only where it is required to preserve every non-line-ending byte exactly.
- Do not duplicate file opening, encoding detection, transport-BOM validation, mode lookup, or error classification when a shared primitive already exists.
- Do not normalize mixed line endings except when the caller explicitly requests conversion.

### Required TDD sequence

- [x] Confirm the existing `TestHandleDetectLineEndings_BOMEncodingConflict` fails for the expected reason.
- [x] Refactor `HandleDetectLineEndings` to use `readTextDocument` or a smaller shared primitive extracted from it.
- [x] Make the focused conflict test pass without changing the public output schema.
- [x] Add or retain tests for UTF-8 BOM, UTF-16 LE/BE BOM, explicit encodings, unsupported encodings, empty files, BOM-only files, CRLF, LF, mixed, and no line endings.
- [x] Review `HandleChangeLineEndings` for duplicated encoding/BOM/path/mode logic.
- [x] Refactor only the duplicated orchestration; preserve the proven byte-exact conversion algorithms where appropriate.
- [x] Add tests proving BOM/encoding conflicts fail before mutation.
- [x] Add tests proving no-op conversions do not rewrite the file or alter mtime.
- [x] Add tests proving non-line-ending bytes remain identical for UTF-8, UTF-16 LE/BE, and representative legacy encodings.
- [x] Revalidate the destination before the atomic write and preserve permissions.
- [x] Run the focused line-ending tests.
- [x] Run the complete handler and repository regression suites.
- [x] Review documentation and runtime metadata; concurrent R1.6 updates to `TOOLS.md`, `CHANGELOG.md`, `filetoolsserver/server.go`, and `server.json` were detected, reviewed, and preserved. `README.md` required no change.

### MQL acceptance coverage for R1.6

MQL is an acceptance domain for the shared text-document refactor, not a separate active feature project.
After the line-ending migration is correct, add small synthetic fixtures under repository test data; never depend on or modify the real MetaQuotes installation.

- [x] Add UTF-16 LE + BOM + CRLF `.mq5` fixture.
- [x] Add UTF-16 LE + BOM + CRLF `.mqh` fixture.
- [x] Add UTF-8 without BOM `.mq5` fixture.
- [x] Include Italian, Cyrillic, CJK, and a surrogate-pair character.
- [x] Verify detect/read/grep/edit/write/convert/change-line-endings behavior across the same fixture set.
- [x] Verify first-line edits preserve exactly one BOM.
- [x] Verify no-op edit/conversion/line-ending operations are byte-identical.
- [x] Verify conversions obey the public BOM policy.

### R1.6 Definition of Done

R1.6 is complete only when:

- [x] Both line-ending handlers reuse the shared document/BOM/encoding primitives for common orchestration.
- [x] Duplicated logic removed by the change is not reintroduced elsewhere; the now-unused legacy `resolveEncoding` and `shouldLoadEntireFile` helpers were removed.
- [x] The existing failing conflict test passes.
- [x] All focused and full tests pass.
- [x] Formatter, Staticcheck, vet, build, race detector, `git diff --check`, and Git status are verified.
- [x] MQL acceptance fixtures cover UTF-16 LE+BOM+CRLF and UTF-8-no-BOM.
- [x] No unrelated files changed.
- [x] Documentation and runtime descriptions reflect BOM/encoding conflict handling; public schemas remain unchanged.
- [x] This checklist was updated before handoff.

# Active next milestone and queued refactoring milestones

## COMPLETED — R2 — Secure filesystem walker

- [x] Inventoried duplicated traversal in `search_files`, `grep_text_files`, `tree`, and `directory_tree`.
- [x] Added one internal deterministic walker contract for exclusion pruning, cancellation, maximum depth, early stop, and caller-defined error handling.
- [x] Added native Windows final-path resolution with `GetFinalPathNameByHandle` because `filepath.EvalSymlinks` does not resolve junctions reliably on Windows.
- [x] Enforced symlink, junction, and reparse-point workspace safety consistently for every traversed entry.
- [x] Rejected deeply nested missing paths whose nearest existing ancestor escapes through a link, while preserving safe missing destinations.
- [x] Migrated `search_files`, `grep_text_files`, `tree`, and `directory_tree` one consumer at a time with focused tests.
- [x] Preserved each tool's public schema, filtering, limits, deterministic order, cancellation, and error behavior.
- [x] Revalidated `tree.showEncoding` immediately before file access.
- [x] Added shared walker unit tests for ordering, metadata, pruning, depth, early stop, cancellation, partial filesystem errors, unsafe root rejection, junction escape, and path-swap revalidation.
- [x] Added handler regressions for deterministic order, cancellation, and Windows junction escape.
- [x] Updated runtime descriptions, `server.json`, `README.md`, `TOOLS.md`, and `CHANGELOG.md` without changing public schemas.
- [x] Ran formatter, focused tests, full tests, vet, Windows/Linux/macOS builds, Staticcheck, coverage, race detector, JSON validation, and `git diff --check`.

TDD record:

- The first junction regression run failed in `search_files`, `tree`, and `directory_tree` because each returned the unsafe `escape` entry.
- A second focused test failed because `ValidatePath` accepted `allowed\\escape\\missing\\nested\\new.txt` when `escape` was a junction outside the allowed directory.
- Both failures now pass through native final-path resolution and nearest-existing-ancestor validation.

### R2 Definition of Done

- [x] All four recursive consumers use the shared walker.
- [x] Duplicated recursive `WalkDir`/`ReadDir` implementations were removed; only the intentionally non-recursive `list_directory` retains direct `os.ReadDir`.
- [x] Windows junction and reparse-point behavior is tested without requiring symlink privileges.
- [x] Cross-platform symlink behavior remains covered.
- [x] Full regression, race, static-analysis, and build checks pass.
- [x] No unrelated files changed.
- [x] R2 was committed as one reviewable change set in `97cd91f`.

## COMPLETED — R3 — Atomic mutation layer

- [x] Inventoried `atomicWriteFile`, `atomicWriteWithBackup`, copy staging, move, delete, directory creation, and intentional read-only permission changes.
- [x] Added `internal/filesystem` snapshots and one shared durable replacement layer with exclusive same-directory staging, file sync, atomic replacement, no-replace creation, cleanup, and platform-specific namespace commits.
- [x] Migrated `write_file`, `edit_file`, `convert_encoding`, `change_line_endings`, BOM strip/add, `copy_file`, `move_file`, and `delete_file`.
- [x] Removed the duplicated handler-local `atomic.go` implementation.
- [x] Made conversion backups transactional: stage and sync the original first, preserve an existing backup until success, restore it on target failure, or remove a newly created backup.
- [x] Preserve a recovery copy and report its path if restoring the previous backup also fails.
- [x] Preserve file mode for replacements and mode/modification time for copies and backups where the platform represents them.
- [x] Defined fsync policy: sync staged files before commit; sync containing directories on Unix; use `MoveFileEx` write-through flags on Windows, where no portable directory fsync exists.
- [x] Added optimistic metadata/digest snapshots and immediate path revalidation for practical concurrent-modification detection.
- [x] Prevented initially missing targets and copy/move destinations from overwriting concurrently created paths through native no-replace commits.
- [x] Added failure-injection tests for staging sync failure, target commit failure, backup rollback, rollback failure recovery, cleanup, and concurrent creation/modification.
- [x] Added public handler regressions for transactional backup replacement and cancellation-safe copy, move, delete, and BOM changes.
- [x] Updated README, tool reference, changelog, publishing notes, plugin/marketplace metadata, Smithery metadata, runtime descriptions, and `server.json`; public MCP schemas remain unchanged.

TDD record:

- The first focused test run failed to compile because the shared mutation contract did not exist.
- After the first implementation, focused tests exposed a leaked temp file on sync failure and invalid POSIX-mode assumptions on Windows; cleanup was corrected and platform-specific assertions were added.
- Devil's-advocate review then exposed a concurrent-create overwrite risk for missing targets and a possible loss of the previous backup when rollback itself fails; both now have focused passing tests.

### R3 Definition of Done

- [x] File replacement, backup, copy, move, and delete behavior use shared filesystem primitives where their semantics overlap.
- [x] Directory creation and explicit `forceWritable` permission changes remain separate, documented policies rather than being forced into replacement semantics.
- [x] Staging data is synced before commit and namespace durability policy is explicit per platform.
- [x] Backup rollback and cleanup failures are surfaced; the last known-good backup is not deleted after rollback failure.
- [x] Practical concurrent modifications and concurrent destination creation are rejected.
- [x] Focused, handler, full, race, static-analysis, coverage, and cross-build checks pass.
- [x] No unrelated files changed.
- [x] R3 was committed locally as one reviewable change set in `79a9ada`.

## COMPLETED — R4 — Typed operation errors

- [x] Added transport-independent `internal/operation` error kinds for invalid input/path, access denial, symlink escape, not found, permissions, encoding, decoding, encoding output, conflicts, cancellation, limits, and filesystem failures.
- [x] Preserved original `Error()` text plus `errors.Is`/`errors.As` behavior while attaching operation and path metadata.
- [x] Converted existing handler, security, encoding, and mutation sentinels to typed errors without changing public messages.
- [x] Annotated exported security, encoding-detection, text-document, snapshot, replacement, copy, move, and delete boundaries.
- [x] Added one centralized handler mapping for MCP error results and `read_multiple_files` per-file codes.
- [x] Removed duplicated `classifyPathError` and `classifyReadError` string/sentinel switches.
- [x] Preserved public MCP schemas and compatibility-sensitive messages.
- [x] Added focused unit and integration tests for standard errors, joined errors, access denial, invalid paths, missing files, permissions, encoding failures, conflicts, cancellation, filesystem failures, and batch codes.

TDD record:

- The first focused test failed to compile because `internal/operation`, `mapOperationError`, and `errorResultFromError` did not yet exist.
- The minimal typed error contract and mapper then made the focused tests pass.
- Integration tests verified that existing security and mutation sentinels still satisfy `errors.Is` while exposing stable typed categories.
- Batch regressions verified unchanged `NOT_FOUND`, `ACCESS_DENIED`, `ENCODING`, and `OPERATION_FAILED` behavior.

### R4 Definition of Done

- [x] Typed categories cover all roadmap classes and standard `context`/`io/fs` sentinels.
- [x] Domain boundaries add categories without replacing human-readable causes.
- [x] MCP and batch conversion share one handler boundary; no duplicated string-based classifier remains.
- [x] Full tests, vet, builds, Staticcheck, coverage, race detector, and `git diff --check` pass.
- [x] No public schema changed and no unrelated repository file was modified.
- [x] R4 was preserved in the dedicated `920eb39` commit before subsequent identity and release-pipeline work.

## COMPLETED — R5 — Bounded ordered concurrency utilities

Implementation, verification, dedicated commit, deployment, and connector runtime verification are complete.

- [x] Inventoried worker-pool duplication in batch read, grep, tree/search, and other parallel operations.
- [x] Added `internal/concurrency.ProcessOrdered`, a generic bounded worker coordinator with deterministic serial commits and explicit cancellation policy.
- [x] Migrated `read_multiple_files` after confirming `readSingleFile` is the centralized single-item operation.
- [x] Migrated `grep_text_files` after confirming `searchSingleFile` is the centralized single-item operation.
- [x] Preserved cancellation, deterministic ordering, per-file partial failure, exact global match limits, and bounded pending results.
- [x] Preserved grep's decreasing per-dispatch match budget through an atomic remaining-budget value.
- [x] Kept `tree`, `directory_tree`, and `search_files` on the serial secure walker because traversal-time pruning, lexical order, hierarchy construction, and early limits are part of their behavior.
- [x] Added deterministic reverse-completion, saturation, cancellation-mode, and early-stop tests.
- [x] Passed focused and full race-detector suites.

TDD record:

- The first focused run failed to compile because `Stats`, `Options`, and `ProcessOrdered` did not exist.
- The minimal generic coordinator made ordering, worker-bound, cancellation, complete-position, and early-stop tests pass.
- `read_multiple_files` and grep focused regressions passed immediately after migration.
- Devil's-advocate review removed the buffered job queue so early stop cannot leave accepted-but-not-started expensive work behind.

### R5 Definition of Done

- [x] Only one production worker-pool implementation remains; `execution.go` retains its unrelated single process-wait goroutine.
- [x] In-flight work and out-of-order pending results are bounded by the worker count.
- [x] Commits occur serially in input order regardless of completion order.
- [x] `read_multiple_files` still produces one result for every requested path after cancellation.
- [x] Grep still preserves deterministic order, exact `maxMatches`, correct exact-limit `truncated` behavior, and early cancellation.
- [x] No public MCP input/output schema changed.
- [x] Documentation, runtime descriptions, publishing notes, changelog, and Registry template are aligned.
- [x] Full tests, vet, Staticcheck, govulncheck, coverage, race detector, manual operations, workflow lint, GoReleaser snapshot, Registry validation, Gitleaks, and `git diff --check` pass.
- [x] Temporary build and Registry-validation artifacts were removed.
- [x] Preserved R5 in dedicated commit `d5513a6`, built the final Windows binary, verified its version/hash, updated the launcher, restarted the tunnel, and verified active PID `28308`.

## COMPLETED — First fork release v1.8.0

Release preparation is committed as `edcffd2`; Unix and Windows CI regressions are corrected in follow-up commits `0ceb55e` and `4941a69`.

- [x] Bumped `plugin/.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` to `1.8.0`.
- [x] Cut `CHANGELOG.md` section `1.8.0 - 2026-07-25` and removed stale no-release wording.
- [x] Added `scripts/verify-release-version.js` and tests for repository consistency, tag mismatch, metadata mismatch, and malformed tags.
- [x] Made the release workflow reject a tag whose version differs from plugin or marketplace metadata.
- [x] Updated normal CI to execute both release-script test suites.
- [x] Passed focused Node and project-identity tests, full Go tests, vet, Staticcheck, govulncheck, manual server tests, race detector, and coverage.
- [x] Passed actionlint/ShellCheck, GoReleaser check, JSON/YAML parsing, PowerShell parsing, 34 local Markdown links, Gitleaks, and `git diff --check`.
- [x] Built a six-target GoReleaser snapshot and verified raw binary checksum names expected by the plugin.
- [x] Built and executed a disposable binary with embedded version `v1.8.0`.
- [x] Generated a checksum-backed `v1.8.0` manifest and passed live non-published MCP Registry validation.
- [x] Removed all snapshot, validation, and synthetic manifest artifacts.
- [x] Committed the preparation as `edcffd2` (`Prepare v1.8.0 release`).
- [x] Pushed R5 and `edcffd2` to `origin/main` through GitHub Desktop.
- [x] Diagnosed the failed Ubuntu `Test` job: an inherited permission test requested a byte-identical UTF-8 conversion, which intentionally creates no backup, then called `stat` on an empty path.
- [x] Corrected the Unix test to force `utf-8` to `utf-16-le`, fail immediately on handler errors, and require a non-empty backup path before checking permissions.
- [x] Passed focused contract tests, Linux test-package compilation, full Go tests, vet, Staticcheck, govulncheck, race detector, release-version verification, and `git diff --check`.
- [x] Committed the CI fix as `0ceb55e` (`Fix Unix backup permission test`).
- [x] Pushed `0ceb55e` to `origin/main` through GitHub Desktop.
- [x] Diagnosed the Windows failures as one root-canonicalization defect: active roots retained DOS 8.3 aliases while repeated validation returned canonical long paths, breaking conversion backups and grep; remaining walker/junction failures compared equivalent short and long paths textually.
- [x] Canonicalized CLI, updated, and merged roots; expanded Windows 8.3 components without following reparse points; retained final resolved containment validation.
- [x] Added an end-to-end Windows 8.3 test covering CLI roots, roots updates, roots merge, conversion backup, and grep, plus semantic path comparisons for walker and junction tests.
- [x] Made conversion and grep tests fail fast instead of cascading into empty-path operations or index panics.
- [x] Passed the exact failed Windows tests, full tests, vet, Staticcheck, govulncheck, race detector, Linux/macOS cross-builds, workflow lint, GoReleaser check, and `git diff --check`.
- [x] Completed a six-target `v1.8.0` release simulation, verified all raw asset checksums and embedded version, and passed live non-published Registry validation.
- [x] Removed all release-simulation artifacts and committed the Windows runtime fix as `4941a69` (`Fix Windows short-path root handling`).
- [x] Pushed `4941a69` to `origin/main` through GitHub Desktop.
- [x] Reproduced the three residual Windows CI failures locally by forcing `TEMP/TMP` through a workspace 8.3 alias.
- [x] Replaced textual path equality in deterministic grep, backup replacement, and deterministic file-search tests with `os.SameFile` identity checks.
- [x] Passed focused reproduction, full `go test -race ./...` under the short-path environment, full tests, vet, Staticcheck, govulncheck, and Linux test-package compilation.
- [x] Committed the final Windows assertion fix as `2aebd71` (`Fix Windows path identity assertions`).
- [x] Pushed `2aebd71` to `origin/main`.
- [x] Removed the incorrectly named remote tag `1.8.0`.
- [x] Created and published tag `v1.8.0` on commit `2aebd71`.
- [x] Verified `Release` jobs `test (ubuntu-latest)`, `test (windows-latest)`, `release`, and `publish-registry / publish` are green.
- [x] Verified the published tag resolves to `2aebd71439e4103bf7b0a48dc7d76fb155728be0`.
- [x] Verified 19 GitHub Release assets, including six raw binaries, platform archives, and `checksums.txt`.
- [x] Independently downloaded `checksums.txt` and `mcp-file-tools_windows_amd64.exe`; verified SHA-256 `0463ded458ae173146dc432d4a158263c776f00603f9de2583168a5f4aba3315` against both GitHub metadata and the checksum file.
- [x] Executed the published Windows amd64 binary and verified embedded version `1.8.0`.
- [x] Verified MCP Registry identity `io.github.zoster81/mcp-file-tools`, version `1.8.0`, status `active`, and `isLatest: true` with all six package checksums.
- [ ] Verify the end-user plugin download path and `check_for_updates` against the published release during the next runtime-upgrade task.

## COMPLETED AND COMMITTED — R6 — Execution preparation and tool metadata

- [x] Added `internal/execution` for shared process-level validation, timeout bounds, output limiting, cancellation, process waiting, and process-tree termination.
- [x] Kept `run_script` and `shell` authorization separate: `run_script` validates the script and cwd, while `shell` validates only cwd and leaves command text unrestricted.
- [x] Revalidated authorized working directories immediately before process launch.
- [x] Added streaming SHA-256 file snapshots and required `run_script` to verify script metadata and content immediately before launch.
- [x] Preserved historical validation order for cwd and timeout errors before interpreter or shell selection.
- [x] Added `internal/toolcatalog/catalog.json` as the authoritative names, titles, descriptions, and annotations source.
- [x] Embedded and validated the catalog in Go; runtime registration consumes it through `catalogTool`.
- [x] Made `server.template.json` keep an intentionally empty `tools` array and made the generator inject the Registry projection from the catalog.
- [x] Added runtime catalog tests for complete registration, uniqueness, descriptions, annotations, and documentation coverage.
- [x] Updated README, TOOLS, changelog, and publishing notes; public MCP input/output schemas remain unchanged.

TDD and adversarial-review record:

- The initial focused tests failed to compile because `internal/execution` and `internal/toolcatalog` did not exist.
- The first runtime-catalog integration test failed because the MCP SDK returns tools alphabetically rather than in registration order; the assertion was corrected to validate the complete unique set by name.
- The first script-replacement regression failed because Windows `os.SameFile` accepted a rapid delete/rename replacement; metadata plus streaming SHA-256 snapshots replaced the identity-only check.
- A first race-detector invocation did not run because `CGO_ENABLED` was disabled; it was rerun with the workspace-contained w64devkit GCC toolchain and passed.

R6 Definition of Done:

- [x] Shared execution mechanics do not authorize script paths or shell commands.
- [x] Script arguments remain separate process arguments and are not interpolated through a shell.
- [x] Timeout, output, cancellation, exit-code, and structured response behavior remain compatible.
- [x] Runtime metadata and Registry generation consume the authoritative catalog.
- [x] README links and TOOLS sections cover every catalog tool.
- [x] Focused, handler, complete, race, static-analysis, vulnerability, coverage, cross-build, manual-server, Registry-schema, JSON, Markdown-link, and diff checks pass.
- [x] No public MCP schema changed, no generated artifacts remain, and no unrelated repository files changed.
- [x] R6 was committed as `279808a`; the Windows amd64 binary was built and verified, and the launcher was updated.
- [x] Pushed `279808a` to `origin/main`, restarted the tunnel, and verified active MCP PID `25720`, tunnel PID `27732`, version `1.8.0-2-g279808a`, binary SHA-256, parent-child relationship, and listener `127.0.0.1:8080`.

# Deferred product work — not part of the active refactor milestone

Do not begin these items while another refactoring milestone is active unless explicitly reprioritized by the user.

- [ ] Improve BOMless UTF-16 structural detection without weakening binary classification.
- [ ] Add malformed UTF-16 validation beyond the guarantees currently provided by the decoder.
- [ ] Decide whether automatic UTF-16 detection applies only to known text/MQL extensions or all structurally valid files.
- [ ] Decide whether `mql-current`, `mql-legacy`, and `preserve` become public profiles or documentation-only presets.
- [ ] Stream large-file read, grep, and conversion paths.
- [ ] Add optional native HTTP/JSON or Streamable HTTP transport after a separate security design.

# Latest handoff — 2026-07-25

R4 implementation summary:

- Added `internal/operation` with stable transport-independent kinds for invalid input/path, access denial, symlink escape, not found, permissions, encoding, decoding, output encoding, conflicts, cancellation, limits, and filesystem failures.
- Added an operation error wrapper carrying kind, operation, path, and cause while preserving the original `Error()` text and `errors.Is`/`errors.As` behavior.
- Converted existing handler, security, and mutation sentinels to typed errors without changing their public messages.
- Annotated exported security validation, encoding detection, shared text-document, snapshot, replacement, copy, move, and delete boundaries.
- Added one handler mapping boundary for MCP error results and `read_multiple_files` per-file codes.
- Removed the duplicated `classifyPathError` and `classifyReadError` switches.
- Preserved all public MCP input/output schemas and compatibility-sensitive error strings.
- Aligned `README.md`, `TOOLS.md`, publishing notes, plugin and marketplace metadata, Smithery metadata, `server.json`, runtime tool descriptions, `CHANGELOG.md`, and this roadmap; public schemas remain unchanged.

TDD and adversarial-review record:

- The initial focused tests failed to compile because `internal/operation`, `mapOperationError`, and `errorResultFromError` did not exist.
- The minimum typed contract and centralized mapper made those tests pass before broader adoption.
- Focused tests verify standard context/filesystem sentinels, joined errors, typed metadata, path denial, missing paths, invalid directories, encoding detection, concurrent modification, destination conflicts, cancellation, and batch codes.
- Existing sentinels remain discoverable through `errors.Is`; wrappers do not prepend or rewrite public messages.
- Import-cycle risk was avoided by keeping `internal/operation` dependent only on the standard library.
- Joined errors use deterministic Go error-tree traversal; the first matching typed operation error is authoritative.

Final verification after the last Go edit, with all temporary directories and Go caches inside `D:\OpenAI-Tunnel`:

- [x] `gofmt -w` across every modified or untracked Go file.
- [x] Parsed repository JSON and YAML metadata, checked plugin JavaScript syntax, parsed both PowerShell launchers, verified six Markdown files with local links, and confirmed no obsolete R4 references remain.
- [x] Focused operation, encoding, security, filesystem, mapper, and batch tests.
- [x] `go test ./filetoolsserver/handler -count=1`.
- [x] `go test ./... -count=1`.
- [x] `go mod verify`.
- [x] `go vet ./...`.
- [x] Windows `go build ./...`.
- [x] Linux amd64 cross-build with `CGO_ENABLED=0`.
- [x] macOS arm64 cross-build with `CGO_ENABLED=0`.
- [x] `D:\OpenAI-Tunnel\tools\staticcheck\bin\staticcheck.exe ./...`.
- [x] `go test -cover ./... -count=1` — handler 74.0%, encoding 73.6%, filesystem 69.3%, operation 62.8%, security 83.4%.
- [x] `go test -race ./... -count=1` using the workspace-contained w64devkit toolchain.
- [x] `git diff --check`.
- [x] Git status, tracked/untracked file list, diff statistics, complete 1503-line patch review, active process, and repository artifact state reviewed; no unrelated repository files or build artifacts changed.
- [x] Built and version-checked a disposable pre-commit Windows binary.
- [x] Committed R4 as `920eb39`, built the final `v1.7.3-18-g920eb39` Windows binary, verified its SHA-256, and updated `start-tunnel.ps1` without restarting the tunnel.

R4 repository files changed:

- Added `internal/operation/errors.go` and `internal/operation/errors_test.go`.
- Added `filetoolsserver/handler/error_mapping.go` and `filetoolsserver/handler/error_mapping_test.go`.
- Added typed-error integration tests under `internal/security` and `internal/filesystem`.
- Updated handler error sentinels, validation, shared text-document consumers, direct MCP error conversion, and batch read mapping.
- Updated encoding detection, security validation, filesystem mutation boundaries, runtime descriptions, `README.md`, `TOOLS.md`, publishing notes, plugin/marketplace/Smithery metadata, `server.json`, `CHANGELOG.md`, and `D:\OpenAI-Tunnel\todo.md`.

R4 completion record:

- R4 commit: `920eb398139c44ea40997dccf2a8d1dd1f142362` (`Add typed operation error model`).
- R4 is included in current `HEAD` `edcffd2326ad25a901cf41847b8d6fc5350d06b7`.
- Local tracking `origin/main` includes both `920eb39` and the subsequent identity commit `daf83eb`; R5 `d5513a6` remains local.
- R4 standalone binary: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_920eb39.exe`, version `v1.7.3-18-g920eb39`, size `10384384` bytes, SHA-256 `54E94448D7E1E89BAC64876F278BDE20A69538D3A0C6AF9498C7261B2DF6FCD5`.
- R4 runtime restart was completed and later superseded by the active R5 `d5513a6` binary.
- `start-tunnel.ps1` points to the active R5 `d5513a6` binary, running as PID `28308`.

Remaining risks and deferred work:

- Operation metadata is internal and intentionally not added to public MCP schemas.
- Handler-local literal validation failures remain direct MCP error results when no reusable domain error exists; propagated domain failures use the centralized boundary.
- Unknown untyped failures conservatively map to `IO_ERROR`; new domain primitives should attach a specific kind at their public boundary.
- The first typed cause in an `errors.Join` tree is authoritative, so callers should order the primary operation failure before cleanup or rollback errors.
- R5 concurrency consolidation is committed as `d5513a6`, built, deployed, and verified; repository push remains pending together with release-preparation commit `edcffd2`.
- Full connector runtime verification of the R4 binary was completed after the user-managed restart.
- Linux and macOS builds were cross-compiled but not runtime-executed in this Windows session.

# R5 implementation handoff — committed and prepared for deployment

## Implementation summary

- Added `internal/concurrency/ordered.go` with generic `ProcessOrdered`, bounded worker selection, unbuffered job dispatch, buffered completion delivery, deterministic serial commits, optional dispatch continuation after parent cancellation, early-stop cancellation, and run statistics.
- Added `internal/concurrency/ordered_test.go` covering reverse completion order, maximum in-flight saturation, cancellation that stops new dispatch, cancellation that still completes every input position, and commit-driven early stop.
- Migrated `filetoolsserver/handler/read_multiple.go` to the shared coordinator while preserving one output per input, input order, partial failures, stable error codes, and cancellation results.
- Migrated `filetoolsserver/handler/grep.go` to the shared coordinator while preserving lexical file order, exact global `maxMatches`, exact-limit `truncated=false`, early stop when a match is omitted, and decreasing per-dispatch match budgets through an atomic remaining value.
- Kept `tree`, `directory_tree`, and `search_files` serial on `internal/filesystem.Walk`; parallelizing traversal would break hierarchy construction, traversal-time pruning, deterministic lexical order, or early limits.
- Updated `README.md`, `TOOLS.md`, `docs/PUBLISHING.md`, `CHANGELOG.md`, `filetoolsserver/server.go`, `server.template.json`, and this handoff. Public MCP input/output schemas remain unchanged.

## Verification completed after the final Go edit

- Focused TDD baseline failed to compile before the coordinator existed, then passed after the minimal implementation.
- Focused normal and race tests passed for `internal/concurrency`, `read_multiple_files`, grep handlers, and single-file grep.
- `go mod verify`, `go test ./... -count=1`, `go vet ./...`, Staticcheck, and govulncheck passed; govulncheck reported no vulnerabilities.
- `go run test_server.go` passed every manual operation check.
- `go test -race ./... -count=1` passed using the workspace-contained Windows CGO toolchain.
- Coverage passed: handler 73.3%, concurrency 87.7%, config 100.0%, encoding 73.6%, filesystem 69.6%, operation 62.8%, security 83.4%, updater 25.0%.
- Node registry-generator tests passed 2/2; actionlint, GoReleaser check, JSON/YAML parsing, both PowerShell parsers, Gitleaks, and `git diff --check` passed.
- Gitleaks scanned 206 commits and found no leaks.
- GoReleaser snapshot built the six unique targets: Windows/Linux/macOS on amd64/arm64; archives and checksums were generated.
- A generated non-published manifest passed MCP Publisher 1.8.0 live validation against the Registry.
- All `dist` output and synthetic Registry-validation files were removed.

## Repository and runtime state

- Branch and committed baseline: local `main` points to `d5513a6129b2727c3d2eabdf1d3612fb78cccdb6`; local tracking `origin/main` remains at `daf83eb5b4b8e9c85566fa786ea5987d70b5639e`.
- R5 status: committed locally as `d5513a6`, not pushed, built, and prepared in the launcher.
- Prepared binary: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_d5513a6.exe`, version `v1.7.3-20-gd5513a6`, size `10404352` bytes, SHA-256 `CC8AFA22446D5D0E23B9B5EB298FC2D1F8217C79920E1B5ABABEAF60EB25045A`.
- Active runtime: PID `28308`, `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_d5513a6.exe`, version `v1.7.3-20-gd5513a6`, containing R1-R5.
- `start-tunnel.ps1` points to the active `d5513a6` binary and passed byte-level plus PowerShell parser verification.
- R5 runtime verification is complete; the active milestone is now the first fork release `v1.8.0`.

## Remaining risks and checks not performed

- Worker callbacks must continue to honor cancellation; the coordinator cannot forcibly interrupt blocking operating-system calls.
- GitHub-hosted CI runners have not executed commit `d5513a6`.
- The Linux and macOS binaries were cross-built but not runtime-executed in this Windows session.
- The final Windows binary was version-checked only; full connector runtime verification remains pending the user-managed restart.
- No push was performed for R5.

# Fork identity alignment — completed and committed

## Purpose

Remove the inherited original-owner identity from source code, scripts, workflows, release metadata, and registry metadata. References to the original project remain only in explicit project-lineage and upstream-synchronization documentation.

## Implemented

- Changed the Go module path to `github.com/zoster81/mcp-file-tools`.
- Rewrote every internal Go import, the manual test harness, and GoReleaser linker flags to the fork module path.
- Replaced the inherited `server.json` with fork-owned `server.template.json`; the template contains the `io.github.zoster81/mcp-file-tools` identity and release-neutral checksum placeholders.
- Reduced the registry description to 93 characters after live MCP Registry validation reported the documented 100-character maximum.
- Added `scripts/generate-server-json.js`, which creates the publishable `server.json` only from a real fork release version and all six raw-binary SHA-256 entries.
- Added Node tests covering successful manifest generation, archive-checksum tolerance, and fail-closed behavior when a binary checksum is missing.
- Converted `.github/workflows/publish-registry.yml` into a fork-only reusable/manual workflow using GitHub OIDC.
- Updated `.github/workflows/release.yml` to call registry publication directly after GoReleaser succeeds, avoiding reliance on a secondary release event.
- Added `internal/projectidentity` regression tests that enforce the fork module, GoReleaser, registry workflow, release workflow, registry template, generator identity, and the exact allowed locations/counts for original-owner references.
- Removed the upstream plugin installation command and all operational upstream references from README/plugin documentation.
- Retained exactly four original-owner references: one lineage reference in `README.md`, one changelog attribution, and two upstream synchronization references in `docs/PUBLISHING.md`.
- Updated build/test workflow path filters so registry-template changes execute CI instead of being ignored.
- Added `scripts/validate-workflows.sh` with pinned actionlint 1.7.12 and ShellCheck 0.11.0 downloads verified by fixed SHA-256 values.
- Pinned Staticcheck v0.7.0, govulncheck v1.1.4, GoReleaser v2.17.0, and MCP Publisher v1.8.0 in CI/release workflows.
- Updated `CHANGELOG.md`, `README.md`, `docs/PUBLISHING.md`, `plugin/README.md`, and `scripts/bump-version.js`.

## TDD and verification

- The initial `internal/projectidentity` test failed on the old module path, 50+ source imports, GoReleaser, registry workflow, inherited manifest, plugin documentation, and excess README/publishing references.
- Final `go test ./internal/projectidentity -count=1` passes.
- `go mod tidy` changed only the module declaration; `go.sum` remained unchanged.
- `go mod verify` passed.
- `go test ./... -count=1` passed under the fork module path.
- `node --test scripts/generate-server-json.test.js` passed both tests.
- `go vet ./...` and workspace Staticcheck passed.
- Repository JSON and YAML metadata parsed successfully.
- `go test -race ./... -count=1` passed with the workspace-contained Windows CGO toolchain.
- Windows build and Linux amd64/macOS arm64 cross-builds passed with fork-path linker flags; the disposable Windows binary returned the expected embedded version.
- `go run test_server.go` passed all manual server-operation checks.
- Thirty-four local Markdown links were verified.
- `git diff --check` passed.
- `actionlint` 1.7.12 with ShellCheck 0.11.0 validated all four GitHub Actions workflows with zero errors.
- `scripts/validate-workflows.sh` passed Bash syntax validation and ShellCheck with LF-only UTF-8 encoding.
- GoReleaser 2.17.0 completed a full snapshot build for Windows, Linux, and macOS on amd64/arm64, produced archives/checksums, and all temporary `dist` artifacts were removed.
- MCP Publisher 1.8.0 live validation passed against `https://registry.modelcontextprotocol.io` using a generated non-published fork manifest with six synthetic checksum entries. Publication, login, and credential flows were not executed.
- The complete 1617-line patch was reviewed; 51 existing Go files were verified as module-path replacement plus deterministic `gofmt` import ordering only.

## Workspace validation toolchain

Installed exclusively under `D:\OpenAI-Tunnel\tools`, with no global PATH or profile changes:

- actionlint 1.7.12;
- ShellCheck 0.11.0;
- govulncheck 1.1.4;
- GoReleaser 2.17.0;
- Gitleaks 8.30.1;
- MCP Publisher 1.8.0;
- Cosign 3.1.2;
- existing Staticcheck 2026.1 / v0.7.0.

Toolchain metadata and activation:

- `D:\OpenAI-Tunnel\tools\toolchain-manifest.json` records exact executable SHA-256 values, source assets, versions, and verification methods.
- `D:\OpenAI-Tunnel\tools\Activate-Toolchain.ps1` prepends the versioned tool directories only to the current PowerShell process and confines Go caches and temporary files to the workspace.
- Downloaded release assets were verified against GitHub release digests; actionlint, GoReleaser, Gitleaks, MCP Publisher, and Cosign were additionally checked against their official checksum files where available.
- Cosign verified its own Sigstore bundle, the GoReleaser checksum bundle, and the MCP Publisher archive bundle using explicit certificate identities and issuers.
- MCP Publisher SBOM and Sigstore metadata are retained under the versioned installation directory.
- `govulncheck ./...` reported no vulnerabilities, Staticcheck passed, and the latest Gitleaks run scanned 206 commits with no leaks found.
- GoReleaser validated `.goreleaser.yml`; actionlint validated four workflows with ShellCheck available.
- The optional actionlint `pyflakes` integration remains unavailable because the repository has no Python workflow steps and does not require that runtime.

## Repository and runtime state

- Branch: `main`, aligned with local tracking `origin/main`.
- `HEAD` and `origin/main`: `279808acb2ceb8d9052de5c4d558234c1e44401d` (`Consolidate execution preparation and tool metadata`).
- Release tag `v1.8.0` remains on `2aebd71` and the GitHub Release is published.
- Repository working tree: clean; no generated repository artifacts or temporary Registry manifests remain.
- Previous release binary retained: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64.exe`, version `1.8.0`, size `10440704` bytes, SHA-256 `0463DED458AE173146DC432D4A158263C776F00603F9DE2583168A5F4ABA3315`.
- Active R6 binary: `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_279808a.exe`, version `1.8.0-2-g279808a`, size `10421248` bytes, SHA-256 `4B5F6DBCEAA49400A3AF73A9FD60D471941AA51BA8E1A27769FFF9424D548CE1`.
- `start-tunnel.ps1` points to the active R6 binary; UTF-8 BOM and LF endings are preserved. File size is `6116` bytes and SHA-256 is `2A7E503F9B114CD550E02B7D5A1BAADE79CF2431A162E60D9D56F21FEB6CA3E1`.
- Active MCP runtime: PID `25720`, parent PID `27732`, executable `D:\OpenAI-Tunnel\mcp-file-tools_windows_amd64_279808a.exe`.
- Tunnel client PID `27732` owns listener `127.0.0.1:8080`; the MCP parent-child relationship is verified.
- Remote release `v1.8.0` is published and MCP Registry lists version `1.8.0` as active/latest.
- R6 source, commit, push, build, launcher update, restart, and active-connector runtime verification are complete.

# R6 implementation handoff — committed, pushed, and deployed

## Files changed

Modified:

- `CHANGELOG.md`
- `README.md`
- `TOOLS.md`
- `docs/PUBLISHING.md`
- `filetoolsserver/handler/execution.go`
- `filetoolsserver/server.go`
- `internal/filesystem/mutation.go`
- `internal/filesystem/mutation_test.go`
- `scripts/generate-server-json.js`
- `scripts/generate-server-json.test.js`
- `server.template.json`

Added:

- `filetoolsserver/handler/execution_test.go`
- `filetoolsserver/tool_catalog_test.go`
- `internal/execution/execution.go`
- `internal/execution/execution_test.go`
- `internal/toolcatalog/catalog.go`
- `internal/toolcatalog/catalog.json`
- `internal/toolcatalog/catalog_test.go`

## Final verification

- [x] `gofmt` on every changed Go file.
- [x] Focused execution, filesystem snapshot, tool catalog, handler-policy, runtime-catalog, and Registry-generator tests.
- [x] `go mod verify`.
- [x] `go test ./... -count=1`.
- [x] `go vet ./...`.
- [x] Windows `go build ./...`.
- [x] Workspace Staticcheck `./...`.
- [x] govulncheck `./...`: no vulnerabilities found.
- [x] Coverage: server 74.1%, handler 78.7%, execution 78.6%, filesystem 70.0%, toolcatalog 70.7%; all package tests passed.
- [x] `go test -race ./... -count=1` with `CGO_ENABLED=1` and workspace w64devkit GCC.
- [x] Linux amd64 and macOS arm64 cross-builds with `CGO_ENABLED=0`.
- [x] `go run test_server.go`: all manual operations passed.
- [x] Node generator and release-version suites: 7/7 passed.
- [x] Generated synthetic `1.8.1` Registry manifest passed live `mcp-publisher validate`; temporary files were removed.
- [x] Repository JSON parsed; local Markdown links passed.
- [x] GoReleaser configuration check, actionlint/ShellCheck workflow validation, and Gitleaks scan passed.
- [x] `git diff --check` passed; status and complete diff were reviewed.

## Remaining risks and checks not performed

- The final path check and process creation cannot be made fully atomic with the current path-based `exec` API; a handle-relative launch design would be required to eliminate the last TOCTOU window.
- `shell` remains intentionally unrestricted after cwd validation and must stay disabled unless explicitly needed in a trusted environment.
- Linux and macOS targets were cross-built but not runtime-executed in this Windows session.
- No new tag or GitHub Release was created for R6; the deployed binary is a commit build, version `1.8.0-2-g279808a`.
- Commit, push, versioned Windows build, launcher update, tunnel restart, and active-connector verification are complete.
- The previous official `1.8.0` binary remains available as a local rollback artifact.

Next chat must read, in order:

1. `D:\OpenAI-Tunnel\todo.md`
2. `D:\OpenAI-Tunnel\mcp-file-tools-src\internal\execution\execution.go`
3. `D:\OpenAI-Tunnel\mcp-file-tools-src\filetoolsserver\handler\execution.go`
4. `D:\OpenAI-Tunnel\mcp-file-tools-src\internal\toolcatalog\catalog.json`
5. `D:\OpenAI-Tunnel\mcp-file-tools-src\filetoolsserver\server.go`
6. `D:\OpenAI-Tunnel\mcp-file-tools-src\scripts\generate-server-json.js`
7. `D:\OpenAI-Tunnel\mcp-file-tools-src\CHANGELOG.md`

# Verification baseline

Run from `D:\OpenAI-Tunnel\mcp-file-tools-src` after each milestone:

```text
gofmt -w <changed Go files>
go test ./filetoolsserver/handler -count=1
go test ./... -count=1
go vet ./...
go build ./...
staticcheck ./...
go test -cover ./...
go test -race ./... -count=1
git diff --check
git status --short --branch
git diff --stat
git diff
```

Use the workspace-contained Staticcheck and Windows CGO toolchain already used by the project when they are required.
Do not claim a check passed unless it was actually executed after the final edit.

# Handoff update rule

Before ending any work session:

1. Update the `ACTIVE` milestone checklist.
2. Record any failing test and why it fails.
3. Record every uncommitted file and whether it must be preserved.
4. Mark completed items `[x]`; do not leave completed work unchecked.
5. Move `ACTIVE` to the next milestone only after the current Definition of Done passes.
6. State the exact files the next chat must read.
7. State checks performed, checks not performed, remaining risks, commit status, push status, active binary, and runtime version.