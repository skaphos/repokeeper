# Quickstart: Recipe-Driven End-to-End Harness

This document describes the intended developer workflow after issue #327 is implemented. The planning artifacts themselves do not yet add the harness.

## Prerequisites

- Go version from `go.mod` (Go 1.26)
- Git available on `PATH` for ad hoc local runs; qualification jobs instead provision the exact patch declared for their cell
- A platform where local filesystem Git remotes are supported
- No network access, hosted repository, or test credential is required

The integration CI jobs use explicit OS-version labels and report `git --version`. The executable claim is a closed set of three Git minor lines—currently `2.53`, `2.54`, and `2.55`—across Linux, macOS, native Windows, and WSL. Unlisted minors are not implicitly supported.

Routine pull-request and main CI use four representative declared cells, one per environment. A full tag-triggered release runs all twelve environment/minor cells, verifies each exact declared patch, and blocks all publication until the complete matrix passes.

## Run the E2E Package

Run only the new black-box package without accepting cached results:

```sh
go test -v -count=1 -tags integration ./test/e2e
```

Run the repository's complete integration tier:

```sh
go -C tools tool task -d .. test-integration
```

Run the full local pull-request gate:

```sh
go -C tools tool task -d .. ci
```

The ordinary fast test command remains untagged and does not run the E2E package:

```sh
go -C tools tool task -d .. test
```

The repository also verifies that the untagged package list/build does not compile any `test/e2e` Go source. Every file in that tree uses `//go:build integration`; platform helpers combine it with their operating-system constraint.

Run the lifecycle/concurrency race tier on a supported native platform:

```sh
go -C tools tool task -d .. test-integration-race
```

## What the Canonical Scenario Does

1. Builds the checked-out RepoKeeper executable once into a suite-owned temporary directory.
2. Materializes a fresh local-only recipe containing clean and dirty checkouts, bare origins, source-controlled repository metadata, and one missing registry entry.
3. Runs real `scan` and `status` CLI processes with JSON output and verifies exit codes, stream separation, persisted registry state, HEADs, and worktree bytes.
4. Materializes a separate fresh copy of the recipe for MCP.
5. Starts `repokeeper mcp` as an owned child process, attaches an MCP client to its real standard streams, initializes, and calls `tools/list`.
6. Requires exact equality between the live tool names and the E2E case map.
7. Calls every discovered tool with valid input and verifies each response or state transition.
8. Exercises refusal and confirmed success for confirmation-gated/destructive tools.
9. Closes stdin, joins the process and stream readers, and validates every recorded stdout frame as JSON-RPC.

All remotes, homes, configuration, caches, temp paths, working trees, and mutation targets are beneath the disposable scenario root.

## Repeatability Qualification

The suite includes a five-materialization semantic determinism assertion. To investigate intermittent process or platform behavior beyond that built-in check, run the package repeatedly:

```sh
go test -v -count=5 -tags integration ./test/e2e
```

Each invocation rebuilds the binary once for that test process and creates fresh scenario roots. Normalization removes only documented volatile timestamps, durations, and disposable absolute prefixes.

## Release Qualification

The executable compatibility declaration under the E2E test data is the source for both routine and full-release matrices. Tests and workflows consume it through one tagged command:

```sh
go run -tags integration ./test/e2e/cmd/compatibility matrix --scope routine
go run -tags integration ./test/e2e/cmd/compatibility matrix --scope release
go run -tags integration ./test/e2e/cmd/compatibility verify-docs
```

Linux, macOS, and WSL cells build a checksum-pinned official Git source archive into a test-owned prefix. Native Windows cells extract a checksum-pinned official MinGit archive. WSL imports Canonical's versioned Noble `20240423` rootfs into WSL1, installs any missing build prerequisites only from declared signed-snapshot package artifacts, and runs cross-compiled Linux compatibility-helper, RepoKeeper, and E2E binaries inside it; no preinstalled distribution or unconstrained package update is assumed.

A release run produces one evidence artifact per declared environment and Git minor cell. Each artifact records the candidate tag, source commit, provisioned inputs, and exact patch actually exercised; the completeness gate rejects missing, duplicate, skipped, failed, tag/commit-mismatched, or version-mismatched cells before GoReleaser or another distribution publisher can run.

Changing the supported matrix or its selected patch versions is a reviewed repository change accompanied by the corresponding `DESIGN.md` update. A temporarily unavailable Git distribution fails qualification; it does not silently shrink the support claim.

A version tag starts qualification but does not itself publish a release. A transient infrastructure failure may rerun the same unchanged tag and commit. Any code, test, provisioning, compatibility, or documentation fix uses a new semantic version tag; failed tags are never moved or reused.

## Add a Scenario

1. Add a portable JSON recipe artifact under `test/e2e/testdata/recipes/`; do not add a shell bootstrap or a second serialized recipe format.
2. Reuse the shared validator and materializer.
3. Declare expected CLI or MCP structured fields and Git/filesystem invariants.
4. If adding an MCP tool, add its keyed canonical success case. If it is destructive or confirmation-gated, add refusal and confined-success assertions too.
5. Run the E2E package and confirm the live `tools/list` coverage gate has no missing or unexpected cases.

## Diagnose a Failure

Failure output is emitted without requiring another verbose run. Look for the fixed diagnostic fields:

- scenario and operation/tool name;
- executable, argument vector, and working directory;
- normalized workspace and config paths;
- start/timeout/exit state and duration;
- bounded raw stdout and stderr, with omitted-byte counts if truncated;
- MCP log contents and protocol-frame validation failures when applicable.

The workspace is disposable. Successful runs clean it automatically; failed assertions must report enough paths and raw output to reproduce the failed operation even if framework cleanup removes the directory.

## Scope Reminder

This harness uses local Git remotes only. Hosted GitHub fixtures, authentication, and network behavior belong to specification 003 and issue #329, not issue #327.
