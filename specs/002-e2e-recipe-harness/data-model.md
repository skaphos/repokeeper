# Phase 1 Data Model: Recipe-Driven End-to-End Test Harness

The models below cover the typed in-memory representations of version-controlled recipe and compatibility artifacts. They are deliberately test-internal contracts, not new serialized RepoKeeper APIs.

## WorkspaceRecipe

Represents one complete deterministic scenario.

| Field | Type | Meaning |
| --- | --- | --- |
| `SchemaVersion` | integer | Required artifact contract version; initial value is `1` |
| `Name` | string | Stable diagnostic name, unique within the suite |
| `Repositories` | []RepositoryRecipe | Live Git repositories to materialize |
| `MissingEntries` | []MissingEntryRecipe | Registry entries whose worktree path is absent |
| `Config` | ConfigRecipe | RepoKeeper defaults, excludes, and registry layout |

Validation rules:

- `SchemaVersion` is exactly `1`, unknown JSON fields are rejected, and malformed JSON fails before validation or materialization.
- Name is non-empty.
- Repository and missing-entry names and logical paths are unique.
- Every path passes the containment rules in the recipe contract.
- All referenced remote, branch, upstream, and related-repository names exist.
- A path cannot be both a live repository and a missing entry.
- Validation completes before materialization begins.

## RepositoryRecipe

Describes a local bare remote and one working checkout. Additional source checkouts can be generated privately by the materializer when it needs to advance a remote independently.

| Field | Type | Meaning |
| --- | --- | --- |
| `Name` | string | Logical fixture identifier |
| `RemotePath` | relative path | Bare repository beneath `remotes/` |
| `CheckoutPath` | relative path | Working tree beneath `workspace/` |
| `Files` | map[path][]byte | Content committed in the initial commit |
| `Branches` | []BranchRecipe | Branches and commits created after the base commit |
| `CurrentBranch` | string | Branch checked out when setup completes |
| `Upstream` | UpstreamRecipe | Remote/branch tracking relation for the current branch |
| `DirtyFiles` | map[path][]byte | Uncommitted content written only after commits and pushes |
| `Metadata` | optional RepoMetadataRecipe | Source-controlled `.repokeeper-repo.yaml` content |
| `Labels` | map[string]string | Machine-local registry labels expected after setup/scan |
| `Annotations` | map[string]string | Machine-local registry annotations used by response assertions |

Validation rules:

- Names and paths are non-empty; all file paths remain within the checkout.
- `CurrentBranch` exists.
- An upstream references an existing branch pushed to the declared local remote.
- Dirty paths do not name directories or escape the checkout.
- At least one committed file exists so commit identities are meaningful.

## BranchRecipe and CommitRecipe

`BranchRecipe` names its base branch and contains an ordered list of commits. `CommitRecipe` declares a stable message and file changes. The materializer applies commits in order with fixed author/committer identity and time, then pushes requested branches to the local bare remote.

State transitions:

```text
declared -> validated -> remote initialized -> checkout cloned
         -> base committed/pushed -> branches committed/pushed
         -> current branch/upstream selected -> dirty files applied -> ready
```

No RepoKeeper process may start before the workspace reaches `ready`.

## MissingEntryRecipe

| Field | Type | Meaning |
| --- | --- | --- |
| `RepoID` | string | Registry identity expected in structured output |
| `CheckoutID` | string | Stable checkout identity, explicitly set to avoid temp-path drift |
| `Path` | relative path | Intentionally absent path beneath `workspace/` |
| `RemoteURL` | string | Test-local identity value |
| `Status` | `missing` | Required registry status |

The materializer verifies the path does not exist before saving the registry.

## ConfigRecipe

Declares only values that scenarios need to vary. The materializer starts from `config.DefaultConfig()`, forces deterministic concurrency/timeouts and a test-owned external or embedded registry choice, then serializes it to the scenario's exact `REPOKEEPER_CONFIG` path.

The actual executable must load the serialized file. Tests may use production config/registry types to create valid setup data, but they may not call command handlers or engine operations to obtain the result under test.

## MaterializedWorkspace

Runtime projection of a validated recipe.

| Field | Meaning |
| --- | --- |
| `Root` | Unique scenario root |
| `WorkspaceRoot` | Only directory scanned by RepoKeeper |
| `RemotesRoot` | Local bare remotes, outside the scan root |
| `SourcesRoot` | Helper/source checkouts, outside the scan root |
| `HomeRoot` | Test-owned home and platform config directories |
| `TempRoot` / `CacheRoot` | Test-owned process scratch locations |
| `ConfigPath` / `RegistryPath` | Exact persisted RepoKeeper paths |
| `Env` | Exact platform-aware child environment |
| `Repositories` | Logical name to absolute checkout/remote/repo-ID/HEAD mapping |
| `MissingEntries` | Logical name to absolute absent path and registry identity mapping |

Invariant: every writable path and every mutation target has `Root` as its path ancestor after canonical lexical resolution.

## ExecutionResult

| Field | Meaning |
| --- | --- |
| `Operation` | Stable scenario step such as `cli scan` or `mcp initialize` |
| `Executable` / `Arguments` | Exact command vector, never a shell string |
| `ExitCode` | Numeric exit status when the process started and exited normally |
| `Stdout` / `Stderr` | Independently captured bytes |
| `Duration` | Measured process duration |
| `TimedOut` | Deadline caused termination |
| `LaunchError` | Failure before a meaningful exit status existed |

Domain non-zero exit codes are data until compared with an `ExpectedOutcome`; launch errors, timeouts, and malformed structured output are always infrastructure failures.

## MCPToolCase

| Field | Meaning |
| --- | --- |
| `Name` | Exact live MCP tool name and map key |
| `Arguments` | Canonical valid-input arguments, derived from materialized paths/IDs |
| `AssertResult` | Structured content and `isError` assertions |
| `CaptureState` | Optional pre-call Git/registry/filesystem snapshot |
| `AssertState` | Optional post-call invariant or transition assertion |
| `SafetyCase` | Optional refusal arguments, confirmed arguments, and separate state assertions |
| `Order` | Explicit sequence position when calls have state dependencies |

Coverage invariant:

```text
set(tools/list names) == set(MCPToolCase.Name values)
```

The comparison reports missing and unexpected names separately and occurs before canonical tool calls.

## ExpectedOutcome and NormalizedOutcome

`ExpectedOutcome` declares allowed exit codes and assertions for output, registry, Git, and filesystem state. `NormalizedOutcome` is the comparable projection used by the five-run determinism test. It uses relative paths and semantic fields, excludes a documented volatile-field allowlist, and never silently discards an unknown field.

## GitCompatibilityMatrix and CompatibilityCell

`GitCompatibilityMatrix` is a strictly decoded, version-controlled test declaration that is both the executable release matrix and the source summarized by the human-readable compatibility table in `DESIGN.md`.

Matrix-level fields include `SchemaVersion`, the ordered `SupportedMinors` set, and `Cells`. `SupportedMinors` contains exactly three distinct ascending `major.minor` values. Its highest member is the current claimed Git minor and its other members are the two immediately preceding lines at the time the declaration is reviewed. The declaration is closed: an unlisted minor does not acquire support merely because it is newer than the lowest member.

Each `CompatibilityCell` contains:

| Field | Meaning |
| --- | --- |
| `Environment` | Stable `linux`, `macos`, `windows`, or `wsl` identity |
| `RunnerLabel` | Explicit non-floating operating-system image label |
| `GitMinor` | Claimed supported `major.minor` line |
| `GitPatch` | Exact patch release selected from that minor line |
| `Provisioner` | `source-build` for Linux/macOS/WSL or `mingit-archive` for native Windows, with immutable source URL and SHA-256 |
| `WSLRootFS` | Required only for WSL: Ubuntu release/image date, immutable Canonical rootfs URL, SHA-256, WSL version `1`, and any exact signed-snapshot build-prerequisite package URLs/checksums |
| `RoutineCI` | Whether this cell belongs to the smaller PR/main selection |

Validation rejects duplicate environment/minor pairs, a cell minor absent from `SupportedMinors`, incomplete coverage of the four-environment by three-minor Cartesian product, patch versions outside their declared minor, floating runner labels, incomplete or non-HTTPS provisioning inputs, malformed SHA-256 values, unsupported environments, invalid WSL metadata, and an environment with anything other than one routine representative. Release expansion returns all twelve declared cells; routine expansion returns four cells.

The stable cell key is `<environment>-git-<major.minor>`. Exact Windows versions may retain a `.windows.N` suffix but must still map to the declared upstream minor.

## CompatibilityCommandResult

The reusable tagged compatibility command emits bounded JSON for workflow consumption:

| Operation | Result |
| --- | --- |
| `matrix` | Scope plus ordered cell objects and stable keys |
| `provision` | Installed prefix, expected version, actual raw version, and verified source identity |
| `verify-version` | Cell key and exact-match result |
| `write-evidence` | One `CompatibilityResult` document path and digest |
| `verify-evidence` | Exact-set completeness result with sorted missing, duplicate, unexpected, failed, skipped, and mismatched keys |
| `verify-docs` | Declaration and `DESIGN.md` agreement result |

Command failures use a non-zero exit status and emit field-qualified diagnostics to stderr. Machine-readable stdout contains only the requested JSON result.

## CompatibilityResult

One release-qualification result identifies the source commit, immutable release-candidate tag, runner image, declared cell, actual `git --version`, provisioned input digests, E2E outcome, and evidence artifact digest. The completeness gate requires a one-to-one successful mapping between all twelve declared cells and results at the tagged commit. A skipped test, provisioning failure, version mismatch, duplicate, missing result, unexpected result, tag/commit mismatch, or evidence-digest mismatch fails the gate.
