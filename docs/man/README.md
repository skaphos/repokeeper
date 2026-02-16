# Manpage Plan

RepoKeeper should ship a generated `man` page for release artifacts and package managers.

## Target

- Main page: `repokeeper(1)`
- Optional subcommand pages later: `repokeeper-status(1)`, `repokeeper-sync(1)`, etc.

## Generation Approach

Use Cobra doc generation (`github.com/spf13/cobra/doc`) from a small generator program that imports the root command and emits manpages into this directory.

Proposed command:

```bash
go run ./cmd/repokeeper-docs man --out docs/man
```

Where:

- `./cmd/repokeeper-docs` is a tiny internal tool that calls Cobra manpage generation.
- `docs/man/repokeeper.1` is committed so release/packaging workflows do not need Go toolchain doc generation at install time.

## Release/CI Integration

1. Add a CI check that regenerates docs and fails on diff.
2. Include `docs/man/*.1` in release artifacts and install packages.
3. Add `man` install instructions in `INSTALL.md`.

## Acceptance Criteria

- `man ./docs/man/repokeeper.1` renders correctly.
- Content reflects current `--help` output.
- Regeneration is deterministic and automated in CI.
