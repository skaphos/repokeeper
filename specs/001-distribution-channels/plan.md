# Implementation Plan: Distribution Channel Conformance

**Branch**: `feature/distribution-channels` | **Date**: 2026-07-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-distribution-channels/spec.md`

## Summary

RepoKeeper adopts `skaphos-resources` DECISIONS/0001, which makes distribution channels a function of
artifact shape. RepoKeeper is both Shape 2 (end-user CLI) and Shape 3 (MCP server), so it inherits
the union of both channel sets. Four channels are delivered — version identity, `.deb`/`.rpm`, an MCP
registry entry, a multi-arch container image — and one required channel, self-update, is deviated
from on measured evidence and recorded in ADR-0016.

The technical approach is deliberately narrow: **almost none of this is application code.** Version
identity is a new stdlib-only `internal/buildinfo` package; the read-only-mount refusal is a small
stdlib-only error translation. Everything else is GoReleaser configuration, a Dockerfile, a JSON
manifest, workflow changes and documentation. Zero dependencies are added — a hard requirement
(FR-037), because the self-update deviation rests entirely on dependency cost.

Two findings from Phase 0 shape the work and have no counterpart in sting's implementation:

1. **`internal/gitx` shells out to the `git` binary**, so the container cannot use the
   distroless-static base sting uses, and a bind-mounted workspace triggers git's dubious-ownership
   refusal on *every* call unless `safe.directory` is configured. Measured, not assumed.
2. **RepoKeeper's registry discovery walks upward from the working directory**, which is what lets
   several purpose-specific workspace roots self-select by position. A container's working directory
   is fixed before any tool call, so this cannot be reproduced — the container serves one explicitly
   named root per configured entry, and the docs must say so rather than let a user be served the
   wrong workspace.

## Technical Context

**Language/Version**: Go 1.26.5 (`go.mod`; toolchain pinned in `.tool-versions`)

**Primary Dependencies**: none added (FR-037). Existing: `spf13/cobra`, `mark3labs/mcp-go`. New code
uses `runtime/debug`, `errors`, `syscall` only.

**Storage**: N/A — registry state is `.repokeeper.yaml` on disk, unchanged by this feature

**Testing**: Ginkgo v2 + Gomega, race-enabled, per-package coverage gate; `go -C tools tool task ci`

**Target Platform**: linux/darwin/windows × amd64/arm64 for the CLI; linux/amd64 + linux/arm64 for
the container image

**Project Type**: single Go module — CLI + MCP server

**Performance Goals**: N/A — no runtime hot path is touched. `buildinfo.Resolve` runs once at startup.

**Constraints**: zero dependency growth; released-binary `version` output byte-identical to today;
no change to the MCP tool surface, sync policy, or registry file format; container image must contain
a `git` runtime

**Scale/Scope**: ~6 new files, ~8 modified. Two small compiled changes; the rest is release
configuration and documentation.

**Release tooling**: GoReleaser 2.17.0 (`nfpms`, `dockers_v2`), `mcp-publisher` (pinned),
`check-jsonschema` (pinned), cosign, syft — all pre-existing in the pipeline except the last three.

## Constitution Check

*GATE: must pass before Phase 0. Re-checked after Phase 1.*

| Principle | Assessment | Verdict |
| --- | --- | --- |
| **I. Explicit State** | No new implicit state. The container's workspace is named explicitly rather than discovered — strengthens the principle. | PASS |
| **II. Git Is the Durable Boundary** | Identical-path mount exists specifically so `.repokeeper.yaml` stays the single statement of where a repository lives. A remapping layer was rejected for creating a second interpretation. | PASS |
| **III. Deterministic** | Version resolution is a pure function of recorded metadata with fixed precedence. Paths resolve identically inside and outside the container. | PASS |
| **IV. Kubernetes-Native** | Not applicable; RepoKeeper is a local developer CLI. The container image is an MCP transport, not a cluster component. | N/A |
| **V. Compose, Don't Trap** | Adds install channels; no new coupling to other Skaphos tools. | PASS |
| **VI. Explainable** | The load-bearing principle here: a build that cannot state its version says so; a mutating tool under a read-only mount names the mount; a missing workspace names the path searched; a channel that did not publish is named. | PASS |
| **VII. Read-Only Degradation** | The container's read-only default is this principle made concrete — full inspection always, mutation refused *with a reason*, never a silently reduced surface. | PASS |
| **VIII. Topology Is State** | Registry remains the encoded model of where repositories live; unchanged. | PASS |
| **IX. Honest Scope** | Requires stating: a `.deb` is not an apt repo; Windows binaries are unsigned; the container is Git-only; one entry serves one root; no self-updater ships and why. | PASS |
| **X. Safe-by-Default VCS** | No new mutation path. The container defaults to a mount that *cannot* mutate. | PASS |
| **XI. Git-First, Multi-VCS Scoped** | Container carries Git only; Mercurial's unavailability there is stated rather than silently failing at `exec.LookPath`. | PASS |
| **XII. CLI-First, Machine-Readable** | Version identity ships in both human and JSON form. Linux packages exist because Linux/WSL is first-class. | PASS |
| **Engineering: dependencies minimized** | FR-037 makes zero growth a *requirement*, verified mechanically. | PASS |
| **Engineering: testing** | Both compiled changes ship with direct coverage; tests touch no network. | PASS |
| **Engineering: attribution** | No `go.mod` change ⇒ no `third_party_licenses/` regeneration. Packages *ship* the existing notices. | PASS |
| **Adopt before build** | sting is adopted for four channels; the two divergences (R2/R3, R4) are documented with the reason the verdict does not transfer. | PASS |
| **Governance: ADR** | ADR-0016 records the dropped required channel, as DECISIONS/0001 demands. | PASS |

**Result: PASS, no violations.** Complexity Tracking is empty.

### Post-Phase 1 re-check

Re-evaluated after the design artifacts. Still PASS. Phase 1 *strengthened* two principles rather
than straining them: the decision to keep mutating tools advertised-but-refusing (rather than hidden)
is Principle VII applied more strictly than the spec required, and the decision to require an
explicit workspace rather than ship a `$HOME` default is Principle IX choosing an honest constraint
over a convenient default that would be silently wrong for multi-root users.

One correction was made to the spec during Phase 0 rather than carried as a violation: FR-037
previously asserted version identity was "the one change to compiled code". The read-only-mount
refusal is a second. Both are stdlib-only, so the normative requirement — no new dependencies — was
never at risk; the parenthetical was simply inaccurate and is fixed. Recorded here because the
requirement predates the container contract being settled.

## Project Structure

### Documentation (this feature)

```text
specs/001-distribution-channels/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output — 12 resolved unknowns
├── data-model.md        # Phase 1 output — 4 entities
├── quickstart.md        # Phase 1 output — 8 validation groups
├── contracts/           # Phase 1 output
│   ├── cli-version.md
│   ├── container-image.md
│   ├── release-artifacts.md
│   └── server-json.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
.
├── Dockerfile                          # NEW — no build stage; copies GoReleaser's binary
├── .dockerignore                       # NEW
├── server.json                         # NEW — MCP registry entry
├── .goreleaser.yaml                    # MOD — + nfpms, + package sboms, + dockers_v2
├── README.md                           # MOD — per-channel upgrade table, container config
├── INSTALL.md                          # MOD — Linux packages, "no apt repo", container
├── RELEASE.md                          # MOD — new channels and the verify job
│
├── cmd/repokeeper/
│   └── version.go                      # MOD — delegate to buildinfo; keep ldflags vars exported
│
├── internal/
│   ├── buildinfo/
│   │   ├── buildinfo.go                # NEW — 3-tier resolution
│   │   └── buildinfo_test.go           # NEW
│   └── mcpserver/
│       ├── readonly.go                 # NEW — EROFS → explained refusal
│       ├── readonly_test.go            # NEW
│       └── serverjson_test.go          # NEW — drift guard vs ReadOnlyToolNames()
│
├── docs/adr/
│   ├── 0016-no-self-update-subcommand.md   # NEW — the deviation record
│   └── README.md                       # MOD — index entry
│
└── .github/workflows/
    ├── ci.yml                          # MOD — + manifests job
    └── release.yml                     # MOD — credential pre-flight, GHCR, registry, verify job
```

**Structure Decision**: existing single-module layout, unchanged. New Go code goes to
`internal/buildinfo` (a new leaf package, no dependents beyond `cmd/repokeeper`) and
`internal/mcpserver` (existing). No new top-level directories, no package moves. `Dockerfile` and
`server.json` sit at the repository root because GoReleaser and `mcp-publisher` expect them there.

## Implementation Phasing

Ordered by dependency and risk. Per the repository's phased-execution directive, each phase is
verified before the next begins, and no phase touches more than five files.

| Phase | Deliverable | Files | Depends on |
| --- | --- | --- | --- |
| **1** | Version identity — `internal/buildinfo` + `version.go` + tests | 3 | — |
| **2** | Linux packages — `nfpms` + package `sboms` | 1 | — |
| **3** | ADR-0016 + README upgrade table | 3 | — |
| **4** | Container — `Dockerfile`, `.dockerignore`, `dockers_v2`, read-only refusal + tests | 5 | 1 |
| **5** | MCP registry — `server.json`, drift tests, CI `manifests` job | 3 | 4 (image identifier) |
| **6** | Release pipeline — credential pre-flight, GHCR, publish, `verify` job | 2 | 2, 4, 5 |
| **7** | Documentation sweep — `INSTALL.md`, `RELEASE.md`, container docs | 3 | all |

Phases 1–3 are independent and could run in parallel; 4 depends on 1 only for the version the image
reports. Phase 6 is last because it verifies channels the earlier phases create.

## Risks

| Risk | Impact | Mitigation |
| --- | --- | --- |
| `safe.directory` omitted from the image | **Every** git call fails; container is non-functional | Explicit quickstart check 5a; measured in R3 |
| Package SBOMs omitted | `.deb`/`.rpm` ship unattested; FR-014 silently unmet | Second `sboms` entry; quickstart check 4 asserts the files |
| ldflags variables moved during refactor | Release stamping breaks silently — no test catches it, because unstamped builds now degrade gracefully | Invariant recorded in data-model; keep `Version`/`Commit`/`Date` exported at the same path |
| Container serves the wrong workspace | Confident, complete, wrong answer | Explicit `--config`, no default location, tools report which registry answered |
| `dockers_v2` behavior differs from expectation | Release-time failure | `goreleaser check` in CI; snapshot build in quickstart check 4 |
| Registry publish becomes load-bearing | A flaky external channel blocks releases | Non-blocking by design (FR-034); `continue-on-error` + reported |

## Complexity Tracking

No constitutional violations. Table intentionally empty.
