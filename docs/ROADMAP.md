# Development Roadmap

This is the authoritative product roadmap for `zoster81/mcp-file-tools`.

Version `2.0.0` is published and deployed through both supported transports. The live stdio connector and an authenticated stateful Streamable HTTP session have each verified the complete 23-tool catalog from the published binary. R14 is complete after the controlled active rollback and final restoration of the published runtime.

Product identity and the fork's independent relationship to upstream are defined in [PROJECT_DIRECTION.md](PROJECT_DIRECTION.md). Current milestone status and completion gates live in this document. Reusable engineering checks live in [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md), contributor workflow in [`CONTRIBUTING.md`](../CONTRIBUTING.md), scoped agent guidance in [`AGENTS.md`](../AGENTS.md), and completed R1-R6 engineering outcomes in [ROADMAP_HISTORY.md](ROADMAP_HISTORY.md).

## Operating rules

- Only one milestone may be `ACTIVE` at a time.
- Complete milestones in order unless maintainers explicitly reprioritize them.
- Keep changes atomic and limited to the active milestone.
- Use content bytes and structural evidence for encoding detection. File extensions must not select or bias an encoding.
- Treat domain-specific files, including MQL sources, as ordinary test fixtures rather than special encoding profiles.
- Preserve stdio support while adding Streamable HTTP.
- Keep `run_script` and `shell` disabled by default on every transport.
- Build internal commit binaries as needed; create later public releases only after their dated changelog, verification, asset, Registry, and deployment gates pass.
- Every milestone must pass [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md) before it is marked complete.

## Milestone overview

| Milestone | Status | Outcome |
|---|---|---|
| R7 | COMPLETE | Replaced the historical roadmap with a clear operating plan and removed domain-specific MQL emphasis. |
| R8 | COMPLETE | Generic, conservative, extension-independent encoding detection, including BOMless UTF-16. |
| R9 | COMPLETE | Real bounded-memory streaming for large-file read, grep, conversion, line-ending, and BOM paths. |
| R10 | COMPLETE | Resolve public API inconsistencies and compatibility debt before the 2.0 boundary. |
| R11 | COMPLETE | Separate transport bootstrap from the shared MCP server and tool policies. |
| R12 | COMPLETE | Approve the Streamable HTTP threat model and security design. |
| R13 | COMPLETE | Implement and verify native MCP Streamable HTTP while preserving stdio. |
| R14 | COMPLETE | Completed hardening, publication, dual-transport deployment, active rollback, restoration, and final handoff for 2.0.0. |
| R15 | PLANNED | Improve agent ergonomics and project-aware workflows without weakening transport, memory, mutation, or security guarantees. |

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

Completed on 2026-07-27. Configuration loading, CLI parsing, server construction, and transport startup are separate. `BuildServer` owns one shared 23-tool registration and process-wide policy; the stdio runner is lifecycle-aware and responds to process cancellation; explicit `--transport=stdio` and `MCP_TRANSPORT=stdio` preserved stdio as the sole implemented transport at the R11 boundary. Multiple connections to one server were verified to expose equivalent tool catalogs and the same configured roots, while provided configuration controls handler behavior. Dynamic client roots are limited to roots-capable stdio clients started without configured directories, and empty updates remove stale dynamic access. The complete race-detector suite and Gitleaks scans of Git history plus the working tree passed. R12 and R13 subsequently added the reviewed security design and native HTTP implementation on top of this architecture.

---

# R12 — Streamable HTTP security design

## Goal

Approve a concrete threat model and secure defaults before exposing filesystem and optional execution tools over an HTTP listener.

The approved design is [`HTTP_SECURITY.md`](HTTP_SECURITY.md). It is the source of truth for R13 HTTP configuration, trust boundaries, middleware order, accepted risks, security tests, and release blockers.

## Checklist

- [x] Document assets, trust boundaries, actors, and supported deployment models.
- [x] Preserve the R11 process-wide root model: all HTTP sessions share startup roots, client roots cannot mutate them, and per-agent isolation uses separate processes when required.
- [x] Bind to loopback by default.
- [x] Require explicit configuration for non-loopback binding.
- [x] Define token authentication and reverse-proxy integration.
- [x] Use constant-time credential comparison where applicable.
- [x] Define secure token loading without command-line secret exposure.
- [x] Validate `Host` and `Origin` and address DNS rebinding.
- [x] Disable CORS by default.
- [x] Define TLS expectations for direct and reverse-proxy deployments.
- [x] Set request body, header, connection, and concurrency limits.
- [x] Set read-header, request, idle, session, and shutdown timeouts without breaking SSE.
- [x] Define session identifiers, expiry, cleanup, and hijacking protections.
- [x] Define CSRF and browser-origin protections.
- [x] Prevent sensitive headers, tokens, file contents, commands, and session identifiers from entering logs.
- [x] Keep `run_script` and `shell` disabled by default.
- [x] Require a distinct explicit opt-in for execution over HTTP.
- [x] Define rate limiting and denial-of-service behavior.
- [x] Define trusted-proxy handling without accepting spoofed forwarding headers.
- [x] Define health and readiness endpoints that expose no sensitive data.
- [x] Define security tests and release-blocking findings.

## Completion gate

A reviewed security design exists, every identified threat has a mitigation or explicit accepted risk, and implementation tests are specified before HTTP code is merged.

## Completion record

Completed on 2026-07-27. [`HTTP_SECURITY.md`](HTTP_SECURITY.md) defines a fail-closed stateful Streamable HTTP profile with loopback binding, mandatory bearer authentication on every MCP request, exact Host and all-method Origin validation, no CORS, explicit non-loopback/TLS/proxy rules, bounded bodies, headers, requests, sessions, rate state, and timeouts, redacted logging, minimal health endpoints, and a second execution opt-in. It preserves R11 process-wide roots, disables HTTP client roots, keeps the initial event store unset, records SDK integration constraints, and lists required negative tests plus release-blocking findings before R13 implementation.

---

# R13 — Native MCP Streamable HTTP

## Goal

Implement the MCP Streamable HTTP transport according to R11 and R12 while preserving stdio behavior.

R13 must implement [`HTTP_SECURITY.md`](HTTP_SECURITY.md) without broadening its accepted risks or adding an unauthenticated compatibility mode.

## Checklist

- [x] Implement the Streamable HTTP endpoint using the shared server and the approved security middleware order.
- [x] Support required JSON-RPC request and streaming response behavior.
- [x] Implement session creation, lookup, expiration, and cleanup.
- [x] Propagate disconnect and request cancellation into tool contexts.
- [x] Implement graceful shutdown for listeners and active sessions.
- [x] Add health and readiness endpoints.
- [x] Apply authentication, Host/Origin, per-request and aggregate size, timeout, rate, session, and concurrency policies from R12.
- [x] Reject malformed content types, methods, JSON, protocol messages, query credentials, and event replay deterministically.
- [x] Keep stdio startup and protocol output unchanged.
- [x] Test simultaneous clients and concurrent sessions.
- [x] Test disconnect, timeout, cancellation, POST-paused idle expiry, SSE-only expiry, cleanup, and shutdown races.
- [x] Test oversized known-length/chunked payloads, oversized headers, saturation, bounded limiter state, and resource cleanup.
- [x] Compare complete tool metadata and representative tool results across stdio/direct and HTTP adapters.
- [x] Verify allowed-directory and dual execution policies on both transports, including shared process roots across simultaneous HTTP sessions.
- [x] Run native HTTP end-to-end tests and the existing stdio manual harness; the OpenAI Secure MCP Tunnel deployment remains stdio and was not changed or restarted.

## Completion gate

All 23 retained tools operate through native Streamable HTTP and stdio with equivalent schemas and policy boundaries, and the R12 security suite passes.

## Completion record

Completed on 2026-07-27. The executable now selects `stdio` or stateful `streamable-http` while constructing the 23 tools once through `BuildServer`. Native HTTP is loopback-bound and bearer-authenticated by default, validates exact Host and all-method Origin values, emits no CORS allow headers, disables HTTP client roots and event replay, supports optional TLS and bounded trusted-proxy handling, exposes minimal health/readiness routes, and coordinates graceful shutdown. Per-request and aggregate body budgets, non-SSE concurrency, live sessions, bounded per-peer rate state, idle cleanup, header limits, and cancellation are enforced before or around the pinned SDK handler. HTTP execution requires its own opt-in in addition to the existing tool authorization, and token-source variables are removed from the process environment after startup snapshotting.

Tests verified multiple simultaneous HTTP clients, unique sessions, DELETE and idle cleanup, SDK-aligned POST-paused and SSE-only expiry, cancellation propagation, authentication on every method, Host/Origin and proxy rejection, known-length and chunked `413` behavior, aggregate/concurrency `429` behavior, oversized-header `431`, log and TLS-path redaction, immutable process roots after an HTTP roots notification, and equivalence with the direct adapter for all tool metadata, CP1251 reads, allowed directories, and representative typed errors. Focused and complete Go tests, `go vet`, Staticcheck, govulncheck, the full race detector, the stdio manual harness, Node release tests, documentation/catalog checks, and Gitleaks history plus working-tree scans passed. No binary build, push, launcher change, service restart, or live deployment was performed.

---

# R14 — Hardening and 2.0.0 release

## Goal

Finish platform, container, CI, documentation, packaging, migration, and release verification for the first public 2.x release.

## Final deployment verification

The controlled active rollback and restoration were completed on 2026-08-03.

## Checklist

- [x] Align Docker builder Go version with `go.mod`.
- [x] Pin the container bases and CI/release action versions.
- [x] Run the container as a non-root user.
- [x] Define mounts, temporary storage, allowed roots, healthcheck, and shutdown behavior.
- [x] Cover all six Windows/Linux/macOS amd64/arm64 targets in CI.
- [x] Ensure documentation and catalog changes trigger consistency tests.
- [x] Add HTTP tests to supported CI platforms.
- [x] Run race detector, vet, Staticcheck, govulncheck, actionlint, ShellCheck, and Gitleaks on the R14 working tree.
- [x] Add bounded fuzzing for detection, decoder chunking, line framing, HTTP parsing, proxy chains, and JSON-RPC inputs.
- [x] Run deterministic load, resource-accounting, cancellation, and session-cleanup soak tests, including 102,400 admitted/rejected requests and repeated race-detector cycles.
- [x] Build and runtime-smoke the Linux/amd64 container locally with UID/GID 10001, a read-only root filesystem, dropped capabilities, `no-new-privileges`, bounded temporary storage, SDK-driven stdio MCP, direct-TLS HTTP, negative security responses, health/readiness, and graceful shutdown.
- [x] Pass the native binary MCP smoke gate on Windows, Linux, and macOS GitHub runners.
- [x] Pass the Ubuntu container gate for non-root execution, hardened stdio, direct-TLS HTTP, security responses, health/readiness, and graceful shutdown.
- [x] Remove the stale GoReleaser Registry TODO and keep Registry publication in the checksum-verified workflow.
- [x] Normalize archive entry metadata to commit-derived values and verify that two independent GoReleaser snapshots produce identical checksums for all 6 raw binaries and 6 platform archives.
- [x] Generate internal prerelease manifests from both the direct six-target build and the reproducible GoReleaser snapshot checksums, then pass `mcp-publisher 1.7.9 validate` without login or publication.
- [x] Verify the known-good R10 rollback binary offline: exact hash/version, 23-tool stdio startup, and byte-identical two-reference launcher reversal/restore.
- [x] Update README, TOOLS, catalog, tunnel/HTTP examples, Smithery metadata, container docs, and publishing notes.
- [x] Finish the 1.8-to-2.0 migration guide.
- [x] Remove the optional Claude Code downloader plugin and marketplace metadata rather than carry a second network installer and cache trust boundary into 2.0.
- [x] Run the complete release checklist in [PUBLISHING.md](PUBLISHING.md).
- [x] Create and push `v2.0.0` only after all prior gates pass.
- [x] Verify release binaries, archives, checksums, signatures where available, and MCP Registry publication.
- [x] Deploy the published 2.0.0 runtime and execute live stdio plus authenticated Streamable HTTP smoke tests.
- [x] Execute the controlled active rollback, restore 2.0.0, and record the final handoff.

## Publication record

Published on 2026-08-02. Fork tag `v2.0.0` resolves to commit `1530fbb1eab529a1ef7236b4b3df8ab84a9a0d1d`. The tag workflow passed the complete Linux, Windows, and macOS test matrix, produced six raw binaries, six deterministic platform archives, and `checksums.txt`, and published `io.github.zoster81/mcp-file-tools` version `2.0.0` to the MCP Registry through GitHub OIDC. All 12 published binary/archive checksums were independently verified against the release checksum file. No separate signature assets were emitted by the configured release pipeline.

Operator deployment completed on 2026-08-02. The published Windows amd64 `2.0.0` binary now runs through both the stdio tunnel path and the native loopback Streamable HTTP path. Live verification confirmed the embedded version, HTTP health/readiness, unauthenticated `401`, authenticated session initialization, and the complete 23-tool catalog. On 2026-08-03, a controlled active rollback to the retained R10 build verified the complete 23-tool stdio catalog while the later HTTP transport was intentionally absent. The published `2.0.0` runtime was then restored and reverified over stdio and authenticated Streamable HTTP, including the complete tool catalog and expected health, readiness, and authentication responses.

## Completion gate

`v2.0.0` is reproducible, verified across supported targets, includes secure native MCP Streamable HTTP and stdio, has complete migration documentation, and passes deployment plus rollback verification.

---

# R15 — Agent ergonomics and project-aware workflows

## Status

Planned. R14 is complete; R15 has not started.

## Goal

Reduce unnecessary tool calls and token usage for common repository and encoding workflows while preserving the fork's bounded-memory pipeline, durable mutation semantics, stable public errors, process-wide root model, and stdio/HTTP equivalence.

## Candidate evaluation backlog

These are design candidates, not accepted API commitments:

- optional absolute line numbers in paged `read_text_file` results;
- grep output modes for full content, matching file paths, and per-file counts;
- grep result paging and plural include/exclude patterns under existing aggregate budgets;
- `.gitignore`-aware traversal with explicit opt-out and secure nested-pattern handling;
- bounded sorting for directory/search results by name, modification time, or size;
- batch encoding-conversion dry runs with machine-readable unsupported-character locations;
- transport-independent MCP prompts for encoding audits, mojibake diagnosis, and controlled UTF-8 migration;
- unified-diff edit input after strict single-file parsing, bounded hunk processing, and ambiguity tests;
- fuzzy edit matching only if a deterministic complexity bound, explicit threshold, unique-match policy, and safe failure behavior are established.

## Design constraints

- Do not copy or mechanically synchronize another repository's implementation; design against this fork's current APIs and invariants.
- Preserve the existing 23-tool contract unless a deliberate versioned API decision justifies a change.
- Keep all read/search additions bounded by `MCP_MAX_*` limits and deterministic ordering.
- Route mutations through the shared encoding-aware document and durable filesystem layers.
- Keep prompts and tool metadata identical over stdio and Streamable HTTP.
- Do not weaken HTTP authentication, Host/Origin validation, session limits, logging redaction, or dual execution authorization.
- Do not add per-session filesystem ACLs or let HTTP clients mutate process roots.

## Completion gate

Accepted R15 features demonstrate measurable call/token reduction, pass normal and adversarial tests on supported platforms, remain bounded under large inputs, preserve exact mutation guarantees, and expose equivalent schemas and behavior through both transports.
