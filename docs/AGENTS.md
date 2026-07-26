# Documentation Agent Guide

This guide applies to files under `docs/`. Follow the root [`AGENTS.md`](../AGENTS.md) first.

## Document responsibilities

- `ROADMAP.md`: authoritative current milestones, design requirements, and completion gates.
- `DEVELOPMENT_CHECKLIST.md`: reusable, portable engineering and verification checks.
- `ROADMAP_HISTORY.md`: concise public engineering history, not an operator session log.
- `PUBLISHING.md`: maintainer release and distribution procedure.
- `MIGRATION_2.0.md`: authoritative intentional breaking changes and migration actions for 1.8 to 2.0.

Keep operational details in their proper source instead of duplicating them across documents.

## Portability rules

Documentation must be usable by an external contributor from a normal clone.

Do not include:

- private workspace or home-directory paths;
- connector instance names, local PIDs, active binary filenames, or workstation hashes;
- private handoff files or launcher state;
- credentials, real tunnel identifiers, or unsanitized configuration;
- instructions that depend on a specific contributor asking an agent to commit, push, or restart a service.

Use repository-relative links, environment variables, and obvious placeholders such as `/path/to/project` or `C:\Path\To\AllowedProject`.

Historical documents should record architectural outcomes, compatibility decisions, public releases, and reproducible validation—not ephemeral branch tracking, local deployment, or process state.

## Consistency

- Keep roadmap status consistent with README and publishing notes.
- Keep current limitations explicit and distinguish implemented behavior from planned work.
- Do not imply that filename extensions influence encoding detection.
- Do not claim streaming, atomicity, sandboxing, or platform support beyond what tests and implementation establish.
- When tool behavior changes, verify links and descriptions against `internal/toolcatalog/catalog.json` and `TOOLS.md`.
- Use English technical prose and stable headings suitable for direct links.

## Verification

For documentation-only changes, run at least:

```bash
go test ./internal/projectidentity ./internal/toolcatalog -count=1
git diff --check
```

Also validate all modified Markdown links. Run broader tests when documentation changes accompany code, metadata, workflow, packaging, or release behavior.
