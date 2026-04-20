# ADR-0008: Replace Release Please with skaphos/actions

**Status:** Accepted
**Date:** 2026-04-19
**Author:** Shawn Stratton

## Context

[ADR-0007](./0007-release-binaries-and-homebrew.md) landed the goreleaser pipeline under `skip-github-release: true` with `googleapis/release-please-action@v4` retained as the version-bump gate, plus a workflow-level tag-push step to work around release-please-action's missing tag-creation under `skip-github-release`. That combination works, but it keeps release-please-action in the tree purely for its PR-open/update surface — a small fraction of what the action does.

After ADR-0007 shipped, three things became true at once:

1. **We have zero dependence on release-please-generated GitHub releases.** Goreleaser owns the GitHub release object end-to-end.
2. **We have zero dependence on a release-please-generated `CHANGELOG.md` as an end-user-visible surface.** The GitHub release body is the changelog surface and is rendered by goreleaser from Conventional Commits.
3. **The `skip-github-release: true` tag-push workaround is a workflow-level behaviour** that release-please-action itself doesn't participate in — it lives in inline shell in `release-please.yml`.

What we actually need from release-please is narrow: *given Conventional-Commit-annotated merges to `main`, open a PR that bumps `.release-please-manifest.json` to the next semver, and push the tag when that PR merges*. Everything else — multi-package manifests, release-type plugins, changelog generation, release-object creation — is duplicate work on our pipeline.

`skaphos/actions` (introduced in this cycle at [github.com/skaphos/actions](https://github.com/skaphos/actions)) provides exactly that narrow scope as two composite actions — `release-pr` and `release-tag` — designed deliberately around the PR-gate + tag-push behaviour. It also drops `CHANGELOG.md` as an in-repo artifact; consumers that want one maintained independently can wire it separately (we don't).

### Why replace

- **Dead weight.** The release-type, plugin, and changelog-generation features in release-please go entirely unused under the post-ADR-0007 pipeline shape.
- **Fragility.** release-please-action v4 silently stopped pushing tags under `skip-github-release: true`, requiring a homegrown fallback in the same workflow ([fix #190](https://github.com/skaphos/repokeeper/pull/190) / [fix #194](https://github.com/skaphos/repokeeper/pull/194)). Every release-please-action upgrade risks re-breaking that fallback.
- **Cross-repo reuse.** We have additional Go tools (`fathom`, `meridian`, the `wake` family) that need the same PR-gate + goreleaser shape. A narrow, purpose-built action is reusable in ways a 500-line shared `release-please-config.json` isn't.

### Alternatives considered

- **Keep release-please, just live with the workaround.** Rejected. The workaround is load-bearing, undocumented in release-please-action itself, and subject to silent breakage on dependency bumps.
- **Drop release-please entirely, trigger goreleaser on every push to `main`.** Rejected in ADR-0007, still rejected here — we want the PR-review gate, not an auto-release on every merge.
- **Replace with `svu` + goreleaser directly (no PR gate).** Rejected in ADR-0007. `svu` tags on every merge; the gate is the point.

## Decision

Adopt `skaphos/actions` as the PR gate + tag push, keep the existing goreleaser-driven release publish.

### Files added

- **`.github/workflows/release-pr.yml`** — triggers on `push: main`, calls `skaphos/actions/release-pr@v1.0.0`. Maintains a single release PR that bumps `.release-please-manifest.json`.
- **`.github/workflows/release-tag.yml`** — triggers on `pull_request: closed` with the `release-pr` label, calls `skaphos/actions/release-tag@v1.0.0`. Pushes the annotated `vX.Y.Z` tag.
- **`docs/adr/0008-replace-release-please-with-skaphos-actions.md`** — this document.

### Files deleted

- `.github/workflows/release-please.yml`
- `release-please-config.json`
- `CHANGELOG.md`

### Files modified

- **`.github/CODEOWNERS`** — drop entries for `release-please.yml` and `release-please-config.json`; add entries for the two new workflow files.
- **`CONTRIBUTING.md`** — replace the "Release Please depends on…" paragraph with an `svu` / Conventional-Commits description; update the cross-reference to `RELEASE.md`.
- **`RELEASE.md`** — rewrite the release procedure to describe the two-workflow model.

### Files kept

- **`.release-please-manifest.json`** — retained at its historical path and name. `skaphos/actions` defaults to this path and key for release-please compatibility; renaming it would churn other skaphos repos that haven't cut over yet.
- **`.github/workflows/release.yml`** — unchanged. Still listens on `push: tags: v*` and runs goreleaser.

### Bot identity

Continues to use `skaphos-release-bot[bot]` via the existing `HOMEBREW_APP_ID` / `HOMEBREW_APP_PRIVATE_KEY` variable/secret pair. The App already has `contents: write` and `pull-requests: write` on this repo — previously consumed by release-please-action, now consumed by `release-pr`. No credential rotation or new app installation required.

### Action pin

Pin `@v1.0.0` (strict) initially rather than `@v1` (floating major). `skaphos/actions` has passed its own repo's CI but has not yet been battle-tested end-to-end against a real release cycle; a strict pin keeps the failure mode bounded. Loosen to `@v1` in a follow-up PR once the first cycle lands cleanly.

## Consequences

### Positive
- One fewer third-party action with a broad permission surface in the critical path.
- Tag-push logic lives in `skaphos/actions/release-tag`, not in inline shell inside this repo. Bug fixes land once in the action and propagate to all consumers via `@v1`.
- No `CHANGELOG.md` drift between release-please's representation and goreleaser's release-body representation. The GitHub release is the single source of truth.
- Other skaphos Go tools can adopt the same pipeline with a ~20-line workflow pair (see [`skaphos/actions/examples/goreleaser-go-cli`](https://github.com/skaphos/actions/tree/main/examples/goreleaser-go-cli)).

### Negative / risks
- **First-cycle risk.** The actions have not been exercised against a real release yet. Mitigated by pinning `@v1.0.0` strictly — any bug that appears in `@v1.0.1+` doesn't affect us automatically.
- **`CHANGELOG.md` deletion is visible** to anyone browsing the file historically. Anyone landing on the old URL will 404. `git log` and the GitHub release page remain authoritative. Acceptable — the audience for a browsable changelog file is small; the audience for the GitHub release page is everyone.
- **Contents API dependency.** `release-pr` commits via the Contents API to produce verified commits (signed by GitHub's web-flow key). If GitHub ever deprecates web-flow signing or changes the Contents API behaviour, the signed-commit policy on this repo would reject the bot's commits. Unlikely near-term; worth watching.

### Verification plan
1. Merge this PR into `main`.
2. `release-pr.yml` fires on the merge → should open (or force-update) a `release/v<next>` PR labelled `release-pr`, with a single-file `.release-please-manifest.json` diff.
3. Review the release PR; confirm the computed version matches expectations (should be `v0.7.2` given the last release was `v0.7.1` and this commit is a non-breaking feat/refactor).
4. Merge the release PR.
5. `release-tag.yml` fires on the PR-merge event → should push `v<next>` as an annotated tag.
6. `release.yml` fires on the tag push → goreleaser publishes the GitHub release, Homebrew cask, SBOMs, cosign signatures, and provenance attestations exactly as before.
7. `gh release view v<next> --json assets` should list the full archive/SBOM/signature set.

## Implementation plan

Single PR against `skaphos/repokeeper` (this one):

- **Commit 1** — add this ADR.
- **Commit 2** — atomic cutover: add the two new workflow files + rewrite `RELEASE.md`; patch `CONTRIBUTING.md` and `.github/CODEOWNERS`; delete `release-please.yml`, `release-please-config.json`, `CHANGELOG.md`.

Atomic cutover so merging the PR transitions the repo from release-please to skaphos/actions in one step. There is no mixed state in which both systems run simultaneously.
