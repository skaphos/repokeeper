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
- [x] `go tool golangci-lint run ./...` passes
- [x] `go tool ginkgo ./...` runs (even if no tests yet)

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
- [x] `go tool golangci-lint run ./...` passes with zero issues
- [x] Overall test coverage >= 80%

### Milestone 6 — CLI Improvements (Ongoing)

- [x] Kubectl-aligned command aliases (additive):
- [x] `get repos`, `describe repo`, `reconcile repos`, `repair upstream`
- [x] map existing commands to new aliases with parity tests
- [x] Output contract alignment:
- [x] converge list commands on shared columns and `-o` semantics
- [x] add `--no-headers` and deterministic sort behavior everywhere
- [x] implement `-o wide` for repo list/reconcile views
- [ ] Color/styling policy hardening:
- [x] auto-color only for TTY table output; disable color for machine formats
- [x] preserve `--no-color` and `NO_COLOR` precedence
- [x] normalize semantic color mapping (healthy/warn/error/info)
- [ ] Selector evolution:
- [x] keep `--only` as shorthand
- [x] add field-selector style filtering (phase rollout)
- [ ] `repair-upstream` command:
- [x] detect missing/wrong upstream tracking refs
- [x] repair to configured/default upstream (`origin/<branch>`) with dry-run support
- [ ] diverged-focused reporting:
- [x] add machine-readable and table views for repos in `diverged` state
- [x] include reason and recommended action (`manual`, `--force`, etc.)
- [ ] remote mismatch detection:
- [x] report registry `remote_url` vs live git remote mismatch
- [x] optional reconcile mode to update registry or git remote (explicit flag)
- [ ] `sync --continue-on-error`:
- [x] continue processing all repos while accumulating failures
- [x] summarize failed repos/actions at end with deterministic ordering
- [ ] richer exit code model for automation:
- [x] preserve existing high-level codes but add structured per-repo outcomes in JSON
- [x] outcome categories: `fetched`, `rebased`, `skipped_no_upstream`, `skipped_diverged`, `failed`, etc.
- [ ] protected-branch safeguards:
- [x] block auto-rebase on protected branch patterns by default
- [x] allow explicit override flag for emergency runs
- [x] document safeguards in `README.md` and `DESIGN.md`
- [ ] confirmation policy for mutating actions:
- [x] require user confirmation for repo/state-mutating actions by default
- [x] add `--yes` global bypass for non-interactive/automation use
- [x] ensure fetch-only/no-op actions do not require confirmation

**Acceptance:**

- [x] RepoKeeper commands feel familiar to kubectl-heavy users without breaking existing workflows.
- [x] Output formats are consistent across list/reconcile/repair commands.
- [x] Color behavior is predictable: rich in interactive terminals, clean for machine-readable output.
- [x] Operators can run one command to identify and repair upstream drift safely.
- [x] Diverged and remote-mismatch repos are clearly surfaced without digging through logs.
- [x] Batch sync runs complete across all repos even with partial failures (`--continue-on-error`).
- [x] CI/automation can rely on stable outcome fields and exit behavior for policy decisions.
- [x] Protected branches are never rebased automatically unless explicitly overridden.
- [x] Mutating actions always require confirmation unless `--yes` is explicitly passed.
- [ ] Milestone remains open as new CLI ergonomics and automation gaps are identified.

### Milestone 6.1 — Code Quality & Refactoring

- [ ] Global state elimination:
  - [x] replace package-level flag variables (`flagVerbose`, `flagQuiet`, `exitCode`, etc.) with a command context struct
  - [x] enable isolated unit testing of commands without state leakage
- [ ] Duplicate sync execution:
- [x] refactor `sync` command to reuse dry-run plan instead of calling `eng.Sync()` twice
- [x] add `Execute(plan)` method to engine that accepts a pre-computed plan
- [ ] Typed error classification:
  - [x] replace string-based error classification in `gitx/error_class.go` with sentinel errors or error types
  - [x] define `ErrAuth`, `ErrNetwork`, `ErrCorrupt`, `ErrMissingRemote` typed errors
  - [x] update `ClassifyError` to use `errors.Is`/`errors.As` instead of string matching
- [ ] Engine method decomposition:
  - [ ] break `Sync()` method (~400 lines) into smaller focused functions
  - [ ] extract dry-run planning, execution, and result collection into separate methods
  - [ ] reduce goroutine body complexity (currently handles 8+ distinct code paths)
- [ ] Eliminate repeated nil-guard pattern:
  - [x] initialize `Adapter` in `engine.New()` constructor with sensible default
  - [x] remove redundant `if e.Adapter == nil` checks from `Scan`, `Status`, `Sync`, `InspectRepo`
- [ ] Extract shared utilities:
  - [x] move `splitCSV()` from `scan.go` to a shared `internal/cli` or `internal/strutil` package
  - [x] extract ANSI color constants and `colorize()` to shared package
  - [ ] create reusable table writer abstraction to deduplicate `writeStatusTable`, `writeSyncTable`, `writeSyncPlan`
  - [x] extract common sorting lambda (`RepoID` then `Path`) into named comparator functions
- [ ] Magic string constants:
  - [x] define constants for error state strings (`"missing"`, `"skipped-local-update:"`, etc.)
  - [ ] consider typed outcome enum for `SyncResult.Outcome`
- [ ] Remove outdated loop capture pattern:
- [x] remove `entry := entry` captures in goroutine loops (unnecessary since Go 1.22)
- [ ] Configurable main branch assumption:
- [x] make hardcoded `/main` suffix check in `pullRebaseSkipReason` configurable
- [x] add `defaults.main_branch` config option or use protected-branches pattern
- [ ] Dependency cleanup:
- [x] run `go mod tidy` to fix `golang.org/x/term` direct/indirect status

**Acceptance:**

- [ ] No package-level mutable state in `cmd/repokeeper/` (flags read via context/struct)
- [x] `sync` command performs repo analysis only once per invocation
- [x] Error classification uses Go error types with `errors.Is`/`errors.As`
- [ ] No single function exceeds 100 lines (excluding table definitions)
- [ ] Shared utilities live in dedicated packages with their own tests
- [x] `go mod tidy` produces no changes
- [x] All existing tests pass with no behavior changes
- [x] Coverage remains >= 80%

### Milestone 7 — Responsive Output & Reflow

- [ ] Replace stdlib `text/tabwriter` with `github.com/liggitt/tabwriter` to align kubectl-style table behavior
- [ ] Evaluate kubectl printer-stack compatibility (`k8s.io/cli-runtime/pkg/printers`) without taking Kubernetes object dependencies
- [ ] Width-aware table rendering for narrow terminals (kubectl-style reflow behavior)
- [ ] Adaptive column strategy by view (`get/status`, `reconcile/sync`, `repair`) with deterministic priority
- [ ] Smart truncation rules and optional wrapping that preserve key identifiers (`PATH`, `REPO`, `ACTION`)
- [ ] Dynamic header/column compaction for small widths while keeping `-o json` stable
- [ ] Add output selectors: `-o jsonpath=...` and `-o custom-columns=...` (custom headers/column maps)
- [ ] Snapshot tests across terminal widths (for example: 80, 100, 120, 160 cols)
- [ ] Ensure color and readability parity in compact/reflow modes

**Acceptance:**

- [ ] Table output remains readable and actionable on small terminal windows without horizontal scrolling for common cases
- [ ] Output remains deterministic across width classes for automation-safe parsing in machine formats
- [ ] Milestone 6 remains free of terminal reflow scope

### Milestone 8 — Multi-VCS Adapters & Feedback Loop

- [x] Add VCS adapter abstraction (Git-first, but extensible)
- [ ] Mercurial (hg) adapter: discovery, status, safe sync (pull --update? define safety)
- [ ] Bazaar (bzr) adapter: discovery, status, safe sync
- [ ] CLI flags to select VCS types (e.g., `--vcs git,hg,bzr`)
- [ ] Update docs + compatibility matrix per VCS tool versions
- [ ] Gather user feedback from milestone 6/7 usage and prioritize improvements
- [ ] Implement highest-impact UX and automation improvements from feedback
- [ ] Re-run docs/help cleanup after feedback-driven changes

**Acceptance:**

- [ ] Git remains default; Hg/Bzr optional and clearly documented as experimental
- [ ] Adapter selection works without regressions to Git behavior
- [ ] Feedback items are captured, triaged, and reflected in shipped improvements

### Milestone 9 — TUI (phase 2)

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

### Milestone 10 — 1.0 Readiness & Release Reset

- [ ] Keep release line on `0.x` until milestone 7, milestone 8, and milestone 9 completion criteria are met
- [ ] Define and publish a 1.0 readiness gate (CLI surface freeze, output contract freeze, docs freeze, compatibility matrix)
- [ ] Archive or remove pre-1.0 GitHub releases/tags to reset public release history
- [ ] Confirm post-reset versioning strategy (for example restart at `v0.1.0` or begin prerelease series)
- [ ] Add migration notes for users once post-milestone-9 releases begin
- [ ] Cut first post-reset release only after milestone 7, milestone 8, and milestone 9 acceptance criteria are met

**Acceptance:**

- [ ] Release history and versioning policy are explicit and consistent with pre-1.0 status
- [ ] No `1.0.0` tag is created until readiness gate is complete
- [ ] Post-reset release process is documented and repeatable

---

## Test Plan

### Testing philosophy

**Test-Driven Development (TDD)** — write tests first, then implement. Every package should have tests before or alongside its implementation code.

**Behavior-Driven Development (BDD)** — use [Ginkgo](https://onsi.github.io/ginkgo/) as the test framework and [Gomega](https://onsi.github.io/gomega/) as the matcher library.

### Coverage requirements

- [x] Target: 80%+ line coverage across all packages
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
