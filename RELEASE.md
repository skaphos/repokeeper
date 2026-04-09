# Release Process

This repository releases via Release Please, Git tags, and GitHub Actions.

## Prerequisites

- You have push access to `main`.
- `RELEASE_PLEASE_TOKEN` should be configured in GitHub Actions secrets with permission to open PRs and create tags/releases on this repository. This should be a non-default token so the downstream tag-triggered release workflow runs.
- Optional for Homebrew publishing: `HOMEBREW_APP_ID` is configured as a repository or organization variable and `HOMEBREW_APP_PRIVATE_KEY` is configured as a GitHub Actions secret so the release workflow can mint a token for `skaphos/homebrew-tools`.
- CI is green on `main`.

If `RELEASE_PLEASE_TOKEN` is not configured, the Release Please workflow falls back to the default `github.token` so the workflow still runs. That fallback is sufficient for maintaining the release PR, but it may not trigger downstream tag-push workflows reliably. Configure `RELEASE_PLEASE_TOKEN` for full release automation.

## 1. Land Releasable Commits on `main`

Release Please opens and maintains the release PR from commits already merged to `main`.

- Use Conventional Commits on the commits that land on `main` so Release Please can determine the next version and changelog entries.
- `feat:` drives a minor bump.
- `fix:` and `perf:` drive a patch bump.
- `docs:`, `test:`, `ci:`, `chore:`, and `refactor:` do not bump by default.
- If you squash-merge pull requests, make sure the final squash commit message is also a Conventional Commit.

## 2. Run Local Release Checks

Use the same checks CI runs:

- `go -C tools tool task ci`
- `go -C tools tool task notices`

Optional version preview:

- `go -C tools tool task version-next`

## 3. Review and Merge the Release PR

When Release Please detects releasable commits on `main`, it opens or updates a release PR.

- Review the generated changelog and version bump.
- Merge the release PR when the proposed release is correct.

Release Please creates the semantic version tag in the format `vX.Y.Z` when the release PR merges.

## 4. GitHub Release Automation

The release PR merge creates the version tag and an initial GitHub release entry. That tag push triggers `.github/workflows/release.yml`, which runs GoReleaser to:

- Builds release artifacts
- Publishes release assets to the GitHub release for that tag
- Generates SPDX JSON SBOMs for the archived release artifacts
- Signs `checksums.txt` with a keyless Sigstore bundle (`checksums.txt.sigstore.json`)
- Publishes GitHub artifact attestations for the release artifacts and release metadata assets
- Updates Homebrew cask in `github.com/skaphos/homebrew-tools` (`Casks/repokeeper.rb`) when the Homebrew GitHub App credentials are configured

Release Please is responsible for the release PR, version bump, changelog commit, and tag creation. GoReleaser is responsible for publishing the final release assets for that tag and may update the GitHub release metadata/body as part of publishing.

No manual GoReleaser invocation or manual tag creation is required for normal releases.

## 5. Verify the Release

After workflow completion:

- Confirm the GitHub Release exists for the tag.
- Confirm expected artifacts are attached.
- Confirm the release archives include `THIRD_PARTY_NOTICES.md` and `third_party_licenses/`.
- Confirm `checksums.txt`, `checksums.txt.sigstore.json`, and the generated `*.sbom.json` files are attached.
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

- The `sha256sum -c` example above is Linux-oriented. On macOS, use `shasum -a 256 -c checksums.txt`; on Windows, use PowerShell `Get-FileHash` to verify the downloaded artifacts against `checksums.txt`.
- `cosign` verifies that the published checksum file was keylessly signed by the release workflow identity for that tag.
- `gh attestation verify` verifies the GitHub-hosted provenance attestation for a downloaded release asset.
- The `*.sbom.json` assets are SPDX JSON SBOMs generated from the published release archives.

## Rollback / Fix Forward

- If the release workflow fails after the release PR merges, fix the workflow issue and re-run the failed workflow or create a follow-up patch release.
- If Release Please generated the wrong version or notes, fix the underlying commits/process and let it regenerate the next release PR.
- Manual tag creation should be reserved for emergency recovery only.

## Notes

- CI workflow is aligned to `Taskfile.yml` targets.
- Release Please is the source of truth for normal release PRs and version tags.
- The GoReleaser workflow remains tag-driven (`v*`).
