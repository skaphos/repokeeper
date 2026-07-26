# Feature Specification: Distribution Channel Conformance

**Feature Branch**: `feature/distribution-channels`

**Created**: 2026-07-26

**Status**: Draft

**Input**: User description: "https://github.com/skaphos/repokeeper/issues/305 — Conform to DECISIONS/0001: distribution channels by artifact shape."

## Problem

`skaphos-resources` [DECISIONS/0001 — Distribution channels by artifact shape][adr0001] makes a
tool's distribution channels a function of what the artifact *is*. RepoKeeper is both an end-user
CLI (**Shape 2**) and an MCP server (**Shape 3**) — it ships `repokeeper mcp`, a stdio MCP server,
plus an in-binary MCP runtime installer ([ADR-0008][rk0008]). It therefore inherits the union of
both channel sets. RepoKeeper and sting are named as the standard's first adoption targets.

RepoKeeper already conforms on five channels: GitHub release archives with a checksum manifest,
per-archive SBOM, cosign bundle and build-provenance attestation; macOS Developer ID signing and
notarization; a Homebrew cask into `skaphos/homebrew-tools`; the in-binary MCP runtime installer;
and a release owned end-to-end by a single GoReleaser invocation ([ADR-0013][rk0013]).

Five required obligations are unmet. Each one is a user who cannot install, cannot tell what they
are running, cannot upgrade, or cannot find the tool at all:

1. **Linux users have no package.** Every Linux install is "download a tarball, extract it, put it
   on `PATH` yourself." No `.deb`, no `.rpm`.
2. **Anyone who installs the way the module path implies gets a binary that does not know its own
   version.** `cmd/repokeeper/version.go:13` defaults `Version` to `"dev"` (and `Commit`/`Date` to
   `"none"`/`"unknown"`) with no build-metadata fallback, so
   `go install github.com/skaphos/repokeeper/cmd/repokeeper@latest` followed by
   `repokeeper version` prints `repokeeper dev`. Release-time ldflags are the only source of
   version information. A user cannot tell what they are running, and neither can a bug report.
3. **There is no upgrade path the tool itself can offer.** Every upgrade runs through whichever
   channel the user originally installed from, and RepoKeeper cannot tell them which one that was.
   *(Deliberately left open — see Out of Scope and ADR-0016. Documented per channel in `README.md`
   instead.)*
4. **RepoKeeper is invisible to MCP clients.** It is an MCP server with no entry in the index those
   clients read — findable only by someone who already knows it exists.
5. **RepoKeeper cannot be run as a container.** A `docker`-based MCP client configuration — the
   portable way to run a server without a local toolchain — has nothing to point at.

Closing these is the work. The constraint that shapes *how* is recorded first-hand in
[ADR-0007][rk0007]: RepoKeeper's own Homebrew cask sat pinned at `0.6.0` across two releases
because a GoReleaser run died before the tap step and nothing said so. A release publishes to two
channels today — the release archives and the cask — and to six once this feature lands, adding
`.deb`, `.rpm`, a container image and the MCP registry entry. That is six places a release can
half-land instead of two. The mitigation is that all of them bar one are fed by a single release
invocation that fails as a unit — and where a channel cannot be held to that (the MCP registry),
its outcome must be *verified*, not assumed.

**This feature surfaces three defects in the upstream record, which together require a
supersession.** DECISIONS/0001 is Accepted, and both the RepoKeeper constitution and the upstream
governance rule hold that ADRs are immutable and superseded, never rewritten — so none of these is
an in-place edit to that file. They are the substance of a successor record, `DECISIONS/0002`,
tracked in `skaphos-resources` and not a blocker for this feature:

1. **The "What ships today" table undercounts RepoKeeper.** It classifies RepoKeeper as `Go CLI`
   only and lists Shape 3 as "sting, and the `wake-*-mcp` family." RepoKeeper has had an MCP server
   since [ADR-0001][rk0001] and a runtime installer since [ADR-0008][rk0008]. It belongs under both
   shapes, which is what makes the Shape 3 work below obligatory rather than optional.
2. **The self-update rules have no answer for a tool that spans Shape 2 and Shape 3.** They sit
   under the Shape 2 heading and were written for a CLI in isolation. A tool that also ships a
   container image has a channel where self-replacement is not merely deferred but *incoherent*:
   writing a new binary into a container's ephemeral writable layer appears to succeed and
   evaporates on the next run. The standard requires a channel-aware rule, not a shape-wide one.
3. **Self-update is listed as required, and both first adoption targets have now deviated from it
   identically, for the same measured reason.** sting reverted a working implementation; RepoKeeper
   declines to build one on the evidence recorded under Prior Art. When every adopter deviates the
   same way, the standard is the thing that needs revising.

[adr0001]: https://github.com/skaphos/skaphos-resources/blob/main/DECISIONS/0001-distribution-channels-by-artifact-shape.md
[rk0001]: ../../docs/adr/0001-mcp-server.md
[rk0007]: ../../docs/adr/0007-release-binaries-and-homebrew.md
[rk0008]: ../../docs/adr/0008-mcp-install-tooling.md
[rk0012]: ../../docs/adr/0012-release-please-owns-release-notes.md
[rk0013]: ../../docs/adr/0013-goreleaser-owns-github-release.md
[sting0011]: https://github.com/skaphos/sting/blob/main/docs/adr/0011-no-self-update-subcommand.md

## Prior Art

sting closed the identical obligation set under `skaphos/sting#121`. Its outcome is a direct
input to this specification, not merely a reference:

- **Version identity, Linux packages, MCP registry entry and container image all shipped** and are
  the template for the equivalent work here.
- **Self-update was specified, implemented, and then reverted on measured cost.** Verifying
  in-process is the only way to satisfy both "verification cannot be skipped" and "no tooling
  required on the user's machine"; the dependency that makes that possible took sting's binary from
  11 MB to 25 MB (+122%) and `go.mod` from 44 to 106 requirements, pulling an unmaintained crypto
  module into the graph. sting removed the command rather than weakening it, and recorded the
  deviation in [sting ADR 0011][sting0011]. RepoKeeper's binary carries a comparable dependency
  budget today (47 direct requirements), so the same cost would land here.
- **The registry namespace question is settled upstream**: sting publishes as `io.skaphos/sting`,
  proving ownership by a DNS TXT record on `skaphos.io`. RepoKeeper inherits that namespace and
  that proof rather than re-deciding it.

The constitution's *Adopt before build* rule makes this binding: where mature prior art exists in
the suite, a plan that diverges must document why the verdict does not apply.

### Measured cost of a conforming self-updater on RepoKeeper

sting's percentage was not assumed to transfer. It was re-measured against RepoKeeper's own module
graph, by adding the same `sigstore-go` verification surface sting used
(`pkg/bundle`, `pkg/root`, `pkg/verify`, `pkg/fulcio/certificate`) to a copy of RepoKeeper at
`48fd6de` and building with the release flags from `.goreleaser.yaml` (`-s -w`, `CGO_ENABLED=0`,
`-trimpath`, `linux/amd64`):

| Measure | Baseline | With in-process verification | Delta |
| --- | --- | --- | --- |
| Binary size | 10,072,226 B (9.6 MiB) | 23,031,970 B (22.0 MiB) | **+12.96 MB (+128.7%)** |
| Direct requirements in `go.mod` | 47 | 106 | +59 |
| Modules in the build graph | 95 | 406 | +311 |
| `go.sum` entries | 70 | 232 | +162 |

The cost is not smaller than sting's — it is marginally worse (sting recorded +122%; RepoKeeper
measures +128.7%, a 2.29× binary). The measurement scaffolding was built outside the repository and
discarded; no dependency was added to RepoKeeper.

### Where a self-updater could actually run

The cost above is only half the case. Once this feature ships, RepoKeeper is installable through six
channels, and the verified-replace path is reachable on exactly one of them:

| Channel | What an `update` command could do |
| --- | --- |
| GitHub release archive, hand-placed | **Replaces the binary** — the only case |
| Homebrew cask | Defers — prints `brew upgrade --cask` |
| `.deb` | Defers — dpkg owns the file |
| `.rpm` | Defers — replacing breaks `rpm -V` |
| `go install` | Defers — the toolchain owns the file |
| Container image | **Incoherent** — see below |

Five of the six take the deferral branch, which requires no verification code at all: it is
install-provenance detection and a printed string. The container case is worse than a no-op —
writing a replacement into a container's ephemeral writable layer appears to succeed and disappears
on the next run — and because FR-021 requires the image be built from the same binaries as the
archives, every containerized MCP user would carry +12.96 MB for a command that cannot function in a
container. Building per-channel binaries behind build tags would avoid that, but it breaks the
same-binary guarantee FR-011 and FR-021 rest on and doubles the signing and SBOM surface.

So the measured cost lands on all six channels to serve one — and that one is the channel whose
users have already demonstrated they can place a binary on `PATH` by hand.

## Clarifications

### Session 2026-07-26

- Q: Ship a `repokeeper update` self-update subcommand, accepting the dependency cost, or record a
  deviation? → A: **Record the deviation; ship no `update` command**, mirroring
  [sting ADR 0011][sting0011]. The decision was gated on measurement rather than on sting's
  precedent: the cost was re-measured on RepoKeeper's own module graph (+128.7% binary, +59 direct
  requirements) and found no better than sting's, while the channel analysis above showed the
  verified-replace path is reachable on one of six channels and incoherent on another. ADR-0016
  records the deviation as DECISIONS/0001 requires of any dropped required channel; `README.md`
  carries a per-channel upgrade table in its place.
- Q: How does the containerized MCP server reach the user's repositories, given `.repokeeper.yaml`
  records absolute host paths? → A: **Mount the workspace at its identical host path, read-only by
  default.** Every absolute path in the registry then resolves unchanged, with no path-translation
  layer between the registry file and what the tools see. The read-only inspection tools work as
  shipped; the mutating tools refuse and name the read-only mount as the reason, which is
  Principle VII's read-only degradation and Principle VI's explained refusal rather than a
  capability the image silently lacks. A user who wants mutation mounts read-write and supplies git
  credentials explicitly.
- Q: What is the mechanism for correcting the upstream record? → A: **A supersession —
  `DECISIONS/0002` in `skaphos-resources`, not an edit to `DECISIONS/0001`.** DECISIONS/0001 is
  Accepted, and both the RepoKeeper constitution and upstream governance hold that ADRs are
  immutable and superseded, never rewritten. The successor carries all three defects listed in the
  Problem section. It is tracked in `skaphos-resources` and is not a dependency of this feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Know which version you are running, however you installed it (Priority: P1)

A developer who installed RepoKeeper from source runs `repokeeper version` and gets the actual
version, revision and build time of the binary in their hand, not a placeholder. The same is true
of a release build, a CI build, and a locally compiled working copy; each reports what it honestly
knows, and says so when it knows nothing.

**Why this priority**: It is the smallest change with the widest blast radius. Today every source
install is indistinguishable from every other source install, which makes a bug report unactionable
against a cross-platform tool whose behavior varies by platform. It is also a Principle VI
(explainability) defect: the tool cannot explain what it is. Every other story here is either
unblocked by this one or unaffected by it.

**Independent Test**: Build the binary without any release-time version stamping, run the version
command, and confirm it reports the module version and revision recorded in the build rather than a
placeholder. Then build it *with* release-time stamping and confirm the stamped values are what
appear.

**Acceptance Scenarios**:

1. **Given** a binary installed directly from the module at a published version, **When** the user
   asks for the version, **Then** the published version and the revision it was built from are
   reported, and no placeholder value appears.
2. **Given** a release binary produced by the release pipeline, **When** the user asks for the
   version, **Then** the values the pipeline stamped in are reported unchanged — release output is
   byte-identical to what it is today.
3. **Given** a binary built from a working copy with uncommitted changes, **When** the user asks for
   the version, **Then** the report identifies it as a modified build rather than presenting it as a
   clean build of the underlying revision.
4. **Given** a binary built in a way that records no version information at all, **When** the user
   asks for the version, **Then** the report says the information is unavailable rather than
   inventing a value.
5. **Given** any of the above, **When** the version is requested in machine-readable form, **Then**
   the same facts are available as structured output, per Principle XII.

---

### User Story 2 - Install on Linux with the system package manager (Priority: P2)

A developer on a Debian- or RPM-based Linux distribution downloads a package from the release page
and installs it with their system package manager in one command, getting the binary on `PATH` and
the license and notice files where their distribution expects them.

**Why this priority**: It is the largest population currently served worst — every Linux user today
does manual tarball placement, and RepoKeeper explicitly supports Linux including WSL as a
first-class platform (Principle XII). It is also the lowest-risk item in the feature, being
additional output from a release run that already produces the binaries, and it is independent of
every other story.

**Independent Test**: Take the packages from a release run and install each on a matching container
image, then confirm the binary runs, reports the release version, and is registered with the package
database; then remove it and confirm nothing is left behind.

**Acceptance Scenarios**:

1. **Given** a published release, **When** a user looks at its assets, **Then** there is a `.deb`
   and an `.rpm` for both 64-bit Intel and 64-bit ARM, alongside the existing archives.
2. **Given** one of those packages, **When** it is installed with the system package manager,
   **Then** the binary is on `PATH`, reports the release version, and the license and third-party
   notice files are installed to the location the distribution expects.
3. **Given** an installed package, **When** the user removes it, **Then** every file it placed is
   removed and the package database is left consistent.
4. **Given** a published release, **When** a user verifies its assets, **Then** the packages are
   listed in the same checksum manifest and covered by the same signature and provenance attestation
   as every other artifact in that release.
5. **Given** a user reading the install documentation, **When** they reach the Linux packages,
   **Then** the documentation states plainly that there is no hosted package repository and that
   upgrades mean downloading the next release's package.

---

### User Story 3 - Find RepoKeeper from an MCP client (Priority: P3)

Someone looking for a multi-repo workspace MCP server finds RepoKeeper in the registry their client
reads, sees what it does and what it needs, and installs it — instead of only ever finding it by
already knowing it exists.

**Why this priority**: Discoverability is the whole point of the channel, and RepoKeeper's MCP
surface is a first-class product ([ADR-0001][rk0001]), not an afterthought. But it reaches users who
have not yet found RepoKeeper rather than users blocked today, and DECISIONS/0001 itself notes this
is the least settled channel with the most movement under it.

**Independent Test**: Validate the checked-in server description against the registry's published
schema in CI, then confirm a release run publishes an entry whose version matches the release and
whose reported outcome — published or not — appears in the release run's output.

**Acceptance Scenarios**:

1. **Given** the repository, **When** it is inspected, **Then** it carries a checked-in server
   description naming the server, what it does, how it is run and what configuration it needs, and
   that file is schema-validated on every change.
2. **Given** a published release, **When** the registry is queried, **Then** RepoKeeper is present
   at the released version.
3. **Given** a release where the registry publish fails, **When** the release run finishes, **Then**
   the failure is surfaced in the run's output rather than passing silently, the rest of the release
   is still valid and complete, and the publish can be retried without cutting a new release.
4. **Given** the server description, **When** RepoKeeper's MCP tool surface changes, **Then** the
   description is updated in the same change, so the registry never describes a surface RepoKeeper
   does not have.
5. **Given** the server description, **When** it names the tool surface, **Then** it distinguishes
   the read-only inspection tools from the mutating ones, so a client can honor the read-only
   default RepoKeeper already enforces.

---

### User Story 4 - Run RepoKeeper as a container from an MCP client config (Priority: P4)

A developer configures their MCP client to run RepoKeeper from a container image and gets a working
server on Intel or ARM, with no Go toolchain and no local install, reading the workspace the client
gives it access to.

**Why this priority**: It removes the local-install prerequisite entirely and is required for Shape
3, but it serves the same discovery-stage user as Story 3 and is the furthest from a user who is
blocked right now. It is also the story where RepoKeeper differs most sharply from sting: sting's
server talks to a remote API and needs only a credential, whereas RepoKeeper's tools operate on
local repository working trees, so containerizing it raises a filesystem-access question sting never
had to answer.

The workspace is mounted at the same absolute path it occupies on the host, so every path already
recorded in `.repokeeper.yaml` resolves unchanged — no remapping, and no second interpretation of
the registry file. It is read-only by default: inspection works in full, and the tools that would
write refuse and say why.

The workspace is also named explicitly, and each configured entry serves exactly one root. This is
the sharpest difference from the native CLI, and it is a property of containers rather than a
shortcoming of the image: RepoKeeper normally finds its registry by walking upward from wherever it
was invoked, which is what lets a developer keep several purpose-specific roots and have the right
one selected by position. A container's working directory and mounts are fixed before the MCP client
ever calls a tool, so there is no "wherever it was invoked" to be relative to. A developer with
several roots configures one entry per root; a developer who wants position-sensitive behavior uses
the native binary. The documentation says this outright rather than letting a user discover it by
being served the wrong workspace.

**Independent Test**: Pull the published image on both architectures and drive the containerized
server with an MCP client over stdio, confirming a workspace inventory query returns the expected
repositories from a workspace mounted read-only at its host path, and that a mutating tool refuses
with the read-only mount named.

**Acceptance Scenarios**:

1. **Given** a published release, **When** the container registry is queried, **Then** an image
   exists at the release version and at a moving current tag, working on both 64-bit Intel and
   64-bit ARM Linux.
2. **Given** that image, **When** it is run with no arguments, **Then** it starts the MCP server on
   stdio, ready for a client to connect.
3. **Given** the workspace mounted read-only at its host path, **When** a client calls an inventory
   or repository-context tool, **Then** the results are identical to what the same query returns
   from a natively installed binary against the same workspace.
4. **Given** that same read-only mount, **When** a client calls a mutating tool, **Then** the tool
   refuses, names the read-only mount as the cause, and states that a read-write mount and supplied
   credentials would enable it — and the inspection tools continue to work unaffected.
5. **Given** the image started with no workspace at the expected path, **When** a client calls an
   inventory tool, **Then** the response names the path that was searched, states that discovery
   walks upward from the working directory, and states how to supply a workspace — rather than
   reporting an empty inventory as a successful scan.
6. **Given** a developer with several purpose-specific workspace roots, **When** they read the
   container documentation, **Then** it states that one configured entry serves one root, shows how
   to configure several entries, and identifies the native binary as the better path for a
   position-sensitive multi-root workflow.
7. **Given** that image, **When** it is inspected, **Then** it runs as an unprivileged user,
   contains no credentials or user-specific configuration, and carries the same SBOM, signature and
   provenance guarantees as the release archives.
8. **Given** the documentation, **When** a user reaches the MCP setup section, **Then** a
   container-based client configuration is shown alongside the existing local-binary one, the
   identical-path mount is shown explicitly, and the read-only default and what lifting it requires
   are stated plainly.

---

### Edge Cases

**Version identity**

- A build that records no version information at all must report the information as unavailable
  rather than substituting a value that could be mistaken for a real version.
- A build produced by the release pipeline and a build produced by the module toolchain must never
  disagree about which source they trust; stamped values win, deterministically (Principle III).

**Linux packages**

- Package install over an existing hand-placed binary at the same path: the package manager's own
  conflict behavior governs; RepoKeeper does not work around it, and the documentation says which
  wins.
- A `.deb` on a release page must not be read as an implied apt repository. The documentation says
  so plainly (Principle IX).

**Container**

- **A mount at anything other than the host path.** `.repokeeper.yaml` names each repository by its
  absolute path on the machine that wrote it. Mounted anywhere else, every one of those paths misses,
  and the inventory reports every repository as missing — a wrong answer delivered confidently, which
  Principle VI treats as a defect. The identical-path contract (FR-024) is what makes this
  unreachable in the documented configuration; where a user mounts elsewhere anyway, the result must
  be diagnosed as a workspace-not-found condition rather than reported as a scan of an empty
  workspace.
- **A read-only mount meeting a mutating tool.** RepoKeeper's sync and registry-editing tools write.
  Under the read-only default the refusal must name the mount as the cause and state the remedy
  (Principle VI), and every inspection tool must continue to work (Principle VII).
- **Credentials for remote operations.** Any tool that fetches from a remote needs credentials the
  container does not have by default. The failure must say the credential is missing, not report the
  remote as unreachable — the two have different remedies.
- **A partially mounted workspace.** Some registry entries resolve and others do not, because only
  part of the tree was mounted. Each unresolved repository must be reported individually with its
  reason, not collapsed into a whole-workspace failure or silently dropped from the inventory.
- **A workspace path that exists but is empty.** Distinguishable from "no workspace mounted" only by
  checking for the registry file; both must produce a message naming which condition was found.
- **A container serving a workspace the user did not mean.** The likeliest container failure and the
  most dangerous, because nothing errors. A user with roots at `~/work/skaphos` and `~/work/alaska`
  configures one entry, forgets it is pinned, and asks about the other root — receiving a complete,
  confident inventory of the wrong workspace. Mitigated by requiring the workspace to be named
  explicitly rather than discovered (FR-026), and by tools reporting which registry they answered
  from, so the answer carries its own scope.
- **An upward crawl that escapes the mount.** Discovery walks toward the filesystem root, so with the
  workspace mounted at a deep path and no registry inside it, the search continues into container
  directories that are not the user's. It must not adopt a `.repokeeper.yaml` found outside the
  mounted workspace — the same guard the native CLI already applies against picking up a stray
  registry from an unrelated ancestor.
- **Self-update inside a container.** RepoKeeper ships no update command (FR-007), so this cannot
  arise — but it is the reason it ships none: a binary replaced inside a container's ephemeral
  writable layer appears to update and reverts on the next run.

**Release pipeline**

- **Release signing secrets unset.** The macOS notarization block is gated on the five `MACOS_*`
  secrets being set and silently skips when they are not, shipping unsigned binaries into every
  downstream channel. Each channel added widens that exposure, so this feature must not make an
  unsigned release *harder* to notice than it already is.
- **A channel publishes the wrong version.** Verification must confirm the version each channel is
  serving, not merely that the channel responds — a cask still pinned to the previous release
  responds successfully and is precisely the ADR-0007 failure being guarded against.
- **Verification cannot reach a channel.** A transient outage at the tap, the container registry or
  the MCP registry must be distinguishable from a channel that genuinely did not publish;
  verification retries before declaring a channel missing, so a third-party outage is not reported
  as a failed release.
- **Registry entry drifts from the release.** An entry naming a version that was never published,
  or missing the current one, is a stale-channel failure of exactly the ADR-0007 kind.
- **The registry namespace's DNS proof lapses.** If the TXT record on `skaphos.io` is removed or the
  publishing credential expires, registry publishing fails while every other channel succeeds.
  Verification must report this as a registry-channel failure naming the DNS record and the
  credential as the things to check, rather than as a generic authentication error.

## Requirements *(mandatory)*

### Functional Requirements

**Version identity**

- **FR-001**: The version command MUST report a meaningful version for a binary built without
  release-time stamping, derived from the build's own recorded module and revision metadata.
- **FR-002**: The version command MUST report the source revision and, where recorded, the build
  time and whether the working tree was modified.
- **FR-003**: Release-time stamped values MUST take precedence over build-recorded metadata, so
  released binaries report exactly what they report today.
- **FR-004**: When neither stamped values nor build metadata are available, the version command MUST
  report the information as unavailable and MUST NOT substitute a value that could be mistaken for a
  real version.
- **FR-005**: The resolved version MUST be available to the rest of the tool as a single value, so
  no two surfaces can disagree about what version is running.
- **FR-006**: Version information MUST be available in machine-readable form alongside the
  human-readable form, consistent with Principle XII.

**Self-update** — *a required Shape 2 channel that this feature deliberately does not deliver.*

- **FR-007**: RepoKeeper MUST NOT ship a self-update subcommand in this feature. The verified-replace
  path is reachable on one of six channels and incoherent on another, at a measured cost of +128.7%
  binary size and +59 direct requirements carried by all six (see Prior Art). There MUST be no
  checksum-only mode and no "verify if a signing tool happens to be present" fallback:
  DECISIONS/0001 is correct that a self-updater ignoring available signing material is worse than
  none, so the choice is a correct updater at that cost or no updater.
- **FR-008**: The dropped channel MUST be recorded in an ADR (`docs/adr/0016-*`) stating what was
  given up, the measurement that drove it, what users get instead, and what would reverse the
  decision — a verification stack whose cost is materially lower. DECISIONS/0001's *Deviating*
  section requires a record in the repo that drops a required channel; this is that record.
- **FR-009**: `README.md` MUST carry a per-channel upgrade table covering every channel RepoKeeper
  ships on, so the user-facing question the update command would have answered — "how do I upgrade
  what I have?" — is answered by documentation instead.
- **FR-010**: No command MAY contact a release or version endpoint, for any purpose, including
  background or opportunistic update checks. DECISIONS/0001's no-implicit-network-calls rule is
  satisfied vacuously here and MUST stay that way; a future update command does not license ambient
  version checks in unrelated commands.

**Linux OS packages**

- **FR-011**: Each release MUST publish `.deb` and `.rpm` packages for 64-bit Intel and 64-bit ARM,
  built from the same binaries as that release's archives.
- **FR-012**: Packages MUST install the binary onto the system `PATH` and MUST install the license
  and third-party notice files to the location their packaging convention expects.
- **FR-013**: Packages MUST declare complete metadata — maintainer, homepage, license and
  description — consistent with the repository's other published artifacts and with its REUSE/SPDX
  attribution.
- **FR-014**: Packages MUST appear in the release's checksum manifest and MUST be covered by the
  same signature and build-provenance attestation as every other artifact in that release.
- **FR-015**: Installation documentation MUST state that no hosted package repository exists, so a
  `.deb` on a release page is not read as an implied apt repository.

**MCP registry**

- **FR-016**: The repository MUST carry a checked-in server description that names the server,
  describes what it does, and states how it is run and what configuration it requires. The server's
  canonical registry identity MUST be `io.skaphos/repokeeper`, matching the namespace sting
  established, and that name is what MCP clients use to refer to it.
- **FR-017**: That description MUST be validated against the registry's published schema in CI, so
  an invalid entry fails before a release rather than at publish time.
- **FR-018**: Each release MUST publish the entry to the MCP registry at the released version.
  Publishing MUST prove ownership of the `io.skaphos` namespace by a DNS TXT record on `skaphos.io`,
  and the credential it authenticates with MUST be organization-scoped rather than tied to an
  individual's account.
- **FR-019**: A registry publish failure MUST be surfaced in the release run's output and MUST NOT
  be silent; it MUST NOT invalidate the remainder of the release, and it MUST be retryable without
  cutting a new release.
- **FR-020**: The description MUST be updated in the same change as any change to RepoKeeper's MCP
  tool surface, and MUST distinguish read-only tools from mutating ones.

**Container image**

- **FR-021**: Each release MUST publish a container image for 64-bit Intel and 64-bit ARM Linux,
  tagged with the release version and with a moving current tag. The image MUST be built and pushed
  by the same release invocation that produces the archives and packages, from the same binaries, so
  that a failure to publish it fails the release as a unit.
- **FR-022**: The image's default behavior when run with no arguments MUST be to start the MCP
  server on stdio.
- **FR-023**: The image MUST run as an unprivileged user and MUST contain no credentials or
  configuration specific to any user or organization.
- **FR-024**: The container's workspace contract MUST be an identical-path mount: the user's
  workspace is made available inside the container at the same absolute path it occupies on the
  host, so every path recorded in `.repokeeper.yaml` resolves unchanged. RepoKeeper MUST NOT
  introduce a path-translation or workspace-root-remapping layer between the registry file and the
  paths its tools resolve; the registry remains the single source of truth for where a repository
  lives (Principles II and III).
- **FR-025**: The documented default MUST be a read-only mount. Under a read-only workspace, every
  read-only inspection tool MUST function as it does natively, and every mutating tool MUST refuse
  with a message naming the read-only mount as the cause and stating what would enable it — a
  read-write mount and explicitly supplied credentials. Refusing without a reason, or silently
  omitting the mutating tools from the advertised surface, is a defect (Principles VI and VII).
- **FR-026**: A containerized server MUST be pointed at its workspace explicitly, and each configured
  server entry serves exactly one workspace root. RepoKeeper's registry discovery walks upward from
  the working directory to find the nearest `.repokeeper.yaml`, which is what lets the native CLI
  serve several purpose-specific roots by position; a container's working directory and mounts are
  fixed before any tool call, so that behavior cannot be reproduced and MUST NOT be implied. The
  image MUST NOT ship a default workspace location — not a fixed path, and not one anchored to a home
  directory — because a default that is correct for single-root users and silently wrong for
  multi-root users is worse than requiring the workspace to be named.
- **FR-027**: When no workspace is available, RepoKeeper MUST report the path it searched, state that
  discovery walks upward from the working directory, and state how to supply a workspace. It MUST NOT
  report an empty inventory as a successful scan, and it MUST NOT serve a different workspace than
  the one the user intended without saying which one it found.
- **FR-028**: The image MUST carry an SBOM, a signature and a build-provenance attestation
  equivalent to those on the release archives.
- **FR-029**: Documentation MUST show a container-based MCP client configuration alongside the
  existing local-binary configuration, and MUST state each of the following plainly: the
  identical-path mount form; which parts of the tool surface are available under the read-only
  default and what enabling the rest requires; that one configured entry serves one workspace root,
  so a multi-root workflow needs one entry per root and the native binary remains the better path for
  it; and that the container supports Git only, Mercurial being unavailable in it. The container is a
  portability and discovery channel, and presenting it as equivalent to the native CLI would
  misrepresent it (Principle IX).

**Pipeline coherence and scope discipline**

- **FR-030**: Every channel except the MCP registry MUST be produced by the single release
  invocation — archives, Linux packages, the Homebrew cask and the container image alike — so a
  failure in any one of them fails the release as a unit rather than half-landing it. The MCP
  registry is the only permitted exception, for the reason stated in FR-019; no further channel may
  be carved out without a recorded justification, because each carve-out reinstates the
  half-landed-release failure mode this requirement exists to prevent.
- **FR-031**: A missing or unusable credential MUST fail the release rather than silently skipping
  the channel it belongs to. A channel that quietly drops out of a release that then reports success
  is the ADR-0007 failure mode.
- **FR-032**: After a release publishes, an independent verification step MUST confirm that each
  channel actually landed at the released version — release assets present and checksummed, the
  Homebrew cask updated in `skaphos/homebrew-tools`, the container image resolvable on both
  architectures, and the registry entry at the released version. Confirmation MUST be obtained by
  querying the channel, never by trusting the publishing step's own report of what it did.
- **FR-033**: Verification MUST fail the release workflow when any channel other than the MCP
  registry is missing or stale. A missing or stale registry entry MUST be surfaced as a distinct,
  visible, non-blocking failure, consistent with FR-019.
- **FR-034**: Verification MUST distinguish a channel that genuinely did not publish from a
  transient failure to reach it, retrying before declaring a channel missing.
- **FR-035**: No requirement in this feature may introduce a code path that mutates a user's working
  tree outside RepoKeeper's existing, explicitly opt-in sync flows. Principle X's safe-by-default
  guarantee and the read-only MCP tool classification are unchanged by this feature.
- **FR-036**: Installation and upgrade documentation MUST be updated for every channel this feature
  adds, and MUST make clear which upgrade path applies to which install path.
- **FR-037**: This feature MUST NOT add a dependency to RepoKeeper's module graph. Most of what it
  delivers is release-pipeline configuration, packaging metadata or documentation; the two changes to
  compiled code — version identity (FR-001 – FR-006) and the read-only-mount refusal (FR-025) — MUST
  use only the standard library. Any proposal that breaches this requires the same written
  justification the constitution demands of dependency growth, and the measurement under Prior Art is
  the standard it is judged against.

### Key Entities

- **Version identity**: what a running binary knows about itself — version, source revision, build
  time, and whether the working tree was clean. Sourced from release-time stamping when present and
  from the build's own recorded metadata otherwise.
- **Release artifact set**: everything one release publishes — archives, Linux packages, the
  container image, the checksum manifest, SBOMs, signatures and attestations — plus which of them
  actually landed.
- **Server description**: the checked-in declaration of RepoKeeper's MCP identity, tool surface,
  runtime and configuration, published to the registry and kept in step with the released version.
- **Container workspace contract**: how a containerized RepoKeeper is given access to the
  repositories it reports on, what that access permits, and which single workspace root it serves —
  an identical-path mount, read-only by default, named explicitly rather than discovered. Defined by
  FR-024 through FR-027.
- **Workspace root**: the directory holding a `.repokeeper.yaml` registry. The native CLI selects one
  by walking upward from the working directory, so several purpose-specific roots coexist and
  self-select by position. A container serves exactly one per configured entry, fixed before any tool
  call — the distinction FR-026 exists to keep honest.
- **Install provenance**: which channel placed the running binary — Homebrew prefix, system package
  database, language toolchain, hand-placed archive, or container layer. RepoKeeper does not detect
  this in code (FR-007); the concept survives as the axis along which `README.md`'s upgrade table is
  organized (FR-009), and it is what a future self-updater would have to determine.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A binary installed from source reports a real version and revision; the placeholder
  value that ships today appears in zero installation paths that produce a usable binary.
- **SC-002**: A user on a mainstream Debian- or RPM-based distribution goes from the release page to
  a working `repokeeper` on `PATH` in a single package-manager command, on both 64-bit Intel and
  64-bit ARM.
- **SC-003**: RepoKeeper is resolvable by name in the MCP registry, at the current released version,
  within one release cycle of this change landing.
- **SC-004**: An MCP client configured to run RepoKeeper from the published container image
  completes a workspace inventory query against a supplied workspace on both supported
  architectures, with no local Go toolchain present.
- **SC-005**: For a workspace mounted read-only at its host path, the containerized server returns
  results identical to a natively installed binary for 100% of read-only inspection tools; and 100%
  of mutating tools refuse with the read-only mount named. Zero tools fail in a way that does not
  state its cause.
- **SC-006**: Zero container configurations serve a workspace silently: every one either answers from
  the root it was explicitly given, or reports that no workspace was found and which path it
  searched. A user with multiple workspace roots can determine from the documentation alone how many
  entries they need, without running the container to find out.
- **SC-007**: Every artifact in a release carries provenance appropriate to its form — archives and
  Linux packages listed in the signed checksum manifest, the container image signed and attested in
  the registry it is published to. Zero unsigned or unattested artifacts ship.
- **SC-008**: Every published release is followed by an independent per-channel confirmation
  obtained by querying each channel. A release in which a required channel did not land is marked
  failed by that check rather than discovered by a user; zero releases complete as apparently
  successful while a channel is missing.
- **SC-009**: A user reading the installation documentation can identify the correct upgrade command
  for their install path without consulting any other source, for every one of the six channels
  RepoKeeper ships on.
- **SC-010**: Zero required channels are absent from both the shipped implementation and the ADR
  record — every gap is either closed or documented. Specifically, self-update is absent from the
  implementation and present in ADR-0016.
- **SC-011**: This feature adds zero entries to `go.mod` and `go.sum`; the binary size delta
  attributable to it is under 1%, against a measured +128.7% for the alternative that was declined.
- **SC-012**: RepoKeeper's existing behavior is unchanged by this feature: the MCP tool surface, the
  read-only tool classification, sync's opt-in mutation gating, and registry file formats behave
  identically before and after.

## Assumptions

- **Windows package managers are out of scope for this feature.** `winget` and `scoop` are
  *recommended*, not required, under DECISIONS/0001. RepoKeeper builds both Windows architectures
  and ships them as bare zips today, but Windows Authenticode signing does not exist yet, and the
  issue records that pushing unsigned binaries into those channels is worse than not pushing them —
  SmartScreen will warn on them. They are therefore deferred behind Authenticode signing, matching
  sting's disposition. This is a scope decision taken from the issue's own guidance, not a silent
  drop of a required channel.
- **The MCP registry is the one channel exempt from failing the release as a unit.** DECISIONS/0001
  states the registry is the least settled channel and that a break there is a smaller emergency
  than a broken cask.
- **The `io.skaphos` registry namespace and its DNS proof are inherited, not re-decided.** sting
  established the namespace and the TXT record on `skaphos.io`; RepoKeeper publishes under the same
  namespace as `io.skaphos/repokeeper`.
- **The existing macOS signing and notarization arrangement is unchanged by this feature.** The known
  gap — that the notarization block silently skips when its `MACOS_*` secrets are unset — is noted as
  an edge case above and is separate follow-up work.
- **Promoting the `MACOS_*` secrets from repository to organization scope is out of scope here.**
  DECISIONS/0001 calls for it; it is an org-administration change tracked in `skaphos-resources`.
- **The ADR-0007 §2 `postflight` quarantine hook in the Homebrew cask stays.** Notarization now
  exists, but `xcrun stapler` cannot staple a ticket to a bare Mach-O in a tarball, so Gatekeeper
  still fetches the ticket online and an offline first run would fail without the hook. Removing it
  is a separate call backed by testing on real hardware.
- **Release infrastructure stays as it is.** The release remains tag-triggered and driven by a
  single GoReleaser invocation, with the GitHub release owned per [ADR-0013][rk0013] and release
  notes per [ADR-0012][rk0012]. This feature adds outputs to that run; it does not restructure it.
- **The container image serves the MCP server use case.** It exists because Shape 3 requires a
  `docker`-runnable server; it is not positioned as a general-purpose way to run the CLI, whose
  natural home is the host filesystem it inspects.
- **The upstream record is corrected by supersession, tracked separately.** DECISIONS/0001 is
  Accepted and therefore immutable; the three defects this feature surfaces are carried by a
  successor record, `DECISIONS/0002`, in `skaphos-resources`. That work is a byproduct of this
  feature, not a dependency of it — RepoKeeper's obligations under the standard as written are what
  this spec delivers against, and the deviation is recorded in ADR-0016 regardless of whether the
  successor lands.
- **Self-update is deviated from, not silently omitted.** DECISIONS/0001 lists it as required for
  Shape 2. This feature declines to deliver it on measured evidence and records ADR-0016 as the
  standard's *Deviating* section requires. If the successor record rescopes the requirement, ADR-0016
  becomes the input to that decision rather than being invalidated by it.

## Out of Scope

- Hosted, signed apt or yum repositories. DECISIONS/0001 excludes them explicitly; they would need
  their own decision record.
- AUR packaging and a Nix flake. Optional under DECISIONS/0001 and not expected.
- Windows Authenticode signing, and the `winget` and `scoop` channels that depend on it.
- Homebrew core, Chocolatey, Snap and Flatpak — all deliberately excluded by DECISIONS/0001.
- Removing the Homebrew cask's quarantine hook.
- Any change to RepoKeeper's inventory model, sync policy, prune-safety classification, MCP tool
  surface, configuration precedence, or registry file formats.
- **A `repokeeper update` self-update subcommand.** Required for Shape 2 by DECISIONS/0001 and
  therefore a recorded deviation, not a silent omission: ADR-0016 documents it. Briefly —
  verification cannot be skipped, verifying in-process is the only way to avoid requiring a signing
  tool on the user's machine, and the dependency that makes that possible more than doubles the
  binary (+128.7%, measured under Prior Art) for every user on every channel, to serve a replace path
  reachable on one channel of six and incoherent on a seventh. Upgrades run through the channel the
  user installed from, documented per channel in `README.md`.
- **Detecting install provenance in code.** With no replace path to gate, ownership detection has no
  caller. It is described under Key Entities and organizes `README.md`'s upgrade table, but ships as
  documentation rather than as a code path.
- **Authoring `DECISIONS/0002`.** The supersession this feature's findings justify belongs to
  `skaphos-resources` and is tracked there.

## Dependencies

- **DECISIONS/0001 (`skaphos-resources`)** — the standard being adopted; defines the required
  channel set per shape and the three self-update rules.
- **sting's implementation of the same standard (`skaphos/sting#121`)** — the reference
  implementation for version identity, Linux packages, the registry entry and the container image,
  and the precedent for the self-update deviation. Adopted, not copied: the cost was re-measured on
  RepoKeeper's own module graph before the same verdict was accepted.
- **Existing release supply chain** — checksum manifest, per-archive SBOM, cosign signature and
  build-provenance attestation. Every new channel extends these rather than sitting beside them.
- **`skaphos/homebrew-tools` and the release bot credentials** — unchanged, but they share the
  release run whose failure semantics this feature tightens.
- **The MCP registry's publishing interface and schema** — external, and the least stable dependency
  in this feature, which is why its failure mode is specified separately.
- **DNS control over `skaphos.io`** — the `io.skaphos` namespace is proven by a TXT record on that
  zone. Together with the publishing credential this is a second expiry-and-rotation surface in the
  release path, alongside the Apple Developer certificate. Both fail quietly; post-release
  verification is what makes either visible.
- **GitHub Container Registry** — the publish target for the container image.

## Constitution Alignment

- **II. Git Is the Durable Desired-State Boundary** — the identical-path container mount exists so
  `.repokeeper.yaml` stays the single statement of where a repository lives; a remapping layer would
  make the container a second, divergent interpretation of that file (FR-024).
- **III. Deterministic, Reconstructible Operation** — version resolution is a pure function of what
  the build recorded and what the release stamped, with a fixed precedence (FR-003, FR-005); paths
  resolve identically inside and outside the container (FR-024).
- **VI. Explainable Reconciliation, Evidence-Grade Audit** — a build that cannot state its version
  says so rather than guessing; a mutating tool under a read-only mount names the mount as the
  reason; a container with no workspace says so instead of reporting an empty scan; a channel that
  did not publish is named (FR-004, FR-019, FR-025, FR-027, FR-032).
- **VII. Read-Only Degradation Over Blindness** — the container's read-only default is this principle
  made concrete: full inspection always, mutation refused with a reason and a stated remedy, never a
  silently reduced surface (FR-025).
- **IX. Technical Precision, Honest Scope** — the documentation must state that shipping a `.deb` is
  not an apt repository, that Windows binaries remain unsigned, which tools work under the read-only
  container default, that the container serves one workspace root per entry, and that RepoKeeper
  ships no self-updater and why (FR-009, FR-015, FR-029, FR-036, and ADR-0016).
- **X. Safe-by-Default VCS Operations** — nothing here adds a path that mutates a working tree
  outside the existing opt-in sync flows, and the container defaults to a mount that cannot
  (FR-025, FR-035).
- **XII. CLI-First With Machine-Readable Output** — version information is available in both
  human-readable and machine-readable form, and the Linux packages exist because Linux and WSL are
  first-class platforms, not courtesies (FR-006, FR-011).
- **Adopt before build** (Specification and Decision Workflow) — sting's implementation is mature
  prior art for four channels and is adopted rather than re-derived. Its self-update verdict was
  *re-measured* rather than inherited on faith, because the constitution requires a documented reason
  when prior art is departed from, and the same standard should apply to accepting it.
- **Engineering Constraints — dependencies minimized** — FR-037 makes zero dependency growth a
  requirement of this feature rather than an outcome, and the Prior Art measurement is the evidence
  behind it.
- **Attribution** — because FR-037 holds, `third_party_licenses/` and `THIRD_PARTY_NOTICES.md` need
  no regeneration. Had the self-updater shipped, +162 `go.sum` entries would have made that a
  substantial review, which is itself part of the cost recorded in ADR-0016.
- **Testing (non-negotiable)** — new behavior ships with meaningful tests in the same change, tests
  do not touch the network, and tests that touch the filesystem isolate `HOME` and `USERPROFILE`.
  The per-package coverage gate applies. This feature adds two pieces of compiled behavior — version
  identity (FR-001 – FR-006), covered for each of its four resolution outcomes, and the
  read-only-mount refusal (FR-025), covered for both the refusal message and the unaffected
  inspection path.
