# Release Process

This repository releases via Release Please (PR-gated version bump and changelog) and `goreleaser` (GitHub release object, artifacts, and Homebrew publish). See [ADR-0013](./docs/adr/0013-goreleaser-owns-github-release.md) for the current release ownership decision and [ADR-0007](./docs/adr/0007-release-binaries-and-homebrew.md) for the binary/Homebrew release design it builds on.

## Pipeline shape

```
commits on main → release-please.yml (opens/updates release PR)
                ↓
            human reviews + merges the release PR
                ↓
       release-please.yml creates the vX.Y.Z tag
                ↓
           release.yml → goreleaser publishes
                          - GitHub release object
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

- Release Please updates `CHANGELOG.md` and `.release-please-manifest.json`.
- `release-please.yml` creates the annotated `vX.Y.Z` tag for the release commit.
- The tag push triggers `release.yml`, which **first verifies the Homebrew tap credential and fails
  the release if it is missing or cannot reach the tap**, then runs GoReleaser to:
  - Create and publish the GitHub release.
  - Build release binaries for `{linux,darwin,windows}/{amd64,arm64}`.
  - Build `.deb` and `.rpm` packages for `linux/{amd64,arm64}` from those same binaries.
  - Generate SPDX-JSON SBOMs per archive **and per package** via `syft`.
  - Build and push the multi-arch container image to `ghcr.io/skaphos/repokeeper`.
  - Sign `checksums.txt` with a keyless Sigstore bundle (`checksums.txt.sigstore.json`).
  - Publish the Homebrew cask to `github.com/skaphos/homebrew-tools`.
  - Publish GitHub artifact attestations for the release binaries and metadata assets.
- After GoReleaser, the workflow publishes the MCP registry entry (`io.skaphos/repokeeper`).
- A separate `verify` job then confirms each channel actually **serves** the new version.

No manual GoReleaser invocation or manual tag creation is required for normal releases.

### Credentials fail the release; they never skip a channel

A missing or unusable credential fails the release rather than silently dropping the channel it
belongs to. This replaced a step that emitted `--skip=homebrew` with a warning and then completed
green — which is exactly the failure [ADR-0007](docs/adr/0007-release-binaries-and-homebrew.md)
records, where the cask sat pinned at `0.6.0` across two releases because a run finished
"successfully" while a channel quietly did not publish.

| Credential | Failure mode |
| --- | --- |
| `RELEASE_BOT_CLIENT_ID` / `RELEASE_BOT_PRIVATE_KEY` | Pre-flight error, release fails before GoReleaser runs |
| GHCR (`github.token`) | Login step fails, release fails |
| `MCP_REGISTRY_KEY` | Warning only — the registry is the one exempt channel |
| `MACOS_*` | Pre-existing behavior: GoReleaser auto-skips notarization when unset (known gap, separate follow-up) |

### Channel verification

The `verify` job queries each channel rather than trusting the publishing step's own report. The
proposition being tested is not "did the release job succeed" — that is already visible. It is
"does each channel now serve the released version". Reachability is not the test: a cask pinned to
the previous release responds perfectly well, and that is the failure being guarded against.

| Channel | Blocking | Confirmed by |
| --- | --- | --- |
| Release assets | yes | GitHub release asset list, including `.deb`/`.rpm` and their SBOMs |
| Homebrew cask | yes | raw `Casks/repokeeper.rb`, matched on version |
| Container image | yes | `docker buildx imagetools inspect`, both architectures present |
| MCP registry | **no** | registry search API — reported, never blocking |

Each check retries with increasing backoff so a third-party outage is distinguishable from a channel
that genuinely did not publish.

## 5. Verify the release

The `verify` job does this automatically; the steps below remain useful when diagnosing a failure.

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
- Release Please is pinned to the immutable commit for `googleapis/release-please-action@v5.0.0`.
- The GoReleaser workflow remains tag-driven (`v*`) and owns the GitHub release object end to end.
