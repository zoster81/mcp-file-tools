# Fork publishing notes

This document describes release and distribution work for the
`zoster81/mcp-file-tools` fork. The original project remains available at
`dimitar-grigorov/mcp-file-tools` and is configured locally as the `upstream`
Git remote.

## Current state

- GitHub repository: `https://github.com/zoster81/mcp-file-tools`
- Primary deployment: ChatGPT Web through the OpenAI Secure MCP Tunnel
- Implemented MCP transport: stdio
- Native HTTP/JSON or Streamable HTTP transport: not implemented
- Fork update checker: `zoster81/mcp-file-tools` GitHub Releases
- Completed foundations: shared text-document core (R1), secure filesystem walker (R2), durable atomic mutation layer (R3), typed operation errors (R4), bounded ordered concurrency (R5), shared execution preparation plus authoritative tool metadata (R6), conservative extension-independent encoding detection including BOMless UTF-16 (R8), and bounded-memory streaming plus disk-staged large-file mutations (R9)
- Active milestone: R10 public API and compatibility cleanup
- Authoritative plan: [ROADMAP.md](ROADMAP.md), with reusable gates in [DEVELOPMENT_CHECKLIST.md](DEVELOPMENT_CHECKLIST.md)
- Development policy: internal commit builds only until the next public release, `2.0.0`
- Release source: the final fork-owned semantic tag whose plugin, marketplace, binary, and Registry versions must match
- Go module path: `github.com/zoster81/mcp-file-tools`

The fork owns its Go module identity and all internal imports. Clone-and-build is the supported path for internal development commits. No intermediate public release is planned; packaged installations remain on the latest published release until the R14 `2.0.0` gate is complete.

## Fork release flow

This flow is reserved for the R14 `2.0.0` release gate. Do not bump plugin metadata, create a tag, or publish intermediate development builds.

1. Ensure R7-R14 are complete and `main` is clean, tested, and pushed to `origin`.
2. Choose a semantic version that has not been used by this fork.
3. Update the version in:
   - `plugin/.claude-plugin/plugin.json`
   - `.claude-plugin/marketplace.json`
4. Verify the prepared version and run the complete test baseline:

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
   - platform archives;
   - raw platform binaries;
   - `checksums.txt`;
   - `README.md`, `TOOLS.md`, `CHANGELOG.md`, `LICENSE`;
   - `examples/start-openai-tunnel.ps1`.
9. Verify the release asset names and SHA-256 values before announcing it.
10. `.github/workflows/release.yml` invokes the reusable `.github/workflows/publish-registry.yml`, which generates and publishes the fork-owned `server.json` from those verified assets.

The optional Claude Code plugin launcher reads its pinned version from `plugin/.claude-plugin/plugin.json` and downloads binaries from the fork release. It is not part of the active OpenAI tunnel deployment. Whether it remains supported will be decided in R14; if retained, its version must match the `2.0.0` GitHub Release and `checksums.txt` asset.

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

The target registry identity is `io.github.zoster81/mcp-file-tools`. Publishing remains dependent on a real fork release with all six platform binaries and `checksums.txt`; inherited tags alone are insufficient.

## Upstream integrations

The upstream Claude Code marketplace and existing MCP Registry listing install
the upstream implementation. They do not include this fork's execution tools,
tunnel compatibility changes, or Windows drive-root fix.

The optional fork plugin references `zoster81/mcp-file-tools`. It remains frozen during internal development. If retained for 2.0, it will be versioned together with the matching platform binaries and `checksums.txt`; the release workflow will continue rejecting a tag whose version differs from plugin or marketplace metadata.

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
- Staticcheck v0.7.0 and govulncheck v1.1.4 for Go analysis;
- GoReleaser v2.17.0 for release generation;
- MCP Publisher v1.8.0 for registry validation and publication.

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
- README, tool reference, changelog, plugin metadata, Smithery metadata, runtime catalog, roadmap, and generated Registry manifest describe the same fork-owned capabilities;
- runtime tool registration and annotations match `internal/toolcatalog/catalog.json`, every catalog tool is linked from README and documented in TOOLS.md, and `server.template.json` keeps `tools` empty;
- secure traversal tests cover Unix symlinks and Windows junction/reparse-point escapes;
- mutation tests cover synced staging, transactional backup rollback, cleanup, no-replace creation, concurrent-modification rejection, and platform cross-builds;
- typed-error tests cover standard and joined causes, security/path categories, encoding categories, mutation conflicts, cancellation, and stable batch error codes;
- ordered-concurrency tests cover bounded in-flight work, deterministic commit order, cancellation modes, early stop, saturation, and race detection;
- Staticcheck and govulncheck succeed at the versions pinned by CI;
- Gitleaks reports no tracked-history secrets;
- GoReleaser configuration targets `zoster81/mcp-file-tools` and passes `goreleaser check`;
- a generated manifest passes `mcp-publisher validate` without publication;
- `scripts/verify-release-version.js` confirms the release tag, plugin version, and marketplace version match before GoReleaser runs;
- release tag, embedded binary version, plugin version, and marketplace version match;
- release assets and checksums are verified after publication.
