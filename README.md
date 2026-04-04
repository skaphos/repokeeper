# RepoKeeper

A cross-platform multi-repo hygiene tool for developers who work across multiple machines and directory layouts.

RepoKeeper inventories your repositories, reports drift and broken tracking, and performs safe sync actions (remote refresh by default, optional local update when explicitly enabled) — without hidden working-tree mutation or submodule recursion.

## Features

- **Discover** repositories across configured root directories (Git by default, optional experimental Hg)
- **Report** per-repo health: dirty/clean, branch, tracking status, ahead/behind, stale upstreams
- **Sync** safely with fetch/prune-first behavior; optional `--update-local` uses `pull --rebase` under explicit conditions
- **Registry** is stored in `.repokeeper.yaml` with staleness detection
- **Repo-local metadata** can be read from `.repokeeper-repo.yaml` or `repokeeper.yaml` and surfaced in JSON, describe output, and the TUI detail view
- **CLI-first** with tabular (`table`/`wide`) and JSON output formats
- **Cross-platform** — macOS, Windows, Linux (incl. WSL)

## Multi-VCS (Experimental)

RepoKeeper is Git-first. The Mercurial (`hg`) adapter is available as an **experimental** backend.

- Default backend remains `git`
- Opt in per command with `--vcs git,hg`
- Mixed roots are auto-detected per repo path when multiple backends are selected

Current experimental limits:

- `hg`: discovery/status and safe `pull`-based fetch are supported
- `hg`: `reconcile --update-local` (rebase/push/stash flows) is intentionally unsupported and is skipped with a reason
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

# Scan/get across mixed git + hg roots
repokeeper scan --vcs git,hg
repokeeper get --vcs git,hg

# Check the health of all repos
repokeeper get

# Preview repo-local metadata for one tracked repo
repokeeper index github.com/org/repo

# Write repo-local metadata after preview
repokeeper index github.com/org/repo --write

# Fetch/prune all repos safely
repokeeper reconcile
```

## Expected User Flow

1. From the directory you want to manage, run `repokeeper init`.
2. `init` creates `.repokeeper.yaml`, sets that directory as the default root, and performs an initial scan.
3. Run `repokeeper get` to review repo health and identify issues (dirty worktrees, gone upstreams, missing repos).
4. Run `repokeeper reconcile` to safely fetch/prune across registered repos.
5. Re-run `repokeeper scan` whenever clones are added, moved, or removed so the embedded registry stays current.
6. If needed, widen scope for a specific run with `repokeeper scan --roots <dir1,dir2,...>`.

## Commands

Detailed command breakdown moved to docs:

- [docs/commands.md](docs/commands.md) - full command reference, flags, and behavior notes
- [docs/skills/README.md](docs/skills/README.md) - installable agent skill for OpenCode/compatible runtimes
- [docs/man/README.md](docs/man/README.md) - manpage generation and release integration plan

Quick highlights:

- `repokeeper get` and `repokeeper reconcile` are direct command forms (`... repos` aliases still supported).
- `repokeeper edit <repo-id-or-path>` opens a single repo entry YAML in your editor (`$VISUAL`/`$EDITOR`), validates, then saves.
- `repokeeper describe <repo-id-or-path>` accepts plain `repo_id`, `repo_id@checkout_id`, or path selectors; plain `repo_id` now fails when multiple local checkouts exist.
- `repokeeper label <repo-id-or-path>` manages machine-local labels via `--set key=value` and `--remove key`.
- `repokeeper index <repo-id-or-path>` interactively proposes repo-local metadata and writes it only when `--write` is passed.
- `repokeeper index repos --local-selector ... --promote-local-labels --write` explicitly bulk-promotes machine-local labels into repo-local metadata for selected repos.
- Running `repokeeper` with no subcommand launches the interactive TUI (`l` edits repo labels, `i` edits or initializes repo-local metadata from detail view).
- `repokeeper skill install [target]` installs or updates the bundled RepoKeeper skill for supported runtimes.
- `get` supports shared label filtering with `-l/--selector` and machine-local label filtering with `--local-selector` (`key` and `key=value`, comma-separated AND).
- `add` supports metadata on create with `--label` and `--annotation` (repeatable `key=value`).

The bundled skill is embedded in the compiled RepoKeeper binary, so `repokeeper skill install` works from packaged builds such as Homebrew installs.

### MCP Server (Agent Integration)

RepoKeeper includes a built-in [MCP](https://modelcontextprotocol.io/) server for agent runtimes that support the Model Context Protocol (Claude Code, Cursor, Windsurf, etc.). MCP is the preferred integration path for inspection and planning workflows — it provides typed tool schemas, structured JSON responses, and automatic tool discovery.

```json
{
  "mcpServers": {
    "repokeeper": {
      "command": "repokeeper",
      "args": ["mcp"]
    }
  }
}
```

The MCP server is intended as a read-and-plan surface with browsable resources. CLI and TUI remain the execution surfaces for state-changing operations. See [docs/mcp-setup.md](docs/mcp-setup.md) for per-runtime setup instructions and the current tool reference.

### Repo-local metadata

RepoKeeper can read optional repo-root metadata from either `.repokeeper-repo.yaml` or `repokeeper.yaml`.

- `.repokeeper.yaml` remains the machine-local workspace config and registry.
- `.repokeeper-repo.yaml` is the source-controlled repo metadata surface.

- Reads are automatic and read-only in `scan`, `get`, `describe`, and the TUI list/detail views.
- `label` remains machine-local only.
- `index --promote-local-labels --write` explicitly promotes machine-local labels into shared repo metadata.
- `index repos --selector ...` / `--local-selector ...` previews or writes that promotion across multiple selected repos.
- Read commands cache repo-metadata snapshots in the machine-local registry and refresh them when the on-disk metadata state changes.
- `--yes` skips the final write confirmation, but does not change the requirement to pass `--write`.
- Existing `repo_metadata.labels` win on key conflicts; promoted local labels only fill missing keys.

RepoKeeper keeps two identity layers:

- `repo_id`: stable upstream identity, used as the cross-machine join key.
- `checkout_id`: machine-local checkout identity, used to distinguish multiple local clones of the same `repo_id`.

Example:

```yaml
apiVersion: repokeeper/v1
kind: RepoMetadata
repo_id: ops-runbooks
name: Ops Runbooks
labels:
  role: runbooks
  domain: ops
entrypoints:
  readme: README.md
paths:
  authoritative:
    - runbooks/
    - templates/
  low_value:
    - generated/
provides:
  - runbook-templates
related_repos:
  - repo_id: internal-docs
    relationship: references
```

### Global flags

- `--verbose` / `-v` — increase verbosity (repeatable: `-vv` for debug)
- `--quiet` / `-q` — suppress non-essential output
- `--config <path>` — override config file location
- `--no-color` — disable colored output (also respects `NO_COLOR` env var)
- `--yes` — accept mutating actions without interactive confirmation

## Configuration

By default, `repokeeper init` writes `.repokeeper.yaml` in your current directory.

Runtime commands (`scan`, `get`, `reconcile`) resolve config in this order:

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
4. `-l/--selector` is applied as an additional shared-label filter on the resulting repo set
5. `--local-selector` is applied as an additional machine-local label filter on the resulting repo set

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

- By default, sync does **not** run checkout, reset, merge, or other hidden branch-navigation behavior
- **Never** updates or recurses into submodules
- Fetch uses `--no-recurse-submodules` and `-c fetch.recurseSubmodules=false` as belt-and-suspenders
- All mutating commands support `--dry-run`
- `scan`, `get`, `describe`, `add`, and `import` do not create or rewrite repo-local metadata files
- `label` and `edit` do not write repo-local metadata files
- `index --write` and the TUI metadata editor (`i` from detail view) are explicit repo-local metadata write paths

Optional local checkout update:

- `repokeeper reconcile --update-local` adds `pull --rebase` after fetch, but only when all of these are true:
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

Branch switching and prune execution are separate workflow areas rather than hidden sync side effects.

## Documentation

- [docs/commands.md](docs/commands.md) - command reference
- [docs/mcp-setup.md](docs/mcp-setup.md) - MCP server setup for agent runtimes
- [docs/skills/README.md](docs/skills/README.md) - user-scope agent skill installation and usage
- [docs/man/README.md](docs/man/README.md) - manpage generation plan
- [DESIGN.md](DESIGN.md) — full design specification and architecture
- [TASKS.md](TASKS.md) — implementation milestones and task tracking
- [CONTRIBUTING.md](CONTRIBUTING.md) — contributor workflow and PR expectations
- [RELEASE.md](RELEASE.md) — release and tagging process
- [INSTALL.md](INSTALL.md) — installation and upgrade paths

## Development

```bash
# List all task targets without installing task globally
go -C tools tool task --list

# Run tests
go run github.com/onsi/ginkgo/v2/ginkgo@v2.28.1 ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run with coverage and enforce per-package thresholds
go -C tools tool task test-cover-check

# Run coverage and print lowest-covered packages/functions
go -C tools tool task coverage-report

# Run performance benchmarks and append historical record
go -C tools tool task perf-bench

# Lint
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...

# Build locally
go build -o repokeeper .

# Build locally (task runner)
go -C tools tool task build

# CI-style full platform build (task runner)
go -C tools tool task build-ci

# Run standard CI pipeline locally (lint/test/staticcheck/vuln/build-ci)
go -C tools tool task ci

# Format imports + code
go -C tools tool task fmt
```

## License

[MIT](LICENSE)

## Third-Party Notices

RepoKeeper includes third-party open source components.

- Runtime dependency notices for distributed binaries live in [`third_party_licenses/runtime/`](third_party_licenses/runtime/) with an inventory in [`third_party_licenses/runtime-report.csv`](third_party_licenses/runtime-report.csv).
- The maintenance workflow and attribution expectations live in [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

When dependencies change, regenerate the notice artifacts before opening a PR:

```bash
go -C tools tool task notices
```
