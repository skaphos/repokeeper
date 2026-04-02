# Release Process

This repository releases via Release Please, Git tags, and GitHub Actions.

## Prerequisites

- You have push access to `main`.
- `RELEASE_PLEASE_TOKEN` is configured in GitHub Actions secrets with permission to open PRs and create tags/releases on this repository. This must be a non-default token so the downstream tag-triggered release workflow runs.
- Optional for Homebrew publishing: `HOMEBREW_APP_ID` is configured as a repository or organization variable and `HOMEBREW_APP_PRIVATE_KEY` is configured as a GitHub Actions secret so the release workflow can mint a token for `skaphos/homebrew-tools`.
- CI is green on `main`.

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
- Updates Homebrew cask in `github.com/skaphos/homebrew-tools` (`Casks/repokeeper.rb`) when the Homebrew GitHub App credentials are configured

Release Please is responsible for the release PR, version bump, changelog commit, and tag creation. GoReleaser is responsible for publishing the final release assets for that tag and may update the GitHub release metadata/body as part of publishing.

No manual GoReleaser invocation or manual tag creation is required for normal releases.

## 5. Verify the Release

After workflow completion:

- Confirm the GitHub Release exists for the tag.
- Confirm expected artifacts are attached.
- Confirm release notes/version metadata look correct.

## Rollback / Fix Forward

- If the release workflow fails after the release PR merges, fix the workflow issue and re-run the failed workflow or create a follow-up patch release.
- If Release Please generated the wrong version or notes, fix the underlying commits/process and let it regenerate the next release PR.
- Manual tag creation should be reserved for emergency recovery only.

## Notes

- CI workflow is aligned to `Taskfile.yml` targets.
- Release Please is the source of truth for normal release PRs and version tags.
- The GoReleaser workflow remains tag-driven (`v*`).
