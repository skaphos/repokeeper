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
- [x] Color/styling policy hardening:
- [x] auto-color only for TTY table output; disable color for machine formats
- [x] preserve `--no-color` and `NO_COLOR` precedence
- [x] normalize semantic color mapping (healthy/warn/error/info)
- [x] Selector evolution:
- [x] keep `--only` as shorthand
- [x] add field-selector style filtering (phase rollout)
- [x] `repair-upstream` command:
- [x] detect missing/wrong upstream tracking refs
- [x] repair to configured/default upstream (`origin/<branch>`) with dry-run support
- [x] diverged-focused reporting:
- [x] add machine-readable and table views for repos in `diverged` state
- [x] include reason and recommended action (`manual`, `--force`, etc.)
- [x] remote mismatch detection:
- [x] report registry `remote_url` vs live git remote mismatch
- [x] optional reconcile mode to update registry or git remote (explicit flag)
- [x] `sync --continue-on-error`:
- [x] continue processing all repos while accumulating failures
- [x] summarize failed repos/actions at end with deterministic ordering
- [x] richer exit code model for automation:
- [x] preserve existing high-level codes but add structured per-repo outcomes in JSON
- [x] outcome categories: `fetched`, `rebased`, `skipped_no_upstream`, `skipped_diverged`, `failed`, etc.
- [x] protected-branch safeguards:
- [x] block auto-rebase on protected branch patterns by default
- [x] allow explicit override flag for emergency runs
- [x] document safeguards in `README.md` and `DESIGN.md`
- [x] confirmation policy for mutating actions:
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

- [x] Global state elimination:
  - [x] replace package-level flag variables (`flagVerbose`, `flagQuiet`, `exitCode`, etc.) with a command context struct
  - [x] enable isolated unit testing of commands without state leakage
- [x] Duplicate sync execution:
- [x] refactor `sync` command to reuse dry-run plan instead of calling `eng.Sync()` twice
- [x] add `Execute(plan)` method to engine that accepts a pre-computed plan
- [x] Typed error classification:
  - [x] replace string-based error classification in `gitx/error_class.go` with sentinel errors or error types
  - [x] define `ErrAuth`, `ErrNetwork`, `ErrCorrupt`, `ErrMissingRemote` typed errors
  - [x] update `ClassifyError` to use `errors.Is`/`errors.As` instead of string matching
- [x] Engine method decomposition:
  - [x] break `Sync()` method (~400 lines) into smaller focused functions
  - [x] extract dry-run planning, execution, and result collection into separate methods
  - [x] reduce goroutine body complexity (currently handles 8+ distinct code paths)
- [x] Eliminate repeated nil-guard pattern:
  - [x] initialize `Adapter` in `engine.New()` constructor with sensible default
  - [x] remove redundant `if e.Adapter == nil` checks from `Scan`, `Status`, `Sync`, `InspectRepo`
- [x] Extract shared utilities:
  - [x] move `splitCSV()` from `scan.go` to a shared `internal/cli` or `internal/strutil` package
  - [x] extract ANSI color constants and `colorize()` to shared package
  - [x] create reusable table writer abstraction to deduplicate `writeStatusTable`, `writeSyncTable`, `writeSyncPlan`
  - [x] extract common sorting lambda (`RepoID` then `Path`) into named comparator functions
- [x] Magic string constants:
  - [x] define constants for error state strings (`"missing"`, `"skipped-local-update:"`, etc.)
  - [x] consider typed outcome enum for `SyncResult.Outcome`
- [x] Remove outdated loop capture pattern:
- [x] remove `entry := entry` captures in goroutine loops (unnecessary since Go 1.22)
- [x] Configurable main branch assumption:
- [x] make hardcoded `/main` suffix check in `pullRebaseSkipReason` configurable
- [x] add `defaults.main_branch` config option or use protected-branches pattern
- [x] Dependency cleanup:
- [x] run `go mod tidy` to fix `golang.org/x/term` direct/indirect status

**Acceptance:**

- [x] No package-level mutable state in `cmd/repokeeper/` (flags read via context/struct)
- [x] `sync` command performs repo analysis only once per invocation
- [x] Error classification uses Go error types with `errors.Is`/`errors.As`
- [x] Function size guideline: target ~100 lines where practical to keep density/complexity low (not a strict hard limit)
- [x] Shared utilities live in dedicated packages with their own tests
- [x] `go mod tidy` produces no changes
- [x] All existing tests pass with no behavior changes
- [x] Coverage remains >= 80%

### Milestone 6.2 — Hardening & Technical Debt

- [x] Error handling improvements:
  - [x] handle I/O errors from `fmt.Fprintf`/`fmt.Fprintln` on critical output paths (sync.go, status.go, scan.go)
  - [x] log or propagate `tabwriter.Flush()` errors instead of discarding
  - [x] log output write failures at debug level (no exit code change; broken pipes are normal CLI behavior)
- [x] Concurrency safety:
  - [x] fix race condition: goroutines in `Status()` access `e.Registry` without synchronization (engine.go:158-201)
  - [x] copy registry entry data before passing to goroutines
  - [x] address channel leak risk in `syncSequentialStopOnError()` on early return (engine.go:446-475)
  - [x] add mutex or value-based updates for registry entry mutation during sync
- [x] Memory/performance:
  - [x] reduce unbounded channel buffer sizes from `len(entries)` to fixed cap (e.g., 100) in engine.go
  - [x] eliminate redundant `eng.Status()` call after sync for table/wide output (sync.go:145-163)
  - [x] remove duplicate sorting (status.go:133 duplicates engine.go:210)
- [x] Security hardening:
  - [x] add validation for upstream format in edit.go (should match `remote/branch` pattern)
  - [x] validate path normalization in `selectRegistryEntryForDescribe` stays within configured roots
- [x] API design cleanup:
  - [x] make Engine struct fields private, add read-only accessors if needed
  - [x] replace direct `gitx.GitRunner{}` instantiation with Adapter interface (edit.go, add.go, repair_upstream.go)
    - [x] edit.go
    - [x] add.go
    - [x] repair_upstream.go
    - [x] status.go (remote mismatch git reconcile path)
    - [x] portability.go (import clone path)
  - [x] extract remote mismatch logic from status.go to dedicated package
- [x] Flag/config consolidation:
  - [x] create flag builder helpers (`addFormatFlag`, `addFilterFlags`) to DRY up duplicate definitions
  - [x] single source of truth for defaults (concurrency, timeout, main_branch) in Config.Defaults
- [x] Test coverage expansion:
  - [x] add unit tests for command RunE functions (sync, status, repair_upstream)
  - [x] add integration tests for edge cases: symlinks, bare repos, missing repos
  - [x] add table-driven tests for command parsing logic
- [x] Documentation:
  - [x] document flag precedence behavior in root.go or README
  - [x] add Godoc comments on public types (SyncResult fields, model types)
  - [x] clarify `--dry-run` default behavior in status command
- [x] Code organization:
  - [x] extract table rendering and confirmation logic from commands to reusable modules
  - [x] replace deep parameter lists with options structs (e.g., `PullRebaseOptions`)
  - [x] create typed `OutcomeKind` enum for `SyncResult.Outcome`
- [x] Minor cleanup:
  - [x] standardize error message formatting (%q vs %s)
  - [x] remove unused local variables flagged by linter
  - [x] audit and clean up unused imports

**Acceptance:**

- [x] No ignored I/O errors on stdout/stderr writes
- [x] `go test -race ./...` passes with no data races
- [x] Channel buffers capped at reasonable fixed size
- [x] Engine fields are private with controlled access
- [x] All commands use Adapter interface instead of direct GitRunner
- [x] Flag definitions are DRY (single helper per common flag pattern)
- [x] Unit test coverage for command handlers >= 70%
- [x] Integration tests cover symlink, bare repo, and missing repo scenarios
- [x] All public types have Godoc comments

### Milestone 7 — Responsive Output & Reflow

- [ ] Replace stdlib `text/tabwriter` with `github.com/liggitt/tabwriter` to align kubectl-style table behavior
- [ ] Evaluate kubectl printer-stack compatibility (`k8s.io/cli-runtime/pkg/printers`) without taking Kubernetes object dependencies
- [x] Width-aware table rendering for narrow terminals (kubectl-style reflow behavior)
- [x] Adaptive column strategy by view (`get/status`, `reconcile/sync`, `repair`) with deterministic priority
- [ ] Smart truncation rules and optional wrapping that preserve key identifiers (`PATH`, `REPO`, `ACTION`)
- [x] Dynamic header/column compaction for small widths while keeping `-o json` stable
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
- [x] CI enforces coverage thresholds
- [x] Per-package goal: 80% minimum when reasonable, with explicit temporary exceptions documented in tooling
- [x] Packages with inherently low testability (`cmd/`, `internal/tui/`) may have lower thresholds but should still have smoke tests

### Unit tests (Ginkgo suites)

- [x] `internal/gitx/` — URL normalization, `for-each-ref` output parsing, `status --porcelain` parsing, error classification
- [x] `internal/discovery/` — glob/exclude matching, symlink handling, bare repo detection
- [x] `internal/config/` — config loading, defaults, platform path resolution
- [x] `internal/config/` — nearest-parent config discovery and global fallback precedence
- [x] `internal/registry/` — registry read/write, staleness detection, merge semantics
- [x] `internal/model/` — JSON serialization/deserialization, schema stability
- [x] `internal/engine/` — orchestration logic (mock git runner), concurrency behavior, dry-run

### Integration tests

- [x] fetch/prune doesn't change working tree files
- [x] upstream gone detection
- [x] bare repo detection and status reporting
- [x] registry staleness (delete a repo path, re-scan, verify `missing` status)
- [x] Integration tests tagged with `//go:build integration`
- [x] integration test: run from nested subdirectory and verify nearest `.repokeeper.yaml` is used

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
