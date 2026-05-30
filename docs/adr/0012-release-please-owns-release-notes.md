# ADR-0012: Release Please Owns Release Notes

**Status:** Accepted
**Date:** 2026-05-30
**Author:** Shawn Stratton
**Supersedes:** [ADR-0009](./0009-replace-release-please-with-skaphos-actions.md)

## Context

ADR-0009 replaced Release Please with `skaphos/actions` because the release flow
at the time treated GoReleaser as the owner of the GitHub release object and
release body. That avoided the previous collision where Release Please created
the release and GoReleaser later tried to replace it.

The desired ownership model has changed: Release Please should own the release
PR, changelog, tag, and GitHub release notes. GoReleaser should attach binaries,
checksums, SBOMs, signatures, attestations, and the Homebrew cask to that
existing release.

The custom actions are still used by other repositories, so this decision only
changes RepoKeeper's release wiring. It does not remove or redesign
`skaphos/actions`.

## Decision

Restore `googleapis/release-please-action@v5` as RepoKeeper's release gate.
Release Please opens and updates the release PR, bumps
`.release-please-manifest.json`, updates `CHANGELOG.md`, creates the `vX.Y.Z`
tag, and creates a draft GitHub release with release notes.

The workflow uses the `skaphos-release-bot` GitHub App token rather than the
default `GITHUB_TOKEN` so Release Please's tag push can trigger the downstream
tag-driven GoReleaser workflow.

GoReleaser remains responsible for artifact publication. It is configured with
`release.use_existing_draft: true`, `release.mode: keep-existing`, and
`changelog.disable: true`, so it uploads artifacts to the existing Release
Please draft without replacing its notes, then publishes the release.

## Consequences

### Positive

- Release notes have one owner: Release Please.
- `CHANGELOG.md` returns as an in-repository release history maintained by the
  same tool that publishes GitHub release notes.
- GoReleaser's release job is narrower and less likely to collide with release
  body edits.
- The draft handoff is compatible with immutable releases because assets are
  uploaded before the release is published and frozen.

### Negative / risks

- The flow depends on Release Please creating a draft release before GoReleaser
  runs. If Release Please publishes directly instead, GoReleaser cannot attach
  assets once GitHub freezes the release.
- The release bot app token is load-bearing. Falling back to the default
  `GITHUB_TOKEN` would risk suppressing the tag-triggered GoReleaser workflow.

## Alternatives Considered

- **Keep `skaphos/actions` and make GoReleaser own notes.** Rejected because it
  conflicts with the desired release-note ownership model and duplicates Release
  Please's release PR behavior.
- **Let GoReleaser replace Release Please notes.** Rejected because it recreates
  the two-writers conflict that this change is meant to remove.
