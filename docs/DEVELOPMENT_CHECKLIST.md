# Development Checklist

Use this checklist for every active roadmap milestone in [ROADMAP.md](ROADMAP.md). It is the reusable engineering gate; milestone-specific requirements stay in the roadmap and current operational state stays in `D:\OpenAI-Tunnel\todo.md`.

## 1. Requirements and edge cases

- [ ] Restate the requested behavior and expected outcome.
- [ ] Identify affected packages, handlers, schemas, metadata, documentation, and deployment files.
- [ ] Identify compatibility constraints and intentional breaking changes.
- [ ] Define valid, empty, missing, malformed, ambiguous, oversized, and unsupported inputs.
- [ ] Define encoding, BOM, and LF/CRLF behavior where relevant.
- [ ] Define filesystem, permission, read-only, missing-path, and partial-failure behavior.
- [ ] Review path traversal, symlink, junction, reparse-point, and workspace-escape risks.
- [ ] Review cancellation, timeout, concurrency, race, and TOCTOU behavior.
- [ ] Review Windows, Linux, macOS, long-path, and cross-platform implications.
- [ ] Identify likely regressions and public behavior that must remain unchanged.

## 2. Architecture and test strategy

- [ ] Define the smallest coherent component and file set before editing.
- [ ] Keep transport, domain logic, filesystem primitives, and MCP adapters separated.
- [ ] Reuse existing domain primitives instead of adding handler-local copies.
- [ ] Define data flow, ownership, memory bounds, and cleanup responsibilities.
- [ ] Define typed errors and public error mapping.
- [ ] Define time and space complexity when relevant.
- [ ] Write or select focused failing tests before implementation when practical.
- [ ] Include normal behavior, edge cases, invalid input, and regression coverage.
- [ ] Include encoding/BOM and LF/CRLF cases where relevant.
- [ ] Include filesystem failure and concurrent-modification cases where relevant.
- [ ] Include cancellation, timeout, saturation, and race cases where relevant.
- [ ] Include platform-specific and cross-build coverage where relevant.
- [ ] Include security-negative tests, not only successful paths.

## 3. Devil's advocate review

- [ ] Identify at least two concrete implementation risks.
- [ ] Review workspace escape and path-based race windows.
- [ ] Review data loss, non-atomic writes, rollback, cleanup, and recovery artifacts.
- [ ] Review unbounded memory, output, lines, requests, sessions, or worker queues.
- [ ] Review nondeterministic ordering and cancellation behavior.
- [ ] Review encoding corruption, malformed Unicode, and binary false positives.
- [ ] Review dependency, platform, API, and documentation drift.
- [ ] Revise the design before implementation if mitigations are insufficient.

## 4. Repository safety before editing

- [ ] Normalize and verify the repository path is exactly inside `D:\OpenAI-Tunnel`.
- [ ] Read `D:\OpenAI-Tunnel\todo.md` and the active roadmap milestone.
- [ ] Verify current branch, `HEAD`, `origin/main`, and working-tree status.
- [ ] Preserve unrelated user changes.
- [ ] Verify the active MCP binary and process when deployment state matters.
- [ ] Inspect each target file's surrounding context.
- [ ] Check size, encoding, BOM, and line endings when relevant.
- [ ] Prefer targeted edits and no-replace moves over full rewrites.
- [ ] Do not use destructive Git commands.
- [ ] Do not commit, push, tag, release, or restart unless explicitly requested.

## 5. TDD and implementation

- [ ] Reproduce the issue or missing behavior with a focused test.
- [ ] Confirm the test fails for the expected reason.
- [ ] Implement the smallest correct production change.
- [ ] Keep public schemas stable unless the active milestone explicitly changes them.
- [ ] Preserve formatting, encoding, BOM state, and line endings.
- [ ] Use explicit error handling and cleanup.
- [ ] Add comments only for genuinely non-obvious constraints.
- [ ] Avoid unrelated refactors.
- [ ] Rerun the focused test and confirm it passes.
- [ ] Review the changed code before broader verification.

## 6. Verification ladder

Run the applicable checks from focused to broad. Record commands and outcomes exactly.

### Focused checks

- [ ] focused package tests;
- [ ] focused handler/integration tests;
- [ ] metadata or script tests;
- [ ] platform-specific focused tests.

### Go baseline

- [ ] `gofmt` on changed Go files;
- [ ] `go mod verify`;
- [ ] `go test ./... -count=1`;
- [ ] `go vet ./...`;
- [ ] Staticcheck at the repository-pinned version;
- [ ] `govulncheck ./...` at the repository-pinned version;
- [ ] race detector with a working CGO compiler;
- [ ] coverage review for affected packages.

### Build and platform checks

- [ ] native build for the active development platform;
- [ ] Windows amd64/arm64 cross-builds;
- [ ] Linux amd64/arm64 cross-builds;
- [ ] macOS amd64/arm64 cross-builds;
- [ ] runtime execution on available platforms when behavior is platform-specific.

### Repository and documentation checks

- [ ] Node release-script tests;
- [ ] JSON and YAML parsing;
- [ ] PowerShell parsing when launchers change;
- [ ] actionlint and ShellCheck when workflows/scripts change;
- [ ] Markdown local-link verification;
- [ ] catalog/runtime/documentation drift tests;
- [ ] `git diff --check`;
- [ ] final diff review;
- [ ] final `git status` with no unexpected files.

### Security and release-adjacent checks

- [ ] Gitleaks for tracked content and history when relevant;
- [ ] GoReleaser configuration check when packaging changes;
- [ ] non-published MCP Registry manifest validation when catalog or packaging changes;
- [ ] no credentials, tokens, private keys, cookies, or real tunnel identifiers added.

## 7. Internal build and deployment

Internal development builds are allowed before 2.0.0. Public releases are not.

- [ ] Build from the exact verified commit.
- [ ] Embed an unambiguous commit-derived version.
- [ ] Use a new versioned filename; do not overwrite the active rollback binary.
- [ ] Record size and SHA-256.
- [ ] Verify `--version` output.
- [ ] Update the private launcher only when explicitly requested.
- [ ] Preserve launcher encoding, BOM, line endings, and credentials.
- [ ] Restart only when explicitly requested.
- [ ] Verify active PID, executable path, parent tunnel PID, version, hash, and listener.
- [ ] Retain a known-good rollback binary.

## 8. Documentation and handoff

- [ ] Update [ROADMAP.md](ROADMAP.md) milestone status and checkboxes.
- [ ] Update only the documents whose behavior or promises changed.
- [ ] Keep examples generic unless a domain-specific example is genuinely necessary.
- [ ] Do not imply encoding detection based on file extensions.
- [ ] Keep current limitations explicit.
- [ ] Update `CHANGELOG.md` for user-visible or architectural changes.
- [ ] Replace `D:\OpenAI-Tunnel\todo.md` with the exact current operational state.
- [ ] Record files changed, tests executed, results, remaining risks, and checks not performed.
- [ ] State commit, push, build, deployment, and runtime status separately.
- [ ] State the exact files the next session must read.

## 9. Commit gate

Only commit when the user explicitly asks.

- [ ] Re-read the active handoff and verify repository state.
- [ ] Stage only files belonging to the active milestone.
- [ ] Review the staged diff and staged file list.
- [ ] Use one concise English commit message.
- [ ] Verify the commit contents and working tree afterward.
- [ ] Do not push unless explicitly requested.

## 10. Public 2.0.0 release gate

Do not perform these steps for internal builds.

- [ ] R7-R14 are complete.
- [ ] migration guide is complete;
- [ ] security design and HTTP test suite pass;
- [ ] plugin retention decision is complete;
- [ ] release metadata is set to `2.0.0`;
- [ ] release tag, binary version, plugin version, marketplace version, and Registry version match;
- [ ] all six platform assets and checksums are verified;
- [ ] release, Registry publication, deployment, smoke tests, and rollback tests succeed.
