# Fork publishing notes

This document describes release and distribution work for the
`zoster81/mcp-file-tools` fork. The original project remains available at
`dimitar-grigorov/mcp-file-tools` and is configured locally as the `upstream`
Git remote.

## Current state

- GitHub repository: `https://github.com/zoster81/mcp-file-tools`
- Supported transports: stdio and native stateful Streamable HTTP, constructed from one shared server and 27-tool unreleased source catalog
- Validated stdio deployment: client-managed local process and ChatGPT Web through the OpenAI Secure MCP Tunnel
- Validated HTTP deployment: persistent authenticated loopback service; non-loopback deployments require the TLS or trusted-proxy controls in [HTTP_SECURITY.md](HTTP_SECURITY.md)
- Fork update checker: `zoster81/mcp-file-tools` GitHub Releases
- Completed foundations: shared text-document core (R1), secure filesystem walker (R2), durable atomic mutation layer (R3), typed operation errors (R4), bounded ordered concurrency (R5), shared execution preparation plus authoritative tool metadata (R6), conservative extension-independent encoding detection (R8), bounded-memory streaming (R9), public API compatibility cleanup with a 23-tool catalog (R10), transport-independent server construction plus lifecycle-aware stdio startup (R11), the approved Streamable HTTP threat model and secure defaults (R12), and native stateful Streamable HTTP with security and equivalence coverage (R13)
- Completed milestone: R14 hardening, 2.0.0 publication, dual-transport deployment, active rollback, restoration, and final handoff
- Completed milestone: R15 attributed agent-ergonomics and project-aware workflow improvements, preserving the existing 23-tool catalog and adding three transport-independent prompts; the implementation remains unreleased and undeployed
- Completed development milestone: R16 verified change workflows; deterministic `fingerprint_paths`, bounded one-shot `edit_file` preview/apply, complete strict `patch_package` inspect/dry-run/apply/verify, and typed read-only `verify_state` checks are verified in the unreleased 26-tool R16 source catalog without changing the published 2.0.0 baseline
- Completed development milestone: R18 persistent backups; the protected store, exact-byte capture/recovery/index/audit core, bounded `backup_store` review, original-target restore, explicit generation-bound GC, approval-bound `edit_file`, and conservative all-target required `patch_package` capture are implemented and verified while the unreleased catalog remains 27 tools and the published runtime remains unchanged
- R14 container baseline: Go 1.26.5 builder, Alpine 3.24.1 runtime, static binary, UID/GID 10001, explicit `/data` root, temporary state under `/tmp`, and `SIGTERM` shutdown
- R14 CI baseline: the GitHub Test workflow passes on Linux, Windows, and macOS, including the complete race detector and native binary MCP smoke; the Build workflow passes all six supported OS/architecture targets plus the hardened Ubuntu container stdio/direct-TLS HTTP gate
- R14 local container baseline: the Linux/amd64 image builds from the pinned Dockerfile and passes UID/GID 10001, read-only root filesystem, dropped-capability, `no-new-privileges`, bounded-tmpfs, SDK-driven stdio MCP, direct-TLS HTTP, `401`/`403`/`405`/no-CORS, readiness, and clean `SIGTERM` runtime checks under rootless Podman
- R14 GoReleaser baseline: archive entry owner, group, mode, and modification time are normalized to commit-derived values; two independent snapshots produce identical checksums for all six raw binaries and six platform archives
- R14 release baseline: fork tag `v2.0.0` points to commit `1530fbb1eab529a1ef7236b4b3df8ab84a9a0d1d`; the release workflow passed on Linux, Windows, and macOS, published all six raw binaries, six archives, and `checksums.txt`, and completed MCP Registry publication
- R14 Registry baseline: the published `io.github.zoster81/mcp-file-tools` version is `2.0.0`; the release-specific manifest contains 6 packages and 23 tools and was published through GitHub OIDC with MCP Publisher 1.7.9
- R14 deployment baseline: the published Windows amd64 binary is active through both stdio and native Streamable HTTP; live checks passed for version identity, health/readiness, unauthenticated `401`, authenticated session initialization, and the complete 23-tool catalog
- R14 rollback baseline: the retained R10 build passed an active 23-tool stdio rollback with the HTTP listener intentionally absent; the verified `2.0.0` launcher and runtime were restored, and repeated stdio plus authenticated Streamable HTTP checks passed
- Authoritative product direction: [PROJECT_DIRECTION.md](PROJECT_DIRECTION.md); plan: [ROADMAP.md](ROADMAP.md); HTTP security design: [HTTP_SECURITY.md](HTTP_SECURITY.md); migration guide: [MIGRATION_2.0.md](MIGRATION_2.0.md); reusable gates: [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md)
- Release policy: semantic tags must match a dated changelog entry; `v2.0.0` is the first public 2.x release
- Release source: the final fork-owned semantic tag, which must match the dated changelog entry, embedded binary version, and generated Registry version
- Go module path: `github.com/zoster81/mcp-file-tools`

The fork owns its Go module identity and all internal imports. Clone-and-build remains supported for development commits, while packaged installations use fork-owned semantic releases and their verified assets.

R11 defines one process-wide authorization model for every transport: all connections or future HTTP sessions share the directories configured when the process starts, together with the same tool catalog, limits, execution flags, and errors. Sessions are lifecycle and concurrency units, not per-agent ACLs. Prompt instructions may narrow an agent's intended write scope, but technical isolation requires separate server processes and, for concurrent Git changes, separate checkouts or worktrees. Dynamic client roots remain a stdio-only fallback when no startup directories are configured; HTTP sessions must not mutate process roots.

The released native HTTP transport preserves the mandatory [HTTP security design](HTTP_SECURITY.md): loopback by default, bearer authentication on every MCP request, exact Host and Origin validation, no CORS, bounded per-request and aggregate body memory, bounded sessions and request resources, redacted logging, and a second explicit execution opt-in.

## Fork release flow

Use this flow for later fork-owned semantic releases. Development commits may be tested or deployed internally, but public tags require a dated changelog entry and the full applicable release gate.

1. Ensure the release-scoped roadmap work is complete and `main` is clean, tested, and pushed to `origin`.
2. Choose a semantic version that has not been used by this fork.
3. Promote the `CHANGELOG.md` unreleased section to a dated `## X.Y.Z - YYYY-MM-DD` release heading.
4. Verify that the semantic tag is represented by that exact changelog release:

   ```bash
   node scripts/verify-release-version.js vX.Y.Z
   ```

5. Push `main` and wait for its GitHub Actions checks to pass.
6. Create and push the release tag:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

7. `.github/workflows/release.yml` revalidates the tag/metadata match, then runs tests and GoReleaser.
8. `.goreleaser.yml` publishes the release to `zoster81/mcp-file-tools` with:
   - reproducible `-trimpath` builds timestamped from the source commit;
   - archive binary and documentation entries with commit-derived timestamps plus fixed owner, group, and modes;
   - `.tar.gz` archives for Linux/macOS and `.zip` archives for Windows;
   - raw binaries for all six supported OS/architecture targets;
   - `checksums.txt`;
   - `README.md`, `TOOLS.md`, `CHANGELOG.md`, `LICENSE`;
   - `examples/start-openai-tunnel.ps1`;
   - `examples/start-streamable-http.ps1`.
9. Verify the release asset names and SHA-256 values before announcing it.
10. `.github/workflows/release.yml` invokes the reusable `.github/workflows/publish-registry.yml`, which generates and publishes the fork-owned `server.json` from those verified assets.

The fork-owned Claude Code downloader plugin is removed for 2.0. This avoids a second release downloader, cache, checksum parser, and platform-mapping trust boundary. Claude Code users can configure the published binary directly as a normal stdio MCP server.

## Public launcher examples

The tracked launchers are intentionally separate, single-transport reference examples. Operators may combine them in a private deployment launcher, but private credentials and machine-specific orchestration must remain outside the repository.

`examples/start-openai-tunnel.ps1` is the public stdio-through-tunnel quick start. It must:

- remain in English;
- contain placeholders only;
- never contain a real Runtime API key or Tunnel ID;
- require the exact `tunnel_` plus 32 lowercase hexadecimal identifier format;
- select `--transport=stdio` explicitly;
- keep `run_script` and `shell` disabled by default;
- validate the tunnel client, MCP binary, and canonical allowed directory;
- run `tunnel-client doctor --explain` before starting the daemon;
- restore all managed process-level environment variables when it exits.

`examples/start-streamable-http.ps1` is the standalone native HTTP reference. It must:

- bind to loopback by default and require explicit non-loopback opt-in;
- require TLS or an explicitly trusted proxy CIDR for non-loopback use;
- load the bearer token from a regular private file rather than a command-line argument;
- keep both execution authorization layers disabled by default;
- clear unrelated control-plane credentials from the server child environment;
- restore all managed process-level environment variables when it exits.

Real credentials belong in private copies outside the Git checkout.

## MCP Registry status

`server.template.json` is a release-neutral template owned by `zoster81/mcp-file-tools`. It contains the fork namespace, repository, homepage, package filenames, zeroed checksum placeholders, and an intentionally empty `tools` array; it is not published directly. The authoritative tool names, descriptions, titles, and annotations live in `internal/toolcatalog/catalog.json`, which is embedded by the Go runtime.

`.github/workflows/publish-registry.yml` runs only in `zoster81/mcp-file-tools`. After a fork release is published, it downloads that release's `checksums.txt`; `scripts/generate-server-json.js` injects the catalog's Registry-facing name/description projection, exact version, fork download URLs, and SHA-256 values into a temporary `server.json`. The generator rejects a duplicated template catalog, invalid or duplicate catalog tools, and missing or unexpected MCP binaries before GitHub OIDC authentication.

The registry identity is `io.github.zoster81/mcp-file-tools`. Version `2.0.0` was generated from the fork-owned release's `checksums.txt`, validated with MCP Publisher 1.7.9, authenticated through GitHub OIDC, and published with all six platform packages and the authoritative 23-tool projection. The inherited upstream `v2.0.0` tag was not used; the fork-owned tag resolves to the release commit recorded above.

## Upstream relationship

The upstream Claude Code marketplace and upstream MCP Registry listing install the independent `dimitar-grigorov/mcp-file-tools` implementation. Upstream has continued to evolve its encoding-focused stdio product and has independently implemented several ideas also explored in this fork, including stronger path containment, durable writes, BOM and line-ending behavior, ordered work, BOMless UTF-16 detection, richer grep/edit workflows, MCP prompts, and `.gitignore`-aware traversal.

R15 explicitly credits the original project for its line-number, richer grep, `.gitignore`, sorting, batch-conversion, prompt, patch, and fuzzy-edit features, together with the implementation approaches reviewed during design. The resulting code is reworked for this fork's stable 23-tool catalog, bounded-memory pipeline, durable mutation semantics, secure walker, process-wide roots, and stdio/Streamable HTTP equivalence rather than mechanically synchronized. This is treated as reciprocal exchange: useful functionality, implementation techniques, tests, and security improvements may flow in either direction.

This fork is not an upstream synchronization branch. Its distinguishing scope includes native stateful Streamable HTTP, the reviewed HTTP security contract, optional execution tools, one process-wide multi-transport policy, stricter bounded-memory and durable-mutation guarantees, fork-owned release/Registry infrastructure, and tunnel/container deployment work. Cross-project ideas must be evaluated against each repository's current schemas and architecture rather than copied mechanically. See [PROJECT_DIRECTION.md](PROJECT_DIRECTION.md).

The fork does not publish a Claude Code marketplace plugin in 2.0. Its supported distribution surfaces are release binaries and archives, the container definition, Smithery metadata, and the fork-owned MCP Registry entry.

## Upstream synchronization

Use the two-remotes model:

```text
origin   -> https://github.com/zoster81/mcp-file-tools.git
upstream -> https://github.com/dimitar-grigorov/mcp-file-tools.git
```

Fetch and review upstream changes without assuming that the histories remain directly mergeable:

```bash
git fetch upstream
git log --oneline --left-right main...upstream/main
git diff main...upstream/main
```

Adopt upstream ideas or isolated changes only after redesigning or reviewing them against this fork's roots, streaming, mutation, execution, transport, update-check, release, and deployment behavior. Do not merge or rebase upstream mechanically merely to reduce commit divergence.

## Validation toolchain

The repository pins the release-validation toolchain instead of resolving
floating `latest` versions during CI:

- actionlint 1.7.12 and ShellCheck 0.11.0 for GitHub Actions workflows;
- actions/checkout 6.0.2, actions/setup-go 7, and actions/upload-artifact 7.0.1;
- Staticcheck v0.7.0 and govulncheck v1.1.4 for Go analysis;
- GoReleaser action 7.2.1 with GoReleaser v2.17.0 for release generation;
- MCP Publisher v1.7.9 for registry validation and publication.

The workflow-linter archives and MCP Publisher archive are verified against
fixed SHA-256 values before extraction. Local release preparation should also
run Gitleaks and use Cosign to verify signed release assets when bundles are
available.

## Release verification checklist

Run this checklist after the applicable release-scoped milestone and completion gates in [ROADMAP.md](ROADMAP.md) pass. Use [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md) for internal milestones and builds.

- release-scoped roadmap work complete, with any intentionally post-publication deployment gate recorded explicitly;
- working tree clean;
- expected branch and HEAD verified;
- no credentials or real tunnel identifiers in tracked files or history;
- `go test -count=1 ./...` succeeds;
- `go vet ./...` succeeds;
- `go mod verify` succeeds;
- PowerShell example parses under Windows PowerShell 5.1;
- JSON, YAML, JavaScript, Markdown, actionlint, and ShellCheck checks succeed;
- README, tool reference, changelog, Smithery metadata, runtime catalog, roadmap, and generated Registry manifest describe the same fork-owned capabilities;
- runtime tool registration and annotations match `internal/toolcatalog/catalog.json`, every catalog tool is linked from README and documented in TOOLS.md, and `server.template.json` keeps `tools` empty;
- secure traversal tests cover Unix symlinks and Windows junction/reparse-point escapes;
- mutation tests cover synced staging, transactional backup rollback, cleanup, no-replace creation, concurrent-modification rejection, and platform cross-builds;
- typed-error tests cover standard and joined causes, security/path categories, encoding categories, mutation conflicts, cancellation, and stable batch error codes;
- ordered-concurrency tests cover bounded in-flight work, deterministic commit order, cancellation modes, early stop, saturation, and race detection;
- Staticcheck and govulncheck succeed at the versions pinned by CI;
- Gitleaks reports no tracked-history secrets;
- the container image is built from pinned bases, runs as UID/GID 10001, and passes stdio plus direct-TLS HTTP smoke tests with the documented mount, tmpfs, health, and shutdown model;
- all six Windows/Linux/macOS amd64/arm64 builds are generated, and representative binaries are runtime-executed where infrastructure permits;
- GoReleaser configuration targets `zoster81/mcp-file-tools`, passes `goreleaser check`, and produces identical checksums across two independent snapshots;
- a generated manifest passes `mcp-publisher validate` without publication;
- `scripts/verify-release-version.js` confirms the release tag has a matching dated changelog entry before GoReleaser runs;
- release tag, changelog release, embedded binary version, and generated Registry version match;
- release assets and checksums are verified after publication;
- the known-good rollback binary and launcher reversal are verified offline before deployment, followed by an active rollback test during the controlled release cutover.
