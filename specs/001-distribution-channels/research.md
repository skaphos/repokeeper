# Phase 0 Research: Distribution Channel Conformance

**Feature**: `specs/001-distribution-channels` | **Date**: 2026-07-26

Every unknown carried into planning, resolved. Findings that changed the design are marked
**⚠ DIVERGENCE** where RepoKeeper cannot follow sting's implementation.

---

## R1 — Version identity: three-tier resolution

**Decision**: Add `internal/buildinfo` resolving identity from three sources in precedence order —
release-time ldflags, then `debug.ReadBuildInfo()`, then nothing. `cmd/repokeeper/version.go` keeps
its exported `Version`/`Commit`/`Date` ldflags variables (the `.goreleaser.yaml` `-X` flags reference
them by path and must not move) and delegates resolution to the new package.

**Rationale**: The tiers are complementary, not redundant. A module-proxy install
(`go install ...@latest`) records `Main.Version` but no VCS settings; a local `go build` records
`vcs.revision`/`vcs.time`/`vcs.modified` but leaves `Main.Version` as `(devel)`. Neither is a
superset of the other, so both are consulted. Two sentinel values must be mapped back to "not
recorded" rather than surfaced: the current ldflags default `"dev"` (and `"none"`/`"unknown"` for
commit/date), and the toolchain's `(devel)`. Surfacing `(devel)` is the same defect as surfacing
`dev` — FR-004 forbids both.

**Alternatives considered**:
- *`ReadBuildInfo` only, drop ldflags* — rejected: FR-003 requires released binaries to report
  exactly what they report today, and the module version is absent from a GoReleaser build.
- *Keep `"dev"` as a fallback string* — rejected by FR-004.
- *Resolve at each call site* — rejected by FR-005; `mcpserver.New(eng, cfgPath, Version, logger)`
  already passes the version into the MCP server, so two resolution paths could disagree about what
  the server advertises versus what `version` prints.

**Prior art**: sting `internal/buildinfo/buildinfo.go` (149 lines, 183 lines of tests). Adopted
structurally; the package doc and sentinel handling transfer directly.

---

## R2 — Container base image ⚠ DIVERGENCE

**Decision**: Alpine-based image carrying the `git` binary. **Not** `gcr.io/distroless/static`,
which sting uses.

**Rationale**: `internal/gitx/gitx.go` states it outright — *"It shells out to the installed git
binary."* `GitRunner.Run` builds `exec.CommandContext(ctx, "git", args...)`. Every inspection tool
RepoKeeper exposes (tracking status, worktree status, remotes) is a `git` subprocess. A
distroless-static image has no shell, no package manager, and no `git`, so every tool would fail at
`exec.LookPath`. This is the sharpest divergence from sting, whose server only makes HTTPS calls and
genuinely needs nothing but CA certificates.

Base pinned by digest per the org container standard; `git` and `ca-certificates` installed with
`apk add --no-cache`. The `git` version then floats with the Alpine tag, which is acceptable and must
be *stated* rather than implied — a digest pin on the base does not pin the package versions
installed on top of it.

**Alternatives considered**:
- *Distroless + copied-in git* — rejected: `git` is not a static binary; it needs libc and invokes
  helper executables. Assembling a working git into distroless by hand is a maintenance liability.
- *`debian:stable-slim`* — viable and boring, but materially larger for the same content.
- *`alpine/git` as the base* — rejected: a third-party image in the trust path for no gain over
  installing the package ourselves.
- *Rewriting `gitx` onto a pure-Go git library* — rejected outright: a rewrite of RepoKeeper's core
  execution layer is not in this feature's scope and would change behavior across every command.

**Consequence for the CLI binary**: none. The binary stays `CGO_ENABLED=0` static; only the image
gains a git runtime.

---

## R3 — Git "dubious ownership" in the container ⚠ DIVERGENCE, measured

**Decision**: Bake `safe.directory=*` into a system-level gitconfig in the image.

**Rationale**: Measured, not assumed. Git 2.35.2+ refuses to operate on a repository whose directory
is owned by a different uid than the running process. A bind-mounted workspace is owned by the host
user (typically uid 1000); an image running as the conventional non-root uid 65532 hits this on
*every single git invocation*. Tested against `alpine/git`, git 2.55.0, with a real repository
bind-mounted read-only at its host path:

| Scenario | Result |
| --- | --- |
| `-u 65532:65532`, no config | `fatal: detected dubious ownership in repository at '…'` — **exit 128** |
| `-u 65532:65532`, `safe.directory=*` | exit 0, clean output |
| `-u 1000:1000` (matches host owner), no config | exit 0, clean output |

Without mitigation, FR-025's "read-only inspection tools function as they do natively" fails for
100% of tools, and the user sees a git internals error rather than anything RepoKeeper explains.

The security posture must be stated honestly (Principle IX). `safe.directory=*` is discouraged on a
shared multi-user host because it lets git act on repositories owned by others. Inside this
container the threat model is different: the filesystem view *is* exactly what the user chose to
mount, so there is no foreign repository to be tricked into trusting. That reasoning belongs in the
Dockerfile as a comment and in the docs, not left implicit.

**Alternatives considered**:
- *Document `--user $(id -u):$(id -g)` and nothing else* — rejected as the sole mitigation: it works
  (row 3), but a user who forgets gets a total, cryptic failure. It is retained as the
  **recommended** form for read-write use, because it also makes any files git creates land with the
  right ownership.
- *`GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_0` env vars* — works (row 2), but pushes required
  configuration into every client's MCP config, where it will be omitted.
- *Run the container as root* — rejected by FR-023.

---

## R4 — Workspace discovery: cwd-relative crawl vs. a frozen container ⚠ DIVERGENCE

**Decision**: The container's workspace is **fixed per configured server entry**, pinned with an
explicit `--config`. Multi-root users configure one entry per root, or use the native binary. This
limitation is documented, not worked around.

**Rationale**: This is the second place RepoKeeper's shape breaks sting's model, and it is a
usability constraint rather than a bug.

`cmd/repokeeper/mcp.go` resolves configuration from the process working directory — `os.Getwd()`,
then `config.ResolveConfigPath(configOverride(cmd), cwd)`, which walks *upward* through ancestors
looking for a `.repokeeper.yaml` regular file. That upward crawl is a deliberate host-ergonomics
feature: it is what lets a developer work in `~/work/skaphos/repokeeper` and have the registry at
`~/work/skaphos/.repokeeper.yaml` found automatically, four or five directories up, without naming
it. Run the same binary from `~/work/alaska/some-service` and it finds *that* root's registry
instead. Multiple roots, each with a distinct purpose, each self-selecting by where you happen to be.

**A container cannot reproduce this.** Its working directory and its mounts are both fixed at
`docker run`, before any tool call. The MCP client launches the container once and it serves whatever
root it was launched against, forever. There is no cwd to be relative *to* — the notion the crawl
depends on does not exist in that process.

Three consequences follow, and all three are contract, not code:

1. **One container entry serves exactly one workspace root.** A developer with N purpose-specific
   roots needs N MCP server entries, each with its own `-v` mount and `--config`. This is honest and
   workable, but it must be stated, because a user who assumes the container behaves like the CLI
   will find it silently serving the wrong workspace — a confidently wrong answer, which
   Principle VI classes as a defect.
2. **Pinning the workspace to a home directory is not an option.** Some users do keep a single
   registry at `~/.repokeeper.yaml`, and for them a `$HOME`-anchored default would work. It would be
   wrong for everyone else, and a default that is right for some users and silently wrong for others
   is worse than no default. The image therefore ships **no** `WORKDIR` default and requires the
   workspace to be named explicitly.
3. **`--config` is preferred over `-w` for the container.** Both work — `-w <path>` restores enough
   cwd context for the crawl to succeed — but `--config` states the registry outright instead of
   depending on a search whose starting point is invisible in the MCP client's configuration. In a
   context where the path is already known and fixed, determinism beats convenience (Principle III).

**The native path is unaffected and remains the multi-root answer.** `repokeeper mcp` launched
directly by an MCP client inherits that client's working directory, so the crawl behaves exactly as
it does on the command line. Nothing in this feature changes it. The container is a portability and
discovery channel — no Go toolchain, no local install — not a replacement for the native binary in a
multi-root workflow, and the documentation must say so rather than presenting the two as equivalent.

**Impact on FR-026**: strengthened. "No workspace available" must name *the path that was searched*
and state that discovery walks upward from the working directory, so a user who mounted at the wrong
place, forgot `--config`, or expected the CLI's crawl behavior can diagnose it. The existing message
in `mcp.go` — `registry not found in %q (run repokeeper scan first)` — names the resolved path and
is a good base, but it is reached only when `ResolveConfigPath` *succeeded* and the registry was
empty. The case that matters here is discovery failing outright, which must be equally explicit.

**Alternatives considered**:
- *Default `WORKDIR` to a fixed path such as `/workspace`* — rejected: it implies a remap contract
  FR-024 forbids, and would be correct only for users who happen to mount there.
- *Anchor to `$HOME` inside the container* — rejected per consequence 2.
- *A `REPOKEEPER_CONFIG` environment variable* — rejected as new surface this feature does not need;
  `--config` already exists and is visible in the client configuration where a reader will look.
- *Mount a parent of all roots and rely on the crawl* — rejected as a general answer: it only works
  when the roots share an ancestor that itself holds a registry, and it silently merges purposes the
  user deliberately separated.

---

## R5 — Linux packages via GoReleaser `nfpms`

**Decision**: An `nfpms` block producing `deb` and `rpm` for `amd64` and `arm64`, with
`file_name_template: "{{ .ConventionalFileName }}"`, installing `LICENSE`,
`THIRD_PARTY_NOTICES.md` and `third_party_licenses/` under `/usr/share/doc/repokeeper/`.

**Rationale**: Native GoReleaser feature consuming binaries already built, so FR-029's
single-invocation guarantee holds with no extra machinery. `ConventionalFileName` yields
`repokeeper_0.8.0_amd64.deb` and `repokeeper-0.8.0-1.x86_64.rpm` — each format's own convention
rather than one shape forced onto both.

**Critical detail**: packages do **not** inherit the archive SBOM. A second `sboms` entry with
`artifacts: package` is required or the `.deb`/`.rpm` ship without one, breaking FR-014. This is
easy to miss because the archives-only config looks complete.

**Prior art**: sting `.goreleaser.yaml` lines 72–112. Transfers directly with names changed.

---

## R6 — Container publishing via `dockers_v2`

**Decision**: A `dockers_v2` block pushing `ghcr.io/skaphos/repokeeper` for `linux/amd64` and
`linux/arm64`, tagged `{{ .Version }}` and `latest` (suppressed on prereleases), with a `Dockerfile`
that has **no build stage** — it copies the binary GoReleaser already built.

**Rationale**: `dockers_v2` assembles a buildx manifest from pre-built binaries laid out per platform
(`linux/amd64/repokeeper`, `linux/arm64/repokeeper`). Compiling inside the Dockerfile would produce a
*different* binary from the one in the archives, breaking the "same binaries" guarantee FR-011 and
FR-021 rest on — the bytes that were signed, notarized, checksummed and attested must be the bytes
that ship. Being inside the GoReleaser run is also what makes a container-publish failure fail the
release as a unit (FR-029).

Requires in the workflow: QEMU setup, Buildx setup, GHCR login, and `packages: write` permission.

**Note**: `dockers_v2` differs from the older `dockers` + `docker_manifests` pair, which builds
per-arch images and stitches a manifest afterwards. `dockers_v2` is the current form and is what
sting uses at GoReleaser 2.17.0 — the version already pinned in RepoKeeper's `.tool-versions`.

---

## R7 — MCP registry entry

**Decision**: Checked-in `server.json` declaring `io.skaphos/repokeeper`, schema-validated in CI with
`check-jsonschema`, plus Go tests guarding it against drift from the live server. Published at
release by `mcp-publisher` authenticating via DNS on `skaphos.io`.

**Rationale**: Two layers catch different failures. Schema validation catches a malformed entry
before merge. The Go tests catch something schema validation cannot: the entry describing a surface
RepoKeeper does not have — which is FR-020's requirement and the more dangerous case, since clients
act on the description. RepoKeeper's tests differ from sting's in one respect: sting asserts *every*
tool is read-only, which is true of sting and false of RepoKeeper. The equivalent assertion here is
that the entry's description of the tool surface agrees with `mcpserver.ReadOnlyToolNames()`, which
derives from the live `ReadOnlyHint` annotations and therefore cannot drift.

Both the top-level `version` and `packages[].version` are stamped from the tag at release time; a
checked-in placeholder (`0.0.0`) that diverges between the two fields would publish an entry naming
an image tag that does not exist.

**Alternatives considered**:
- *Schema validation only* — rejected: cannot detect a truthful-looking but wrong description.
- *`mcp-publisher@latest`* — rejected: makes releases non-reproducible and lets an upstream change
  break a release. Pin, and bump deliberately.
- *GitHub-anchored namespace* — settled upstream by sting; `io.skaphos` is inherited (FR-016).

---

## R8 — Release pipeline: credentials and verification

**Decision**: Two changes to `.github/workflows/release.yml`. First, replace the current
credential-driven skip with a hard pre-flight failure. Second, add a `verify` job that queries each
channel after publishing.

**Rationale**: RepoKeeper's release workflow today contains exactly the failure mode FR-030 forbids.
The `Determine GoReleaser args` step emits `--skip=homebrew` and a `::warning::` when the token is
missing or cannot reach the tap, then completes green. A release that silently drops the cask and
reports success *is* the ADR-0007 incident — the cask pinned at `0.6.0` across two releases — encoded
as workflow logic. The fix is to fail before GoReleaser runs, with a message naming which credential
to check.

Verification is a separate job (`needs: release`) because it tests a different proposition. That the
release job succeeded is already visible; what is not visible is a channel that responded correctly
while serving the *previous* version. So reachability is not the test — the version served is the
test. Each check retries with increasing backoff (FR-033) so a third-party outage is distinguishable
from a channel that did not publish.

Per-channel disposition:

| Channel | Query | On failure |
| --- | --- | --- |
| Release assets | GitHub API asset list + checksums | fail |
| Homebrew cask | raw `Casks/repokeeper.rb`, grep the version | fail |
| Container image | `docker buildx imagetools inspect --raw`, assert both arches present in the manifest | fail |
| MCP registry | registry search API for the version | **warn, `continue-on-error`** |

The registry is the sole non-blocking channel, per FR-032 and DECISIONS/0001's own statement that a
break there is a smaller emergency than a broken cask. Its publish step is likewise
`continue-on-error` with an `always()` reporting step naming the DNS TXT record and the credential as
the things to check — the two things that fail quietly.

---

## R9 — Explaining a read-only mount to a mutating tool

**Decision**: A small stdlib-only helper that recognizes a read-only-filesystem write failure and
returns a message naming the mount and the remedy. Applied where the mutating MCP tools write.

**Rationale**: FR-025 requires a refusal that *names the read-only mount as the cause and states what
would enable it*. Without translation, `executeSync` surfaces a raw git error and the registry-writing
tools surface a raw `*fs.PathError: read-only file system` — both are failures, but neither is an
explained one, and Principle VI treats "failed without a reason and a next safe action" as a defect.

Detection uses `errors.Is(err, syscall.EROFS)`, which is reliable and needs no string matching. The
container is Linux-only, so the check lives behind `//go:build unix` with a no-op fallback elsewhere;
Windows has no read-only bind mount to detect.

**Alternatives considered**:
- *Probe writability at startup and hide the mutating tools* — rejected: FR-025 explicitly forbids
  silently omitting them from the advertised surface. A tool that exists and explains why it cannot
  run is honest; a tool that vanishes is not.
- *String-match the error text* — rejected as locale- and version-fragile. Note `gitx.Run` already
  forces `LC_ALL=C` for exactly this reason, which shows the cost of the pattern.

**⚠ Spec inconsistency found.** FR-036's parenthetical asserts "the one change to compiled code —
version identity". This work is a second compiled change. The *normative* requirement — MUST NOT add
a dependency — is unaffected, since both changes are stdlib-only. The parenthetical is inaccurate and
is corrected in `spec.md` as part of this plan. Recorded here rather than silently fixed, because the
requirement was written before the container contract was settled.

---

## R10 — Mercurial in the container

**Decision**: The image carries `git` only. Mercurial support is unavailable in the container, stated
in the docs.

**Rationale**: Principle XI already scopes Mercurial as experimental and opt-in per command, and
Principle IX requires stating plainly what RepoKeeper is not. Installing `hg` — a Python runtime —
into the image to serve an experimental backend is a poor trade. The honest statement costs nothing;
the silent absence would be a defect, since an `--vcs hg` call in the container would fail at
`exec.LookPath` with no explanation.

---

## R11 — ADR-0016 and the deviation record

**Decision**: `docs/adr/0016-no-self-update-subcommand.md`, Accepted, with the measurement table from
the spec's Prior Art section, and an entry in `docs/adr/README.md`.

**Rationale**: DECISIONS/0001's *Deviating* section requires a record in the repo that drops a
required channel. 0016 is the next free number (0001–0015 exist). Immutable per the constitution: if
the cost profile changes, 0016 is superseded, not edited.

**Reversal condition to state explicitly**: a verification path whose cost is materially lower —
whether that is a lighter in-process verifier, or DECISIONS/0002 rescoping the requirement for tools
that span Shape 2 and Shape 3.

---

## R12 — Zero dependency growth

**Decision**: Verify FR-036 mechanically in CI rather than trusting review.

**Rationale**: The whole self-update decision rests on dependency cost. A feature justified by
avoiding +59 requirements should not add any by accident. `go.mod` and `go.sum` are unchanged by
every item in this plan: `internal/buildinfo` uses `runtime/debug`, the read-only helper uses
`errors`/`syscall`, and every other deliverable is YAML, JSON, a Dockerfile, or Markdown. The
existing `build` CI job already fails on a dirty tree after `go mod tidy` in most Skaphos repos —
confirm and rely on it rather than adding new machinery.

---

## Resolved unknowns summary

| # | Unknown | Resolution |
| --- | --- | --- |
| R1 | How does a non-release build learn its version? | Three-tier `internal/buildinfo` |
| R2 | Which container base? | Alpine **+ git** — not distroless ⚠ |
| R3 | Does git work on a bind-mounted repo as non-root? | No — needs `safe.directory=*` ⚠ measured |
| R4 | How does the container find the registry? | Explicit `--config`; one entry per root ⚠ |
| R5 | How are `.deb`/`.rpm` produced and covered? | `nfpms` + a **second** `sboms` entry |
| R6 | How is the image built from the same bytes? | `dockers_v2`, no build stage in the Dockerfile |
| R7 | How is `server.json` kept truthful? | Schema validation **and** drift tests |
| R8 | How is a half-landed release caught? | Pre-flight credential failure + `verify` job |
| R9 | How does a mutating tool explain a read-only mount? | `errors.Is(err, syscall.EROFS)` translation |
| R10 | Does the container support Mercurial? | No — git only, documented |
| R11 | Where does the deviation live? | ADR-0016 |
| R12 | How is zero dependency growth enforced? | Existing tidy-drift gate in CI |

No `NEEDS CLARIFICATION` items remain.
