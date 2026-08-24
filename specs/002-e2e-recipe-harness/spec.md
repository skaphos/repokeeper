# Feature Specification: Recipe-Driven End-to-End Test Harness

**Feature Branch**: `test/e2e-recipe-harness`

**Created**: 2026-08-23

**Status**: Draft

**Input**: User description: "Create a separate issue and feature for recipe-driven end-to-end tests that scaffold disposable repositories and exercise an actual RepoKeeper run in an isolated temporary workspace. See issue #327."

## Clarifications

### Session 2026-08-23

- Q: Does one automated scenario need to share a workspace between CLI and MCP execution, or may one suite contain separate isolated scenarios? → A: One automated suite contains separate isolated CLI and MCP scenarios.
- Q: Which Git versions and platforms define compatibility for the end-to-end harness? → A: Use the Git CLI compatibility matrix in `DESIGN.md`.
- Q: Where should generated recipe identity data—names, relative paths, and relationships—be persisted? → A: Store a portable version-controlled artifact in the repository's test area, outside the feature specification; generate absolute temporary paths only at runtime.
- Q: How much of the declared Git compatibility matrix must run before a full release? → A: Exercise every claimed environment and Git minor-version combination, using at least one exact patch release from each claimed minor line, before publication.
- Q: Does the compatibility claim include WSL? → A: Yes. WSL is a distinct declared environment and must be exercised in both representative and exhaustive qualification.
- Q: Does a minimum Git version imply support for every newer minor? → A: No. The claim is a closed rolling set of three explicitly declared Git minor lines; unlisted minors, including newer ones, are not qualified until added by review.
- Q: What happens when qualification fails after a version tag is created? → A: The tag is an immutable release-candidate trigger, not a published release. Infrastructure failures may rerun the same tag and commit, but code, compatibility, or declaration fixes require a new version and tag.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Verify Real CLI Workflows Safely (Priority: P1)

A RepoKeeper maintainer can describe a representative workspace containing healthy, dirty, and missing repositories, run the packaged command-line behavior against that workspace, and verify both the reported results and the resulting repository state without touching real user data.

**Why this priority**: Package and in-process tests cannot detect failures in executable startup, command wiring, configuration discovery, process environment handling, exit codes, or interactions with real repository metadata. This is the primary confidence gap the harness must close.

**Independent Test**: Materialize one disposable workspace containing a clean checkout, a dirty checkout, local remotes, and a missing registry entry; run discovery and status commands through the actual executable; then verify structured output, exit status, registry state, commit identity, and worktree contents.

**Acceptance Scenarios**:

1. **Given** an isolated workspace with clean and dirty repositories backed by local remotes, **When** RepoKeeper scans the workspace, **Then** every repository is discovered once, machine-readable output is valid, and the dirty worktree and checked-out commits are unchanged.
2. **Given** a registry entry whose path no longer exists beneath the scanned workspace, **When** RepoKeeper scans and reports status, **Then** the entry is reported as missing, the documented warning or error exit status is returned, and no unrelated repository state is changed.
3. **Given** host-level RepoKeeper and version-control configuration exists, **When** the end-to-end workflow runs, **Then** only the test-owned configuration and repositories influence the result.

---

### User Story 2 - Verify MCP Over a Real Process Boundary (Priority: P2)

A RepoKeeper maintainer can launch the packaged MCP server as a subprocess, negotiate the protocol over standard input and output, discover the registered tools, and exercise every registered tool against disposable repository state.

**Why this priority**: In-process MCP coverage verifies handlers and response contracts but cannot catch failures in standard-stream framing, process lifecycle, executable arguments, configuration discovery, or accidental diagnostic output on the protocol stream.

**Independent Test**: Launch RepoKeeper's MCP mode against a materialized recipe, complete initialization, list tools, invoke every returned tool through the standard-stream transport with valid inputs, exercise required confirmation or refusal paths for safety-sensitive tools, and verify each structured result or state transition before shutting the process down cleanly.

**Acceptance Scenarios**:

1. **Given** a valid isolated RepoKeeper workspace, **When** an MCP client initializes through the executable's standard streams, **Then** protocol negotiation succeeds and the server identifies itself correctly.
2. **Given** an initialized MCP session, **When** the client lists tools, **Then** every returned tool has a corresponding end-to-end case and the suite fails if a registered tool is omitted.
3. **Given** the complete registered tool set, **When** the client invokes every tool with valid inputs, **Then** every tool succeeds over the real standard-stream transport and its structured response or resulting workspace state matches the recipe.
4. **Given** a tool that requires confirmation or can perform destructive mutation, **When** its safety precondition is absent, **Then** the tool refuses without changing repository state, and **When** the precondition is satisfied, **Then** the expected change remains confined to the disposable workspace.
5. **Given** the MCP workflow completes or fails, **When** the test ends, **Then** the child process and all standard-stream readers terminate within a bounded time and useful diagnostics are retained on failure.

---

### User Story 3 - Add Scenarios Without Rebuilding the Harness (Priority: P3)

A RepoKeeper maintainer can add another repository topology or expected outcome by declaring a new recipe and its assertions, while reusing the same workspace construction, process isolation, execution, timeout, and diagnostic behavior.

**Why this priority**: The harness only remains valuable if new regression scenarios are inexpensive to express and do not accumulate subtly different copies of repository setup code.

**Independent Test**: Generate and check in a second small portable recipe artifact using the shared fixture vocabulary, then verify it can be materialized and executed without duplicating repository initialization or subprocess management.

**Acceptance Scenarios**:

1. **Given** the shared harness exists, **When** a maintainer generates and checks in a portable recipe artifact defining a new combination of repositories, remotes, worktree state, and missing entries, **Then** the scenario uses the existing lifecycle and isolation behavior without custom bootstrap commands.
2. **Given** a recipe is invalid or internally inconsistent, **When** it is materialized, **Then** the test fails before RepoKeeper runs and identifies the invalid recipe field or relationship.
3. **Given** RepoKeeper exits unexpectedly or emits invalid structured output, **When** the scenario fails, **Then** the report includes the invoked operation, exit status, captured output, and relevant workspace paths.

---

### User Story 4 - Qualify Full Releases Against the Declared Git Matrix (Priority: P4)

A RepoKeeper releaser can create a release-candidate tag and receive conclusive compatibility evidence for every claimed Git minor on Linux, macOS, native Windows, and WSL before any release artifact or distribution channel is published.

**Why this priority**: The harness provides confidence only for environments that actually execute it. Release qualification turns the documented support claim into a complete, reviewable gate while preserving a smaller representative matrix for routine feedback.

**Independent Test**: Expand the version-controlled compatibility declaration, provision and verify every exact Git patch, run the same E2E suite for every declared cell without fail-fast cancellation, and prove the publisher remains blocked until an exact-set evidence check succeeds.

**Acceptance Scenarios**:

1. **Given** the closed compatibility declaration, **When** routine pull-request or main validation runs, **Then** one exact declared representative for Linux, macOS, native Windows, and WSL is exercised and reported.
2. **Given** a release-candidate tag, **When** release qualification expands the declaration, **Then** every declared environment and Git-minor cell runs, including WSL, and each result records the expected and actual exact Git version.
3. **Given** any cell fails, is skipped, cannot provision its pinned inputs, reports a version mismatch, or lacks unique evidence, **When** the completeness gate evaluates the run, **Then** all publication jobs remain blocked.
4. **Given** every declared cell has exactly one successful matching result, **When** the completeness gate succeeds, **Then** the publisher may create release artifacts and update distribution channels with publishing authority unavailable to qualification jobs.
5. **Given** qualification exposes a product or compatibility defect after tagging, **When** maintainers correct it, **Then** the failed tag remains immutable and the corrected commit is qualified under a new version tag; a transient infrastructure retry may rerun only the unchanged tag and commit.

### Edge Cases

- A health command can intentionally return a documented non-zero status for dirty or missing repositories; the harness must distinguish that expected domain result from failure to start or execute RepoKeeper.
- A subprocess can hang, close one standard stream early, or leave a reader waiting after exit; every process interaction needs a deadline and deterministic cleanup.
- Repository names and paths can contain spaces or platform-specific separators; recipes must not rely on shell tokenization or hard-coded Unix paths.
- A fixture can reference an absolute path, parent traversal, or destination outside the disposable workspace; materialization must reject it before writing.
- The host can define global signing, identity, default-branch, credential, proxy, or RepoKeeper settings; fixture creation and execution must not inherit behavior that makes results user-specific.
- A local remote or checkout can fail to initialize; diagnostics must identify the exact fixture operation and preserve enough output to reproduce the failure.
- Structured output can be syntactically valid while missing required fields; assertions must validate the documented response shape, not merely parseability.
- Cleanup can run after a partial setup or failed assertion; all artifacts and child processes must remain bounded to the test-owned lifecycle.
- A claimed Git minor can become unavailable from its configured distribution source; release qualification must fail with the affected matrix cell identified rather than skip it or publish with partial coverage.
- A hosted Windows image can lose WSL support or fail to import the pinned Linux root filesystem; the WSL cell must fail rather than silently fall back to native Windows or disappear from the matrix.
- A newer Git minor can be released while RepoKeeper still declares an older closed support set; the new minor remains unclaimed until a reviewed declaration and documentation update adds it and retires another line.
- A release-candidate tag can fail qualification after it exists; retries may not move or recreate the tag at a different commit, and code fixes require a new version tag.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Maintainers MUST be able to produce and version-control a portable workspace recipe artifact in the repository's test area, outside the feature specification, containing stable repository names, relative logical paths, relationships, remotes, initial content, branch and tracking relationships, worktree state, and missing registry entries; machine-specific absolute paths MUST be generated only during materialization and MUST NOT be persisted in the artifact.
- **FR-002**: The harness MUST materialize each recipe beneath one unique disposable root and MUST reject recipe paths that escape that root.
- **FR-003**: Repository fixtures MUST use local resources only; executing an end-to-end scenario MUST NOT require network access, credentials, hosted repositories, or external services.
- **FR-004**: Fixture creation MUST use an explicit test identity and MUST be independent of host-level signing, credential, default-branch, and repository configuration.
- **FR-005**: RepoKeeper execution MUST use isolated home, configuration, cache, and temporary locations and MUST NOT read or modify a real user configuration or repository.
- **FR-006**: The harness MUST build or otherwise select the actual RepoKeeper executable once for an end-to-end suite and MUST exercise it as an operating-system process rather than calling command handlers directly.
- **FR-007**: The initial CLI workflow MUST scan and inspect a recipe containing at least one clean checkout, one dirty checkout, their local remotes, and one missing registry entry.
- **FR-008**: CLI assertions MUST cover process exit status, standard output, standard error, structured response fields, persisted registry/configuration state, checked-out commit identity, and worktree contents.
- **FR-009**: The harness MUST allow documented warning and health-error exit statuses to be declared as expected outcomes without masking launch errors, timeouts, crashes, or malformed output.
- **FR-010**: The initial MCP workflow MUST launch the actual RepoKeeper MCP process, initialize a protocol session, list the registered tools, and successfully invoke every registered tool against disposable recipe state.
- **FR-011**: The end-to-end MCP case matrix MUST be checked against the live tool-discovery result so that adding a registered tool without adding an end-to-end case fails the suite.
- **FR-012**: Each registered MCP tool MUST have at least one valid-input success case that verifies its structured response and, when applicable, its resulting registry, repository, or filesystem state.
- **FR-013**: Tools with confirmation gates or destructive behavior MUST additionally be exercised without the required safety precondition, MUST refuse without mutation, and MUST only perform their success-path mutation inside the disposable workspace.
- **FR-014**: MCP assertions MUST verify that protocol responses remain on the protocol stream, diagnostic output does not corrupt framing, and every tool result reflects the recipe workspace.
- **FR-015**: Every scenario-level RepoKeeper, fixture-Git, and MCP subprocess interaction MUST have a deadline of 30 seconds or less, deterministic shutdown behavior, and captured diagnostics that are reported when the scenario fails. Compatibility download, build, import, and cross-compilation operations MUST also be explicitly bounded, but MAY use a longer documented provisioning deadline.
- **FR-016**: Recipe validation failures MUST identify the invalid field or relationship and MUST occur before any path outside the disposable root can be affected.
- **FR-017**: Common fixture construction, environment isolation, command execution, and failure reporting MUST be reusable across scenarios without copying bootstrap logic.
- **FR-018**: The end-to-end suite MUST run in the repository's dedicated integration-test tier and MUST be documented with its invocation, scope, isolation guarantees, and failure-diagnosis guidance.
- **FR-019**: The faster in-process contract suite from issue #301 MUST remain the detailed schema and malformed-input layer, but it MUST NOT substitute for real-process end-to-end execution of any registered tool.
- **FR-020**: Adding this harness MUST NOT change production RepoKeeper behavior or weaken the guardrail that RepoKeeper does not unexpectedly modify working trees.
- **FR-021**: The supported Git compatibility declaration MUST define a closed rolling set of exactly three explicitly named Git minor lines, enumerate every claimed environment and Git-minor combination in a version-controlled form that automation can consume, and pin one exact Git patch version plus integrity-checked provisioning inputs for each cell. Unlisted Git minors MUST NOT be described as supported or inferred from a minimum version.
- **FR-022**: The declared environments MUST include Linux, macOS, native Windows, and WSL; routine validation MUST exercise at least one declared exact-patch representative for each environment.
- **FR-023**: Before publishing a full release, automation MUST run the end-to-end suite for every declared compatibility cell, MUST report the exact environment and Git patch version exercised for each cell, and MUST prevent publication when any cell fails, is skipped, is duplicated, lacks evidence, or cannot be provisioned.
- **FR-024**: Compatibility matrix expansion, exact version verification, evidence creation, evidence completeness checking, and documentation agreement MUST be exposed through one reusable executable interface consumed by tests and workflows rather than reimplemented in workflow expressions.
- **FR-025**: Every end-to-end Go source file MUST remain excluded from the ordinary untagged build and test tier, and platform-specific files MUST require both the integration tier and their operating-system constraint.
- **FR-026**: Continuous integration MUST exercise concurrency and lifecycle-sensitive harness behavior under automated data-race detection and fail on any detected race.
- **FR-027**: A version tag MUST be treated as an immutable release-candidate trigger. Publication MUST occur only after compatibility qualification; transient infrastructure retries MAY reuse the unchanged tag and commit, while any source or compatibility-declaration correction MUST use a new version tag.

### Key Entities

- **Workspace Recipe**: A generated, version-controlled portable artifact in the repository's test area, outside the feature specification, that declares stable repository names, relative logical paths, relationships, topology, initial state, registry state, operation to exercise, and expected outcomes for one end-to-end scenario. Materialization resolves its relative paths beneath a unique runtime root.
- **Repository Fixture**: A test-owned repository with its local remote, commits, branches, tracking relationship, checkout state, and optional uncommitted changes.
- **Materialized Workspace**: The unique disposable directory tree produced from a recipe, including isolated user/configuration locations and paths to every fixture.
- **Execution Result**: The exit status, standard output, standard error, timing, and termination state of one RepoKeeper process invocation.
- **Expected Outcome**: The declared structured-output, registry, filesystem, and repository invariants against which an execution result is evaluated.
- **Compatibility Cell**: One claimed environment and Git minor-version combination, including the exact patch, immutable provisioning inputs, and evidence produced by its end-to-end run. Environments are Linux, macOS, native Windows, and WSL.

### Scope Boundaries

- The first delivery covers Git repositories only; experimental alternate version-control backends are not part of the initial harness.
- The recipe format is an internal testing contract, not a new user-facing RepoKeeper configuration format or public compatibility promise.
- The MCP scenario covers every registered tool, including the required refusal modes for confirmation-gated or destructive tools. Exhaustive combinations of equivalent optional inputs and malformed values remain in the faster contract suite.
- Hosted remotes, authentication, network-failure behavior, validation of built release archive or container contents, and interactive terminal behavior are outside the initial scope. Gating publication of release archives, containers, and distribution channels on compatibility qualification is in scope.
- Performance and large-workspace scalability tests remain separate from deterministic end-to-end correctness scenarios.
- Pull-request validation may use a smaller representative compatibility selection for fast feedback; exhaustive declared-matrix execution is mandatory for full release qualification.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: One automated suite contains separate isolated scenarios that exercise the actual CLI process and a separate actual MCP process against fully disposable workspaces and pass without contacting an external service.
- **SC-002**: One CLI workflow verifies at least two live repositories in distinct worktree states plus one missing registry entry, with 100% of created and modified paths contained beneath the scenario's disposable root.
- **SC-003**: The dirty repository's checked-out commit and uncommitted file contents are byte-for-byte identical before and after scan and status operations, and the clean repository remains clean.
- **SC-004**: The MCP workflow completes initialization and successfully invokes 100% of the tools returned by live tool discovery, with no registered tool lacking an end-to-end case.
- **SC-005**: Every scenario-level child process has an enforced deadline of 30 seconds or less; every compatibility provisioning operation completes within a 10-minute deadline; and no child process or stream reader remains active after its test or qualification cell completes.
- **SC-006**: A failed command reports its operation, arguments, exit status, standard output, and standard error without requiring a maintainer to rerun the suite in verbose mode.
- **SC-007**: Five consecutive executions from the same recipe produce equivalent repository classifications and structured results after excluding documented volatile values such as timestamps and temporary absolute prefixes.
- **SC-008**: A maintainer can add a new repository-state scenario by generating and checking in one portable recipe artifact and its expectations without duplicating executable build, environment isolation, process lifecycle, or repository bootstrap logic.
- **SC-009**: The dedicated integration-test command runs the new suite in continuous integration while the ordinary fast build and test commands neither compile nor run the end-to-end package.
- **SC-010**: Every confirmation-gated or destructive MCP tool demonstrates both non-mutating refusal without its safety precondition and a successful, expected operation confined to the disposable workspace.
- **SC-011**: Every full release records a conclusive result for 100% of the environment and Git minor-version cells declared supported, including Linux, macOS, native Windows, WSL, and the exact Git patch exercised; no release artifact or distribution channel is published unless all cells pass.
- **SC-012**: The concurrency and process-lifecycle E2E coverage completes under automated data-race detection with zero reported races in continuous integration.

## Assumptions

- Git compatibility is defined by the Git CLI Strategy & Compatibility Matrix in `DESIGN.md` and its version-controlled executable declaration. The declaration carries the current Git minor and the two immediately preceding minors as a closed rolling claim; adding a new line and retiring the oldest is a reviewed repository change, not a runtime inference.
- The lowest declared Git minor is the minimum only within that closed set. RepoKeeper makes no compatibility claim for omitted historical or newly released minors.
- WSL is qualified as a Linux execution environment hosted on the pinned Windows runner, using a checksum-pinned Ubuntu 24.04 root filesystem; it does not reuse the native Windows RepoKeeper or Git executable.
- A tag-triggered run creates no published release by itself. The failed tag remains immutable; only an unchanged tag/commit may be retried for transient infrastructure failure, while code or compatibility changes require a new semantic version.
- Local filesystem remotes provide sufficient fidelity for the initial executable, configuration, discovery, status, and MCP boundaries; hosted service behavior is not required.
- The existing integration-test tier is the appropriate cadence because the harness creates processes and real repositories and is intentionally heavier than unit or in-process tests.
- Existing per-tool MCP tests remain authoritative for exhaustive schema, malformed-input, and equivalent-parameter coverage; this feature still executes every registered tool through the real process boundary with a canonical success case and every required safety refusal.
- Volatile timestamps, generated temporary prefixes, and platform path separators are normalized or excluded when comparing repeated outcomes.
- Test artifacts are disposable and do not need retention after successful runs; failure output must contain enough context to diagnose a failed workspace before cleanup.
