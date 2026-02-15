# RepoKeeper — Implementation Tasks

## Milestones

### Milestone 0 — Repo skeleton

- [x] Initialize Go module (`github.com/skaphos/repokeeper`)
- [x] Add Cobra scaffolding (`repokeeper`, `init`, `scan`, `status`, `sync`, `version`)
- [x] Add config + registry paths and basic I/O
- [x] Bootstrap Ginkgo test suites for each package (`ginkgo bootstrap`)
- [x] Add `.golangci.yml` with linter configuration
- [x] Add `.goreleaser.yaml` with build configuration
- [x] Add GitHub Actions CI workflow (lint, test, build)
- [x] Add `ldflags` for version embedding

**Deliverable:**

- [x] `repokeeper version` prints version/commit/date
- [x] `repokeeper init` creates a config file
- [x] `repokeeper init` defaults to `.repokeeper.yaml` in the current directory
- [x] runtime commands resolve nearest `.repokeeper.yaml` by walking parent directories
- [x] `repokeeper status --help`
- [ ] `golangci-lint run ./...` passes
- [ ] `ginkgo ./...` runs (even if no tests yet)

### Milestone 1 — Discovery + registry

- [x] **Tests first:** Ginkgo specs for URL normalization, glob/exclude matching, registry read/write, staleness detection
- [x] Implement scan: walk configured roots
- [x] Implement scan: detect `.git` dirs (handle bare/linked git dirs)
- [x] Implement scan: store repo path + normalized repo_id
- [x] Implement scan: registry staleness detection (missing/moved)
- [x] Write registry persistence

**Acceptance:**

- [x] `repokeeper scan` produces registry with N repos
- [x] All discovery/registry Ginkgo specs pass
- [x] Coverage >= 80% for `internal/discovery/` and `internal/registry/`

### Milestone 2 — Status engine

- [x] **Tests first:** Ginkgo specs for `for-each-ref` parsing, `status --porcelain` parsing, bare repo detection, multiple remote handling, error classification
- [x] Implement dirty status (skip for bare repos)
- [x] Implement branch/detached detection
- [x] Implement all remotes + primary remote selection
- [x] Implement submodule presence detection
- [x] Implement tracking status + ahead/behind counts via `for-each-ref`
- [x] Implement bare repo detection
- [x] Implement table output
- [x] Implement JSON output

**Acceptance:**

- [x] `repokeeper status --format json` returns accurate info across many repos
- [x] All gitx/model Ginkgo specs pass
- [x] Coverage >= 80% for `internal/gitx/` (`internal/model` has no executable statements)

### Milestone 3 — Sync engine

- [x] **Tests first:** Ginkgo specs for engine orchestration (mock git runner), concurrency limits, timeout behavior, dry-run output
- [x] Implement worker pool for syncing repos
- [x] Implement `git fetch --all --prune --prune-tags --no-recurse-submodules`
- [x] Record per-repo success/failure
- [x] Integration tests with real temporary git repos

**Acceptance:**

- [x] `repokeeper sync` updates remote refs and prunes stale remote-tracking branches without touching worktrees/submodules
- [x] `repokeeper sync --dry-run` shows intended operations without executing
- [x] All engine Ginkgo specs pass
- [x] Integration test suite passes

### Milestone 4 — Polish for daily use

- [x] `--only` filters (including `missing` for stale registry entries)
- [x] Exit codes per DESIGN.md §5.1 (0 success, 1 warnings, 2 errors, 3 fatal)
- [x] Better error classification (auth, network, corrupt, missing remote)
- [x] Stable sorting/grouping in output
- [x] `--verbose` / `--quiet` behavior
- [x] `--no-color` and `NO_COLOR` env var support

**Acceptance:**

- [x] Actionable output for "what broke and where"
- [x] Exit codes are correct and tested
- [ ] `golangci-lint run ./...` passes with zero issues
- [ ] Overall test coverage >= 80%

### Milestone 5 — TUI (phase 2)

- [ ] k9s-style primary repo list as default view
- [ ] keyboard-first navigation (`j`/`k` + arrows), filter mode, and selection set
- [ ] filter repos by id/path/branch/tracking/error state
- [ ] contextual actions from list/details (sync, edit metadata, repair upstream)
- [ ] trigger batch actions for selected repos
- [ ] progress updates + detail/action view
- [ ] keybindings baseline: `/` filter, `space` select, `a` select all, `s` sync, `e` edit, `r` repair upstream, `enter` details/actions, `q` quit

**Acceptance:**

- [ ] Use TUI as primary operations dashboard without losing CLI automation parity
- [ ] Core interaction model feels familiar to k9s users (list-first, filter-first, action-driven)

### Milestone 6 — CLI Improvements (Ongoing)

- [x] Kubectl-aligned command aliases (additive):
- [x] `get repos`, `describe repo`, `reconcile repos`, `repair upstream`
- [x] map existing commands to new aliases with parity tests
- [ ] Output contract alignment:
- [ ] converge list commands on shared columns and `-o` semantics
- [ ] add `--no-headers` and deterministic sort behavior everywhere
- [x] implement `-o wide` for repo list/reconcile views
- [ ] Color/styling policy hardening:
- [x] auto-color only for TTY table output; disable color for machine formats
- [x] preserve `--no-color` and `NO_COLOR` precedence
- [ ] normalize semantic color mapping (healthy/warn/error/info)
- [ ] Selector evolution:
- [ ] keep `--only` as shorthand
- [ ] add field-selector style filtering (phase rollout)
- [ ] `repair-upstream` command:
- [ ] detect missing/wrong upstream tracking refs
- [ ] repair to configured/default upstream (`origin/<branch>`) with dry-run support
- [ ] diverged-focused reporting:
- [ ] add machine-readable and table views for repos in `diverged` state
- [ ] include reason and recommended action (`manual`, `--force`, etc.)
- [ ] remote mismatch detection:
- [ ] report registry `remote_url` vs live git remote mismatch
- [ ] optional reconcile mode to update registry or git remote (explicit flag)
- [ ] `sync --continue-on-error`:
- [ ] continue processing all repos while accumulating failures
- [ ] summarize failed repos/actions at end with deterministic ordering
- [ ] richer exit code model for automation:
- [ ] preserve existing high-level codes but add structured per-repo outcomes in JSON
- [ ] outcome categories: `fetched`, `rebased`, `skipped_no_upstream`, `skipped_diverged`, `failed`, etc.
- [ ] protected-branch safeguards:
- [ ] block auto-rebase on protected branch patterns by default
- [ ] allow explicit override flag for emergency runs
- [ ] document safeguards in `README.md` and `DESIGN.md`

**Acceptance:**

- [ ] RepoKeeper commands feel familiar to kubectl-heavy users without breaking existing workflows.
- [ ] Output formats are consistent across list/reconcile/repair commands.
- [ ] Color behavior is predictable: rich in interactive terminals, clean for machine-readable output.
- [ ] Operators can run one command to identify and repair upstream drift safely.
- [ ] Diverged and remote-mismatch repos are clearly surfaced without digging through logs.
- [ ] Batch sync runs complete across all repos even with partial failures (`--continue-on-error`).
- [ ] CI/automation can rely on stable outcome fields and exit behavior for policy decisions.
- [ ] Protected branches are never rebased automatically unless explicitly overridden.
- [ ] Milestone remains open as new CLI ergonomics and automation gaps are identified.

### Milestone 9 — Multi-VCS adapters (stretch)

- [x] Add VCS adapter abstraction (Git-first, but extensible)
- [ ] Mercurial (hg) adapter: discovery, status, safe sync (pull --update? define safety)
- [ ] Bazaar (bzr) adapter: discovery, status, safe sync
- [ ] CLI flags to select VCS types (e.g., `--vcs git,hg,bzr`)
- [ ] Update docs + compatibility matrix per VCS tool versions

**Acceptance:**

- [ ] Git remains default; Hg/Bzr optional and clearly documented as experimental
- [ ] Adapter selection works without regressions to Git behavior

---

## Test Plan

### Testing philosophy

**Test-Driven Development (TDD)** — write tests first, then implement. Every package should have tests before or alongside its implementation code.

**Behavior-Driven Development (BDD)** — use [Ginkgo](https://onsi.github.io/ginkgo/) as the test framework and [Gomega](https://onsi.github.io/gomega/) as the matcher library.

### Coverage requirements

- [ ] Target: 80%+ line coverage across all packages
- [ ] CI enforces coverage thresholds
- [ ] Packages with inherently low testability (`cmd/`, `internal/tui/`) may have lower thresholds but should still have smoke tests

### Unit tests (Ginkgo suites)

- [x] `internal/gitx/` — URL normalization, `for-each-ref` output parsing, `status --porcelain` parsing, error classification
- [x] `internal/discovery/` — glob/exclude matching, symlink handling, bare repo detection
- [x] `internal/config/` — config loading, defaults, platform path resolution
- [x] `internal/config/` — nearest-parent config discovery and global fallback precedence
- [x] `internal/registry/` — registry read/write, staleness detection, merge semantics
- [x] `internal/model/` — JSON serialization/deserialization, schema stability
- [x] `internal/engine/` — orchestration logic (mock git runner), concurrency behavior, dry-run

### Integration tests

- [ ] fetch/prune doesn't change working tree files
- [ ] upstream gone detection
- [ ] bare repo detection and status reporting
- [ ] registry staleness (delete a repo path, re-scan, verify `missing` status)
- [ ] Integration tests tagged with `//go:build integration`
- [ ] integration test: run from nested subdirectory and verify nearest `.repokeeper.yaml` is used

### Linting & code quality

- [x] golangci-lint configured via `.golangci.yml`
- [x] Lint runs on every PR and push to main
- [ ] Lint failures block merge

### CI pipeline (GitHub Actions)

- [x] `lint` job: golangci-lint
- [x] `test` job: go test (Ginkgo) + coverage check
- [x] `inttest` job: go test -tags integration
- [x] `build` job: go build (all platforms via matrix)
- [x] `release` job: GoReleaser on tag push

---

## Release & Distribution

**Repository:** `github.com/skaphos/repokeeper`

**Go module path:** `github.com/skaphos/repokeeper`

- [x] `.goreleaser.yaml` configured with binary name, archive formats, ldflags
- [x] GitHub Actions release workflow triggers on tag push
- [x] Target platforms: macOS (arm64, amd64), Windows (amd64), Linux (amd64, arm64)
- [x] Snapshot builds for development (`goreleaser build --snapshot`)
