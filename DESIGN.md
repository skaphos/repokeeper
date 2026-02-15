# RepoKeeper — Design Spec

## 1. Summary

**RepoKeeper** is a cross-platform multi-repo hygiene tool for developers who work across multiple machines and directory layouts. It inventories git repos, reports drift and broken tracking, and performs safe sync actions (fetch/prune) without touching working trees or submodules.

Primary interfaces:

* **CLI** (scriptable, automation-friendly)
* **TUI** (Bubble Tea) for interactive workflows when a "GUI-like" experience is desirable in terminal environments ([GitHub][1])

Key safety rule: **never update submodules** (no recursion).

## 2. Problem Statement

You have many git repos across **macOS, Windows, Linux (incl. WSL/VM)** with:

* different directory layouts per machine
* inconsistent sets of repos per machine
* repos out of sync with remotes
* stale remote-tracking branches due to deleted PR branches
* local branches tracking remotes that no longer exist ("upstream gone")

You want "bring everything back in line" behavior **en masse**, but:

* the working tree may be dirty
* you sometimes need CLI-only operation
* you want a terminal UI when available
* you must avoid touching submodules

## 3. Goals

### 3.1 Functional goals (v1 / "80%")

1. **Discover** repos under configured roots.
2. **Identify** repos consistently across machines (remote URL normalization).
3. **Report** per-repo state:

    * dirty/clean
    * current branch / detached HEAD
    * remote URL(s), presence of `origin`
    * upstream tracking status (including "gone")
    * ahead/behind relative to upstream (where possible)
    * has submodules (presence detection only)
4. **Sync** safely:

    * run `git fetch --all --prune --prune-tags` with submodule recursion disabled
    * do not checkout branches, do not reset, do not pull
5. **Persist per-machine registry** mapping repo-id → local path.
6. **Output**:

    * human-readable table
    * machine-readable JSON

### 3.2 Non-goals (defer)

* Auto-clone or credential management.
* Auto-updating `main`/`master` (beyond fetching remote refs).
* Auto-deleting local branches (even if merged).
* Desktop GUI.
* Multi-VCS support beyond Git (see Stretch Goals §9.1).

## 4. Safety & Policy

### 4.1 Submodules: do not update

RepoKeeper **must not** update submodules and must defend against user git configs that enable recursive submodule fetching.

Implementation requirements:

* Never run `git submodule …`.
* Always fetch with recursion disabled using either:

    * `git fetch --no-recurse-submodules …` ([Git][2])
    * and/or `git -c fetch.recurseSubmodules=false fetch …` (belt-and-suspenders)

### 4.2 Working tree safety

* Fetch/prune is allowed even if dirty (it does not modify working tree files).
* RepoKeeper must **not**:

    * `checkout`, `pull`, `reset`, `rebase`, `merge` in v1.

### 4.3 Dry-run

All commands that can change repo metadata must support `--dry-run` (prints intended operations).

## 5. User Experience

### 5.1 CLI commands (v1)

#### Global flags (apply to all commands)

* `--verbose` / `-v` — increase output verbosity (show per-repo git commands being run, timing info). Repeatable (`-vv` for debug-level).
* `--quiet` / `-q` — suppress non-essential output; only errors and requested data.
* `--config <path>` — override config file location (default resolution: nearest local `.repokeeper.yaml`, then platform config dir fallback; see §6.2.1).
* `--no-color` — disable colored output (also respected via `NO_COLOR` env var).

#### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success — all operations completed, no issues found |
| 1 | Warnings — operations completed but some repos have issues (dirty, gone upstreams, etc.) |
| 2 | Errors — one or more operations failed (network, auth, corrupt repo, etc.) |
| 3 | Fatal — RepoKeeper itself could not run (bad config, missing git binary, etc.) |

Commands that produce status output (`status`, `scan`) use exit code 1 when any repo has a warning-level condition, making them useful in CI/scripts (`repokeeper status && echo "all clean"`).

#### `repokeeper version`

Prints the RepoKeeper version, Go version, and build metadata (commit SHA, build date).

#### `repokeeper init`

Bootstrap that creates a RepoKeeper config file.

Behavior:

* Uses the config file directory as the default root.
* Uses default exclude patterns (`node_modules`, `.terraform`, `dist`, `vendor`).
* Runs an initial scan immediately after writing config.
* Writes `.repokeeper.yaml` in the current working directory by default (see §6.2.1).
* If a config already exists, errors unless `--force` is provided.

Flags:

* `--force` — overwrite existing config without prompting

#### `repokeeper scan`

Scans roots for git repos and updates local registry.

Flags:

* `--roots <comma-separated>`
* `--exclude <comma-separated globs>` (e.g., `node_modules,.terraform`)
* `--follow-symlinks` (default false)
* `--write-registry` (default true)
* `-o, --format table|json` (default table)

#### `repokeeper status`

Reads registry (and optionally scans roots) and prints repo health.

Flags:

* `--roots …` (optional)
* `--registry <path>` (optional)
* `-o, --format table|wide|json` (default table)
* `--only errors|dirty|clean|gone|diverged|remote-mismatch|missing|all` (default all)

#### `repokeeper describe <repo-id-or-path>`

Alias form: `repokeeper describe repo <repo-id-or-path>`

Shows detailed status for a single repo selected by repo ID, cwd-relative path, or config-root-relative path.

Flags:

* `--registry <path>` (optional)
* `-o, --format table|json` (default table)

#### `repokeeper add <path> <git-repo-url>`

Clones and registers a repository entry.

Flags:

* `--branch <name>` (optional; checkout clone)
* `--mirror` (optional; full mirror clone, bare with no working tree)
* `--registry <path>` (optional)

#### `repokeeper delete <repo-id-or-path>`

Removes a repository entry from the registry.

Flags:

* `--registry <path>` (optional)

#### `repokeeper edit <repo-id-or-path>`

Updates per-repo metadata and tracking.

Flags:

* `--set-upstream <remote/branch>` (required; updates git branch upstream and registry branch metadata)
* `--registry <path>` (optional)

#### `repokeeper sync`

Runs safe fetch/prune on repos (all or selected).
Shows a preflight plan and prompts for confirmation before executing unless `--yes` is passed.

Flags:

* `--only errors|dirty|clean|gone|diverged|remote-mismatch|missing|all`
* `--concurrency <n>` (default: min(8, CPU))
* `--timeout <duration>` (default 60s/repo)
* `--continue-on-error` (default true; continue syncing remaining repos after per-repo failures)
* `--dry-run`
* `--yes` (skip confirmation prompt and execute immediately)
* `--update-local` (optional; after fetch, run local branch updates based on tracking state)
* `--push-local` (optional; when branch is ahead, run `git push` instead of skipping)
* `--rebase-dirty` (optional; stash, rebase, then pop for dirty worktrees)
* `--force` (optional; allow rebase when branch is diverged)
* `--protected-branches` (default `main,master,release/*`; block auto-rebase on matching branches)
* `--allow-protected-rebase` (optional; override protected branch safeguard)
* `--checkout-missing` (optional; clone repos marked missing from registry metadata)
* `-o, --format table|wide|json`

#### `repokeeper repair-upstream`

Alias form: `repokeeper repair upstream`

Inspects registered repositories for missing or mismatched upstream tracking and optionally repairs them.

Flags:

* `--registry <path>` (optional)
* `--dry-run` (default true; preview only)
* `--only all|missing|mismatch`
* `-o, --format table|json`

#### `repokeeper export`

Exports config and optional registry into a single YAML bundle for migration.

Flags:

* `--output <path|->` (default `repokeeper-export.yaml`)
* `--include-registry` (default true)

#### `repokeeper import`

Imports a previously exported YAML bundle.

Flags:

* Positional bundle input: `<path>` or `-` (stdin). When omitted, import reads stdin.
* `--force` (overwrite existing config)
* `--include-registry` (default true)
* `--preserve-registry-path` (default false; by default registry path is rewritten beside imported config)
* `--clone` (default true; clone imported registry repos into current directory)
* `--dangerously-delete-existing` (dangerous; delete existing target paths before clone)
* `--file-only` (config only; disables registry import and cloning)

### 5.2 TUI command (phase 2)

#### `repokeeper tui`

Interactive terminal UI built with Bubble Tea + Bubbles + Lipgloss ([GitHub][1]).

**Design principle:** The TUI should follow a **k9s-style interaction model**: one primary repo list, keyboard-first filtering/navigation, and contextual actions on selected rows. Keep it as a thin presentation layer over existing engine APIs.

Core interaction model:

* Primary, filterable repo list as the default view.
* `/` enters filter mode; filter by repo id, path, branch, tracking state, and error class.
* Arrow keys / `j` / `k` navigate rows; `enter` opens a repo detail/action view.
* `space` toggles selection, `a` selects all visible rows.
* Action keys trigger repo operations from the list or detail view (sync, edit metadata, repair upstream, open path).
* Batch actions operate on current selection and stream progress in-place.

Non-goals for TUI:

* No independent business logic; all operations route through engine/CLI primitives.
* No mandatory mouse support.
* No complex plugin system in milestone 5.

> Note: TUI is a frontend; it must call the same core engine APIs as CLI. All business logic lives in `internal/engine/`.

### 5.3 Kubectl-Style CLI Alignment (Milestone 6+)

RepoKeeper CLI should align with common `kubectl` conventions where practical, with one intentional delta: preserve colorized table output when terminal capability allows.

#### 5.3.1 Command shape

Target command grammar (additive, backwards-compatible aliases during migration):

* `repokeeper get repos` (status/list view)
* `repokeeper describe repo <repo-id-or-path>`
* `repokeeper edit repo <repo-id-or-path>`
* `repokeeper delete repo <repo-id-or-path>`
* `repokeeper reconcile repos` (sync/reconciliation workflows)
* `repokeeper repair upstream` (tracking repair workflows)

Existing commands (`status`, `sync`, `repair-upstream`, etc.) remain supported during migration and should map to the new internal actions.

#### 5.3.2 Output contracts

List-style commands (`get`, `reconcile`, `repair`) should converge on:

* `-o table|wide|json|yaml|name` (phase 1: `table|json`, phase 2 extends format set)
* `--no-headers`
* stable sorting and deterministic rows
* explicit per-row machine outcome fields for automation (example: `outcome=fetched|rebased|pushed|skipped_*|failed_*`)

Table baseline for repos:

* `NAME` (repo id short name), `PATH`, `BRANCH`, `TRACKING`, `DIRTY`, `STATUS`

`-o wide` extends with:

* `PRIMARY_REMOTE`, `UPSTREAM`, `AHEAD`, `BEHIND`, `ERROR_CLASS`

#### 5.3.3 Styling and color policy (intentional delta vs kubectl)

RepoKeeper should keep color by default for human table output when:

* stdout is a TTY,
* terminal supports ANSI,
* `--no-color` is not set,
* `NO_COLOR` is not set.

RepoKeeper should suppress color for machine-focused output (`json`, `yaml`, `name`) regardless of TTY.

Recommended semantic colors:

* green: healthy (`up to date`, success outcomes),
* yellow: warnings/action-needed (`dirty`, non-fatal skips),
* red: failures/errors (`diverged`, operation failures),
* blue: informational states (`mirror`, local-only).

#### 5.3.4 Filter and selector direction

Current `--only` filters remain supported.
Future selector syntax should move toward kubectl-like field filtering semantics (for example, `--field-selector tracking.status=diverged`), with `--only` retained as shorthand aliases.

#### 5.3.5 Migration strategy

1. Add new kubectl-style command aliases and shared output primitives.
2. Move documentation/examples to new command forms.
3. Keep old forms available through at least one minor release cycle.
4. Emit deprecation notices only after parity is achieved.

## 6. Data Model

### 6.1 Repo identity

RepoKeeper uses a stable `repo_id` based on normalized remote URL, preferably `origin`.

Normalization rules (examples):

* Strip protocol/user:

    * `git@github.com:Org/Repo.git` → `github.com/Org/Repo`
    * `https://github.com/Org/Repo.git` → `github.com/Org/Repo`
* Lowercase host; preserve path case (GitHub is case-insensitive, but be conservative).
* Strip trailing `.git`.
* Strip trailing slashes.

### 6.2 Config files

#### 6.2.1 Machine config

Config path is resolved in this order for runtime commands (`scan`, `status`, `sync`):

1. `--config` flag (if provided).
2. `REPOKEEPER_CONFIG` environment variable (if set).
3. Nearest `.repokeeper.yaml` in current directory or any parent directory.
4. Platform default:

| Platform | Default path |
|----------|-------------|
| Linux | `$XDG_CONFIG_HOME/repokeeper/config.yaml` (falls back to `~/.config/repokeeper/config.yaml`) |
| macOS | `~/Library/Application Support/repokeeper/config.yaml` (falls back to `~/.config/repokeeper/config.yaml`) |
| Windows | `%APPDATA%\\repokeeper\\config.yaml` |

`repokeeper init` resolves write target in this order:

1. `--config` flag (if provided).
2. `REPOKEEPER_CONFIG` environment variable (if set).
3. Current directory: `.repokeeper.yaml`.

The registry is embedded directly in `.repokeeper.yaml` under the `registry` key. For backward compatibility, older configs may still contain `registry_path`.

Implementation: use Go's `os.UserConfigDir()` as the base, which already returns the correct platform directory.

```yaml
apiVersion: "skaphos.io/repokeeper/v1beta1"
kind: "RepoKeeperConfig"
exclude:
  - "**/node_modules/**"
  - "**/.terraform/**"
  - "**/dist/**"
registry:
  updated_at: "2026-02-10T16:00:00-06:00"
  repos: []
defaults:
  remote_name: "origin"
  concurrency: 8
  timeout_seconds: 60
```

The effective default root is the directory containing the active config file.

#### 6.2.2 Registry (embedded in machine config by default)

Per-machine mapping of repo-id to local path.

```yaml
updated_at: "2026-02-10T16:00:00-06:00"
repos:
  - repo_id: "github.com/alaskaairlines/sdp-foo"
    path: "/Users/shawn/code/sdp-foo"
    remote_url: "git@github.com:alaskaairlines/sdp-foo.git"
    type: "checkout"    # checkout | mirror
    branch: "main"      # optional preferred branch for checkout clones
    last_seen: "2026-02-10T16:00:00-06:00"
    status: "present"   # present | missing | moved
```

**Registry staleness detection:**

During `scan`, RepoKeeper validates every existing registry entry:

* If the path still exists and contains a valid git repo → update `last_seen`, mark `present`.
* If the path no longer exists on disk → mark `missing`. The entry is **retained** (not deleted) so users can see what disappeared.
* If the same `repo_id` is found at a different path → mark the old entry `moved` and update the path.

`repokeeper status` surfaces missing/moved repos so the user can act:

* `--only missing` — show only repos whose paths no longer exist.
* Missing repos older than a configurable threshold (default: 30 days, `registry_stale_days` in config) can be auto-pruned with `repokeeper scan --prune-stale`.

*(Optional future)* Global manifest for cross-machine "missing repos" reconciliation.

### 6.3 Status JSON schema (v1)

Top-level:

```json
{
  "generated_at": "…",
  "repos": [
    {
      "repo_id": "github.com/org/repo",
      "path": "…",
      "bare": false,
      "remotes": [
        { "name": "origin", "url": "git@github.com:org/repo.git" },
        { "name": "upstream", "url": "git@github.com:upstream-org/repo.git" }
      ],
      "primary_remote": "origin",
      "head": { "branch": "main", "detached": false },
      "worktree": { "dirty": true, "staged": 1, "unstaged": 2, "untracked": 0 },
      "tracking": {
        "upstream": "origin/main",
        "status": "ahead|behind|diverged|equal|gone|none",
        "ahead": 2,
        "behind": 0
      },
      "submodules": { "has_submodules": true },
      "last_sync": { "ok": true, "at": "…", "error": "" }
    }
  ]
}
```

Field notes:

* **`bare`** — `true` for bare repos. When bare, `worktree` is `null` (no working tree to inspect).
* **`remotes`** — all configured remotes for the repo, not just one. The `primary_remote` field indicates which remote was used for `repo_id` derivation.
* **`primary_remote`** — preference order: `origin` > first alphabetically. Used for repo identity and tracking status.
* **`tracking.ahead`** / **`tracking.behind`** — integer counts. Both `0` when `status` is `"equal"`. Both `null` when `status` is `"gone"` or `"none"` (no upstream to compare against).

## 7. Git Operations (Engine Contract)

RepoKeeper shells out to the installed `git` binary for parity with real-world behavior. Use Go git libraries only when the CLI is a poor fit (performance, missing capability, or brittle parsing), and document any such fallback.

### 7.0 Git CLI Strategy & Compatibility Matrix

We maintain a lightweight compatibility matrix to track minimum supported and tested Git versions across platforms. Update this table when changing git invocation behavior or adding CLI flags.

| Platform | Minimum Supported | Tested In CI | Notes |
| --- | --- | --- | --- |
| Linux | TBD | TBD | Fill in once CI pins a Git version. |
| macOS | TBD | TBD | Fill in once CI pins a Git version. |
| Windows | TBD | TBD | Fill in once CI pins a Git version. |

## 9. Stretch Goals

### 9.1 Multi-VCS Support (Hg/Bzr)

RepoKeeper is **Git-first**, but the architecture should stay adapter-friendly so future Mercurial (hg) and Bazaar (bzr) support doesn’t require a rewrite. This affects layout decisions early:

* Keep core orchestration in `internal/engine/` and add a thin VCS adapter interface (discover/status/sync).
* Implement Git as the default adapter; treat Hg/Bzr as optional, stretch targets.
* Maintain a per-VCS compatibility matrix (minimum supported + tested tool versions).
* Keep CLI flags extensible (example: `--vcs git,hg,bzr`) without changing defaults.

### 7.1 Detection commands

* **Verify repo:** `git rev-parse --is-inside-work-tree`
* **Detect bare repo:** `git rev-parse --is-bare-repository` — returns `true` for bare repos.
* **Determine git dir:** `git rev-parse --git-dir`
* **List all remotes:** `git remote` — enumerate all configured remotes.
* **Remote URL (per remote):** `git remote get-url <name>` — called for each remote. Primary remote selection: prefer `origin`, fall back to first remote alphabetically.
* **Dirty state:** `git status --porcelain=v1` — **skip for bare repos** (no working tree).
* **Current branch:** `git symbolic-ref --quiet --short HEAD` (if fails → detached) — **skip for bare repos**.
* **Submodule presence** (no recursion):

    * check file `.gitmodules` exists AND has at least one `submodule.*.path` entry:

        * `git config --file .gitmodules --get-regexp ^submodule\\..*\\.path$`

#### Bare repo handling

Bare repos (cloned with `--bare`) have no working tree. RepoKeeper detects them via `git rev-parse --is-bare-repository` and adjusts behavior:

* `worktree` in status output is `null`.
* `head.branch` / `head.detached` are still reported (bare repos have HEAD).
* Fetch/prune operates normally on bare repos.
* Bare repos are flagged in table output with a `[bare]` indicator.

### 7.2 Tracking status (preferred)

Use `git for-each-ref` to get upstream + ahead/behind + gone status in one pass.

Example:

* `git for-each-ref refs/heads --format="%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort)"`

To get exact ahead/behind counts (when upstream exists and is not gone):

* `git rev-list --left-right --count <branch>...@{upstream}`

Notes:

* `%(upstream:track)` can emit `"[gone]"` when the upstream ref is missing ([Git][3])
* This is the cleanest way to detect "dangling upstream"

### 7.3 Sync operation (core action)

Per repo:

* `git fetch --all --prune --prune-tags --no-recurse-submodules`

`--no-recurse-submodules` explicitly disables recursive fetching of submodules ([Git][2])

Optional additional defense:

* `git -c fetch.recurseSubmodules=false fetch …`

### 7.4 Error handling guidelines

* Capture combined stdout/stderr.
* Classify errors:

    * auth/permission
    * remote missing
    * not a repo / corrupted repo
    * network/timeouts
* Persist last error in status output.

## 8. Architecture

### 8.1 Packages (Go)

Module: `github.com/skaphos/repokeeper`

```
/cmd/repokeeper/              # cobra commands (main entry point)
/internal/config/             # config load/save, platform path resolution
/internal/registry/           # registry load/save, staleness detection
/internal/discovery/          # scan roots for repos, glob/exclude
/internal/gitx/               # git execution helpers, output parsing
/internal/model/              # RepoStatus structs, JSON types
/internal/engine/             # orchestration: status, sync, scan (core logic)
/internal/tui/                # bubble tea frontend (phase 2)
/.github/workflows/           # CI: lint, test, build, release
/.golangci.yml                # linter configuration
/.goreleaser.yaml             # release configuration
```

Each `internal/` package has a corresponding `*_suite_test.go` (Ginkgo bootstrap) and `*_test.go` files.

### 8.2 CLI framework

Use Cobra for subcommands, flags, help text ([GitHub][4])

Config loading: YAML (either viper or lightweight YAML parsing).

### 8.3 Concurrency model

* A worker pool processes repos for `status` and `sync`.
* Concurrency is bounded by `--concurrency`.
* Each repo action has a context timeout.

### 8.4 TUI model (phase 2)

Bubble Tea implements the Elm architecture (Model → Update → View):

* **Model:** repo list state (`[]RepoStatus`), active filters, cursor position, selection set, active view (`list` or `details`), and action progress/errors.
* **Update:** routes key events to list/filter/action reducers and receives async engine results (sync/update/repair completion).
* **View:** k9s-style list-first presentation with compact status columns plus contextual detail/action panel on demand.

Prefer Bubbles components for list/table/text input/spinner; allow small custom view composition where needed for action menus and details. ([GitHub][5])

## 9. Future: Cross-Machine Registry Sync

A key future capability is **pushing your repo structure to other machines** so you can reconcile what's cloned where and potentially auto-clone missing repos.

### 9.1 Architecture considerations (factor in now)

The v1 data model should be designed so the registry is **portable and mergeable**:

* **`repo_id` is the join key** — the normalized remote URL is machine-independent by design. This is already the case.
* **Registry is serializable** — YAML/JSON, no machine-specific binary state. Already the case.
* **Timestamps on entries** — `last_seen` per repo enables conflict resolution (last-writer-wins or newest-wins). Added in §6.2.2.

### 9.2 Planned sync mechanisms (future, not v1)

* **File-based:** Baseline export/import is implemented in v1 (`repokeeper export` / `repokeeper import`). Future work can extend this with remote targets (cloud drive, git repo, S3 bucket).
* **Network-based:** Lightweight HTTP/gRPC server mode (`repokeeper serve`) that other machines can push/pull registries to/from.
* **Git-based:** Store registries in a dedicated git repo (for example, a `repokeeper/registries/` folder) and sync via git itself — dogfooding.

### 9.3 Reconciliation (future)

Given registries from multiple machines, RepoKeeper can produce a **global manifest** showing:

* Which repos exist on which machines.
* Which repos are missing from a given machine.
* Optionally: `repokeeper clone-missing` to clone repos that exist elsewhere but not locally.

> **v1 action item:** No sync code is needed, but ensure the registry schema and `repo_id` normalization are stable enough to be shared across machines without breaking.

## 10. Open Questions (explicitly deferred)

* Auto-update `main` safely (future).
* Auto-clean local branches (future).
* Better handling for worktrees and submodules status display (future).
* Network sync transport choice (HTTP vs gRPC vs git-based) (future).
* Authentication/authorization for network sync (future).
* Conflict resolution strategy when two machines modify the same repo entry (future).

---

## Appendix A — Why `for-each-ref` is preferred for tracking/gone

`git for-each-ref` supports upstream fields and can emit `"[gone]"` when upstream refs are missing, making it a reliable way to detect stale tracking relationships ([Git][3])

## Appendix B — Why Bubble Tea + Bubbles

Bubble Tea is a Go TUI framework suitable for simple/complex terminal apps ([GitHub][1]) and Bubbles provides reusable TUI components to speed up implementation ([GitHub][5])

[1]: https://github.com/charmbracelet/bubbletea "charmbracelet/bubbletea: A powerful little TUI framework - GitHub"
[2]: https://git-scm.com/docs/git-fetch "Git - git-fetch Documentation"
[3]: https://git-scm.com/docs/git-for-each-ref "git-for-each-ref Documentation - Git"
[4]: https://github.com/spf13/cobra "spf13/cobra: A Commander for modern Go CLI interactions - GitHub"
[5]: https://github.com/charmbracelet/bubbles "charmbracelet/bubbles: TUI components for Bubble Tea - GitHub"
[6]: https://onsi.github.io/ginkgo/ "Ginkgo - A Go Testing Framework"
[7]: https://onsi.github.io/gomega/ "Gomega - A Go Matcher Library"
[8]: https://golangci-lint.run/ "golangci-lint - Fast Go linters runner"
[9]: https://goreleaser.com/ "GoReleaser - Release Go projects as fast and easily as possible"
