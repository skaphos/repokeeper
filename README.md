# RepoKeeper

A cross-platform multi-repo hygiene tool for developers who work across multiple machines and directory layouts.

RepoKeeper inventories your repositories, reports drift and broken tracking, and performs safe sync actions (fetch/prune) — without touching working trees or submodules.

## Features

- **Discover** repositories across configured root directories (Git by default, optional experimental Hg)
- **Report** per-repo health: dirty/clean, branch, tracking status, ahead/behind, stale upstreams
- **Sync** safely with `git fetch --all --prune` (never checkout/reset; optional `--update-local` uses `pull --rebase`)
- **Registry** is stored in `.repokeeper.yaml` with staleness detection
- **CLI-first** with tabular (`table`/`wide`) and JSON output formats
- **Cross-platform** — macOS, Windows, Linux (incl. WSL)

## Multi-VCS (Experimental)

RepoKeeper is Git-first. The Mercurial (`hg`) adapter is available as an **experimental** backend.

- Default backend remains `git`
- Opt in per command with `--vcs git,hg`
- Mixed roots are auto-detected per repo path when multiple backends are selected

Current experimental limits:

- `hg`: discovery/status and safe `pull`-based fetch are supported
- `hg`: `sync --update-local` (rebase/push/stash flows) is intentionally unsupported and is skipped with a reason
- Repair and remote mismatch reconciliation flows remain Git-oriented

## Install

See [INSTALL.md](INSTALL.md) for full install and upgrade instructions.

### Homebrew (cask)

```bash
brew tap skaphos/tools
brew install --cask skaphos/tools/repokeeper
```

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

# Scan/status across mixed git + hg roots
repokeeper scan --vcs git,hg
repokeeper status --vcs git,hg

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
6. If needed, widen scope for a specific run with `repokeeper scan --roots <dir1,dir2,...>`.

## Commands

Detailed command breakdown moved to docs:

- [docs/commands.md](docs/commands.md) - full command reference, flags, and behavior notes
- [docs/man/README.md](docs/man/README.md) - manpage generation and release integration plan

Quick highlights:

- `repokeeper get` and `repokeeper reconcile` are direct command forms (`... repos` aliases still supported).
- `repokeeper edit <repo-id-or-path>` opens a single repo entry YAML in your editor (`$VISUAL`/`$EDITOR`), validates, then saves.
- `repokeeper label <repo-id-or-path>` manages labels via `--set key=value` and `--remove key`.
- `status`/`get` support label filtering with `-l/--selector` (`key` and `key=value`, comma-separated AND).
- `add` supports metadata on create with `--label` and `--annotation` (repeatable `key=value`).

### Global flags

- `--verbose` / `-v` — increase verbosity (repeatable: `-vv` for debug)
- `--quiet` / `-q` — suppress non-essential output
- `--config <path>` — override config file location
- `--no-color` — disable colored output (also respects `NO_COLOR` env var)
- `--yes` — accept mutating actions without interactive confirmation

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

Flag precedence (highest to lowest):
1. Explicit command flags (`--config`, `--only`, `--field-selector`, `--selector`, etc.)
2. Environment variables where supported (`REPOKEEPER_CONFIG`, `NO_COLOR`)
3. Values loaded from the resolved config file
4. Built-in command defaults

Selector precedence:
1. `--field-selector` when set
2. `--only` when `--field-selector` is not set
3. Providing both in one command is rejected
4. `-l/--selector` is applied as an additional label filter on the resulting repo set

Example config:

```yaml
apiVersion: "skaphos.io/repokeeper/v1beta1"
kind: "RepoKeeperConfig"
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

The default scan/display root is inferred from the directory containing the active config file.

## Safety

RepoKeeper is designed to be safe to run on repos with dirty working trees:

- By default, **never** runs `checkout`, `pull`, `reset`, `rebase`, or `merge`
- **Never** updates or recurses into submodules
- Fetch uses `--no-recurse-submodules` and `-c fetch.recurseSubmodules=false` as belt-and-suspenders
- All mutating commands support `--dry-run`

Optional local checkout update:

- `repokeeper sync --update-local` adds `pull --rebase` after fetch, but only when all of these are true:
- working tree is clean (or `--rebase-dirty` is set)
- branch is not detached
- branch tracks an upstream
- branch is not ahead
- branch is not diverged unless `--force` is set
- branch is not matched by `--protected-branches` (default: none) unless `--allow-protected-rebase` is set
- `--rebase-dirty` stashes changes, rebases, then pops the stash
- `--push-local` pushes local commits when a branch is ahead (instead of skipping with "local commits to push")
- `--continue-on-error` keeps processing all repos after per-repo failures (default true)
- In dry-run/preflight mode, these checks are evaluated up front so the plan calls out which repos are candidates for `fetch + rebase` versus `skip local update (...)`.

## Documentation

- [docs/commands.md](docs/commands.md) - command reference
- [docs/man/README.md](docs/man/README.md) - manpage generation plan
- [DESIGN.md](DESIGN.md) — full design specification and architecture
- [TASKS.md](TASKS.md) — implementation milestones and task tracking
- [CONTRIBUTING.md](CONTRIBUTING.md) — contributor workflow and PR expectations
- [RELEASE.md](RELEASE.md) — release and tagging process
- [INSTALL.md](INSTALL.md) — installation and upgrade paths

## Development

```bash
# Run tests
go tool ginkgo ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run with coverage and enforce per-package thresholds
go tool task test-cover-check

# Run coverage and print lowest-covered packages/functions
go tool task coverage-report

# Run performance benchmarks and append historical record
go tool task perf-bench

# Lint
go tool golangci-lint run ./...

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
