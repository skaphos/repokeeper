# Release Process

This repository releases via `skaphos/actions` (PR-gated version bump + annotated tag push) and `goreleaser` (artifact build + GitHub release publish). See [ADR-0009](./docs/adr/0009-replace-release-please-with-skaphos-actions.md) for the rationale behind the current shape and [ADR-0007](./docs/adr/0007-release-binaries-and-homebrew.md) for why goreleaser owns the GitHub release object.

## Pipeline shape

```
commits on main → release-pr.yml (opens/updates release PR)
                ↓
            human reviews + merges the release PR
                ↓
       release-tag.yml (pushes annotated vX.Y.Z tag)
                ↓
           release.yml → goreleaser publishes
                          - GitHub release + release notes
                          - Binaries, checksums, SBOMs, cosign signatures
                          - Homebrew cask in skaphos/homebrew-tools
                          - Build-provenance attestations
```

## Prerequisites

- You have push access to `main` (for regular merges — release tagging is automated).
- The `skaphos-release-bot` GitHub App is installed on `skaphos/repokeeper` and `skaphos/homebrew-tools` with `contents: write` and `pull-requests: write`.
- `HOMEBREW_APP_ID` is set as a repository or organization variable, and `HOMEBREW_APP_PRIVATE_KEY` is set as a GitHub Actions secret. Both `release-pr.yml` and `release-tag.yml` mint short-lived installation tokens from this pair on the fly.
- CI is green on `main`.

## 1. Land releasable commits on `main`

`skaphos/actions/release-pr` infers the next version via [`svu`](https://github.com/caarlos0/svu) from Conventional Commits on `main` since the last `v*` tag:

- `feat:` → minor bump
- `fix:` / `perf:` → patch bump
- `docs:`, `test:`, `ci:`, `chore:`, `refactor:` → no bump by default
- Any `!` in the type/scope or a `BREAKING CHANGE:` footer → major bump

If you squash-merge pull requests, the final squash commit message must also follow Conventional Commit format.

## 2. Run local release checks

Use the same checks CI runs:

- `go -C tools tool task ci`
- `go -C tools tool task notices`

Optional version preview:

- `go -C tools tool task version-next`

## 3. Review and merge the release PR

On every push to `main`, `release-pr.yml` recomputes the next version and opens or force-updates a `release/v<next>` PR labelled `release-pr`. The PR contains exactly one file diff: `.release-please-manifest.json`.

- Review the version bump.
- If the computed version is wrong (rare — happens when `svu`'s commit-type inference disagrees with intent), hand-edit the manifest in the PR branch before merging. Whatever version is in the manifest at merge time is the version that gets tagged.
- Merge the release PR when ready.

## 4. Tag push + GitHub release automation

When the release PR merges:

- `release-tag.yml` fires on `pull_request: closed`. It reads the version from the manifest at the merge commit and pushes an annotated `vX.Y.Z` tag.
- The tag push triggers `release.yml`, which runs GoReleaser to:
  - Build release binaries for `{linux,darwin,windows}/{amd64,arm64}`.
  - Generate SPDX-JSON SBOMs per archive via `syft`.
  - Sign `checksums.txt` with a keyless Sigstore bundle (`checksums.txt.sigstore.json`).
  - Create the GitHub release with notes rendered from Conventional Commits, grouped as Features / Bug Fixes / Performance / Others.
  - Publish the Homebrew cask to `github.com/skaphos/homebrew-tools` when the Homebrew GitHub App token is reachable.
  - Publish GitHub artifact attestations for the release binaries and metadata assets.

No manual GoReleaser invocation or manual tag creation is required for normal releases.

## 5. Verify the release

After workflow completion:

- Confirm the GitHub release exists for the tag.
- Confirm expected artifacts are attached.
- Confirm `checksums.txt`, `checksums.txt.sigstore.json`, and the generated `*.sbom.json` files are attached.
- Confirm the release archives include `THIRD_PARTY_NOTICES.md` and `third_party_licenses/`.
- Confirm release notes/version metadata look correct.

Example verification flow for `vX.Y.Z`:

```bash
mkdir -p /tmp/repokeeper-release && cd /tmp/repokeeper-release
gh release download vX.Y.Z --repo skaphos/repokeeper \
  --pattern 'checksums.txt' \
  --pattern 'checksums.txt.sigstore.json' \
  --pattern '*.sbom.json' \
  --pattern 'repokeeper_vX.Y.Z_linux_amd64.tar.gz'
sha256sum -c checksums.txt --ignore-missing
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/skaphos/repokeeper/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
gh attestation verify repokeeper_vX.Y.Z_linux_amd64.tar.gz --repo skaphos/repokeeper
```

Notes:

- `sha256sum -c` is Linux-oriented. On macOS, use `shasum -a 256 -c checksums.txt`; on Windows, use PowerShell `Get-FileHash` to verify artifacts against `checksums.txt`.
- `cosign` verifies that `checksums.txt` was keylessly signed by the release workflow identity for that tag.
- `gh attestation verify` verifies the GitHub-hosted provenance attestation for a downloaded release asset.
- The `*.sbom.json` assets are SPDX-JSON SBOMs generated from the published release archives.

## Rollback / fix forward

- If `release.yml` (goreleaser) fails after the tag is pushed, fix the workflow issue and re-run the failed workflow, or cut a follow-up patch release from `main`.
- If `release-pr` computes the wrong version or the PR body is wrong, let additional commits land — the next push to `main` will regenerate the PR.
- Manual tag creation is reserved for emergency recovery only.

## Notes

- CI workflow is aligned to `Taskfile.yml` targets.
- `skaphos/actions/release-pr@v1.0.0` is currently pinned strictly; after one successful full release cycle under this pipeline, the pin will be relaxed to `@v1`.
- The GoReleaser workflow remains tag-driven (`v*`) and is otherwise unchanged.
