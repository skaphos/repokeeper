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

| Command | Description |
|---------|-------------|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper status` | Report repo health summary (path, branch, dirty, tracking) |
| `repokeeper get repos` | Kubectl-style alias for status/list view |
| `repokeeper describe <repo-id-or-path>` | Show detailed status for one repository |
| `repokeeper describe repo <repo-id-or-path>` | Kubectl-style describe form for a single repository |
| `repokeeper add <path> <git-repo-url>` | Clone and register a repository (`--branch` or `--mirror`) |
| `repokeeper delete <repo-id-or-path>` | Remove a repository from the registry |
| `repokeeper edit <repo-id-or-path>` | Update repo metadata/tracking (`--set-upstream`) |
| `repokeeper repair-upstream` | Repair missing/mismatched upstream tracking across registered repos |
| `repokeeper repair upstream` | Kubectl-style alias for upstream repair |
| `repokeeper sync` | Fetch and prune all repos safely |
| `repokeeper reconcile repos` | Kubectl-style alias for sync/reconciliation |
| `repokeeper export` | Export config (and registry) for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

`repokeeper sync --format table` shows `PATH`, a summarized `ACTION` (`fetch`, `fetch + rebase`, `skip ...`), status context (`BRANCH`, `DIRTY`, `TRACKING`), outcome (`OK`, `ERROR_CLASS`, `ERROR`), and `REPO` as the trailing identifier column.
Use `-o wide` (or `--format wide`) on `status`/`get repos` and `sync`/`reconcile repos` for additional remote/upstream/ahead/behind context.
`repokeeper sync` shows a preflight plan. Confirmation is requested only when the plan includes local-branch-changing actions (`pull --rebase`/stash+rebase) or checkout-missing clones; fetch-only plans run without a prompt. Use `--yes` to skip confirmation when it is required.
`repokeeper sync` supports `--only diverged` and `--only remote-mismatch` for targeted remediation runs.
`repokeeper status --only diverged` now includes a diverged reason and recommended action in table output, and adds a machine-readable `diverged` guidance array in JSON output.
`repokeeper status --only remote-mismatch --reconcile-remote-mismatch registry|git --dry-run=false` can explicitly reconcile mismatched remotes by updating either registry `remote_url` values or live git remote URLs. Apply mode prompts unless `--yes` is passed.
For `status`, `--dry-run` defaults to `true` and only affects remote-mismatch reconcile actions (preview vs apply); regular status reporting itself never mutates repos.

`repokeeper describe` and `repokeeper describe repo` both accept a repo ID, a path relative to your current working directory, or a path relative to the directory containing `.repokeeper.yaml`.

`repokeeper add` accepts `--branch <name>` for a single-branch checkout clone or `--mirror` for a full mirror clone (bare, no working tree). Mirror repos are tracked and shown in status as `TRACKING=mirror`.

`repokeeper edit <repo-id-or-path> --set-upstream <remote/branch>` updates both local git tracking and the registry metadata branch.

`repokeeper export --output -` and `repokeeper import` support shell piping (for example: `repokeeper import < repokeeper-export.yaml`).

`repokeeper import` defaults to merge mode (`--mode=merge`) so existing local config/registry can be synchronized with an exported bundle without overwrite. Use `--mode=replace --force` for the previous full-replace behavior.

In merge mode, conflicts on the same `repo_id` can be resolved with `--on-conflict skip|bundle|local` (default `bundle`).

`repokeeper import` does not clone by default (`--clone=false`). Use `--clone` to clone imported entries into the current directory layout. In merge mode against an existing config, clone only applies to merge-selected bundle entries (new repos and bundle-wins conflicts). If a target repo path already exists, import reports conflicting paths unless `--dangerously-delete-existing` is set.

Use `repokeeper import --file-only` to import only the config file without registry data or cloning.

Use `repokeeper sync --checkout-missing` to clone registry entries currently marked missing (using their `remote_url`, `branch`, and mirror type).

Use `repokeeper repair-upstream --dry-run` to preview upstream tracking fixes, then `repokeeper repair-upstream --dry-run=false` to apply. Use `--only missing` or `--only mismatch` to focus the repair set.
When `repair-upstream --dry-run=false` would modify tracking, RepoKeeper prompts for confirmation by default; use `--yes` for non-interactive runs.

`scan`, `status`, and `sync` accept `--vcs git,hg` (default `git`) to choose one or more repository backends.

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
1. Explicit command flags (`--config`, `--only`, `--field-selector`, etc.)
2. Environment variables where supported (`REPOKEEPER_CONFIG`, `NO_COLOR`)
3. Values loaded from the resolved config file
4. Built-in command defaults

Selector precedence:
1. `--field-selector` when set
2. `--only` when `--field-selector` is not set
3. Providing both in one command is rejected

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
- branch tracks `*/main`
- branch is not ahead
- branch is not diverged unless `--force` is set
- branch is not matched by `--protected-branches` (default: none) unless `--allow-protected-rebase` is set
- `--rebase-dirty` stashes changes, rebases, then pops the stash
- `--push-local` pushes local commits when a branch is ahead (instead of skipping with "local commits to push")
- `--continue-on-error` keeps processing all repos after per-repo failures (default true)
- In dry-run/preflight mode, these checks are evaluated up front so the plan calls out which repos are candidates for `fetch + rebase` versus `skip local update (...)`.

## Documentation

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
