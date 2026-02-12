# RepoKeeper

A cross-platform multi-repo hygiene tool for developers who work across multiple machines and directory layouts.

RepoKeeper inventories your git repos, reports drift and broken tracking, and performs safe sync actions (fetch/prune) — without touching working trees or submodules.

## Features

- **Discover** git repos across configured root directories
- **Report** per-repo health: dirty/clean, branch, tracking status, ahead/behind, stale upstreams
- **Sync** safely with `git fetch --all --prune` (never checkout, pull, reset, or touch submodules)
- **Registry** tracks repos per-machine with staleness detection
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
# Bootstrap config for this directory
repokeeper init

# Scan your roots for git repos
repokeeper scan

# Check the health of all repos
repokeeper status

# Fetch/prune all repos safely
repokeeper sync
```

## Commands

| Command | Description |
|---------|-------------|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper status` | Report repo health (dirty, tracking, ahead/behind) |
| `repokeeper sync` | Fetch and prune all repos safely |
| `repokeeper export` | Export config (and registry) for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

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
machine_id: "my-laptop"
roots:
  - "/home/user/code"
  - "/home/user/work"
exclude:
  - "**/node_modules/**"
  - "**/.terraform/**"
  - "**/vendor/**"
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

# Snapshot build (all platforms)
goreleaser build --snapshot --clean
```

## License

[MIT](LICENSE)
