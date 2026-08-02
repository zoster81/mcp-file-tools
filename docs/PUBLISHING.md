# Fork publishing notes

This document describes release and distribution work for the
`zoster81/mcp-file-tools` fork. The original project remains available at
`dimitar-grigorov/mcp-file-tools` and is configured locally as the `upstream`
Git remote.

## Current state

- GitHub repository: `https://github.com/zoster81/mcp-file-tools`
- Primary deployment: ChatGPT Web through the OpenAI Secure MCP Tunnel
- Implemented MCP transports in the development source: stdio and native stateful Streamable HTTP
- Primary validated OpenAI Secure MCP Tunnel deployment: stdio; native HTTP has direct end-to-end coverage and remains in internal pre-release validation rather than active tunnel deployment
- Fork update checker: `zoster81/mcp-file-tools` GitHub Releases
- Completed foundations: shared text-document core (R1), secure filesystem walker (R2), durable atomic mutation layer (R3), typed operation errors (R4), bounded ordered concurrency (R5), shared execution preparation plus authoritative tool metadata (R6), conservative extension-independent encoding detection (R8), bounded-memory streaming (R9), public API compatibility cleanup with a 23-tool catalog (R10), transport-independent server construction plus lifecycle-aware stdio startup (R11), the approved Streamable HTTP threat model and secure defaults (R12), and native stateful Streamable HTTP with security and equivalence coverage (R13)
- Active milestone: R14 hardening, CI, packaging, migration review, and 2.0.0 release
- R14 container baseline: Go 1.26.5 builder, Alpine 3.24.1 runtime, static binary, UID/GID 10001, explicit `/data` root, temporary state under `/tmp`, and `SIGTERM` shutdown
- R14 CI baseline: documentation-sensitive tests on Linux, Windows, and macOS plus independent builds for all six supported OS/architecture targets
- R14 GoReleaser baseline: archive entry owner, group, mode, and modification time are normalized to commit-derived values; two independent snapshots produce identical checksums for all six raw binaries and six platform archives
- R14 Registry baseline: internal manifests generated from both the direct six-target checksums and the reproducible GoReleaser snapshot pass `mcp-publisher 1.7.9 validate` without login or publication
- R14 rollback baseline: the known-good R10 binary passes exact hash/version and 23-tool stdio checks, and the private launcher filename update is byte-identically reversible; an active rollback still requires a controlled restart
- Authoritative plan: [ROADMAP.md](ROADMAP.md), HTTP security design in [HTTP_SECURITY.md](HTTP_SECURITY.md), migration guide in [MIGRATION_2.0.md](MIGRATION_2.0.md), and reusable gates in [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md)
- Development policy: internal commit builds only until the next public release, `2.0.0`
- Release source: the final fork-owned semantic tag, which must match the dated changelog entry, embedded binary version, and generated Registry version
- Go module path: `github.com/zoster81/mcp-file-tools`

The fork owns its Go module identity and all internal imports. Clone-and-build is the supported path for internal development commits. No intermediate public release is planned; packaged installations remain on the latest published release until the R14 `2.0.0` gate is complete.

R11 defines one process-wide authorization model for every transport: all connections or future HTTP sessions share the directories configured when the process starts, together with the same tool catalog, limits, execution flags, and errors. Sessions are lifecycle and concurrency units, not per-agent ACLs. Prompt instructions may narrow an agent's intended write scope, but technical isolation requires separate server processes and, for concurrent Git changes, separate checkouts or worktrees. Dynamic client roots remain a stdio-only fallback when no startup directories are configured; HTTP sessions must not mutate process roots.

R12's [HTTP security design](HTTP_SECURITY.md) remains release-blocking through R14. The R13 implementation is fail-closed: loopback by default, bearer-authenticated on every MCP request, exact Host and Origin validation, no CORS, bounded per-request and aggregate body memory, bounded sessions and request resources, redacted logging, and a second explicit execution opt-in. It remains an internal pre-release capability and has not been published or deployed.

## Fork release flow

This flow is reserved for the R14 `2.0.0` release gate. Do not create a tag or publish intermediate development builds.

1. Ensure R7-R14 are complete and `main` is clean, tested, and pushed to `origin`.
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
   - `examples/start-openai-tunnel.ps1`.
9. Verify the release asset names and SHA-256 values before announcing it.
10. `.github/workflows/release.yml` invokes the reusable `.github/workflows/publish-registry.yml`, which generates and publishes the fork-owned `server.json` from those verified assets.

The fork-owned Claude Code downloader plugin is removed for 2.0. This avoids a second release downloader, cache, checksum parser, and platform-mapping trust boundary. Claude Code users can configure the published binary directly as a normal stdio MCP server.

## OpenAI Tunnel example

`examples/start-openai-tunnel.ps1` is the public quick-start launcher. It must:

- remain in English;
- contain placeholders only;
- never contain a real Runtime API key or Tunnel ID;
- keep `run_script` and `shell` disabled by default;
- validate the tunnel client, MCP binary, and allowed directory;
- run `tunnel-client doctor --explain` before starting the daemon;
- remove process-level credential variables when it exits.

Real credentials belong in a private copy outside the Git checkout.

## MCP Registry status

`server.template.json` is a release-neutral template owned by `zoster81/mcp-file-tools`. It contains the fork namespace, repository, homepage, package filenames, zeroed checksum placeholders, and an intentionally empty `tools` array; it is not published directly. The authoritative tool names, descriptions, titles, and annotations live in `internal/toolcatalog/catalog.json`, which is embedded by the Go runtime.

`.github/workflows/publish-registry.yml` runs only in `zoster81/mcp-file-tools`. After a fork release is published, it downloads that release's `checksums.txt`; `scripts/generate-server-json.js` injects the catalog's Registry-facing name/description projection, exact version, fork download URLs, and SHA-256 values into a temporary `server.json`. The generator rejects a duplicated template catalog, invalid or duplicate catalog tools, and missing or unexpected MCP binaries before GitHub OIDC authentication.

The target registry identity is `io.github.zoster81/mcp-file-tools`. Internal prerelease manifests generated from the verified direct-build checksum set and the reproducible GoReleaser snapshot checksum set both passed `mcp-publisher 1.7.9 validate` without authentication or publication. Final Registry validation remains dependent on a real fork release with all six platform binaries and `checksums.txt`; inherited tags alone are insufficient.

## Upstream integrations

The upstream Claude Code marketplace and existing MCP Registry listing install
the upstream implementation. They do not include this fork's execution tools,
tunnel compatibility changes, or Windows drive-root fix.

The fork does not publish a Claude Code marketplace plugin in 2.0. The supported distribution surfaces are release binaries and archives, the container definition, Smithery metadata, and the fork-owned MCP Registry entry.

## Upstream synchronization

Use the two-remotes model:

```text
origin   -> https://github.com/zoster81/mcp-file-tools.git
upstream -> https://github.com/dimitar-grigorov/mcp-file-tools.git
```

Fetch and review upstream changes without rewriting local history:

```bash
git fetch upstream
git log --oneline --left-right main...upstream/main
git diff main...upstream/main
```

Integrate upstream changes only after reviewing conflicts with fork-specific
roots, execution, update-check, release, and tunnel behavior.

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

Run this checklist only after every milestone and completion gate in [ROADMAP.md](ROADMAP.md) passes. Use [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md) for internal milestones and builds.

- R7-R14 complete;
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
