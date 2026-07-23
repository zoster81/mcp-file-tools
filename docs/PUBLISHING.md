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
- Go module path: intentionally retained as `github.com/dimitar-grigorov/mcp-file-tools`
  for source compatibility with the upstream codebase

Because the module path remains upstream-compatible, fork users should clone and
build the repository or download a fork release. A `go install` command using the
upstream module path installs upstream code rather than the custom fork.

## Fork release flow

1. Ensure `main` is clean, tested, and pushed to `origin`.
2. Choose a semantic version that has not been used by this fork.
3. Update the version in:
   - `plugin/.claude-plugin/plugin.json`
   - `.claude-plugin/marketplace.json`
4. Run the complete test and verification baseline.
5. Create and push the release tag:

   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

6. `.github/workflows/release.yml` runs tests and GoReleaser.
7. `.goreleaser.yml` publishes the release to `zoster81/mcp-file-tools` with:
   - platform archives;
   - raw platform binaries;
   - `checksums.txt`;
   - `README.md`, `TOOLS.md`, `CHANGELOG.md`, `LICENSE`;
   - `examples/start-openai-tunnel.ps1`.
8. Verify the release asset names and SHA-256 values before announcing it.

The plugin launcher reads its pinned version from
`plugin/.claude-plugin/plugin.json` and downloads binaries from the fork release.
A plugin version must therefore have a matching GitHub Release and
`checksums.txt` asset.

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

The checked-in `server.json` describes the existing upstream MCP Registry entry
and its upstream release hashes. It must not be presented or published as the
fork entry.

`.github/workflows/publish-registry.yml` is guarded so it does not publish from
this fork. Before enabling registry publication for `zoster81/mcp-file-tools`,
all of the following are required:

1. approve a fork-specific registry namespace;
2. update repository and homepage metadata;
3. include all fork tools and security descriptions;
4. publish matching fork release assets;
5. calculate and verify every asset SHA-256;
6. test OIDC publication from the fork repository.

## Upstream integrations

The upstream Claude Code marketplace and existing MCP Registry listing install
the upstream implementation. They do not include this fork's execution tools,
tunnel compatibility changes, or Windows drive-root fix.

The fork contains plugin files for future compatibility. Their repository and
download references point to `zoster81/mcp-file-tools`, but they should not be
advertised as ready until a matching fork release has been tested end to end.

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

## Release verification checklist

- working tree clean;
- expected branch and HEAD verified;
- no credentials or real tunnel identifiers in tracked files or history;
- `go test -count=1 ./...` succeeds;
- `go vet ./...` succeeds;
- `go mod verify` succeeds;
- PowerShell example parses under Windows PowerShell 5.1;
- JSON, YAML, JavaScript, and Markdown checks succeed;
- GoReleaser configuration targets `zoster81/mcp-file-tools`;
- release tag, embedded binary version, plugin version, and marketplace version match;
- release assets and checksums are verified after publication.
