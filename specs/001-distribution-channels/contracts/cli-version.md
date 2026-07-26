# Contract: `repokeeper version`

**Requirements**: FR-001 – FR-006 | **Package**: `internal/buildinfo`, `cmd/repokeeper/version.go`

The command's observable behavior. Output text is a stable contract (Principle III); the exact
wording may be refined during implementation but the *facts present* and the *distinctions drawn*
may not.

## Invocation

```
repokeeper version [--output json]
```

`--output json` follows the existing convention (Principle XII). If the root command already supplies
an output flag, reuse it rather than adding a command-local one.

## Human-readable output

### Release build — ldflags stamped (FR-003)

Byte-identical to today's output for a released binary. This is a hard requirement: the change must
be invisible to anyone installing from a release.

```
repokeeper v0.8.0
  commit:  9f2c1ab4e5d6789012345678901234567890abcd
  built:   2026-07-26T14:22:31Z
  go:      go1.26.5
  os/arch: linux/amd64
```

### Module install — `go install ...@v0.8.0` (FR-001)

```
repokeeper v0.8.0 (build metadata)
  commit:  unavailable
  built:   unavailable
  go:      go1.26.5
  os/arch: linux/amd64
```

The module proxy records a version but no VCS settings. `commit` and `built` read `unavailable`
rather than being omitted — an absent line invites the reader to assume it was not applicable.

### Local build from a clean tree (FR-002)

```
repokeeper (devel) — no released version (build metadata)
  commit:  9f2c1ab4e5d6789012345678901234567890abcd
  built:   2026-07-26T14:22:31Z
  go:      go1.26.5
  os/arch: linux/amd64
```

The version line must not print a bare `(devel)` as though it were a version. It reports the absence
in words and gives the revision, which is the identifying fact available.

### Local build from a modified tree (FR-002)

```
repokeeper (devel) — no released version, modified working tree (build metadata)
  commit:  9f2c1ab4e5d6789012345678901234567890abcd (dirty)
  built:   2026-07-26T14:22:31Z
  ...
```

The `modified` flag must be visible. A bug report against a dirty build is not a bug report against
the revision it names.

### Nothing recorded (FR-004)

```
repokeeper — version information unavailable
  commit:  unavailable
  built:   unavailable
  go:      go1.26.5
  os/arch: linux/amd64
```

**Forbidden in every case**: `dev`, `(devel)` presented as a version, `none`, `unknown`, an empty
version rendered as though it were one, or any value a reader could mistake for a real release.

## Machine-readable output (FR-006)

```json
{
  "version": "v0.8.0",
  "revision": "9f2c1ab4e5d6789012345678901234567890abcd",
  "built": "2026-07-26T14:22:31Z",
  "modified": false,
  "source": "release build",
  "go": "go1.26.5",
  "os": "linux",
  "arch": "amd64"
}
```

**Rules**:

- Unknown string fields are `""`, not omitted and not `"unknown"`. A consumer testing for emptiness
  gets one answer; a consumer matching the literal `"unknown"` would be parsing prose.
- `source` is one of `release build`, `build metadata`, `unavailable`.
- `modified` is always present as a boolean; absent VCS data means `false`, and `source` is what
  disambiguates "known clean" from "not recorded".

## Exit codes

| Condition | Exit |
| --- | --- |
| Any of the above, including fully unknown | `0` |

Reporting that the version is unknown is a successful execution. The command answered the question
truthfully; it did not fail.

## Consistency invariant (FR-005)

The version the MCP server advertises during handshake and the version this command prints resolve
from the same `buildinfo.Info`. `cmd/repokeeper/mcp.go` passes it into `mcpserver.New(...)`; a second
independent resolution would let the two disagree about the same binary.

## Test obligations

| Case | Asserts |
| --- | --- |
| ldflags stamped | Tier 1 wins; output unchanged from today |
| ldflags `"dev"` sentinel + module version | Falls through to tier 3, not reported as `dev` |
| ldflags `"none"`/`"unknown"` sentinels | Mapped to empty, not printed literally |
| `Main.Version == "(devel)"` + revision | Tier 4: version empty, revision reported |
| No build info at all | `SourceUnknown`, no invented value |
| `vcs.modified == "true"` | Surfaces as modified |
| JSON shape | Unknown fields are `""`; `source` is one of three literals |

Resolution logic is tested against a synthetic `*debug.BuildInfo` — `debug.ReadBuildInfo()` reads the
running test binary and cannot be faked, so the exported `Resolve` wraps an unexported, injectable
core.
