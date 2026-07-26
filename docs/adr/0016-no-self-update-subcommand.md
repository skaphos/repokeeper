# ADR-0016: No Self-Update Subcommand

**Status:** Accepted
**Date:** 2026-07-26
**Author:** Shawn Stratton

## Context

`skaphos-resources` [DECISIONS/0001 — Distribution channels by artifact shape][adr0001] lists a
`<tool> update` self-update subcommand as **required** for every Shape 2 tool, subject to three
rules: verify before replacing, defer to the package manager, and make no implicit network calls.
RepoKeeper is a Shape 2 tool, so the requirement applies.

That record also says what happens when a required channel is not delivered:

> What does not deviate silently is dropping a **required** channel — that needs a record in the
> repo that drops it.

This is that record.

The first rule is the binding one. Verification cannot be skipped, and DECISIONS/0001 is explicit
that a self-updater ignoring the signing material a release already publishes "is strictly worse
than no self-updater." Satisfying it without requiring `cosign` on the user's machine — which is
nearly every end user's machine — means verifying in-process, which means linking a Sigstore
verification stack into the binary.

`skaphos/sting` reached this point first. It specified, implemented and shipped a conforming
`sting update`: cosign bundle and checksum verification with the signer identity pinned to its own
release workflow, package-manager deferral, atomic replacement, explicit target versions for
rollback. It then measured the result and reverted it, recording
[sting ADR 0011][sting0011]. The command worked; the cost was the problem.

## Decision

**RepoKeeper ships no `update` subcommand.**

There is no checksum-only mode and no "verify if a signing tool happens to be present" fallback. The
choice is a correct updater at the cost measured below, or no updater. A weakened updater is the
option DECISIONS/0001 rules out, and it is not taken here.

Upgrades run through whichever channel the user installed from. `README.md` carries a per-channel
upgrade table in place of the command.

The no-implicit-network-calls rule is satisfied vacuously and must stay that way: no RepoKeeper
command may contact a release or version endpoint for any purpose, including background or
opportunistic update checks. A future `update` command would not license ambient version checks in
unrelated commands.

## Rationale

### The cost was measured on RepoKeeper, not inherited from sting

sting's percentage was not assumed to transfer. The same verification surface sting used
(`sigstore-go`'s `pkg/bundle`, `pkg/root`, `pkg/verify`, `pkg/fulcio/certificate`) was added to a
copy of RepoKeeper at `48fd6de` and built with the release flags from `.goreleaser.yaml`
(`-s -w`, `CGO_ENABLED=0`, `-trimpath`, `linux/amd64`):

| Measure | Baseline | With in-process verification | Delta |
| --- | --- | --- | --- |
| Binary size | 10,072,226 B (9.6 MiB) | 23,031,970 B (22.0 MiB) | **+12.96 MB (+128.7%)** |
| Direct requirements in `go.mod` | 47 | 106 | +59 |
| Modules in the build graph | 95 | 406 | +311 |
| `go.sum` entries | 70 | 232 | +162 |

The result is not better than sting's — it is marginally worse. sting recorded +122%; RepoKeeper
measures +128.7%, a 2.29× binary. The measurement scaffolding was built outside the repository and
discarded; no dependency was added.

This is a direct conflict with the constitution's Engineering Constraints, which require external
dependencies to be minimized, and with its Attribution constraint — +162 `go.sum` entries would make
`third_party_licenses/` regeneration and `THIRD_PARTY_NOTICES.md` review a substantial recurring
cost.

### The benefit reaches one channel of six

Cost alone did not settle it. Once this feature lands, RepoKeeper is installable six ways, and the
verified-replace path is reachable on exactly one:

| Channel | What an `update` command could do |
| --- | --- |
| GitHub release archive, hand-placed | **Replaces the binary** — the only case |
| Homebrew cask | Defers — prints `brew upgrade --cask` |
| `.deb` | Defers — dpkg owns the file; replacing it breaks the package database |
| `.rpm` | Defers — replacing it breaks `rpm -V` |
| `go install` | Defers — the toolchain owns the file |
| Container image | **Incoherent** — see below |

Five of the six take the deferral branch, which requires no verification code at all: it is
install-provenance detection and a printed string. The +12.96 MB buys the replace path for the one
remaining channel — whose users have already demonstrated they can place a binary on `PATH` by hand.

### The container case is worse than a no-op

RepoKeeper is also a Shape 3 tool and publishes a container image. A binary replaced inside a
container's ephemeral writable layer *appears* to update and reverts on the next `docker run`. And
because the image is built from the same binaries as the release archives — the property that keeps
every channel fed from one build — every containerized MCP user would carry +12.96 MB for a command
that cannot function in a container at all.

Building per-channel binaries behind build tags would avoid that, at the cost of breaking the
same-binary guarantee and doubling the signing and SBOM surface. Rejected.

## Consequences

### Positive

- `go.mod` and `go.sum` are unchanged by the distribution-channels feature. `govulncheck` exposure
  does not grow; the unmaintained `golang.org/x/crypto/openpgp` that the Sigstore tree pulled into
  sting's graph never enters RepoKeeper's.
- The binary stays at ~10 MB for every user on every channel.
- No self-replacing code path exists, so the attack surface DECISIONS/0001 describes as "a real
  attack surface" — a process that overwrites its own binary — is absent rather than mitigated.
- `third_party_licenses/` and `THIRD_PARTY_NOTICES.md` need no regeneration.

### Negative

- **RepoKeeper does not conform to DECISIONS/0001 on a required channel.** This is a deviation, and
  it is visible here rather than inferred from a missing command.
- A user who installed a release archive by hand must download and replace the binary themselves.
  This is the one case a self-updater would genuinely have served.
- RepoKeeper cannot tell a user which channel they installed from. The per-channel upgrade table in
  `README.md` requires the user to know how they installed, which most will but some will not.
- The upgrade path is documentation rather than code, so it can drift from reality in a way a
  command could not.

## Reversal conditions

This decision is reversed by either of:

1. **A verification path of materially lower cost.** The decision is about the cost of in-process
   Sigstore verification, not about self-update as an idea. A verifier that fits in a small fraction
   of the current binary changes the arithmetic entirely.
2. **DECISIONS/0002 rescoping the requirement.** DECISIONS/0001's self-update rules sit under the
   Shape 2 heading and were written for a CLI in isolation; they have no answer for a tool that also
   ships a container image. Both of the standard's first adoption targets — sting and RepoKeeper —
   have now deviated from this required channel identically and for the same measured reason, which
   is evidence about the standard rather than about the adopters.

ADRs are immutable. If either condition is met, this record is superseded, not edited.

## Alternatives Considered

- **Ship the full conforming self-updater.** Rejected on the measurement above: +128.7% binary and
  +59 direct requirements carried by all six channels to serve one.
- **Ship a checksum-only updater**, verifying the checksum manifest but not its signature. Rejected —
  DECISIONS/0001 identifies this as strictly worse than shipping nothing, because it presents the
  appearance of verification while ignoring the signing material the release already publishes.
- **Verify by shelling out to `cosign` when present.** Rejected: it makes verification conditional on
  the user's machine, which is the same defect in a different place, and the command would fail on
  nearly every end-user machine.
- **A deferral-only `update` command** that detects install provenance and prints the correct upgrade
  command without ever replacing anything. This was genuinely attractive — zero dependencies, and it
  is what five of six channels need. Rejected for this feature to stay consistent with sting, whose
  ADR 0011 took the same position, and because it is additive: it can be introduced later without
  reversing anything decided here. The concept survives as the organizing axis of `README.md`'s
  upgrade table.

## Links

- [DECISIONS/0001 — Distribution channels by artifact shape][adr0001] — the standard being deviated
  from; its *Deviating* section requires this record
- [sting ADR 0011 — No self-update subcommand][sting0011] — the same decision, reached first, with
  the implementation that produced the original measurement
- [ADR-0007](./0007-release-binaries-and-homebrew.md) — release binaries and Homebrew; the
  stale-cask failure mode that motivates channel verification
- [ADR-0013](./0013-goreleaser-owns-github-release.md) — GoReleaser owns the GitHub release
- `specs/001-distribution-channels/` — the specification, research and plan this record accompanies

[adr0001]: https://github.com/skaphos/skaphos-resources/blob/main/DECISIONS/0001-distribution-channels-by-artifact-shape.md
[sting0011]: https://github.com/skaphos/sting/blob/main/docs/adr/0011-no-self-update-subcommand.md
