# Phase 0 Research: Recipe-Driven End-to-End Test Harness

## Decision 1: Use a dedicated integration-tagged Ginkgo package

**Decision**: Place the black-box suite and test-internal compatibility command under `test/e2e`, guard every Go source file with the `integration` build tag, and run scenarios through the repository's `test-integration` task.

**Rationale**: RepoKeeper already uses Ginkgo/Gomega and has an integration tier. A dedicated tree makes the process boundary and runtime cost visible while excluding it from ordinary tests and coverage. Routine and release workflows can deliberately add the new four-environment matrices without changing the ordinary unit-test matrix.

**Alternatives considered**:

- Add scenarios to `internal/engine/integration_test.go`: rejected because those tests call the engine in-process and would blur black-box and component integration responsibilities.
- Use a standalone shell harness: rejected because path portability, exact argument handling, structured assertions, and process cleanup are safer and clearer in Go.
- Add a separate CI workflow: deferred because the existing integration job already provides the required cadence.

## Decision 2: Store portable JSON recipes and decode them into typed Go values

**Decision**: Store version-controlled scenario recipes as JSON under `test/e2e/testdata/recipes/` and strictly decode them into internal test structs for workspace, repository, commit, branch/upstream, dirty-file, metadata, registry, and expected-state declarations. Every artifact carries `schema_version: 1`; unknown fields and unsupported versions fail before materialization.

**Rationale**: A portable checked-in artifact keeps stable names, relative paths, and relationships reviewable outside the feature specification. JSON uses the standard library, while typed decoding and direct Gomega assertions preserve clear validation and test ergonomics. The artifact remains an internal test contract rather than a RepoKeeper feature or public format.

**Alternatives considered**:

- Go-only recipe declarations: rejected because the durable recipe identities and relationships must remain reviewable as repository test artifacts outside the feature specification.
- YAML recipes: rejected because JSON provides the required portable artifact with standard-library decoding and no additional dependency.
- Imperative setup per spec: rejected because it duplicates bootstrap logic and makes invalid topology harder to detect before writes begin.

## Decision 3: Validate paths and relationships before materialization

**Decision**: Recipe validation runs as a pure preflight before creating repositories. Logical paths must be non-empty, relative, clean, and contained beneath the scenario root after `filepath.Abs`/`filepath.Rel`; duplicate names, unknown remotes, unknown upstreams, and inconsistent missing/live declarations fail with field-qualified errors.

**Rationale**: Lexical containment alone is insufficient for absolute paths, `..`, and platform-specific separators. Relationship validation prevents half-built fixtures and directly satisfies the fail-before-run requirement. Test code creates every destination itself, so rejecting symlinked parent components is straightforward.

**Alternatives considered**:

- Rely only on `filepath.Clean`: rejected because a cleaned path may still be absolute or escape through `..`.
- Validate while executing Git commands: rejected because failures would leave ambiguous partial state and weaken the outside-root guarantee.

## Decision 4: Build the real executable once per suite

**Decision**: A suite-level Ginkgo setup resolves the module root, creates a suite temporary directory, and normally runs `go build -trimpath -o <suite-root>/bin/repokeeper[.exe] .` once with `Cmd.Dir` set to that module root. WSL qualification instead supplies an explicit test-owned, CGO-free Linux binary cross-compiled from the same commit. The suite validates that selection once; scenarios reuse the absolute binary path but receive fresh materialized roots.

**Rationale**: This exercises the actual entrypoint and Cobra wiring while avoiding repeated compilation across CLI and all MCP cases. An absolute path and platform-specific executable suffix avoid PATH ambiguity.

**Alternatives considered**:

- `go run` for every command: rejected because it obscures executable exit behavior and adds compilation latency to every assertion.
- Use an installed `repokeeper`: rejected because it may not represent the checked-out source.
- Build RepoKeeper repeatedly inside each scenario: rejected because it adds latency without increasing process-boundary coverage.

## Decision 5: Use local bare repositories and argument-vector Git commands

**Decision**: Materialize bare remotes under `<scenario>/remotes`, working checkouts under `<scenario>/workspace`, and unscanned source clones under `<scenario>/sources`. Represent remotes with correctly constructed local `file://` URLs so RepoKeeper's URL normalizer produces stable non-empty IDs on Windows as well as Unix. Invoke Git with `exec.CommandContext` and argument slices, explicitly create `main`, assign upstreams, and use deterministic file content and commit metadata.

**Rationale**: Local remotes cover clone, fetch, push, tracking, discovery, and sync without network or credentials. Separating remotes and source clones from the scan root avoids accidental discovery. Argument vectors handle spaces and platform separators without shell tokenization.

**Alternatives considered**:

- Plain standalone repositories: rejected because `add_repository`, sync, tracking, and remote identity need real remotes.
- Hosted fixtures: deferred to specification 003 and issue #329 because network reliability and governance are different concerns.

## Decision 6: Construct an exact test-owned environment

**Decision**: Build child environments from a small platform-aware allowlist needed to locate executables and support Windows process startup, then set test-owned `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `APPDATA`, `LOCALAPPDATA`, temp/cache paths, `REPOKEEPER_CONFIG`, `GIT_CONFIG_GLOBAL`, and `GIT_CONFIG_NOSYSTEM=1`. Set deterministic Git identity, disable signing and credential helpers, disable prompts, and force an explicit initial branch.

**Rationale**: mcp-go's default stdio constructor appends values to `os.Environ`, which does not isolate host settings. Its pinned v0.58.0 transport provides `WithCommandFunc`; the harness can set `exec.Cmd.Env` and `Dir` exactly for the MCP child. CLI and fixture Git commands use the same environment builder.

**Alternatives considered**:

- Append overrides to `os.Environ`: rejected because duplicate environment keys are platform-sensitive and unrelated proxy, credential, or RepoKeeper settings remain inherited.
- Trust repository-local Git config alone: rejected because system/global signing, credentials, hooks, and default-branch configuration can affect repository creation and network behavior.

## Decision 7: Model process results instead of treating every non-zero exit as infrastructure failure

**Decision**: A shared runner returns an `ExecutionResult` containing operation, executable, arguments, exit code, stdout, stderr, duration, timeout flag, and launch error. Expected domain exit codes are asserted by each case; start failures, signals, timeouts, crashes, and malformed structured output remain hard failures.

**Rationale**: RepoKeeper health commands intentionally use non-zero exit codes for dirty or missing states. Preserving the distinction prevents false failures without masking process defects. A single formatted diagnostic ensures useful output is always attached to failed assertions.

**Alternatives considered**:

- Require exit zero and special-case individual specs: rejected because behavior would be duplicated and easy to misclassify.
- Parse stdout before recording process state: rejected because malformed output must still report all raw diagnostics.

## Decision 8: Own the MCP child process and attach mcp-go to its pipes

**Decision**: Create `repokeeper mcp --config <path>` with a harness-owned `exec.Cmd`, exact environment, working directory, 30-second lifetime context, platform process-tree boundary, and explicit stdin/stdout/stderr pipes. Wrap recorded stdout and stdin with mcp-go's public `transport.NewIO`, then use `client.NewClient` for initialization, pagination, and calls. Drain the real stderr separately and validate every recorded stdout frame after EOF.

**Rationale**: Reusing the SDK avoids implementing JSON-RPC, but its convenience constructor starts with `context.Background()` and owns pipes internally. Its response loop can skip non-JSON stdout, so successful calls alone do not prove protocol-stream purity. Manual process ownership provides a hard lifetime, exact environment, raw-frame recording, and joined cleanup while `NewIO` retains compatible protocol behavior.

**Alternatives considered**:

- Manually encode JSON-RPC: rejected because it adds protocol-client bugs that are unrelated to RepoKeeper.
- Use an in-process MCP client: rejected because it misses startup, stdio, environment, and lifecycle failures that define this feature.
- Use `NewStdioMCPClientWithOptions`: rejected because it starts with a background process context, hides raw stdout, and owns cleanup with fixed internal waits.
- Use `transport.NewStdioWithOptions` directly: rejected because it can accept a bounded start context but still hides raw stdout and process-tree ownership.

## Decision 9: Couple a canonical case map to live tool discovery

**Decision**: Store one `MCPToolCase` per registered tool, keyed by tool name. After `tools/list`, compare the discovered and declared name sets for exact equality before executing cases in a declared safe sequence. Each case supplies valid arguments, response assertions, and before/after state assertions. Confirmation-gated/destructive tools also provide a refusal call and a confirmed success call.

**Rationale**: Exact set equality turns tool registration into the source of truth and fails on both uncovered new tools and stale cases. Per-tool state callbacks keep the matrix declarative without pretending all responses have the same schema.

**Alternatives considered**:

- Assert only the current count of 14: rejected because renames or substitutions could preserve the count while losing coverage.
- Generate cases from schemas: rejected because schemas do not encode fixture preconditions or state assertions.

## Decision 10: Use separate CLI and MCP materializations and deterministic MCP order

**Decision**: Build the binary once, but materialize separate fresh instances of the same canonical recipe for CLI and MCP. Seed the MCP registry with only the deliberate missing entry so the server can start and `scan_workspace` has observable new entries. In one real session, run scan, read tools, plan, safety refusal/confirmed sync, labels, add, and destructive remove last.

**Rationale**: Reusing a CLI-scanned registry would make MCP scan mostly idempotent and couple assertions to earlier history. The server reloads config before each handler, so a declared safe order keeps mutations visible and deterministic. Distinct clean, dirty, missing, and add/remove fixtures prevent safety assertions from consuming later preconditions.

**Alternatives considered**:

- Restore files after every tool: rejected because Git refs and registry rollback are error-prone and could conceal mutation.
- Start one MCP process for every tool: rejected because a single initialized stdio lifecycle better validates sustained framing and is not needed when the mutation order is explicit.

## Decision 11: Compare semantic normalized outcomes five times

**Decision**: Materialize the same deterministic CLI recipe five times and compare a normalized projection: repository IDs, relative paths, status/classification, branch/tracking state, relevant output fields, and content hashes. Exclude declared volatile timestamps, durations, generated checkout IDs if path-derived, and temporary absolute prefixes.

**Rationale**: Semantic normalization proves reconstructibility without snapshotting incidental values. Re-materialization, rather than rerunning mutations on one workspace, avoids accumulated state.

**Alternatives considered**:

- Byte-compare raw JSON: rejected because timestamps and unique temporary roots are intentionally volatile.
- Repeat on one workspace: rejected because scan timestamps and registry state make the inputs different after the first run.

## Decision 12: Reuse existing modules and add no ADR

**Decision**: Reuse mcp-go and the already resolved `golang.org/x/sys` module. If the Windows Job Object helper imports x/sys directly, update its existing go.mod classification and review notices, but do not introduce another module or version. Document the test-internal design here and in `test/e2e/README.md`; do not add an ADR.

**Rationale**: The design is confined to tests, introduces no runtime behavior or public schema, and is readily reversible. It does not meet the repository's hard-to-reverse architecture threshold.

## Decision 13: Run the E2E boundary in four supported environments

**Decision**: Change integration CI to a non-fail-fast matrix covering Linux on `ubuntu-24.04`, macOS on `macos-15`, native Windows on `windows-2025`, and Ubuntu 24.04 under WSL1 on `windows-2025`. WSL is a separate environment, not an alias for Windows or Linux hosted-runner evidence. Narrow `test-integration` to `./internal/engine` and `./test/e2e`, and retain a generous Go test-binary timeout distinct from the per-child 30-second deadline.

**Rationale**: Executable suffixes, stdio shutdown, environment-key casing, file locking, path separators, paths with spaces, and WSL interoperation are exactly the boundary under test. GitHub documents that floating `*-latest` aliases migrate to newer OS images, so explicit labels prevent unreviewed operating-system migrations. The Windows 2025 runner manifest documents WSL1 availability but no installed distribution; importing a pinned root filesystem makes that precondition explicit and reproducible. Package targeting avoids repeating the ordinary test suite that already has its own OS matrix.

**Alternatives considered**:

- Keep integration Linux-only: rejected because it would leave the highest-risk portability behavior unexecuted.
- Run `go test -tags integration ./...` in every environment: rejected because it needlessly repeats all ordinary packages already covered by the test matrix.
- Keep `ubuntu-latest`, `macos-latest`, and `windows-latest`: rejected because GitHub migrates those aliases and a single run can otherwise change the operating-system baseline without a repository change.
- Treat WSL as covered by native Windows or hosted Linux: rejected because WSL has a distinct process, path, filesystem, and executable boundary.

**Primary sources**: [GitHub runner image label and migration policy](https://github.com/actions/runner-images/blob/main/README.md), [Windows 2025 runner manifest](https://github.com/actions/runner-images/blob/main/images/windows/Windows2025-Readme.md), [Microsoft WSL distribution import documentation](https://learn.microsoft.com/en-us/windows/wsl/use-custom-distro), [Canonical Ubuntu WSL root filesystems](https://cloud-images.ubuntu.com/wsl/releases/noble/current/).

## Decision 14: Separate representative CI from exhaustive release qualification

**Decision**: Maintain a version-controlled, strictly validated compatibility declaration under `test/e2e/testdata/` as the executable source for a closed rolling three-minor Git support claim. At planning time the claim is Git `2.53`, `2.54`, and `2.55`; a newly released minor remains unclaimed until a pull request explicitly adds it and retires the oldest line. Every minor is declared for Linux, macOS, native Windows, and WSL, yielding twelve release cells. Pull-request and main CI select one exact declared patch per environment. The release workflow runs all twelve cells with `fail-fast: false`, then completes an exact-set evidence gate before the existing GoReleaser publisher may start.

**Rationale**: A numeric minimum normally implies an open-ended range, which the release matrix cannot prove when new Git minors appear. A closed three-line set is precise, bounded, and reviewable: its lowest member can be described as the minimum only within the declared set. Minor-granularity qualification makes the claim falsifiable without requiring every patch release. Pinning exact patches prevents package-manager updates from silently changing evidence. Keeping the exhaustive twelve-cell matrix at release cadence preserves useful pull-request latency, while placing it before GoReleaser matters because the current publisher updates GitHub, container, and Homebrew channels as one unit.

The release evidence for each cell contains the declared OS label, claimed Git minor, exact expected patch, actual `git --version`, source revision, test result, and artifact identity. A final completeness check requires exactly one successful result for every declared cell and rejects unexpected, duplicated, skipped, or missing cells. Failure to download, install, or verify a declared Git patch is a failed compatibility cell, not permission to continue publishing.

**Alternatives considered**:

- Run the exhaustive matrix on every pull request: rejected because it multiplies process-heavy E2E work across every supported Git minor when a representative PR matrix provides faster feedback.
- Test only each platform minimum and hosted-runner current version at release: rejected because it leaves the intervening minor versions unverified while continuing to claim them.
- Resolve an unpinned latest patch within each minor at runtime: rejected because two releases from the same commit could exercise different Git binaries and supply-chain inputs.
- Publish first and run compatibility qualification afterward: rejected because a failing cell would be discovered only after immutable release channels had already moved.

## Decision 15: Centralize matrix, provisioning, and evidence behavior in a tagged command

**Decision**: Implement reusable compatibility logic in `test/e2e/internal/compatibility` and expose it through `test/e2e/cmd/compatibility`. Every Go file under `test/e2e` carries the `integration` build constraint, so workflows invoke the command with `go run -tags integration`. The command provides machine-readable `matrix --scope routine|release`, `provision`, `verify-version`, `write-evidence`, `verify-evidence`, and `verify-docs` operations. Workflow YAML consumes outputs and arranges jobs; it does not parse the declaration or reproduce validation logic.

**Rationale**: `_test.go` helpers cannot be invoked by GitHub Actions, while duplicated shell or workflow expressions would allow test validation and release behavior to drift. A tagged test-internal command gives unit specs and both workflows one executable contract without adding production CLI surface. Requiring the tag on every E2E file also proves the ordinary build does not compile the package.

**Alternatives considered**:

- Keep compatibility helpers only in `_test.go`: rejected because workflows need an executable interface.
- Parse JSON independently in each workflow: rejected because declaration, version, documentation, and evidence checks would have multiple implementations.
- Add compatibility commands to the production RepoKeeper CLI: rejected because this is repository test/release infrastructure, not user functionality.

## Decision 16: Provision immutable Git inputs per environment

**Decision**: Linux, macOS, and WSL download the official kernel.org Git source archive declared for their cell, verify its SHA-256, and build into a test-owned prefix. Native Windows downloads the declared official Git-for-Windows MinGit archive, verifies SHA-256, and extracts it into a test-owned prefix. The provisioner prepends that prefix to an isolated `PATH`, verifies the exact `git --version`, and never falls back to hosted-runner Git or a package-manager latest version.

For WSL, the Windows host verifies and imports Canonical's versioned Ubuntu 24.04 image `wsl/releases/noble/20240423/ubuntu-noble-wsl-amd64-24.04lts.rootfs.tar.gz` (SHA-256 `2a790896740b14d637dbdc583cce1ba081ac53b9e9cdb46dc09a2f73abbd9934`) with `wsl --import --version 1`. Because the image manifest contains runtime Git but not a complete compiler toolchain, the declaration pins any required build-prerequisite packages to signed timestamped Ubuntu snapshot URLs and checksums; the job never runs an unconstrained package update. The host cross-compiles the Linux compatibility helper, RepoKeeper, and E2E test binaries with `CGO_ENABLED=0`, copies them into the imported distribution, uses the helper there to build and verify Linux Git, and invokes the E2E binary inside WSL. The WSL distribution name and install path are unique to the job and cleaned up after evidence collection.

**Rationale**: Official immutable archives plus checksums make exact patch selection reproducible across moving runner images. Source builds provide the same upstream Git lines on Linux, macOS, and WSL; MinGit is the official portable Git-for-Windows distribution. WSL1 matches the capability documented by the pinned Windows runner and avoids assuming a preinstalled distribution or nested virtualization.

**Alternatives considered**:

- Use runner-provided Git: rejected because hosted images update weekly and cannot provide all declared minors.
- Use Homebrew, apt, or winget latest: rejected because resolution changes over time and older lines may disappear.
- Run the native Windows RepoKeeper binary from WSL: rejected because that would not qualify RepoKeeper's Linux execution boundary.
- Install a full Go toolchain inside WSL: rejected because cross-compiled CGO-free test and application binaries reduce mutable provisioning inputs while still executing the system-under-test and Git inside WSL.

**Primary sources**: [official Git source archives](https://www.kernel.org/pub/software/scm/git/), [official Git source SHA-256 manifest](https://www.kernel.org/pub/software/scm/git/sha256sums.asc), [Git for Windows release and version policy](https://github.com/git-for-windows/git/security), [Microsoft WSL import documentation](https://learn.microsoft.com/en-us/windows/wsl/use-custom-distro), [versioned Canonical Ubuntu WSL rootfs and signed checksum files](https://cloud-images.ubuntu.com/wsl/releases/noble/20240423/), [Ubuntu snapshot service](https://snapshot.ubuntu.com/).

## Decision 17: Make race coverage and release-candidate recovery explicit

**Decision**: Add a native `ubuntu-24.04` CI job that runs concurrency and process-lifecycle integration coverage with `go test -race -count=1`, an explicit timeout, and a declared representative Git cell. It is a routine required check but is not multiplied across all release cells. Treat a `v*` tag as an immutable release-candidate trigger: a transient infrastructure failure may rerun the unchanged tag and commit, while any source, test, or compatibility-declaration correction requires a new version tag. GitHub Release creation, GoReleaser artifacts, packages, images, and distribution updates occur only downstream of qualification.

**Rationale**: The harness owns concurrent stream readers, child lifecycles, and cleanup, so ordinary functional success is insufficient to meet the constitution's race-enabled CI requirement. One representative native race job provides the relevant detector without making the twelve-cell release matrix disproportionately expensive. Candidate-tag semantics avoid moving tags or publishing partially qualified versions.

**Alternatives considered**:

- Run `-race` in every matrix cell: rejected because support evidence concerns behavior and exact Git compatibility, while the race detector has platform/toolchain constraints and high cost.
- Delete or move a failed tag after a code fix: rejected because mutable release tags weaken provenance and retry interpretation.
- Create the GitHub Release before qualification and leave it as a draft: rejected because release objects and downstream automation can be externally observable; no publication should begin before the gate.
