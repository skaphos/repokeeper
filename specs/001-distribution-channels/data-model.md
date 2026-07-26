# Phase 1 Data Model: Distribution Channel Conformance

**Feature**: `specs/001-distribution-channels` | **Date**: 2026-07-26

Four entities. Only the first is a Go type; the rest are the shapes of files and published artifacts
this feature defines. Each lists its validation rules and where they are enforced.

---

## 1. Version Identity

The one new Go type. Package `internal/buildinfo`.

### `Info`

| Field | Type | Meaning | Empty means |
| --- | --- | --- | --- |
| `Version` | `string` | Semantic version, e.g. `v0.8.0` | Not recorded — **never** substituted |
| `Revision` | `string` | Full VCS revision | Not recorded |
| `Time` | `string` | RFC3339 build time | Not recorded |
| `Modified` | `bool` | Built from a tree with uncommitted changes | — |
| `Source` | `Source` | Where the above came from | — |

Every string field is best-effort: empty means *not recorded*, never *zero*. This distinction is
FR-004 — a field that cannot be known must read as unknown, not as a plausible value.

### `Source`

| Value | String form | Meaning |
| --- | --- | --- |
| `SourceUnknown` | `unavailable` | Nothing usable recorded |
| `SourceBuildInfo` | `build metadata` | From the Go toolchain's own record |
| `SourceLDFlags` | `release build` | Stamped at release time |

`Source` exists so output can distinguish an authoritative version from an inferred one. Reporting
`v0.8.0` without saying whether the release pipeline asserted it or the module proxy implied it
would collapse two different confidence levels (Principle VI).

### Resolution rules — FR-001 through FR-005

Evaluated in order; first match wins.

| Tier | Condition | Result |
| --- | --- | --- |
| 1 | ldflags `Version` set and not the `"dev"` sentinel | `SourceLDFlags`, ldflags values, `"none"`/`"unknown"` sentinels mapped to empty |
| 2 | `debug.ReadBuildInfo()` unavailable | `SourceUnknown` |
| 3 | `Main.Version` present and not `(devel)` | `SourceBuildInfo`, with VCS settings where recorded |
| 4 | VCS revision recorded | `SourceBuildInfo`, `Version` **empty**, revision reported |
| 5 | otherwise | `SourceUnknown` |

**Two sentinels must not escape.** `"dev"` is the current ldflags default and `(devel)` is what the
module system records for a non-released build. Both are placeholders; surfacing either is the exact
defect FR-004 names. Tier 4 is the subtle case — a local `go build` has a real revision but no
version, so it reports the revision and leaves `Version` empty rather than inventing one.

**Single resolution point (FR-005).** `cmd/repokeeper/mcp.go` already passes the version into
`mcpserver.New(eng, cfgPath, Version, logger)`, so the MCP server advertises a version to every
connected client. If `version` and the MCP handshake resolved independently they could disagree.
One resolved `Info`, consumed by both.

**Invariant**: the ldflags variables `Version`, `Commit`, `Date` stay exported at
`github.com/skaphos/repokeeper/cmd/repokeeper` — `.goreleaser.yaml` references them by full path in
`-X` flags. Moving them silently breaks release stamping, which no test would catch because a
non-stamped build now falls back gracefully.

---

## 2. Release Artifact Set

What one release publishes. Not a code type — the contract the pipeline satisfies and the `verify`
job checks.

| Artifact | Platforms | Checksummed | SBOM | Signature | Provenance |
| --- | --- | --- | --- | --- | --- |
| `tar.gz` archive | linux, darwin × amd64, arm64 | yes | per-archive | via checksum bundle | yes |
| `zip` archive | windows × amd64, arm64 | yes | per-archive | via checksum bundle | yes |
| `.deb` | linux × amd64, arm64 | yes | **per-package** | via checksum bundle | yes |
| `.rpm` | linux × amd64, arm64 | yes | **per-package** | via checksum bundle | yes |
| Container image | linux × amd64, arm64 | n/a | attached | cosign | yes |
| `checksums.txt` | — | is the manifest | — | cosign bundle | yes |

**Validation rules**:

- Packages carry their **own** SBOM. The `sboms` block needs a second entry with
  `artifacts: package`; archive SBOMs do not extend to packages, and the omission is silent (FR-014).
- Every artifact derives from the same build. The container copies a pre-built binary rather than
  compiling, so the bytes signed and notarized are the bytes shipped (FR-011, FR-021).
- One release invocation produces all of it. A failure in any channel fails the release (FR-031).
- Package file naming follows each format's own convention via `{{ .ConventionalFileName }}`.
- Packages install `LICENSE`, `THIRD_PARTY_NOTICES.md` and `third_party_licenses/` under
  `/usr/share/doc/repokeeper/` (FR-012).

**States a release moves through**, and the failure this models:

| State | Meaning |
| --- | --- |
| `credentials-verified` | Pre-flight passed; every channel's credential is present and usable |
| `built` | GoReleaser produced every artifact |
| `published` | Every channel accepted its artifact |
| `confirmed` | Each channel independently **queried** and serving the released version |
| `half-landed` | Published reported success; a channel serves a stale version — **the ADR-0007 failure** |

`published` → `confirmed` is the transition this feature adds. A release that stops at `published`
and is assumed `confirmed` is how the cask sat at `0.6.0` across two releases.

---

## 3. Server Description

The checked-in `server.json`. Registry identity `io.skaphos/repokeeper`.

| Field | Value | Rule |
| --- | --- | --- |
| `$schema` | MCP registry schema URL | Validated in CI (FR-017) |
| `name` | `io.skaphos/repokeeper` | Namespace derived from the DNS-proven domain; changing it breaks publishing or publishes under an uncontrolled name |
| `description` | What RepoKeeper's MCP surface does | Must not describe tools that do not exist (FR-020) |
| `repository.url` | `https://github.com/skaphos/repokeeper` | — |
| `version` | Stamped from the tag at release | Must equal `packages[].version` |
| `packages[].registryType` | `oci` | — |
| `packages[].identifier` | `ghcr.io/skaphos/repokeeper` | Must match the image `dockers_v2` publishes |
| `packages[].version` | Stamped from the tag | Divergence publishes an entry naming a non-existent image tag |
| `packages[].transport.type` | `stdio` | Must match what `repokeeper mcp` actually serves |

**Drift rules — the part schema validation cannot check**:

- The described tool surface must agree with `mcpserver.ReadOnlyToolNames()`, which derives from live
  `ReadOnlyHint` annotations and therefore cannot drift from what is registered.
- **RepoKeeper diverges from sting here.** sting asserts every tool is read-only. RepoKeeper has both
  kinds — 8 read-only, 6 mutating — so the entry must represent a *mixed* surface honestly. Claiming
  read-only would misrepresent `execute_sync`, `add_repository`, `remove_repository`, `set_labels`,
  `scan_workspace` and `plan_sync`.
- The entry must state that the container serves one explicitly named workspace root (FR-026), since
  the registry is where a prospective user first learns how the server is configured.

**Checked-in placeholder**: `0.0.0` in both version fields, stamped at release. Both fields move
together or the entry is internally inconsistent.

---

## 4. Container Workspace Contract

How a containerized RepoKeeper is given repositories, and what it may do with them. The entity with
no sting equivalent.

| Property | Value | Requirement |
| --- | --- | --- |
| Mount path | Identical to the host path | FR-024 |
| Default access | Read-only | FR-025 |
| Workspace selection | Explicit `--config`; no discovery default | FR-026 |
| Roots per entry | Exactly one | FR-026 |
| Path translation | **None** | FR-024 |
| Git ownership | `safe.directory=*` in image gitconfig | R3 |
| VCS support | Git only; no Mercurial | FR-029 |

### Why identical-path

`.repokeeper.yaml` records absolute host paths. Mounted anywhere else, every registry entry misses
and the inventory reports every repository as missing — confidently wrong. An identical-path mount
makes the registry mean the same thing inside and outside the container, keeping it the single source
of truth (Principle II) and making resolution deterministic (Principle III). A remapping layer would
create a second interpretation of the same file.

### Tool surface under a read-only mount

| Class | Count | Behavior |
| --- | --- | --- |
| Read-only (`ReadOnlyHint: true`) | 8 | Function identically to native |
| Mutating | 6 | Refuse, naming the read-only mount and the remedy |

Mutating tools stay **advertised**. Hiding them would be a silently reduced surface; FR-025 requires
a refusal that explains itself (Principles VI and VII).

### Workspace Root — the sub-entity that distinguishes the channels

A directory containing a `.repokeeper.yaml` registry.

| Context | Selection | Multi-root |
| --- | --- | --- |
| Native CLI / native `repokeeper mcp` | Walks upward from the working directory | Roots self-select by position |
| Container | Named explicitly, fixed at `docker run` | One entry per root |

The native crawl is a feature, not an accident: it lets several purpose-specific roots coexist and
resolve by where the user is. A container has no working-directory context to be relative to — its
cwd and mounts are fixed before the MCP client calls anything — so that behavior cannot be
reproduced and must not be implied.

**Validation rules**:

- No default workspace location ships in the image. Not a fixed path, not `$HOME`-anchored. A default
  correct for single-root users and silently wrong for multi-root users is worse than requiring the
  workspace be named (FR-026).
- Discovery must not adopt a registry found outside the mounted workspace.
- A failed lookup reports the path searched and that discovery walks upward (FR-027).
- Tools report which registry they answered from, so an answer carries its own scope.
