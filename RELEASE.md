# Release Process

This repository releases via Release Please (PR-gated version bump, changelog, tag, and draft GitHub release notes) and `goreleaser` (artifact build and publish). See [ADR-0012](./docs/adr/0012-release-please-owns-release-notes.md) for the current release-note ownership decision and [ADR-0007](./docs/adr/0007-release-binaries-and-homebrew.md) for the binary/Homebrew release design it updates.

## Pipeline shape

```
commits on main → release-please.yml (opens/updates release PR)
                ↓
            human reviews + merges the release PR
                ↓
       Release Please creates vX.Y.Z tag + draft GitHub release notes
                ↓
           release.yml → goreleaser attaches
                          - Binaries, checksums, SBOMs, cosign signatures
                          - Homebrew cask in skaphos/homebrew-tools
                          - Build-provenance attestations
```

## Prerequisites

- You have push access to `main` (for regular merges — release tagging is automated).
- The `skaphos-release-bot` GitHub App is installed on `skaphos/repokeeper` and `skaphos/homebrew-tools` with `contents: write` and `pull-requests: write`.
- `RELEASE_BOT_APP_ID` is set as a repository or organization variable, and `RELEASE_BOT_PRIVATE_KEY` is set as a GitHub Actions secret. `release-please.yml` and `release.yml` mint short-lived installation tokens from this pair on the fly.
- CI is green on `main`.

## 1. Land releasable commits on `main`

Release Please infers the next version and release notes from Conventional Commits on `main` since the last `v*` tag:

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

On every push to `main`, `release-please.yml` recomputes the next version and opens or updates a Release Please PR. The PR updates `.release-please-manifest.json` and `CHANGELOG.md`.

- Review the version bump.
- Review the generated changelog entry and release notes.
- If the computed version is wrong, adjust the commit history or configure Release Please explicitly before merging.
- Merge the release PR when ready.

## 4. Tag push + GitHub release automation

When the Release Please PR merges:

- Release Please updates `CHANGELOG.md`, creates the `vX.Y.Z` tag, and creates a draft GitHub release with release notes.
- The tag push triggers `release.yml`, which runs GoReleaser to:
  - Build release binaries for `{linux,darwin,windows}/{amd64,arm64}`.
  - Generate SPDX-JSON SBOMs per archive via `syft`.
  - Sign `checksums.txt` with a keyless Sigstore bundle (`checksums.txt.sigstore.json`).
  - Attach artifacts to the existing draft GitHub release without replacing Release Please notes, then publish it.
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
- If Release Please computes the wrong version or PR body, fix the commit history/configuration and let the next push to `main` regenerate the PR.
- Manual tag creation is reserved for emergency recovery only.

## Notes

- CI workflow is aligned to `Taskfile.yml` targets.
- Release Please is pinned to the latest major action (`googleapis/release-please-action@v5`).
- The GoReleaser workflow remains tag-driven (`v*`) and attaches artifacts to the Release Please draft release before publishing it.
