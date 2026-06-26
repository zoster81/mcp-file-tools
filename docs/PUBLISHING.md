# Publishing notes

Internal notes on how this server is distributed and what's left to do. Not meant
for the public repo.

## Current state

We're already live in the MCP Registry as `io.github.dimitar-grigorov/mcp-file-tools`
(v1.7.0, 14 versions). The entry is valid: packages point to real release binaries
with matching SHA-256s. Smithery is also configured (smithery.yaml).

The thing that wasn't done: registry publishing was manual, and there was no Claude
Code plugin. Both are addressed below.

## How a release publishes to the registry

1. `git tag vX.Y.Z && git push origin vX.Y.Z`
2. release.yml runs GoReleaser, which builds binaries and creates the GitHub Release
   (including checksums.txt).
3. publish-registry.yml now triggers on `release: published`, rewrites server.json
   with the version and checksums, then runs `mcp-publisher login github-oidc` and
   `mcp-publisher publish`.

So tag push to registry publish is now automatic. The workflow still has a
workflow_dispatch input for re-publishing a specific tag by hand.

Note: "auto-registered" only means the registry entry updates on release. It does not
mean Claude Code or Desktop auto-install the server. Users still install via .mcp.json,
the plugin, or a .mcpb bundle.

## Allowed directories

The server needs to know which directories it may touch. It gets them from the client
via the MCP roots protocol (filetoolsserver/roots.go), so in Claude Code the open
workspace is allowed automatically, no config or install prompt. CLI args still work as
a fallback for clients that don't send roots. Smithery prompts via configSchema.

## Claude Code plugin

Files:
- .claude-plugin/marketplace.json lets users run `/plugin marketplace add dimitar-grigorov/mcp-file-tools`
- plugin/.mcp.json declares the MCP server, launched as `bash bin/run.sh`
  (plugin/.claude-plugin/plugin.json holds only the metadata; an inline mcpServers
  block there is not picked up, the server must live in .mcp.json)
- plugin/bin/run.sh downloads the pinned release binary on first run, verifies its
  SHA-256, caches it under CLAUDE_PLUGIN_DATA, and execs it
- plugin/bin/run.ps1 is the PowerShell equivalent (not wired up by default)

The server can't ship binaries in a git repo, so the launcher downloads them. Claude
Code spawns MCP servers without a shell and can't run a bare .sh, so plugin.json calls
`bash bin/run.sh`. That works on macOS/Linux and on Windows with Git Bash on PATH.

Tested on Windows: both launchers do download + checksum + exec on a cold cache and
reuse the cache when warm. `claude plugin validate` passes.

Not yet done: an end-to-end `/plugin install` from the marketplace, and a run on
macOS/Linux.

## Self-update

internal/updater/updater.go is notification-only, not an auto-updater. On startup it
checks GitHub for a newer release and prints a message. It's gated by
MCP_NO_UPDATE_CHECK=1 and skipped on dev builds.

Two issues, low priority: it makes a network call on every startup, and the
"re-download the binary" message is wrong for registry/Smithery/package installs.
Plan: make the check opt-in, or suppress the message for non-manual installs. Don't
build real auto-update for a filesystem server.

## TODO

1. Run `/plugin marketplace add` + `/plugin install` end to end; verify on macOS/Linux.
2. Make self-update opt-in.
3. Optional: ship a .mcpb bundle for one-click Claude Desktop installs.
4. Bump VERSION in plugin.json, marketplace.json, and both launcher scripts each release.

## Notes

The native GoReleaser `mcp:` block in .goreleaser.yml stays disabled because GoReleaser
can't emit fileSha256 for the mcpb type yet (goreleaser#6251). The standalone
publish-registry.yml is the workaround.
