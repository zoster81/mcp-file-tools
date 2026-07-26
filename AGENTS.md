# Repository Agent Guide

## Scope and precedence

This file applies to the entire repository. A nested `AGENTS.md` adds or overrides instructions only for files below its directory. Read this file, then the nearest scoped guide before editing.

Do not copy private workstation state, local process details, credentials, or operator-specific paths into tracked files. Public content must be reproducible by an external contributor from a normal clone.

## Sources of truth

- Product status and milestone gates: [`docs/ROADMAP.md`](docs/ROADMAP.md)
- Reusable engineering checks: [`docs/DEVELOPMENT_CHECKLIST.md`](docs/DEVELOPMENT_CHECKLIST.md)
- Contributor workflow: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Tool behavior and examples: [`TOOLS.md`](TOOLS.md)
- Release procedure: [`docs/PUBLISHING.md`](docs/PUBLISHING.md)
- Authoritative MCP tool metadata: [`internal/toolcatalog/catalog.json`](internal/toolcatalog/catalog.json)

Link to these documents instead of duplicating their detailed content.

## Repository map

- `cmd/mcp-file-tools`: CLI entry point and transport bootstrap.
- `filetoolsserver`: MCP server construction, roots, and tool registration.
- `filetoolsserver/handler`: MCP adapters and shared text-document behavior.
- `internal/encoding`: encoding registry and content-based detection.
- `internal/security`: path normalization, resolution, and allowed-root enforcement.
- `internal/filesystem`: secure traversal and durable mutation primitives.
- `internal/operation`: transport-independent error categories.
- `internal/concurrency`: bounded deterministic worker coordination.
- `internal/toolcatalog`: embedded tool metadata and drift checks.
- `scripts`: release metadata and workflow validation utilities.
- `plugin`: optional Claude Code packaging, frozen until the final 2.0 decision.

## Working method

For non-trivial changes:

1. Restate the intended behavior and identify compatibility, security, encoding, filesystem, concurrency, and platform edge cases.
2. Define the smallest coherent design and focused tests before implementation.
3. Review at least two concrete failure or regression risks and their mitigations.
4. Implement the smallest correct change, review the complete diff, and report exactly what was verified.

Use focused TDD when practical: reproduce, confirm the expected failure, implement, rerun focused tests, then run the relevant regression suite.

## Project invariants

- Encoding detection is derived from bytes and decoded-content evidence, never filenames or extensions.
- Unicode BOM evidence is authoritative. Ambiguous data must not be classified with unjustified confidence.
- New files default to UTF-8; existing files preserve a confidently detected encoding unless explicitly overridden.
- Preserve encoding, BOM policy, and line endings exactly where a tool promises preservation.
- All filesystem access must remain inside validated allowed roots after symlink, junction, and reparse-point resolution.
- Missing paths must be validated through their nearest existing ancestor.
- Mutations must preserve the durable staging, snapshot, no-replace, rollback, cleanup, and platform-sync guarantees in `internal/filesystem`.
- Public error schemas and tool metadata remain stable unless a roadmap milestone explicitly changes them.
- `run_script` and `shell` remain disabled by default. Do not weaken their distinct authorization boundaries.
- Preserve stdio behavior while transport work is in progress.

## Verification commands

Start with the narrowest applicable tests, then expand as needed:

```bash
go test ./path/to/affected/package -count=1
go mod verify
go test ./... -count=1
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
node --test scripts/generate-server-json.test.js scripts/verify-release-version.test.js
go run test_server.go
```

Run `go test -race ./...` only where a working CGO compiler is available. Run `bash scripts/validate-workflows.sh` when workflows or shell scripts change. Use the full release checks from `docs/PUBLISHING.md` only for release-related work.

Always run `gofmt` on changed Go files and `git diff --check`. Review final `git status` for unexpected files.

## Change rules

- Inspect surrounding code and tests before editing.
- Preserve existing formatting, encoding, BOM state, and line endings.
- Prefer shared internal primitives over handler-local copies.
- Keep changes atomic and avoid unrelated refactors.
- Treat paths, file contents, environment variables, process output, and network data as untrusted.
- Never add credentials, real tunnel identifiers, workstation paths, PIDs, private binary hashes, or operator handoff state to tracked files.
- Do not manually edit generated release output such as `server.json`; update its source template/catalog or generator instead.
- When tool metadata changes, update the catalog, runtime behavior, README links, TOOLS reference, tests, and release projection together.
- Do not change dependencies, public schemas, release versions, workflows, or packaging incidentally.

## Scoped guides

Additional instructions exist in:

- [`docs/AGENTS.md`](docs/AGENTS.md)
- [`filetoolsserver/handler/AGENTS.md`](filetoolsserver/handler/AGENTS.md)
- [`internal/encoding/AGENTS.md`](internal/encoding/AGENTS.md)
- [`internal/filesystem/AGENTS.md`](internal/filesystem/AGENTS.md)
- [`internal/security/AGENTS.md`](internal/security/AGENTS.md)
- [`scripts/AGENTS.md`](scripts/AGENTS.md)

Do not add another scoped guide unless that subtree has distinct commands, invariants, generated artifacts, or security constraints that cannot be stated clearly here.

## Completion report

State files changed, behavior affected, tests executed and their results, checks not performed, remaining risks, and repository status. Distinguish review, modification, compilation, testing, build, publication, and deployment; do not imply a step occurred when it did not.
