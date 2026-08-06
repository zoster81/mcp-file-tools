# MCP 2026-07-28 Adoption Design

## Status

**R20 phase-1 design baseline. No dependency, protocol, transport, configuration, tool schema, or runtime behavior has changed.**

This document defines the compatibility boundary, transport architecture, security invariants, implementation phases, and verification gate for adopting Model Context Protocol version `2026-07-28` while retaining the existing `2025-11-25` behavior.

The current source baseline uses `github.com/modelcontextprotocol/go-sdk v1.6.1`. The first Go SDK line that implements `2026-07-28` is currently pre-release. R20 must not place a pre-release SDK in the main runtime dependency graph. Implementation begins only after an official stable SDK release supports the final specification and its release notes, module checksums, API surface, and compatibility behavior have been reviewed.

Authoritative external references:

- [MCP 2026-07-28 specification announcement](https://blog.modelcontextprotocol.io/posts/2026-07-28/)
- [MCP Go SDK releases](https://github.com/modelcontextprotocol/go-sdk/releases)
- [MCP Go SDK protocol guide](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/protocol.md)
- [SEP-2577 roots, sampling, and logging deprecation](https://modelcontextprotocol.io/seps/2577-deprecate-roots-sampling-and-logging)

## Current baseline

The existing implementation intentionally provides two stable transport profiles:

- stdio through one SDK transport and the shared `filetoolsserver.BuildServer` server;
- authenticated stateful Streamable HTTP through a hardened outer handler and the SDK stateful handler.

The HTTP profile currently depends on protocol-level sessions:

- initialization creates an `Mcp-Session-Id`;
- an application session gate reserves capacity before initialization;
- later `POST`, `GET`, and `DELETE` requests reauthenticate and address that session;
- session expiry, shutdown, SSE behavior, request admission, and cancellation have explicit tests;
- `Last-Event-ID` replay remains disabled because no event store is configured.

The stdio profile may use legacy MCP client roots only when no startup allowed directories were supplied. Roots are informational compatibility input, not an access-control boundary. Process-wide normalized allowed directories remain authoritative.

## Protocol changes relevant to this server

MCP `2026-07-28` changes the transport assumptions that affect this repository:

- protocol-level initialization and sessions are removed;
- each request carries protocol, client, and capability metadata;
- `server/discover` provides optional capability discovery;
- Streamable HTTP requests use standardized `Mcp-Method` and `Mcp-Name` headers;
- server-initiated calls are replaced by Multi Round-Trip Requests;
- list responses may include cache hints;
- roots, sampling, and logging are deprecated;
- legacy protocol versions remain relevant during the compatibility window.

This server does not currently initiate sampling or elicitation calls, expose resources, or depend on protocol logging. The main migration risks are therefore HTTP session coexistence, stdio roots compatibility, version routing, cancellation, header/body consistency, and downgrade behavior.

## Goals

R20 must:

- support the final `2026-07-28` protocol through an official stable Go SDK;
- preserve supported legacy versions and the published `2025-11-25` stateful behavior;
- preserve the same tool catalog, tool schemas, prompts, limits, allowed-root policy, error categories, and execution gates;
- keep stdio and Streamable HTTP behavior equivalent at the tool layer;
- add stateless HTTP without weakening authentication, Host, Origin, proxy, rate, concurrency, body, timeout, logging, or shutdown controls;
- prevent the new protocol from depending on deprecated client roots;
- retain deterministic downgrade and rejection behavior for old, new, malformed, missing, and unsupported protocol-version inputs;
- avoid pre-release dependencies and avoid adopting unrelated protocol extensions.

## Non-goals

R20 does not add:

- MCP Apps;
- Tasks;
- application-managed OAuth or an authorization server;
- Enterprise Managed Authorization;
- Multi Round-Trip Requests for tool confirmations or missing input;
- list-result cache hints;
- distributed tracing export;
- durable subscriptions or event replay;
- server-side protocol session state for `2026-07-28`;
- per-client roots or per-client filesystem ACLs;
- new MCP tools, prompts, resources, or public file-operation schemas;
- removal of legacy roots support before the compatibility policy permits it.

Each of these requires separate evidence and design if later needed.

## Stable SDK adoption gate

No implementation phase may update `go.mod` until all of the following are true:

1. the official Go SDK publishes a stable release that explicitly supports final protocol version `2026-07-28`;
2. the release is not marked pre-release and does not require a pseudo-version;
3. release notes and protocol documentation identify supported legacy versions and stateless HTTP behavior;
4. the module checksum is fetched through the existing Go module trust path;
5. the public APIs needed for version filtering, stateless HTTP, discovery, request cancellation, and legacy fallback are available without an SDK fork;
6. known security advisories and compatibility flags for the candidate SDK are reviewed;
7. a temporary qualification run passes outside the committed dependency graph before `go.mod` changes.

If the stable SDK cannot satisfy the design through public APIs, R20 stops for a new design review. It must not vendor or fork the SDK opportunistically.

## Transport architecture

### Shared server

Both protocol generations use the same `*mcp.Server` built by `filetoolsserver.BuildServer`. Tool registration, middleware, roots policy, backup-store authority, limits, execution policy, and lifecycle context remain shared.

Protocol selection must never create a second tool catalog or handler implementation.

### Stdio

Stdio remains the default transport.

When startup allowed directories are configured:

- the server may advertise `2026-07-28` and supported legacy versions;
- no request depends on client roots;
- tool behavior is identical across negotiated protocol versions.

When no startup allowed directories are configured:

- the server must not negotiate `2026-07-28` while filesystem authority would depend on deprecated client roots;
- protocol negotiation is capped to the highest legacy version that supports the existing roots compatibility path;
- legacy initialization and roots notifications continue to populate the process-wide root set according to the existing rules;
- startup does not silently broaden access and does not substitute the current working directory.

Phase 2 must confirm that the stable SDK exposes a supported way to filter advertised protocol versions by server configuration or transport. If it does not, the stdio adoption design must be revised before implementation.

### Streamable HTTP

The existing endpoint path remains unchanged. R20 adds two internal SDK handlers behind the same hardened outer middleware:

- **legacy stateful handler:** preserves the current `2025-11-25` and older behavior, session gate, authenticated `POST`/`GET`/`DELETE`, session timeout, and SSE semantics;
- **new stateless handler:** accepts `2026-07-28` requests, creates no protocol session, emits no `Mcp-Session-Id`, and uses no session gate or event store.

The outer handler selects the protocol path only from the normalized `MCP-Protocol-Version` header:

- exact `2026-07-28` routes to the stateless handler;
- supported legacy versions and legacy initialization without a version header route to the stateful handler;
- malformed, repeated, contradictory, or unsupported version headers fail before SDK dispatch;
- `Mcp-Method` and `Mcp-Name` are never trusted for authorization or filesystem policy;
- header/body method-name consistency remains an SDK protocol-validation responsibility and must be covered by negative tests.

No middleware may pre-read or duplicate the JSON body merely to choose the handler. Body limits and aggregate reservations continue to wrap the single body stream before SDK decoding.

### Stateless HTTP method policy

For `2026-07-28`:

- authenticated `POST` is the MCP request method;
- protocol-level `GET` and `DELETE` are rejected because there is no session stream or session termination operation;
- `Last-Event-ID` remains rejected;
- any received `Mcp-Session-Id` is rejected rather than ignored;
- no session capacity is reserved;
- the normal non-SSE concurrency semaphore applies;
- client disconnect cancellation propagates to the tool handler through the stable SDK option intended for stateless requests;
- request timeout, body limits, rate limiting, authentication, Host, Origin, proxy, logging, and shutdown behavior remain unchanged.

Health and readiness routes are protocol-independent.

## Request admission order

The common outer pipeline remains security-significant:

1. identify peer and trusted-proxy status;
2. enforce proxy-only plaintext boundaries;
3. validate Host;
4. validate Origin for every method;
5. apply peer rate limiting;
6. serve health or readiness where applicable;
7. reject shutdown or non-ready work;
8. validate path and empty query;
9. authenticate the bearer token;
10. validate transport method and protocol-version header shape;
11. apply concurrency and body-budget admission;
12. route to the stateless or stateful SDK handler;
13. update legacy session admission only for the stateful path;
14. emit one category-only redacted access log.

Protocol generation does not alter bearer authority, process roots, or execution policy.

## Header handling

The new standard headers are untrusted network data.

- Reject multiple values for singleton MCP headers.
- Bound header bytes through the existing HTTP server limit.
- Require the exact final protocol version string for stateless routing.
- Do not accept version aliases, whitespace variants, date normalization, or prefix matching.
- Do not use `Mcp-Method` or `Mcp-Name` to bypass JSON-RPC decoding, tool lookup, authentication, rate limiting, or authorization.
- Do not log arbitrary header values.
- Negative tests must cover absent, duplicate, malformed, unsupported, conflicting, and body-mismatched headers.

Reverse proxies may route on standardized headers, but the application continues to validate the complete request independently.

## Discovery and capability projection

`server/discover` is provided by the stable SDK, not hand-written locally.

The advertised capability projection must:

- expose only protocol versions actually accepted by the selected transport/configuration;
- preserve the same server identity, instructions, tools, and prompts as legacy initialization;
- omit capabilities the server does not implement;
- avoid advertising deprecated roots support on `2026-07-28`;
- preserve legacy roots capability when the stdio compatibility path requires it;
- remain deterministic for the same binary and startup configuration.

No client-supplied discovery metadata changes process roots or tool registration.

## Tool-list caching

R20 does not initially emit `ttlMs` or `cacheScope` hints.

Although this server's catalog is static after startup, caching semantics affect dynamic roots, prompt availability, execution policy, and future protocol behavior. Cache hints may be designed later after client interoperability evidence is available. Absence of cache hints preserves conservative existing behavior.

## Multi Round-Trip Requests

The current tool handlers complete within one request and do not make server-to-client requests. R20 therefore advertises no MRTR-dependent capability and introduces no `input_required` flow.

Approval-bound edit, patch-package, restore, and garbage-collection operations retain their existing explicit preview/capability/apply contracts. They are not migrated to MRTR implicitly.

## Roots deprecation strategy

Roots remain a legacy stdio compatibility feature only.

- Startup roots remain the preferred and authoritative configuration for every protocol version.
- HTTP never accepts client roots.
- `2026-07-28` never relies on roots.
- Legacy stdio roots remain supported while the protocol compatibility window requires them.
- No warning is written to stdout; any deprecation notice belongs only in bounded stderr developer logging or documentation.
- Removing roots support requires a later milestone with migration evidence and explicit compatibility impact.

## Error and downgrade behavior

- A `2026-07-28` request sent to a binary that has not completed R20 receives a deterministic unsupported-version error from the existing stack.
- After R20, exact new-version requests use stateless semantics.
- Legacy clients continue to initialize and use stateful sessions unchanged.
- A new SDK client may discover and negotiate the highest mutually supported version.
- If stateless discovery fails, client-side fallback behavior is owned by the client SDK; the server does not emulate legacy initialization inside the stateless path.
- Unsupported versions fail rather than silently downgrade an individual request.
- The selected protocol generation cannot change inside one legacy session.
- JSON-RPC errors remain path-free and stack-free.

## Configuration impact

R20 should not introduce a protocol-mode environment variable unless stable-SDK qualification proves same-endpoint dual routing impossible or unsafe.

The preferred outcome is automatic backward-compatible routing:

- existing operators retain stateful clients without changing configuration;
- new clients can use stateless `2026-07-28` on the same authenticated endpoint;
- session limits apply only to legacy sessions;
- all other resource limits remain shared.

Any required new setting must have a secure default, hard bounds where applicable, startup validation, documentation, and configuration tests before implementation.

## Test strategy

### Dependency qualification

- stable SDK tag and module version only;
- `go mod tidy -diff` and `go mod verify`;
- inspect release compatibility flags and public API changes;
- no unexpected new direct dependencies;
- vulnerability and license review;
- clean downgrade back to the previous module files during the qualification spike.

### Stdio compatibility

- startup roots with a new client negotiate `2026-07-28`;
- startup roots with legacy clients negotiate supported legacy versions;
- no startup roots cap negotiation to the legacy roots-compatible protocol;
- roots notifications remain stdio-only and legacy-only;
- all 27 tools and prompts remain identical;
- representative read, write, preview/apply, backup, error, cancellation, and output-limit behavior remains equivalent.

### HTTP protocol routing

- exact new-version header reaches the stateless handler;
- legacy initialize without a version header reaches the stateful handler;
- supported legacy version headers reach the stateful handler;
- malformed, duplicate, empty, and unsupported version headers fail deterministically;
- `Mcp-Method` and `Mcp-Name` mismatch the body and are rejected;
- routing never requires reading the body twice;
- stateless requests emit no session header and consume no session slot;
- stateful session admission, expiry, GET, DELETE, and shutdown remain unchanged;
- stateless GET, DELETE, `Last-Event-ID`, and session headers are rejected;
- concurrent new and legacy clients remain isolated at protocol state while sharing process policy.

### Security and limits

Repeat the complete HTTP negative matrix for both protocol generations:

- authentication;
- Host and Origin;
- trusted and untrusted proxies;
- body and aggregate-body limits;
- request concurrency and peer rate limits;
- timeout and cancellation;
- logging redaction;
- execution dual opt-in;
- shutdown and readiness;
- malformed JSON and content types;
- no CORS headers;
- no path, token, body, complete session identifier, or header-value leakage.

### Interoperability and conformance

- official Go SDK client against stdio and HTTP;
- at least one independent current MCP client when available;
- official protocol conformance tests supported by the stable SDK;
- stateful legacy and stateless new requests in the same process;
- native Windows external smoke;
- Linux container direct-TLS HTTP smoke;
- six supported OS/architecture command and test compilation targets.

## Devil's advocate findings

### Risk: pre-release SDK behavior becomes production authority

A pre-release dependency could change protocol semantics, public APIs, or security defaults before stabilization. R20 blocks all runtime implementation until an official stable SDK release exists and passes a temporary qualification gate.

### Risk: same-endpoint dual routing creates an authentication or body-limit bypass

Two SDK handlers behind separate middleware stacks could diverge. R20 requires one common outer security/admission pipeline and routing only after authentication and bounded request admission. Protocol routing cannot bypass shared controls or consume the body twice.

### Risk: stateless requests leak into the legacy session gate

Reserving legacy session capacity for stateless traffic would create artificial denial of service and incorrect cleanup. The stateless path never creates, acquires, or releases an application session record; tests assert session counts remain unchanged.

### Risk: deprecated roots remain a hidden authority in the new protocol

Advertising the new protocol while depending on client roots could start the server with an empty or unsafe authority model. R20 caps stdio negotiation to a legacy version when startup roots are absent and never enables roots over HTTP.

### Risk: protocol headers become trusted authorization metadata

Gateways can route on `Mcp-Method` and `Mcp-Name`, but clients control them. The application continues to authenticate first and relies on SDK header/body consistency checks; filesystem and execution authorization remain based on decoded tool calls and process policy.

### Risk: enabling the new version silently changes published HTTP behavior

Legacy stateful sessions remain accepted on the same endpoint. New stateless behavior is selected only by the exact new protocol version. The completion gate includes direct regression evidence for the existing `2025-11-25` profile.

## Implementation phases

1. **Design and readiness — active.** Approve this contract, inventory current protocol coupling, and document the stable-SDK adoption gate.
2. **Stable SDK qualification.** Evaluate the first stable Go SDK release supporting final `2026-07-28` in a temporary reversible module change; confirm public APIs and full backward compatibility before committing dependency changes.
3. **Stdio version gating.** Add discovery/version support when startup roots are present and retain legacy roots negotiation when they are absent.
4. **Dual-generation HTTP.** Add shared version validation and separate stateful/stateless SDK handlers behind the existing hardened middleware, with request cancellation and no stateless session admission.
5. **Compatibility and conformance.** Complete official conformance, old/new client interoperability, security failure injection, race, fuzz, six-target, native, and container checks.
6. **Documentation and completion.** Update protocol/security references, migration guidance, publishing notes, and the completion record without changing the published runtime unless separately authorized.

## Completion gate

R20 is complete only when an official stable Go SDK supports final protocol `2026-07-28`; stdio and HTTP accept the new version without relying on deprecated roots or protocol sessions; supported legacy clients retain their current behavior; stateless and stateful HTTP share every security and resource-control boundary; tool catalogs and results remain equivalent; malformed and downgrade cases fail deterministically; and the complete focused, regression, race, static, vulnerability, conformance, six-target, native, container, documentation, and security verification matrix passes.

No push, tag, release, deployment, launcher change, or runtime restart is part of the milestone without separate explicit authorization.
