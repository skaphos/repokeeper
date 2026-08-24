---

description: "Dependency-ordered implementation tasks for the recipe-driven end-to-end harness"
---

# Tasks: Recipe-Driven End-to-End Test Harness

**Input**: Design documents from `/specs/002-e2e-recipe-harness/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: The feature is itself a test harness, and the specification explicitly requires direct coverage. Validation specs for shared helpers are written before their implementations; each user story ends with an independently runnable acceptance checkpoint.

**Organization**: Tasks are grouped by four user stories so the CLI MVP, MCP process boundary, recipe extensibility, and release qualification can be implemented and verified as distinct increments after the shared foundation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with sibling tasks after stated prerequisites because it changes different files
- **[Story]**: Maps the task to a prioritized user story from `spec.md`
- Every task names the exact file or files it changes or validates

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the integration-only package boundary and one-build-per-suite lifecycle.

- [X] T001 Create the integration-tagged Ginkgo suite, resolve the module root, build the real RepoKeeper executable once with a 30-second context, select the platform executable suffix, and register bounded cleanup in `test/e2e/e2e_suite_test.go`
- [X] T002 Add an integration-tagged source-policy spec that scans every Go file under `test/e2e`, requires `//go:build integration`, requires compound `integration && windows` or `integration && !windows` constraints for platform files, and proves ordinary untagged package listing/building excludes the E2E tree in `test/e2e/build_tags_spec_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement reusable recipe, environment, process, containment, and assertion infrastructure required by every user story.

**⚠️ CRITICAL**: No user story work begins until this phase passes its checkpoint.

### Foundational Validation Specs

- [X] T003 [P] Add failing specs for environment allowlisting, case-insensitive key replacement, isolated home/config/cache/temp paths, Git prompt/signing/credential suppression, and host-variable exclusion in `test/e2e/environment_spec_test.go`
- [X] T004 [P] Add failing table-driven specs for JSON recipe decoding, absolute paths, traversal, backslashes, drive/UNC paths, duplicate names, collisions, unknown relationships, and fail-before-write behavior in `test/e2e/recipe_validation_spec_test.go`
- [X] T005 [P] Add failing specs for normal/non-zero exits, launch failures, timeouts, bounded stdout/stderr capture, diagnostic rendering, and idempotent cleanup in `test/e2e/process_spec_test.go`

### Foundational Implementation

- [X] T006 Implement the platform-aware exact child environment builder, test-owned Git global configuration, deterministic identity/dates/locale, `GIT_ALLOW_PROTOCOL=file`, signing/prompt/credential isolation, and no process-global environment mutation in `test/e2e/environment_test.go`
- [X] T007 Implement `ExecutionResult`, bounded output buffers, argument-vector command execution, domain exit-code capture, 30-second-or-less contexts, `Cmd.WaitDelay`, and fixed diagnostic formatting in `test/e2e/process_test.go`
- [X] T008 [P] Implement Unix process-group creation, graceful termination, forced group termination, and reader/process joining with the `integration && !windows` constraint in `test/e2e/process_unix_test.go`
- [X] T009 [P] Implement Windows kill-on-close Job Object ownership, graceful termination, forced tree termination, and reader/process joining with the `integration && windows` constraint in `test/e2e/process_windows_test.go`
- [X] T010 Implement strict JSON loading into typed workspace/repository/branch/commit/metadata/missing/config recipes plus pure field-qualified validation, portable slash-path conversion, and canonical containment checks in `test/e2e/recipe_test.go`
- [X] T011 Extend the validated recipe implementation with deterministic local `file://` bare remotes, source clones, commits, branches, upstreams, metadata, dirty files, missing registry entries, isolated config persistence, and ready-state verification in `test/e2e/recipe_test.go`
- [X] T012 [P] Implement reusable config/registry reloads, Git HEAD/ref/porcelain snapshots, byte-content hashes, mutation containment checks, semantic path normalization, and structured failure assertions in `test/e2e/assertions_test.go`
- [X] T013 Add the shared canonical portable recipe artifact with clean metadata, dirty, missing, and reserved add/remove fixtures in `test/e2e/testdata/recipes/canonical.json`, then implement its separate CLI and missing-only-seeded MCP loading/materialization modes in `test/e2e/recipes_test.go`
- [X] T014 Run the uncached foundational specs, including the build-tag/default-tier boundary, and resolve all failures in `test/e2e/build_tags_spec_test.go`, `test/e2e/environment_spec_test.go`, `test/e2e/recipe_validation_spec_test.go`, and `test/e2e/process_spec_test.go`

**Checkpoint**: Recipes materialize beneath disposable roots, commands run in isolated environments, failures terminate and report deterministically, and ordinary untagged builds do not compile the E2E tree.

---

## Phase 3: User Story 1 - Verify Real CLI Workflows Safely (Priority: P1) 🎯 MVP

**Goal**: Run the real RepoKeeper CLI against clean, dirty, and missing repository state and prove its reports and persistence are correct without changing worktrees.

**Independent Test**: Run `go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus "Real CLI workflows"`; scan must return the documented warning exit, status must return the documented missing-path error exit, JSON and registry state must agree, and clean/dirty HEADs and files must remain unchanged.

### Tests and Harness Implementation for User Story 1

- [X] T015 [P] [US1] Define typed scan/status JSON response decoders, required-field assertions, expected domain exit-code helpers, and volatile-field normalization in `test/e2e/cli_contract_test.go`
- [X] T016 [US1] Implement the black-box `scan --roots <workspace> --write-registry --format json` and `status --format json` acceptance scenario with exit, stdout, stderr, registry, HEAD, clean-worktree, and byte-stable dirty-file assertions in `test/e2e/cli_spec_test.go`
- [X] T017 [US1] Implement five fresh canonical materializations and compare semantic normalized CLI classifications, relative paths, tracking state, commit IDs, and content hashes in `test/e2e/determinism_spec_test.go`
- [X] T018 [US1] Run and stabilize the independently focused CLI story without cached results against `test/e2e/cli_spec_test.go` and `test/e2e/determinism_spec_test.go`

**Checkpoint**: User Story 1 independently provides the executable-level CLI MVP.

---

## Phase 4: User Story 2 - Verify MCP Over a Real Process Boundary (Priority: P2)

**Goal**: Negotiate MCP over the real executable's standard streams and exercise every live registered tool, including destructive refusal and confined success behavior.

**Independent Test**: Run `go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus "Real MCP stdio"`; initialization and `tools/list` must succeed, discovered names and declared cases must match exactly, every case must succeed with valid input, safety calls must refuse without mutation and then succeed inside the scenario root, every stdout frame must be JSON-RPC, and all processes/readers must terminate.

### MCP Lifecycle and Contract Specs

- [X] T019 [P] [US2] Add failing specs for MCP process-lifetime deadlines, stdin EOF shutdown, forced cleanup, one `Wait` call, joined stdout/stderr readers, bounded diagnostics, and non-JSON stdout rejection in `test/e2e/mcp_client_spec_test.go`
- [X] T020 [P] [US2] Define case-specific typed structured-result decoders and required response/state assertion helpers for object and list-shaped MCP results in `test/e2e/mcp_contract_test.go`

### MCP Process and Tool Matrix Implementation

- [X] T021 [US2] Implement the harness-owned `repokeeper mcp --config <path>` command, exact environment, process-tree boundary, recorded stdout, drained stderr, `transport.NewIO`, `client.NewClient`, bounded RPC contexts, and idempotent session cleanup in `test/e2e/mcp_client_test.go`
- [X] T022 [US2] Implement the ordered `MCPToolCase` registry, duplicate detection, exact bidirectional `tools/list` set comparison, sorted drift diagnostics, and annotation-to-safety-case validation in `test/e2e/mcp_cases_test.go`
- [X] T023 [P] [US2] Add valid-input cases and structured/no-mutation assertions for `list_repositories`, `get_repository_context`, `get_workspace_config`, `build_workspace_inventory`, `select_repositories`, `get_repo_metadata`, `get_authoritative_paths`, `get_related_repositories`, and `plan_sync` in `test/e2e/mcp_cases_read_test.go`
- [X] T024 [P] [US2] Add valid-input cases and persisted state assertions for `scan_workspace`, `set_labels`, and `add_repository` in `test/e2e/mcp_cases_mutation_test.go`
- [X] T025 [P] [US2] Add refusal snapshots and confirmed confined-success cases for `execute_sync` and `remove_repository`, including fetched-versus-skipped result semantics and delete-target containment, in `test/e2e/mcp_cases_safety_test.go`
- [X] T026 [US2] Implement the real-session scenario in safe order—initialize, live coverage gate, scan, reads, plan, execute refusal/success, labels, add, remove refusal/success, persisted/global invariants, EOF close, and frame validation—in `test/e2e/mcp_spec_test.go`
- [X] T027 [US2] Run and stabilize the independently focused MCP story without cached results against `test/e2e/mcp_client_spec_test.go` and `test/e2e/mcp_spec_test.go`

**Checkpoint**: User Story 2 independently provides real-stdio success coverage for 100% of live tools and refusal plus confined success for every safety-sensitive tool.

---

## Phase 5: User Story 3 - Add Scenarios Without Rebuilding the Harness (Priority: P3)

**Goal**: Demonstrate that maintainers can declare another topology and reuse validation, materialization, environment, execution, and diagnostics without custom bootstrap code.

**Independent Test**: Run `go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus "Recipe extensibility"`; a second recipe must materialize through shared helpers, an inconsistent relationship must fail before RepoKeeper starts, and a forced command failure must include the complete diagnostic contract.

### Tests and Harness Demonstration for User Story 3

- [X] T028 [P] [US3] Add a second small portable repository-state artifact using only the shared recipe vocabulary and no imperative Git/bootstrap commands in `test/e2e/testdata/recipes/extension.json`
- [X] T029 [US3] Implement shared loading, validation, materialization, and expected-outcome helpers for the extension recipe in `test/e2e/recipes_extension_test.go`
- [X] T030 [US3] Add the recipe-extensibility acceptance scenario covering shared lifecycle reuse, field-qualified preflight failure before writes, and complete unexpected-exit diagnostics in `test/e2e/recipe_extension_spec_test.go`
- [X] T031 [US3] Run and stabilize the independently focused extensibility story without cached results against `test/e2e/recipes_extension_test.go` and `test/e2e/recipe_extension_spec_test.go`

**Checkpoint**: User Story 3 is independently complete, and a new scenario requires only recipe data and assertions.

---

## Phase 6: User Story 4 - Qualify Full Releases Against the Declared Git Matrix (Priority: P4)

**Goal**: Make the closed three-minor Git claim executable across Linux, macOS, native Windows, and WSL, with four representative routine cells and all twelve cells gating publication.

**Independent Test**: Expand both matrix scopes through the tagged compatibility command, provision and exactly verify each declared kind in focused tests, validate a synthetic twelve-cell evidence set, then prove workflow contract tests reject any publication dependency, permission, timeout, WSL, race, or evidence-completeness regression.

### Compatibility Contract Specs

- [X] T032 [P] [US4] Add failing specs for strict compatibility JSON decoding, unknown/trailing fields, the ordered closed three-minor rule, complete four-environment-by-three-minor coverage, duplicate cells, floating labels, patch/minor mismatches, provisioner/environment mismatches, WSL rootfs metadata, exactly one routine cell per environment, and `DESIGN.md` agreement in `test/e2e/compatibility_spec_test.go`
- [X] T033 [P] [US4] Add failing command/evidence specs for stable JSON output, stderr separation, routine/release expansion, exact Git version verification, tag/commit binding, and missing/duplicate/unexpected/skipped/failed/mismatched evidence in `test/e2e/compatibility_command_spec_test.go`
- [X] T034 [P] [US4] Add failing repository contract specs that inspect `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.github/workflows/release-please.yml`, and `Taskfile.yml` for pinned runner/action inputs, explicit job timeouts, read-only qualification permissions, four routine environments, WSL execution, a required race job, twelve-cell release expansion, candidate-tag behavior, evidence-gate dependencies, and publisher-only secrets in `test/e2e/workflow_contract_spec_test.go`

### Declaration and Reusable Compatibility Interface

- [X] T035 [US4] Audit every Git command and flag defined or delegated by `internal/gitx/gitx.go`, `internal/gitx/localbranch.go`, `internal/vcs/adapter.go`, and `internal/vcs/command_runner.go`; record the closed `2.53`/`2.54`/`2.55` claim, all twelve exact patches, official immutable source URLs, SHA-256 values, the Canonical Noble `20240423` rootfs and exact signed-snapshot WSL build prerequisites, and four routine representatives in `test/e2e/testdata/git-compatibility.json`, then make `DESIGN.md` summarize that declaration without an open-ended minimum claim
- [X] T036 [US4] Implement strict declaration types, stable cell keys, closed-window and Cartesian-set validation, routine/release expansion, runner/provisioner/rootfs validation, and `DESIGN.md` agreement in `test/e2e/internal/compatibility/declaration.go`
- [X] T037 [P] [US4] Implement checksum verification, 10-minute operation deadlines, test-owned prefixes, exact version parsing, and official Git source build provisioning for Linux/macOS/WSL with `integration && !windows` constraints in `test/e2e/internal/compatibility/provision_unix.go`
- [X] T038 [US4] After T037, implement 10-minute operation deadlines, checksum-verified MinGit extraction, Canonical Noble `20240423` WSL1 import/unregister, exact signed-snapshot build-prerequisite installation without package-manager latest/update, CGO-free Linux compatibility-helper/RepoKeeper/E2E cross-compilation, in-WSL source provisioning through the T037 Unix helper, suite-owned binary selection, and WSL invocation with `integration && windows` constraints in `test/e2e/internal/compatibility/provision_windows.go` and `test/e2e/e2e_suite_test.go`
- [X] T039 [US4] Implement bounded `CompatibilityResult` writing, tag/commit/input binding, artifact digests, and exact-set completeness diagnostics in `test/e2e/internal/compatibility/evidence.go`
- [X] T040 [US4] Implement `matrix`, `provision`, `verify-version`, `write-evidence`, `verify-evidence`, and `verify-docs` with JSON-only stdout, diagnostic stderr, and non-zero failures in `test/e2e/cmd/compatibility/main.go`
- [X] T041 [US4] Run the focused compatibility, command, evidence, and build-tag specs without cached results and resolve all failures in `test/e2e/compatibility_spec_test.go`, `test/e2e/compatibility_command_spec_test.go`, `test/e2e/internal/compatibility/*.go`, `test/e2e/cmd/compatibility/main.go`, `test/e2e/testdata/git-compatibility.json`, and `DESIGN.md`

### Routine and Full-Release Automation

- [X] T042 [US4] Update `test-integration` to use uncached tagged `./internal/engine` and `./test/e2e` execution, add `test-integration-race` with `-race -count=1` and explicit timeout, and add a default-tier exclusion check in `Taskfile.yml`
- [X] T043 [US4] Update `.github/workflows/ci.yml` to consume only `matrix --scope routine`, provision and exactly verify four non-fail-fast Linux/macOS/native-Windows/WSL representatives on explicit runner labels, add the native Ubuntu race job, pin actions by immutable commit SHA, set 30-minute-or-shorter job timeouts, retain least-privilege permissions, and record runner/Git identity
- [X] T044 [US4] Update `.github/workflows/release.yml` with read-only jobs limited to 30 minutes that consume `matrix --scope release`, run all twelve cells with `fail-fast: false`, provision and verify immutable Git/WSL inputs, run the E2E suite, and always upload bounded per-cell JSON evidence using commit-pinned actions
- [X] T045 [US4] Add the always-run exact-set evidence gate and make every publisher depend on it in `.github/workflows/release.yml`; scope publishing permissions, OIDC, packages, and secrets only to downstream publisher jobs, and retain `.github/workflows/release-please.yml` behavior in which a version tag is a candidate trigger and no GitHub Release is created before qualification
- [X] T046 [US4] Document the closed claim, four environments, WSL provisioning, routine versus exhaustive matrices, race coverage, immutable candidate tags, evidence artifacts, failure/retry rules, and publisher permission boundary in `test/e2e/README.md` and `RELEASE.md`
- [X] T047 [US4] Run and stabilize the independent release-qualification story against `test/e2e/compatibility_spec_test.go`, `test/e2e/compatibility_command_spec_test.go`, and `test/e2e/workflow_contract_spec_test.go`, including synthetic success and one-at-a-time failure cases for all twelve evidence cells

**Checkpoint**: User Story 4 independently proves representative coverage for all four environments, race-enabled lifecycle CI, and an exhaustive twelve-cell gate before any release publication.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Finish contributor documentation, dependency metadata, repository gates, and adversarial verification.

- [X] T048 [P] Add concise integration, race, E2E, and compatibility-command development entry points with a link to `test/e2e/README.md` in `README.md`
- [X] T049 Promote the already resolved `golang.org/x/sys` module to a direct test dependency for the Windows Job Object helper, run module tidy, regenerate/review dependency notices, and verify `go.mod`, `go.sum`, `THIRD_PARTY_NOTICES.md`, and `third_party_licenses/*`
- [X] T050 Run goimports/gofmt and resolve lint/static-analysis findings across `test/e2e/**/*.go`, `Taskfile.yml`, `.github/workflows/ci.yml`, and `.github/workflows/release.yml`
- [X] T051 Run the uncached integration tier, native race tier, ordinary untagged exclusion check, and supported-target compile checks; resolve portability failures in `test/e2e/**/*.go`, `Taskfile.yml`, `.github/workflows/ci.yml`, and `.github/workflows/release.yml`
- [X] T052 Run the full local repository gate and resolve test, lint, staticcheck, vulnerability, build, manifest, and notice failures attributable to `test/e2e/`, `Taskfile.yml`, `.github/workflows/`, `go.mod`, and generated notice artifacts
- [X] T053 Perform an adversarial review against every contract in `specs/002-e2e-recipe-harness/contracts/`, then close every safety, containment, lifecycle, live-tool-coverage, build-tag, WSL, race, support-claim, immutable-input, evidence, permission, candidate-tag, or publication-gating gap found in `test/e2e/`, `Taskfile.yml`, and `.github/workflows/`
- [X] T054 Run `graphify update .` after code changes and verify the resulting E2E, compatibility-command, WSL, race, and release-gate relationships in `graphify-out/graph.json` and `graphify-out/wiki/index.md`

**Checkpoint**: The documented integration and race commands are green, routine CI exercises four declared representatives, full-release automation qualifies all twelve cells before publication, and the full repository gate passes.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; T001–T002 start first.
- **Foundational (Phase 2)**: Depends on Setup and blocks every user story.
- **User Story 1 (Phase 3)**: Depends on T003–T014 and is the suggested MVP.
- **User Story 2 (Phase 4)**: Depends on T003–T014 but not on User Story 1 because the canonical recipe is foundational.
- **User Story 3 (Phase 5)**: Depends on T003–T014 but not on User Stories 1 or 2.
- **User Story 4 (Phase 6)**: Depends on the completed E2E foundation and scenarios because it qualifies that suite. T032–T034 may start together; T035 precedes T036–T040; T036 precedes T037 and T039–T040; T037 precedes T038; T041 precedes T042–T046; T043–T046 precede T047.
- **Polish (Phase 7)**: Depends on all four stories selected for delivery.

### User Story Dependency Graph

```text
Phase 1 Setup
      |
      v
Phase 2 Foundation
      |
      +------------------+------------------+
      v                  v                  v
US1: CLI MVP        US2: MCP stdio     US3: Extensibility
      \                  |                  /
       +-----------------+-----------------+
                         v
                US4: Release qualification
                         |
                         v
                    Phase 7 Polish
```

### Within Each User Story

- Write helper and contract specs before their implementations and confirm they initially fail for the intended reason.
- Complete typed result contracts before scenario or workflow orchestration.
- Capture pre-mutation state before invoking any mutating or safety-gated operation.
- Complete the story's focused uncached test before considering that story done.
- Keep destructive success cases last in any shared process lifecycle.
- Treat declaration, `DESIGN.md`, routine matrix, release matrix, and evidence as one consistency boundary.

### Parallel Opportunities

- T003, T004, and T005 can be authored concurrently after Setup.
- T008 and T009 are platform-specific siblings and can proceed concurrently after T007 defines their interface.
- T012 can proceed alongside late recipe materialization work once its snapshot inputs are agreed.
- After T014, US1, US2, and US3 can proceed in parallel because they consume the same completed foundation.
- Within US2, T019 and T020 can proceed concurrently; after T022, T023–T025 can proceed concurrently in separate files.
- T032–T034 can proceed concurrently because they define separate declaration, command/evidence, and workflow contracts.
- T037 establishes the Linux-side source provisioner before T038 wires that helper into WSL orchestration.
- T043, T044, and T046 can proceed in parallel after T041–T042; T045 follows the release workflow shape in T044.

## Parallel Execution Examples

### Foundation

```text
Task T003: Environment isolation specs in test/e2e/environment_spec_test.go
Task T004: Recipe validation specs in test/e2e/recipe_validation_spec_test.go
Task T005: Process lifecycle specs in test/e2e/process_spec_test.go
```

### User Stories After Foundation

```text
Track A: T015–T018, real CLI MVP
Track B: T019–T027, complete MCP stdio matrix
Track C: T028–T031, recipe extensibility proof
```

### Release Qualification

```text
Task T032: Declaration and WSL matrix specs
Task T033: Compatibility command and evidence specs
Task T034: Workflow, race, permission, and tag contract specs

After declaration types are stable:
Task T037: Linux/macOS/WSL source provisioner

After T037:
Task T038: Native Windows/WSL host provisioner and execution
```

## Implementation Strategy

### MVP First: User Story 1

1. Complete T001–T014 to establish the safe reusable foundation.
2. Complete T015–T018 for the CLI executable boundary.
3. Stop and validate the focused CLI story independently.
4. The MVP catches startup, Cobra wiring, config loading, exit-code, JSON, persistence, and working-tree regressions.

### Incremental Delivery

1. **Foundation**: Safe recipe/process infrastructure with a verified integration-only build boundary.
2. **US1**: CLI black-box protection and five-run semantic determinism.
3. **US2**: Complete live MCP stdio boundary and safety matrix.
4. **US3**: Declarative scenario extensibility proof.
5. **US4**: Closed support declaration, reusable provision/evidence interface, four-environment routine CI, race job, and exhaustive pre-publication release qualification.
6. **Polish**: Contributor entry points, dependency notices, adversarial review, and full quality gates.

## Notes

- `[P]` means different files and no dependency on another incomplete sibling task; it does not override phase dependencies.
- Every Go file under `test/e2e` carries the integration constraint, including command and internal-package files; platform files also carry their OS constraint.
- Every subprocess interaction remains 30 seconds or less even though enclosing Go test and workflow timeouts are longer.
- The current MCP tool count is deliberately not encoded; live `tools/list` names are the contract.
- The in-process #301 contract suite remains authoritative for exhaustive malformed-input/schema permutations and is not duplicated here.
- Routine CI is representative only: exactly one declared cell per Linux, macOS, native Windows, and WSL environment.
- A full release requires exactly one successful result for all twelve declared environment × Git-minor cells before any publishing credential or distribution mutation is used.
- An unavailable pinned Git patch or WSL rootfs is a failed cell, not an automatic support-matrix reduction or platform substitution.
- A failed candidate tag is immutable; only transient infrastructure may rerun the unchanged tag/commit, while any corrective change uses a new version tag.
- Hosted GitHub/network fixtures remain specification 003 / issue #329 and are excluded from these tasks.
- No task changes production RepoKeeper behavior or weakens its working-tree guardrails.
