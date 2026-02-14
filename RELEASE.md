# Release Process

This repository releases via Git tags and GitHub Actions.

## Prerequisites

- You have push access to `main`.
- Local branch is up to date:
  - `git checkout main`
  - `git pull --ff-only`
- CI is green on `main`.

## 1. Run Local Release Checks

Use the same checks CI runs:

- `go tool task ci`

Optional version preview:

- `go tool task version-next`

## 2. Choose and Create the Version Tag

Use semantic version tags in the format `vX.Y.Z`.

Example:

```bash
git tag v1.4.0
git push origin v1.4.0
```

## 3. GitHub Release Automation

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which runs GoReleaser:

- Builds release artifacts
- Publishes a GitHub Release

No manual GoReleaser invocation is required for normal releases.

## 4. Verify the Release

After workflow completion:

- Confirm the GitHub Release exists for the tag.
- Confirm expected artifacts are attached.
- Confirm release notes/version metadata look correct.

## Rollback / Fix Forward

- If the release workflow fails before publishing, fix and re-push the tag if needed.
- If a bad release is published, create a new patch release tag (preferred over rewriting history).

## Notes

- CI workflow is aligned to `Taskfile.yml` targets.
- Release workflow is tag-driven only (`v*`).
