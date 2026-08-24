# Implementation Plan: Recipe-Driven End-to-End Test Harness

**Branch**: `test/e2e-recipe-harness` | **Date**: 2026-08-23 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-e2e-recipe-harness/spec.md`

## Summary

Add an integration-tagged Go harness that strictly decodes version-controlled JSON recipe artifacts from the repository's test area, materializes deterministic Git workspaces beneath unique temporary roots, builds or selects the real RepoKeeper executable once per suite, and exercises CLI and MCP behavior through separate operating-system process scenarios. The CLI scenario covers clean, dirty, and missing repositories. The MCP scenario negotiates a real stdio session and couples valid-input and safety cases to the live `tools/list` result so every registered tool is exercised. Exact environment construction, path containment, deadlines, output capture, and post-operation Git/filesystem assertions are shared by all scenarios. A reusable test-internal compatibility command consumes a closed rolling three-minor declaration, provisions exact Git versions, and drives representative Linux/macOS/native-Windows/WSL CI plus an exhaustive twelve-cell release gate before publication.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Standard library (`os/exec`, `context`, `path/filepath`, `encoding/json`); Ginkgo v2 + Gomega; existing `github.com/mark3labs/mcp-go` v0.58.0 client; existing transitive `golang.org/x/sys` for Windows job ownership if required; provisioned Git CLI

**Storage**: Version-controlled JSON recipe artifacts under `test/e2e/testdata/recipes/`; a version-controlled Git compatibility declaration under `test/e2e/testdata/`; disposable filesystem trees; local bare Git repositories; RepoKeeper YAML configuration and registry files

**Testing**: Integration-tagged Ginkgo v2/Gomega suite invoked by `go -C tools tool task test-integration`; strict recipe and compatibility-declaration validation specs; a negative build-boundary check proving untagged commands exclude `test/e2e`; a native Linux `go test -race` lifecycle job; representative non-fail-fast pull-request CI; exhaustive non-fail-fast full-release environment × Git-minor qualification before publication

**Target Platform**: Linux (`ubuntu-24.04`), macOS (`macos-15`), native Windows (`windows-2025`), and Ubuntu 24.04 under WSL1 on `windows-2025`; labels are explicit rather than floating `*-latest` aliases

**Project Type**: Go CLI test infrastructure; no production behavior change

**Performance Goals**: Each scenario-level child-process interaction has a deadline of 30 seconds or less; each compatibility download/build/import/cross-compile operation has a 10-minute deadline; each CI job has a 30-minute timeout; the binary is built or explicitly selected once per suite; five equivalent recipe runs produce stable normalized outcomes

**Constraints**: End-to-end scenarios are offline after declared toolchain inputs have been provisioned; no test credentials or hosted repositories; all writes beneath a per-scenario root; exact test-owned Git and RepoKeeper environment; no shell command construction inside the harness; stdout/stderr captured independently; deterministic child shutdown; every E2E Go file requires `integration` and platform files use compound constraints; the closed support claim contains the current Git minor plus two preceding minors only; release artifacts and distribution publication cannot begin until every claimed environment × Git-minor cell passes

**Scale/Scope**: One reusable recipe vocabulary, one representative CLI topology, all tools returned by live MCP discovery, and explicit refusal/success coverage for safety-sensitive tools

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle or constraint | Design response | Result |
| --- | --- | --- |
| Explicit and reconstructible state (I–III, VIII) | Version-controlled JSON recipes declare stable topology and expectations; strict typed validation precedes materialization; volatile output is normalized explicitly. | Pass |
| Explainable operation (VI, IX) | Process failures retain operation, arguments, exit status, stdout, stderr, timeout state, and scenario paths. | Pass |
| Read-only degradation and safe VCS behavior (VII, X) | CLI scan/status assertions prove clean and dirty worktrees are unchanged; destructive MCP refusal and confined success are mandatory cases. | Pass |
| Git-first, honest scope (XI) | Initial recipes use real local Git remotes only; hosted remotes and other VCS backends are explicitly deferred; the executable compatibility declaration drives CI, and full releases qualify every claimed environment × Git-minor cell. | Pass |
| CLI/MCP machine-readable parity (V, XII) | The executable is tested as a CLI process and a separate MCP stdio process; structured fields and stream separation are asserted. | Pass |
| Go, Ginkgo, and meaningful tests | The harness is Go, uses Ginkgo/Gomega, and lives in the existing integration tier. | Pass |
| Race-enabled CI | Concurrency and process-lifecycle coverage runs under `go test -race` on a representative native Linux compatibility cell; any race fails CI. | Pass |
| Supported environments, including WSL | The executable declaration includes Linux, macOS, native Windows, and WSL. WSL executes Linux test and RepoKeeper binaries against Git inside an imported Ubuntu root filesystem. | Pass |
| Minimal dependencies | Existing mcp-go is reused for protocol behavior; existing x/sys may become a direct test dependency for Windows process-tree cleanup, without adding a new module/version. | Pass |
| Documentation | Harness invocation, guarantees, and diagnostics are documented in `test/e2e/README.md` and linked from contributor-facing README content if needed. | Pass |
| Architecture decisions | The harness is test-only and reversible; no production boundary or hard-to-reverse decision is introduced, so no ADR is required. | Pass |

The post-design check found no constitutional violations. The complexity table is intentionally omitted.

## Project Structure

### Documentation (this feature)

```text
specs/002-e2e-recipe-harness/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── execution-contract.md
│   ├── git-compatibility-contract.md
│   ├── mcp-tool-matrix.md
│   └── recipe-contract.md
└── tasks.md                 # Created later by speckit-tasks, not by this plan
```

### Source Code (repository root)

```text
test/e2e/
├── e2e_suite_test.go        # integration build tag, suite build/select/cleanup lifecycle
├── build_tags_spec_test.go # integration/default-tier source boundary contract
├── recipe_test.go           # typed JSON recipe loader, validation, materialization
├── recipe_validation_spec_test.go
├── recipes_test.go          # canonical recipe loading/materialization modes
├── recipes_extension_test.go
├── recipe_extension_spec_test.go
├── internal/
│   └── compatibility/
│       ├── declaration.go  # tagged strict model, expansion, docs agreement
│       ├── provision_unix.go    # integration && !windows source provisioner
│       ├── provision_windows.go # integration && windows MinGit/WSL host provisioner
│       └── evidence.go     # tagged version/evidence exact-set gates
├── cmd/
│   └── compatibility/
│       └── main.go         # tagged reusable workflow/test interface
├── compatibility_spec_test.go
├── compatibility_command_spec_test.go
├── workflow_contract_spec_test.go
├── environment_test.go      # exact isolated Git/RepoKeeper process environment
├── environment_spec_test.go
├── process_test.go          # bounded CLI execution and diagnostic capture
├── process_spec_test.go
├── process_unix_test.go     # Unix process-group ownership and termination
├── process_windows_test.go  # Windows Job Object ownership and termination
├── mcp_client_test.go       # manually owned child + SDK NewIO protocol client
├── mcp_client_spec_test.go
├── mcp_contract_test.go
├── assertions_test.go       # Git, registry, filesystem, and normalized-output checks
├── cli_contract_test.go
├── cli_spec_test.go         # clean/dirty/missing black-box CLI workflow
├── determinism_spec_test.go
├── mcp_spec_test.go         # initialize, tools/list coverage equality, case execution
├── mcp_cases_test.go        # one canonical success case per live tool plus safety cases
├── mcp_cases_read_test.go
├── mcp_cases_mutation_test.go
├── mcp_cases_safety_test.go
├── testdata/
│   ├── git-compatibility.json # executable environment/Git-minor claim and exact patches
│   └── recipes/
│       ├── canonical.json     # shared CLI/MCP topology and expected identities
│       └── extension.json     # second topology proving recipe extensibility
└── README.md                # invocation, isolation guarantees, failure diagnosis

Taskfile.yml                 # narrow integration task to tagged engine + E2E packages
.github/workflows/ci.yml     # representative pinned compatibility selection for PRs/main
.github/workflows/release.yml # exhaustive compatibility qualification gates publication
.github/workflows/release-please.yml # candidate tags without pre-gate release publication
README.md                    # optional concise contributor link to test/e2e/README.md
DESIGN.md                    # human-readable compatibility policy linked to executable data
RELEASE.md                   # qualification, evidence, and immutable-tag recovery procedure
```

**Structure Decision**: Keep all harness code test-only under `test/e2e`. Every Go file in that tree starts with `//go:build integration`; Unix and Windows files use `integration && !windows` and `integration && windows`. Portable recipe artifacts live under `test/e2e/testdata/recipes/` as JSON and are decoded into test-internal typed Go models before validation and materialization. Compatibility logic that workflows need lives in a tagged non-test internal package and command, invoked as `go run -tags integration ./test/e2e/cmd/compatibility ...`; workflow YAML only wires its machine-readable outputs and does not duplicate matrix or evidence logic. Machine-specific absolute paths exist only in runtime materializations.

## Compatibility and Provisioning Design

- `test/e2e/testdata/git-compatibility.json` declares a closed rolling window of exactly three Git minor lines: the current upstream minor and the two immediately preceding minors. At planning time these are `2.53`, `2.54`, and `2.55`; a future Git release changes no claim until a pull request updates the declaration and `DESIGN.md` together. “Minimum” means the lowest member of this closed set, not every omitted version above it.
- The full matrix is four environments by three minors: Linux on `ubuntu-24.04`, macOS on `macos-15`, native Windows on `windows-2025`, and WSL1 hosted on `windows-2025`. Routine CI marks one exact declared patch per environment as representative; releases run all twelve cells.
- Linux, macOS, and WSL obtain Git from the official kernel.org source archive pinned by URL and SHA-256, build it into a test-owned prefix, prepend only that prefix to the isolated `PATH`, and verify the exact reported version. Native Windows obtains the corresponding official Git-for-Windows MinGit archive pinned by release URL and SHA-256 and extracts it into a test-owned prefix. Package-manager `latest`, hosted-runner Git, and silent fallback are prohibited.
- The WSL cell imports Canonical's versioned Ubuntu 24.04 WSL root filesystem at `wsl/releases/noble/20240423/ubuntu-noble-wsl-amd64-24.04lts.rootfs.tar.gz` with SHA-256 `2a790896740b14d637dbdc583cce1ba081ac53b9e9cdb46dc09a2f73abbd9934` via `wsl --import --version 1`; no preinstalled distribution is assumed. The declaration also pins any missing Git build-prerequisite packages to signed, timestamped Ubuntu snapshot URLs and SHA-256 values rather than running an unconstrained package update. The Windows host cross-compiles the Linux compatibility helper, RepoKeeper, and E2E test binaries with `CGO_ENABLED=0`, places them in the imported environment, uses the helper there to build/verify Linux Git, and runs the E2E binary inside WSL. The suite's binary-selection hook accepts only that explicit test-owned RepoKeeper binary for this mode.
- The compatibility command exposes machine-readable `matrix --scope routine|release`, `provision`, `verify-version`, `write-evidence`, `verify-evidence`, and `verify-docs` operations. It validates declaration and documentation agreement before expansion, never prints credentials, and supplies stable cell keys to workflow matrices and artifacts.
- A dedicated native `ubuntu-24.04` race job runs the concurrency and process-lifecycle packages with `-race`, `-count=1`, and an explicit timeout against a declared representative Git cell. Release cells do not duplicate the race job.
- A version tag starts qualification but is not a published release. Transient infrastructure failure may rerun the unchanged tag and commit. A product, test, compatibility, or declaration change leaves the failed tag untouched and proceeds under a new semantic version tag.

## Delivery Sequence

1. Establish the integration-tagged suite lifecycle, build the binary once, and create exact isolated process environments.
2. Implement strict JSON recipe loading, validation, and deterministic materialization of local bare remotes, checkouts, commits, upstreams, metadata, dirty files, and missing registry entries.
3. Add reusable bounded process execution, output capture, normalization, and Git/filesystem assertion helpers.
4. Add a fresh CLI instance of the canonical recipe and prove scan/status exit-code, structured-output, registry, HEAD, and worktree invariants.
5. Add a separately materialized MCP instance seeded only with the deliberate missing registry entry, then exercise real stdio initialization, live tool coverage, canonical cases, and safety paths in a deterministic order.
6. Add and validate the reusable compatibility package/command, the closed three-minor declaration, exact platform provisioners, WSL import/execution, and the corresponding `DESIGN.md` matrix.
7. Keep pull-request/main integration representative across all four environments, add the native Linux race job, and generate a non-fail-fast twelve-cell release matrix with exact evidence and a publisher dependency on the completeness gate.
8. Add repeated-run determinism coverage and contributor/release documentation, verify candidate-tag recovery semantics, then run the integration suite and full local CI gate.

## Risk and Rollback Boundaries

- Harness construction is isolated from production packages. Each delivery step can be reverted by removing `test/e2e` and its documentation without changing RepoKeeper runtime behavior.
- The pull-request/main integration job expands to a representative non-fail-fast Linux/macOS/native-Windows/WSL matrix using explicit OS-version labels. To contain routine cost, it targets only the integration-tagged packages (`internal/engine` and `test/e2e`) instead of rerunning every ordinary package.
- The tag-triggered release workflow first expands all twelve declared compatibility cells and runs the same E2E package with `fail-fast: false`. Each cell verifies the exact provisioned Git patch and uploads a result record; the publishing job depends on a completeness gate that rejects failures, skips, duplicates, missing cells, and undeclared substitutions.
- Qualification jobs receive read-only repository permissions, no publishing credentials, and explicit timeouts. Write permissions, identity tokens, package access, and release secrets remain scoped to publishing jobs that depend on the successful completeness gate.
- Compatibility claims and exact patch selections change through reviewable repository updates. If a patch can no longer be provisioned, the release fails; reducing the claim requires an explicit documentation and declaration change rather than an automatic skip.
- Exact provisioning is centralized in the compatibility command. If runner prerequisites or WSL1 import support disappear, the affected cell fails visibly; workflows do not fall back to runner Git, another WSL version, or a native Windows execution.
- Release-candidate tags are immutable. Rerunning unchanged infrastructure is recoverable; correcting source or compatibility data crosses the rollback boundary and requires a new version tag.
- Destructive tool tests resolve and validate their destination under the scenario root immediately before invocation and verify refusal state before allowing the confirmed call.
- The MCP case map is keyed by discovered tool name. Set equality fails before tool invocation if registration and coverage drift, preventing silent partial coverage.
- CLI and MCP use separate materializations of the same recipe so CLI history cannot weaken MCP mutation assertions.
- The harness owns the MCP `exec.Cmd` and pipes, then attaches mcp-go through `transport.NewIO`. This avoids the convenience constructor's unbounded background process context and permits validation of every recorded stdout frame.
