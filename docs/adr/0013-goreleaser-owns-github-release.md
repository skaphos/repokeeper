# ADR-0013: GoReleaser Owns the GitHub Release

**Status:** Accepted
**Date:** 2026-05-30
**Author:** Shawn Stratton
**Supersedes:** [ADR-0012](./0012-release-please-owns-release-notes.md)

## Context

ADR-0012 restored a draft-release handoff where Release Please created the tag
and draft GitHub release, then GoReleaser attached binaries, checksums, SBOMs,
signatures, attestations, and the Homebrew cask before publishing that draft.

That shape is fragile under GitHub immutable releases. The handoff only works if
Release Please creates a mutable draft and GoReleaser reaches it before GitHub
freezes the release object. If Release Please publishes the release, or if the
release is otherwise immutable before GoReleaser uploads assets, GoReleaser
fails with upload errors and users get a release with missing binaries.

RepoKeeper already had a proven tag-only Release Please flow in ADR-0007, and
the same flow was validated in `skaphos/sting`: Release Please owns the release
PR, manifest bump, and in-repository changelog; the workflow creates the tag;
GoReleaser owns the GitHub release object and all release assets.

## Decision

Release Please remains the PR gate. It opens and updates the release PR, bumps
`.release-please-manifest.json`, and updates `CHANGELOG.md`, but it must not
create a GitHub release object.

The Release Please workflow passes `skip-github-release: true` to
`googleapis/release-please-action@v5`. After the action runs, the workflow
checks out the repository and creates an annotated `vX.Y.Z` tag only when the
current `main` commit is the merged Release Please release commit and that
commit changed `.release-please-manifest.json`.

The tag push triggers the GoReleaser workflow. GoReleaser creates and publishes
the GitHub release, uploads binaries, checksums, SBOMs, signatures, and
attestations, and publishes the Homebrew cask to `skaphos/homebrew-tools`.
GoReleaser is not configured to reuse a Release Please draft.

## Consequences

### Positive

- GitHub release object creation has one owner: GoReleaser.
- Immutable releases are compatible with the pipeline because assets are
  uploaded as part of GoReleaser's release creation path before the release is
  frozen.
- The human release gate remains a reviewed Release Please PR.
- `CHANGELOG.md` remains maintained in the repository by Release Please.
- Homebrew publishing remains in the same GoReleaser run that uploads release
  binaries, so the cask version and release assets advance together.

### Negative / risks

- GoReleaser creates the GitHub Release object (with `changelog.disable: true`
  in `.goreleaser.yaml`, the release body is empty or minimal). The authoritative,
  curated release notes live in `CHANGELOG.md` (maintained by Release Please).
  The GitHub release primarily serves as the immutable container for binaries
  and metadata assets.
- The inline tag-creation step is load-bearing until Release Please supports
  tag-only releases natively.
- The release bot app token is still load-bearing. Falling back to the default
  `GITHUB_TOKEN` could suppress the tag-triggered GoReleaser workflow.

## Alternatives Considered

- **Keep the ADR-0012 draft handoff.** Rejected because it is sensitive to
  immutable-release timing and creates a two-tool dependency on one GitHub
  release object.
- **Let Release Please own the GitHub release and upload assets later.**
  Rejected because immutable releases reject those later uploads.
- **Drop Release Please entirely.** Rejected for now because the reviewed
  release PR and in-repository changelog remain useful.
