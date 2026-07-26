# Development Roadmap to 2.0.0

This is the authoritative product roadmap for `zoster81/mcp-file-tools`.

The project is in internal development. Development commits may be built and deployed locally, but no intermediate public release is planned. The next public release target is `2.0.0`, after the native MCP Streamable HTTP server and all completion gates in this document are finished.

Current milestone status and completion gates live in this document. Reusable engineering checks live in [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md), contributor workflow in [`CONTRIBUTING.md`](../CONTRIBUTING.md), scoped agent guidance in [`AGENTS.md`](../AGENTS.md), and completed R1-R6 engineering outcomes in [ROADMAP_HISTORY.md](ROADMAP_HISTORY.md).

## Operating rules

- Only one milestone may be `ACTIVE` at a time.
- Complete milestones in order unless maintainers explicitly reprioritize them.
- Keep changes atomic and limited to the active milestone.
- Use content bytes and structural evidence for encoding detection. File extensions must not select or bias an encoding.
- Treat domain-specific files, including MQL sources, as ordinary test fixtures rather than special encoding profiles.
- Preserve stdio support while adding Streamable HTTP.
- Keep `run_script` and `shell` disabled by default on every transport.
- Build internal commit binaries as needed; do not create public tags or releases before the 2.0 release gate.
- Every milestone must pass [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md) before it is marked complete.

## Milestone overview

| Milestone | Status | Outcome |
|---|---|---|
| R7 | COMPLETE | Replaced the historical roadmap with a clear operating plan and removed domain-specific MQL emphasis. |
| R8 | COMPLETE | Generic, conservative, extension-independent encoding detection, including BOMless UTF-16. |
| R9 | COMPLETE | Real bounded-memory streaming for large-file read, grep, conversion, line-ending, and BOM paths. |
| R10 | COMPLETE | Resolve public API inconsistencies and compatibility debt before the 2.0 boundary. |
| R11 | COMPLETE | Separate transport bootstrap from the shared MCP server and tool policies. |
| R12 | ACTIVE | Approve the Streamable HTTP threat model and security design. |
| R13 | QUEUED | Implement and verify native MCP Streamable HTTP while preserving stdio. |
| R14 | QUEUED | Complete hardening, CI, packaging, migration documentation, and the 2.0.0 release. |

---

# R7 — Roadmap and documentation reset

## Goal

Create one pragmatic development sequence, make the current limitations explicit, and remove the incorrect impression that MQL filenames receive special encoding behavior.

## Checklist

- [x] Publish a concise, contributor-facing R1-R6 engineering history.
- [x] Create this authoritative R7-R14 roadmap.
- [x] Create one reusable development and verification checklist.
- [x] Separate public roadmap, contributor guidance, and engineering history from private operator state.
- [x] State consistently that internal builds continue until the next public release, `2.0.0`.
- [x] Move Claude Code plugin verification out of active development and into the final 2.0 release gate.
- [x] Remove MQL-specific product claims from README, tool documentation, plugin documentation, runtime instructions, and tool catalog descriptions.
- [x] Replace MQL-specific examples with neutral filenames and content-based guidance.
- [x] Rename MQL acceptance tests and fixture directories to generic encoding acceptance names.
- [x] Keep coverage for UTF-16, UTF-8, multilingual text, BOMs, and CRLF/LF behavior after the rename.
- [x] Document that current BOMless UTF-16 auto-detection is incomplete and explicit encoding may still be required.
- [x] Correct documentation that currently implies large files are streamed when the shared document path still uses `os.ReadFile`.
- [x] Link README and publishing notes to this roadmap and the development checklist.
- [x] Add a root `AGENTS.md`, scoped subsystem guides, and a portable `CONTRIBUTING.md`.
- [x] Add a regression test that rejects private operator paths and connector identifiers in tracked text.
- [x] Run catalog, documentation, Go, Node, formatting, link, and diff verification.

## Completion record

Completed on 2026-07-26. Public planning, contributor checks, scoped agent guidance, and engineering history are separated by responsibility; private operator state is excluded from tracked text; public descriptions are extension-independent; generic UTF-16/UTF-8 acceptance fixtures pass under `.data`, extensionless, and `.random` filenames; and documentation/catalog drift checks are green.

---

# R8 — Generic encoding detection

## Goal

Infer encodings from byte structure and decoded-content evidence, independently of filenames and extensions. Prefer an explicit ambiguous result over a confident false classification.

## Design requirements

- BOM detection remains authoritative for UTF-8, UTF-16, and UTF-32 signatures.
- Extension, basename, directory, or language-specific content must not influence the selected encoding.
- BOMless UTF-16 LE/BE detection must combine multiple independent signals rather than rely only on alternating NUL bytes.
- Malformed Unicode input must be rejected or left ambiguous rather than silently repaired.
- Binary classification must occur after candidate decoding as well as on raw structural evidence.
- Detection results must remain deterministic for the same byte sequence.

## Checklist

- [x] Add structural BOMless UTF-16 LE and BE candidate detection.
- [x] Measure NUL distribution on even and odd byte positions.
- [x] Require even byte length for UTF-16 candidates.
- [x] Validate UTF-16 code units and surrogate pairs.
- [x] Reject isolated high/low surrogates and truncated pairs.
- [x] Measure printable, whitespace, replacement, control, and NUL rune ratios after decoding.
- [x] Verify decode/encode round-trip consistency for candidates.
- [x] Define conservative confidence thresholds and a minimum evidence size.
- [x] Avoid forcing a candidate when evidence is insufficient.
- [x] Integrate the same decision logic into sample, chunked, and full modes.
- [x] Verify candidate decisions across chunk boundaries.
- [x] Add fixtures with `.txt`, `.dat`, no extension, random extensions, and identical content under different names.
- [x] Add Latin, Cyrillic, Greek, Hebrew, Arabic, CJK, emoji, and mixed-script fixtures.
- [x] Add empty, BOM-only, very short, odd-length, truncated, and malformed UTF-16 cases.
- [x] Add executable, image, archive, random-byte, sparse-NUL, and binary-structure false-positive tests.
- [x] Fuzz detection and Unicode validation.
- [x] Document confidence semantics and ambiguous results.

## Completion gate

The same byte sequence must produce the same result under any filename. BOMless UTF-16 must be recognized only when structural and decoded-text evidence agree, and binary false-positive tests must pass.

## Completion record

Completed on 2026-07-26. Detection now uses one conservative UTF-16 LE/BE classifier with code-unit and surrogate validation, decoded-text metrics, NUL-byte parity, round-trip checks, deterministic confidence, and explicit ambiguity. Sample, chunked, and full modes share the same decision semantics; chunked analysis preserves surrogate state across 128 KiB boundaries. Tests cover identical bytes under unrelated filenames, multilingual scripts and emoji, malformed and short input, BOM-only files, legacy encodings, executable/image/archive/random data, and public read/grep integration. A bounded fuzz smoke test and the applicable verification ladder passed; the race detector was unavailable and no standalone deployment build was performed.

---

# R9 — Bounded-memory streaming pipeline

## Goal

Make large-file behavior match the documented memory guarantees. Streaming operations must not load complete source files, while operations that inherently require full-document state must reject oversized input before allocation.

## Architecture requirements

- Detection, BOM handling, decoding, line framing, and consuming operations must be separable streaming stages.
- Multibyte sequences, UTF-16 code units, CRLF pairs, and regex context may span chunks.
- Mutation output must be staged and synced before atomic commit without first materializing the complete target as `[]byte`.
- Memory bounds must account for concurrent workers, decoded expansion, result buffers, and exceptionally long lines.

## Checklist

- [x] Define explicit per-operation memory and line-length limits.
- [x] Add a shared incremental decoder for all registered encodings.
- [x] Add chunk-boundary tests for multibyte text, surrogate pairs, BOMs, CRLF, and lone CR/LF.
- [x] Stream `read_text_file` while preserving offset, limit, total line count, and character truncation semantics.
- [x] Bound aggregate memory in `read_multiple_files`, not only worker count.
- [x] Stream `grep_text_files` with a bounded previous-line ring buffer and bounded following context.
- [x] Preserve deterministic file and match order with streaming workers.
- [x] Stream `convert_encoding` from decoder to staged encoder output.
- [x] Preserve exact CRLF, LF, CR, and mixed line-ending sequences during conversion.
- [x] Add writer/reader-based mutation staging APIs.
- [x] Stream `detect_line_endings` and `change_line_endings`.
- [x] Stream `manage_bom` prefix inspection and staged copy.
- [x] Define and enforce an explicit full-document size limit for `edit_file` and unified diff generation.
- [x] Remove or make private `DetectSample` after all byte-slice consumers migrate.
- [x] Remove inaccurate streaming comments and warnings from configuration and documentation.
- [x] Test cancellation, read failures, write failures, disk-full simulation, cleanup, and concurrent source changes mid-stream.
- [x] Benchmark memory and throughput on representative small, medium, and large files.

## Completion gate

Every operation documented as streaming must have a verified bounded-memory path. Large-file tests must demonstrate that memory does not scale linearly with complete input size except for explicitly bounded full-document operations such as editing.

## Completion record

Completed on 2026-07-26. A shared read session now separates random-access detection from one sequential SHA-256 pass; incremental decoders support all 24 registered encodings; bounded line framing rejects decoded lines above 16 MiB; and `MCP_MEMORY_THRESHOLD` is enforced as the default hard budget for single-read output, aggregate batch output, retained grep state, inconsistent-line results, and full-document editing. Read, batch, grep, encoding conversion, line-ending detection/conversion, and BOM mutation now use bounded streams or disk staging while preserving BOM, encoding, ordering, cancellation, no-op, backup, and concurrent-modification behavior. Benchmarks on 1, 16, and 64 MiB inputs showed constant allocation footprints for line scanning and line-ending transformation; the applicable Go, static-analysis, vulnerability, manual MCP, documentation, and repository gates passed. The race detector and release-adjacent multi-platform checks remain deferred to R14.

---

# R10 — Public API and compatibility cleanup

## Goal

Use the major-version boundary to resolve inconsistent schemas, defaults, deprecated tools, and unsupported promises before the 2.x API is stabilized.

## Checklist

- [x] Remove the deprecated `directory_tree` tool and retain `tree` as the single recursive tree API.
- [x] Use UTF-8 as the international default for newly created files; retain `MCP_DEFAULT_ENCODING` and explicit legacy encodings such as `cp1251` as overrides.
- [x] Define behavior for empty and ambiguous files across every text tool.
- [x] Keep UTF-32 as BOM-management only rather than an incomplete registered text encoding.
- [x] Normalize public output JSON to camelCase, including `hasBOM`.
- [x] Define one stable public error-code vocabulary for single-tool `_meta` and batch items.
- [x] Define configurable limits for file size, decoded characters, line length, batch size, matches, output, and sessions.
- [x] Review every tool input/output schema and preserve fields not explicitly listed as breaking changes.
- [x] Remove obsolete `directory_tree` code and full API types; retain the documented stringified-array repair for current MCP client interoperability.
- [x] Produce a 1.8-to-2.0 migration table before implementation is finalized.
- [x] Update catalog, docs, manual tests, and schema compatibility tests together.

## Completion gate

All intentional breaking changes are explicit, tested, and listed in the migration guide. No deprecated or internally inconsistent public API remains accidentally carried into 2.x.

## Completion record

Completed on 2026-07-26. The public catalog now contains 23 tools after removing `directory_tree`; `tree` is the sole recursive tree API. Output fields use camelCase, single-tool errors expose `_meta.errorCode`, and batch errors share the same stable vocabulary. Empty files are explicitly assumed UTF-8, ambiguous non-empty content requires an encoding override, and UTF-32 remains BOM-management only. Separate `MCP_MAX_*` limits cover file input, decoded characters, line length, batch size, matches, output, and future HTTP sessions while `MCP_MEMORY_THRESHOLD` remains a deprecated file/output fallback. The complete change set and migration table are protected by schema, catalog, configuration, encoding-policy, manual MCP, and regression tests.

---

# R11 — Transport-independent server architecture

## Goal

Run one shared MCP server implementation through stdio and Streamable HTTP without duplicating tool registration, policies, roots, or error behavior.

## Architecture decision

- Allowed directories are process-wide policy. Every connection or future HTTP session attached to one server process shares the same configured roots, tool catalog, limits, execution flags, and error behavior.
- Sessions isolate protocol lifecycle, requests, cancellation, and concurrency; they are not per-agent filesystem identities, ACLs, or sandboxes.
- Prompt instructions may restrict an agent to writing in one project while reading shared projects, documentation, or libraries, but the server does not enforce those per-agent conventions.
- Technical isolation requires separate server processes with narrower roots and, where concurrent Git writes are possible, separate checkouts or worktrees.
- Startup directories are authoritative and immutable for the process. Dynamic MCP client roots remain a stdio-only compatibility path when no startup directories are configured. Future HTTP sessions will not mutate process roots.
- R11 does not add an HTTP listener, authentication, session manager, or network policy; those remain ordered work for R12 and R13.

## Checklist

- [x] Separate configuration loading from CLI parsing.
- [x] Separate server construction from transport startup.
- [x] Keep one authoritative tool catalog and registration path.
- [x] Define a transport-neutral server lifecycle abstraction.
- [x] Define explicit CLI/config transport selection.
- [x] Preserve stdio as a supported transport.
- [x] Keep allowed-directory policy authoritative and transport-independent.
- [x] Keep execution feature flags and authorization identical across transports.
- [x] Make logging, cancellation, graceful shutdown, and update checks lifecycle-aware.
- [x] Add equivalence tests for tools/list metadata and representative tool calls across transport adapters.

## Completion gate

The stdio executable uses the new architecture without behavior regression, and a second transport can be attached without duplicating handlers or weakening policy boundaries.

## Completion record

Completed on 2026-07-27. Configuration loading, CLI parsing, server construction, and transport startup are separate. `BuildServer` owns one shared 23-tool registration and process-wide policy; the stdio runner is lifecycle-aware and responds to process cancellation; explicit `--transport=stdio` and `MCP_TRANSPORT=stdio` selection preserve stdio as the sole implemented transport. Multiple connections to one server are verified to expose equivalent tool catalogs and the same configured roots, while provided configuration controls handler behavior. Dynamic client roots are limited to roots-capable stdio clients started without configured directories, and empty updates remove stale dynamic access. The complete race-detector suite and Gitleaks scans of Git history plus the working tree passed. HTTP listeners, authentication, and session policy remain unimplemented pending R12 and R13.

---

# R12 — Streamable HTTP security design

## Goal

Approve a concrete threat model and secure defaults before exposing filesystem and optional execution tools over an HTTP listener.

## Checklist

- [ ] Document assets, trust boundaries, actors, and supported deployment models.
- [ ] Preserve the R11 process-wide root model: all HTTP sessions share startup roots, client roots cannot mutate them, and per-agent isolation uses separate processes when required.
- [ ] Bind to loopback by default.
- [ ] Require explicit configuration for non-loopback binding.
- [ ] Define token authentication and reverse-proxy integration.
- [ ] Use constant-time credential comparison where applicable.
- [ ] Define secure token loading without command-line secret exposure.
- [ ] Validate `Host` and `Origin` and address DNS rebinding.
- [ ] Disable CORS by default.
- [ ] Define TLS expectations for direct and reverse-proxy deployments.
- [ ] Set request body, header, connection, and concurrency limits.
- [ ] Set read-header, read, write, idle, session, and shutdown timeouts.
- [ ] Define session identifiers, expiry, cleanup, and hijacking protections.
- [ ] Define CSRF and browser-origin protections.
- [ ] Prevent sensitive headers, tokens, and file contents from entering logs.
- [ ] Keep `run_script` and `shell` disabled by default.
- [ ] Require a distinct explicit opt-in for execution over HTTP.
- [ ] Define rate limiting and denial-of-service behavior.
- [ ] Define trusted-proxy handling without accepting spoofed forwarding headers.
- [ ] Define health and readiness endpoints that expose no sensitive data.
- [ ] Define security tests and release-blocking findings.

## Completion gate

A reviewed security design exists, every identified threat has a mitigation or explicit accepted risk, and implementation tests are specified before HTTP code is merged.

---

# R13 — Native MCP Streamable HTTP

## Goal

Implement the MCP Streamable HTTP transport according to R11 and R12 while preserving stdio behavior.

## Checklist

- [ ] Implement the Streamable HTTP endpoint using the shared server.
- [ ] Support required JSON-RPC request and streaming response behavior.
- [ ] Implement session creation, lookup, expiration, and cleanup.
- [ ] Propagate disconnect and request cancellation into tool contexts.
- [ ] Implement graceful shutdown for listeners and active sessions.
- [ ] Add health and readiness endpoints.
- [ ] Apply authentication, host/origin, size, timeout, and concurrency policies from R12.
- [ ] Reject malformed content types, methods, JSON, and protocol messages deterministically.
- [ ] Keep stdio startup and protocol output unchanged.
- [ ] Test simultaneous clients and concurrent sessions.
- [ ] Test disconnect, reconnect, timeout, cancellation, and shutdown races.
- [ ] Test oversized payloads, slow clients, saturation, and resource cleanup.
- [ ] Compare tool catalogs and representative tool results across stdio and HTTP.
- [ ] Verify allowed-directory and execution policies on both transports, including shared process roots across simultaneous HTTP sessions.
- [ ] Run direct HTTP end-to-end tests and OpenAI tunnel compatibility tests where applicable.

## Completion gate

All 23 retained tools operate through native Streamable HTTP and stdio with equivalent schemas and policy boundaries, and the R12 security suite passes.

---

# R14 — Hardening and 2.0.0 release

## Goal

Finish platform, container, CI, documentation, packaging, migration, and release verification for the first public 2.x release.

## Known cleanup items

- `Dockerfile` currently uses Go 1.24 while `go.mod` requires Go 1.26.5.
- normal Build/Test workflows ignore Markdown-only changes even though documentation/catalog consistency is tested.
- the Build workflow does not build all six release targets.
- `.goreleaser.yml` contains a stale TODO for native Registry checksum support; the project already uses a separate verified Registry workflow.
- the Claude Code plugin is optional and is not part of the active OpenAI tunnel deployment.

## Checklist

- [ ] Align Docker builder Go version with `go.mod`.
- [ ] Pin the final runtime image and dependencies.
- [ ] Run the container as a non-root user.
- [ ] Define mounts, temporary storage, allowed roots, healthcheck, and shutdown behavior.
- [ ] Cover all six Windows/Linux/macOS amd64/arm64 targets in CI.
- [ ] Ensure documentation and catalog changes trigger consistency tests.
- [ ] Add HTTP tests to supported CI platforms.
- [ ] Run race detector, vet, Staticcheck, govulncheck, actionlint, ShellCheck, and Gitleaks.
- [ ] Add fuzzing for detection, decoding, line framing, HTTP parsing, and JSON inputs.
- [ ] Run sustained load, memory, cancellation, and session cleanup tests.
- [ ] Runtime-execute representative Windows, Linux, and macOS builds where infrastructure permits.
- [ ] Remove or update the stale GoReleaser Registry TODO.
- [ ] Update README, TOOLS, catalog, Smithery, tunnel examples, container docs, and publishing notes.
- [ ] Finish the 1.8-to-2.0 migration guide.
- [ ] Decide whether to retain the Claude Code plugin.
- [ ] If retained, verify plugin download, checksum, cache, platform mapping, and update behavior against 2.0.0 assets.
- [ ] Set plugin and marketplace metadata to `2.0.0` only at release preparation.
- [ ] Run the complete release checklist in [PUBLISHING.md](PUBLISHING.md).
- [ ] Create and push `v2.0.0` only after all prior gates pass.
- [ ] Verify release binaries, archives, checksums, signatures where available, and MCP Registry publication.
- [ ] Deploy the 2.0.0 runtime, execute smoke and rollback tests, and record the final handoff.

## Completion gate

`v2.0.0` is reproducible, verified across supported targets, includes secure native MCP Streamable HTTP and stdio, has complete migration documentation, and passes deployment plus rollback verification.
