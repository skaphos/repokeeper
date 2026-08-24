# Execution Contract

## CLI Process Boundary

Every CLI operation invokes the suite-built absolute RepoKeeper binary with an argument vector, an explicit working directory inside the scenario root, an exact test-owned environment, and a context deadline no longer than 30 seconds.

Fixture Git and MCP scenario subprocesses use the same 30-second maximum. Compatibility downloads, Git source builds, WSL imports, and cross-compilation occur before scenario execution and use explicit operation deadlines no longer than 10 minutes plus a 30-minute CI job timeout. All process trees and readers are joined at the end of their operation or cell.

The runner returns raw stdout and stderr separately. It records normal non-zero exit codes without converting them into launch failures. Tests then distinguish:

- expected domain status: command started, exited within the deadline, produced the declared exit code, and emitted valid structured output;
- infrastructure failure: command could not start, was signaled/crashed, timed out, or produced malformed/missing structured output.

On assertion failure, diagnostics MUST include:

```text
operation
executable and argument vector
working directory
scenario root and config path
exit code or launch error
timeout state and duration
stdout
stderr
```

Environment values that could contain credentials are never inherited and therefore never printed.

## Exact Environment

The environment builder preserves only platform values required for executable lookup and process startup, then sets test-owned values. The implementation MUST account for case-insensitive environment keys on Windows.

Required controls include:

- `PATH` and platform startup variables such as `SystemRoot`, `ComSpec`, and `PATHEXT` where required;
- `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `APPDATA`, and `LOCALAPPDATA` beneath the scenario root;
- `TMPDIR`, `TMP`, and `TEMP` beneath the scenario root;
- `REPOKEEPER_CONFIG` pointing to the exact scenario config file;
- `GIT_CONFIG_NOSYSTEM=1` and a test-owned `GIT_CONFIG_GLOBAL`;
- `GIT_TERMINAL_PROMPT=0`, disabled credential helpers, and disabled signing;
- deterministic locale/timezone/color settings where supported.

Proxy, credential, SSH-command, signing, and unrelated RepoKeeper variables MUST NOT be inherited.

## Git Compatibility Boundary

The integration workflow runs on explicit OS-version labels rather than floating `*-latest` aliases. A tagged reusable compatibility command, not workflow expressions, validates the closed three-minor declaration, checks `DESIGN.md` agreement, expands routine or release cells, provisions declared inputs, verifies exact versions, and creates or validates evidence.

Before materialization, each job MUST:

1. select only a cell returned by `matrix --scope routine|release`;
2. integrity-check and provision the cell's exact Git patch into a test-owned prefix without runner-Git fallback;
3. require exact `git --version` agreement with the cell, not merely a numeric minimum; and
4. preserve the runner image, provisioned input identities, and raw actual version as evidence.

The declared set is closed. Numeric ordering never implies compatibility with an omitted historical or newer Git minor.

Every Go source file under `test/e2e` requires the `integration` build constraint. Platform files use compound constraints. A default-tier check MUST prove that ordinary untagged build/list/test commands neither compile nor execute the E2E package.

Concurrency and lifecycle-sensitive E2E coverage MUST also run in a native Linux CI job under Go's race detector, uncached and with an explicit outer timeout. A detected race is a failed required check.

### WSL execution

WSL is a first-class environment hosted on `windows-2025`:

1. verify the versioned Canonical Ubuntu 24.04 rootfs checksum and any exact signed-snapshot build-prerequisite package checksums;
2. import it under a unique job-owned name and path using WSL1;
3. provision and verify the declared Linux Git source build inside that distribution;
4. cross-compile CGO-free Linux compatibility-helper, RepoKeeper, and E2E test binaries on the Windows host;
5. use the helper to build/verify Git, then execute RepoKeeper through the E2E binary inside WSL with Linux paths and the same isolation contract; and
6. collect evidence and unregister the job-owned distribution during always-run cleanup.

Failure to import WSL1, provision Git, or execute the Linux binaries is a failed WSL cell. Native Windows execution, WSL2, runner Git, or a different distribution is not an automatic substitute.

### Full-release qualification

The version-controlled executable compatibility declaration enumerates all twelve environment and Git-minor combinations RepoKeeper claims to support. Each cell pins one exact patch and immutable provisioning metadata. Before any release artifact or distribution channel is published, the tag-triggered workflow MUST:

1. expand every declared cell without filtering or implicit exclusions;
2. provision and verify the exact declared Git patch on the explicit environment and runner label;
3. run the same end-to-end package with `fail-fast: false` so all cells produce evidence;
4. record the source commit, runner image, declared minor, expected patch, actual `git --version`, and result in a per-cell artifact; and
5. require a final completeness gate with exactly one successful result per declared cell before the publishing job can start.

Provisioning failures, version mismatches, timeouts, skipped tests, missing evidence, duplicate results, and unavailable declared versions are failures. They MUST NOT be converted to warnings, `continue-on-error`, or automatic compatibility-claim reductions. Pull-request and main CI may select representative declared cells, but that smaller selection does not qualify a full release.

The `v*` tag is an immutable release-candidate trigger, not a published release. A transient infrastructure failure may rerun the unchanged tag and commit. Any source, test, provisioning declaration, compatibility claim, or documentation correction requires a new version tag; failed tags are not moved or reused. Qualification jobs have read-only permissions and no publishing secrets. Only the publisher downstream of the completeness gate may acquire release permissions and credentials.

## CLI Scenario Contract

Use a fresh materialization and run real scan and status commands with JSON output. Assert:

- documented exit codes for the clean, dirty, and missing mix;
- stdout parses into the command's documented JSON shape and required fields are present;
- stderr matches the command's documented diagnostic behavior and contains no structured success payload;
- each live repository appears once and the seeded absent path is missing;
- persisted config/registry reload and match the reported state;
- the clean and dirty baseline HEADs are unchanged;
- the clean worktree remains clean;
- the dirty file bytes and porcelain state are unchanged.

## MCP Stdio Boundary

Create and own `exec.Cmd` for `repokeeper mcp --config <scenario-config>`, including its exact environment, working directory, lifetime context, platform process-tree boundary, and three pipes. Attach mcp-go v0.58.0 using `transport.NewIO` and `client.NewClient`; mcp-go owns protocol behavior but not the child process.

Lifecycle:

1. Start the command under a background-derived process-lifetime context of at most 30 seconds, and start exactly one `Wait` goroutine.
2. Continuously drain stderr into a concurrency-safe bounded buffer and wrap stdout in a recording reader before giving it to `NewIO`.
3. Start the MCP client transport and initialize with an explicit client identity and a per-call deadline.
4. Call `tools/list`, including SDK pagination.
5. Compare discovered tool names with the declared case map for exact set equality.
6. Execute the declared case order, applying a fresh context deadline to every call.
7. Close stdin to request EOF shutdown, wait briefly for the child and readers, then cancel/terminate the owned process tree if needed.
8. Join the sole `Wait` goroutine and every stream reader, close the client transport idempotently, and report any cleanup failure. No child or reader may survive test cleanup.

Protocol stdout is owned solely by mcp-go. After EOF, validate that every recorded line is a valid JSON-RPC 2.0 frame; non-JSON or unrelated output is a hard failure even if the SDK skipped it and calls succeeded. Any framing or parse failure includes captured stderr. Diagnostics MUST never be intentionally written to stdout.

The session is process-tree scoped: Unix starts a new process group; Windows assigns the child to a kill-on-close Job Object. Cleanup first attempts normal EOF, then terminates the tree. Go's `Cmd.WaitDelay` bounds pipe shutdown, and `Close` is idempotent because explicit shutdown and test cleanup may both call it.

## Determinism Contract

Run five fresh materializations of the same recipe. Compare a typed normalized projection rather than raw output. The projection may exclude only named volatile categories:

- timestamps and measured durations;
- unique scenario-root prefixes after replacement with a stable token;
- path-derived checkout IDs when their semantic repository mapping is separately asserted.

Repository classifications, relative paths, branch/tracking state, response keys, registry statuses, commit IDs generated from fixed metadata, and file-content hashes MUST be equal. Adding an excluded field requires an explicit update to the allowlist and rationale.
