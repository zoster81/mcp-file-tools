# Project Direction and Upstream Relationship

`zoster81/mcp-file-tools` is an independently versioned downstream fork of [`dimitar-grigorov/mcp-file-tools`](https://github.com/dimitar-grigorov/mcp-file-tools). It preserves the original project's encoding-aware text-file purpose and GPL-3.0 lineage, while maintaining its own module path, MCP Registry identity, release pipeline, public API decisions, transport architecture, and deployment documentation.

This repository is not a compatibility branch intended for routine merging with upstream. Both projects may develop useful ideas independently, but changes must be reviewed and implemented against each project's current architecture rather than copied or synchronized mechanically.

## Product scope

The fork is a secure, encoding-aware MCP filesystem service for local, tunneled, containerized, and explicitly secured network deployments. Its supported scope includes:

- the same authoritative 27-tool unreleased source catalog over stdio and native stateful Streamable HTTP;
- process-wide allowed-directory policy with symlink, junction, reparse-point, and missing-ancestor validation;
- bounded-memory decoding, reading, grep, conversion, line-ending, and BOM operations across 24 registered encodings;
- durable staged mutations with practical concurrent-change detection, no-replace creation, backup rollback, and platform-specific synchronization;
- an optional dedicated persistent backup store with immutable content-addressed objects, checksummed manifests, bounded management/audit, approval-bound pre-state capture for prepared edits and patch packages, one-shot original-target restore, explicit generation-bound garbage collection, and a separate mutation-free offline diagnostic command for existing stores;
- transport-independent error categories and tool metadata;
- a controlled MCP `2026-07-28` adoption path that preserves legacy stdio and stateful HTTP until a stable official Go SDK can provide stateless compatibility without weakening the shared security boundary;
- optional `run_script` and unrestricted `shell` tools, both disabled by default, with an additional execution gate for HTTP;
- reproducible multi-platform releases, checksum-driven MCP Registry publication, and a non-root transport-neutral container.

Binary or media interpretation remains outside the server's scope. Per-agent filesystem ACLs are also outside the current model: every connection to one process shares its startup roots and policy. Deploy separate processes when technical isolation is required.

## Supported transports

| Transport | Intended use | Authentication boundary | Directory policy |
|---|---|---|---|
| stdio | Client-managed local processes, desktop/CLI MCP clients, and secure tunnel bridges | Operating-system process and client configuration | Startup directories are authoritative; dynamic client roots are accepted only when no directories were configured at startup |
| stateful Streamable HTTP | Persistent local services, containers, trusted reverse proxies, and explicitly secured remote deployments | Bearer token on every MCP request; loopback by default; TLS or a trusted proxy boundary for non-loopback listeners | Startup directories are immutable and shared by every HTTP session; HTTP client roots are disabled |

Both transports construct the server through the same `BuildServer` path and expose the same tools, limits, execution policy, encoding behavior, and typed errors. The HTTP-specific threat model and deployment rules are defined in [HTTP_SECURITY.md](HTTP_SECURITY.md).

## Relationship to upstream

Upstream remains the source of the original encoding-aware file-tool implementation and continues to evolve as its own product. This fork retains attribution and tracks upstream developments for ideas, bug reports, and security lessons, but it does not promise source-level or schema compatibility with later upstream releases.

Potential upstream suggestions should be concept-level and narrowly scoped to the original project's product boundaries. Features that depend on this fork's native HTTP service, execution tools, process-wide multi-transport policy, fork-owned Registry identity, or deployment infrastructure should remain fork-specific unless upstream independently chooses those directions.

Conversely, upstream agent-experience improvements may be considered here only after they are adapted to this fork's bounded-memory pipeline, durable mutation layer, stable public schemas, transport equivalence requirements, and security model.

## Reciprocal feature exchange

R15 explicitly credits the original project as the source for optional read line numbers, richer grep modes and paging, `.gitignore`-aware traversal, bounded result sorting, batch encoding dry runs, encoding workflow prompts, unified-patch editing, and opt-in fuzzy matching. Both the user-facing concepts and the original implementation approaches informed this fork's evaluation of behavior, edge cases, and trade-offs. The resulting code was reworked specifically for the fork's secure walker, bounded-memory streaming, durable mutation layer, stable 23-tool catalog, process-wide roots, and stdio/Streamable HTTP equivalence rather than mechanically synchronized.

This attribution is intended as reciprocal engineering exchange rather than one-way ownership. Improvements developed in either project may inspire the other, and useful functionality, implementation techniques, tests, and security findings may flow in either direction through concept-level discussion or normal GPL-3.0-compatible contributions. Neither repository is expected to accept the other's code unchanged, and shared work does not erase their separate APIs, release histories, security models, or maintenance decisions.

## Maintenance policy

- Release and API decisions are made for this fork's users rather than to minimize upstream merge conflicts.
- Upstream changes are reviewed selectively; no automatic merge or rebasing policy exists.
- Public documentation must distinguish shared lineage from current fork behavior.
- Cross-project proposals should describe the user problem and desired behavior, credit the project where the idea was observed, and not assume that either repository can accept the other's implementation unchanged.
