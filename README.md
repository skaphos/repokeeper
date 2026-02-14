# RepoKeeper

A cross-platform multi-repo hygiene tool for developers who work across multiple machines and directory layouts.

RepoKeeper inventories your git repos, reports drift and broken tracking, and performs safe sync actions (fetch/prune) — without touching working trees or submodules.

## Features

- **Discover** git repos across configured root directories
- **Report** per-repo health: dirty/clean, branch, tracking status, ahead/behind, stale upstreams
- **Sync** safely with `git fetch --all --prune` (never checkout, pull, reset, or touch submodules)
- **Registry** is stored in `.repokeeper.yaml` with staleness detection
- **CLI-first** with table and JSON output formats
- **Cross-platform** — macOS, Windows, Linux (incl. WSL)

## Install

### From release binaries

Download the latest release from the [Releases](https://github.com/skaphos/repokeeper/releases) page.

### From source

```bash
go install github.com/skaphos/repokeeper@latest
```

## Quick Start

```bash
# Bootstrap config and run initial scan for this directory
repokeeper init

# Scan your roots for git repos
repokeeper scan

# Check the health of all repos
repokeeper status

# Fetch/prune all repos safely
repokeeper sync
```

## Expected User Flow

1. From the directory you want to manage, run `repokeeper init`.
2. `init` creates `.repokeeper.yaml`, sets that directory as the default root, and performs an initial scan.
3. Run `repokeeper status` to review repo health and identify issues (dirty worktrees, gone upstreams, missing repos).
4. Run `repokeeper sync` to safely fetch/prune across registered repos.
5. Re-run `repokeeper scan` whenever clones are added, moved, or removed so the embedded registry stays current.
6. If needed, widen scope with `repokeeper scan --roots <dir1,dir2,...>` and keep those roots in `.repokeeper.yaml`.

## Commands

| Command | Description |
|---------|-------------|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper status` | Report repo health summary (path, branch, dirty, tracking) |
| `repokeeper describe <repo-id-or-path>` | Show detailed status for one repository |
| `repokeeper add <path> <git-repo-url>` | Clone and register a repository (`--branch` or `--mirror`) |
| `repokeeper delete <repo-id-or-path>` | Remove a repository from the registry |
| `repokeeper sync` | Fetch and prune all repos safely |
| `repokeeper export` | Export config (and registry) for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

`repokeeper sync --format table` mirrors the `status` columns (`PATH`, `BRANCH`, `DIRTY`, `TRACKING`) and appends sync outcome columns (`OK`, `ERROR_CLASS`, `ERROR`, `ACTION`).

`repokeeper describe` accepts a repo ID, a path relative to your current working directory, or a path relative to configured roots.

`repokeeper add` accepts `--branch <name>` for a single-branch checkout clone or `--mirror` for a full mirror clone (bare, no working tree). Mirror repos are tracked and shown in status as `TRACKING=mirror`.

`repokeeper export --output -` and `repokeeper import --input -` support shell piping.

`repokeeper import` now clones repos from the imported registry by default (`--clone=true`). Run imports in a blank directory. If a target repo path already exists, import fails unless `--dangerously-delete-existing` is set, which removes conflicting target paths before cloning.

Use `repokeeper import --file-only` to import only the config file without registry data or cloning.

Use `repokeeper sync --checkout-missing` to clone registry entries currently marked missing (using their `remote_url`, `branch`, and mirror type).

### Global flags

- `--verbose` / `-v` — increase verbosity (repeatable: `-vv` for debug)
- `--quiet` / `-q` — suppress non-essential output
- `--config <path>` — override config file location
- `--no-color` — disable colored output (also respects `NO_COLOR` env var)

## Configuration

By default, `repokeeper init` writes `.repokeeper.yaml` in your current directory.

Runtime commands (`scan`, `status`, `sync`) resolve config in this order:

1. `--config <path>`
2. `REPOKEEPER_CONFIG` environment variable
3. Nearest `.repokeeper.yaml` found by walking upward from the current directory
4. Global fallback in platform config dir:
- Linux: `$XDG_CONFIG_HOME/repokeeper/config.yaml` (default `~/.config/repokeeper/config.yaml`)
- macOS: `~/Library/Application Support/repokeeper/config.yaml`
- Windows: `%APPDATA%\\repokeeper\\config.yaml`

Example config:

```yaml
roots:
  - "/home/user/code"
  - "/home/user/work"
exclude:
  - "**/node_modules/**"
  - "**/.terraform/**"
  - "**/vendor/**"
registry:
  updated_at: "2026-02-14T10:00:00Z"
  repos: []
defaults:
  concurrency: 8
  timeout_seconds: 60
```

## Safety

RepoKeeper is designed to be safe to run on repos with dirty working trees:

- **Never** runs `checkout`, `pull`, `reset`, `rebase`, or `merge`
- **Never** updates or recurses into submodules
- Fetch uses `--no-recurse-submodules` and `-c fetch.recurseSubmodules=false` as belt-and-suspenders
- All mutating commands support `--dry-run`

Optional local checkout update:

- `repokeeper sync --update-local` adds `pull --rebase` after fetch, but only when all of these are true:
- working tree is clean
- branch is not detached
- branch tracks `*/main`
- branch is not ahead/diverged (no local commits pending push)

## Documentation

- [DESIGN.md](DESIGN.md) — full design specification and architecture
- [TASKS.md](TASKS.md) — implementation milestones and task tracking
- [CONTRIBUTING.md](CONTRIBUTING.md) — contributor workflow and PR expectations
- [RELEASE.md](RELEASE.md) — release and tagging process

## Development

```bash
# Run tests
ginkgo ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Lint
golangci-lint run ./...

# Build locally
go build -o repokeeper .

# Build locally (task runner)
go tool task build

# CI-style full platform build (task runner)
go tool task build-ci

# Run standard CI pipeline locally (lint/test/staticcheck/vuln/build-ci)
go tool task ci

# Format imports + code
go tool task fmt

# List all task targets
go tool task --list
```

## License

[MIT](LICENSE)
