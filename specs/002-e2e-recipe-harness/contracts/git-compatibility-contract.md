# Git Compatibility Declaration Contract

The version-controlled declaration at `test/e2e/testdata/git-compatibility.json` is the executable source for the Git support matrix summarized in `DESIGN.md`. It is test and release infrastructure, not a user-facing RepoKeeper configuration format.

## Closed support claim

The declaration contains exactly three ordered Git minor lines: the current claimed upstream minor and the two immediately preceding minors. At planning time the set is `2.53`, `2.54`, and `2.55`. This is a closed set:

- the lowest member is the minimum only within the declared set;
- omitted historical minors are unsupported;
- a newer upstream minor is unqualified and unsupported until a reviewed change adds it, removes the oldest line, and updates `DESIGN.md`;
- automation never infers support from numeric ordering.

Each minor appears once for each required environment: `linux`, `macos`, `windows`, and `wsl`. The release matrix therefore contains exactly twelve cells. Routine CI contains exactly one cell per environment selected from those twelve.

## Document shape

The document is one strict JSON object with this conceptual shape:

```json
{
  "schema_version": 1,
  "supported_minors": ["2.53", "2.54", "2.55"],
  "cells": [
    {
      "environment": "linux",
      "runner_label": "ubuntu-24.04",
      "git_minor": "2.53",
      "git_patch": "2.53.0",
      "provisioner": {
        "kind": "source-build",
        "source_url": "https://www.kernel.org/pub/software/scm/git/git-2.53.0.tar.xz",
        "sha256": "64 lowercase hexadecimal characters"
      },
      "routine_ci": true
    }
  ]
}
```

The implementation MUST reject unknown fields, trailing JSON, unsupported schema versions, unordered/non-consecutive/duplicate minor values, anything other than the complete four-by-three Cartesian set, duplicate cells, floating runner labels, patch/minor mismatches, incomplete provisioning data, malformed integrity values, invalid environment/provisioner pairs, and anything other than one routine cell per environment.

## Environment and provisioner rules

| Environment | Runner | Git provisioner | Additional requirement |
| --- | --- | --- | --- |
| `linux` | `ubuntu-24.04` | Official kernel.org source archive, SHA-256 verified, built into a test-owned prefix | Execute Linux RepoKeeper and tests natively |
| `macos` | `macos-15` | Official kernel.org source archive, SHA-256 verified, built into a test-owned prefix | Execute macOS RepoKeeper and tests natively |
| `windows` | `windows-2025` | Official Git-for-Windows MinGit archive, SHA-256 verified, extracted into a test-owned prefix | Execute native Windows RepoKeeper and tests |
| `wsl` | `windows-2025` | Official kernel.org source archive, SHA-256 verified, built inside WSL into a test-owned prefix | Import checksum-pinned Canonical Ubuntu 24.04 rootfs as WSL1; execute cross-compiled Linux compatibility-helper, RepoKeeper, and E2E binaries inside it |

WSL cells additionally declare `rootfs.release`, `rootfs.image_date`, `rootfs.source_url`, `rootfs.sha256`, `rootfs.wsl_version: 1`, and any missing compiler/build prerequisites as exact package URLs and SHA-256 values from a signed timestamped Ubuntu snapshot. The initial rootfs is Canonical's immutable Noble `20240423` amd64 image with SHA-256 `2a790896740b14d637dbdc583cce1ba081ac53b9e9cdb46dc09a2f73abbd9934`. Non-WSL cells reject rootfs metadata. No cell may use package-manager latest resolution, an unconstrained package update, runner-provided Git, floating URLs, missing integrity data, or fallback to another environment/version after failure.

## Reusable executable interface

Tests and workflows invoke one tagged command:

```text
go run -tags integration ./test/e2e/cmd/compatibility <operation>
```

Required operations are:

- `matrix --scope routine|release`: validate the declaration and `DESIGN.md`, then emit an ordered JSON matrix;
- `provision --cell <key> --prefix <path>`: integrity-check and install the declared inputs for one cell;
- `verify-version --cell <key> --git <path>`: require exact `git --version` agreement;
- `write-evidence --cell <key> ...`: write one bounded result document tied to tag and commit;
- `verify-evidence --directory <path>`: enforce exact-set completeness and success;
- `verify-docs`: require the human-readable and executable declarations to agree.

Workflow YAML may arrange runners, artifacts, and job dependencies, but MUST NOT independently parse support semantics, choose versions, or implement evidence completeness. Machine-readable stdout contains only JSON; diagnostics use stderr and a non-zero exit.

## Release evidence

Each cell produces one bounded JSON evidence artifact containing the immutable candidate tag, source commit, environment, runner label/image, declared minor, expected exact patch, raw actual `git --version`, provisioner and rootfs identities/digests, test result, and artifact digest. Evidence contains no credentials or inherited environment values.

The completeness gate compares declared cell keys with evidence keys for exact set equality, reports sorted missing and unexpected keys, rejects duplicates, requires matching tag/commit and integrity values, and requires every result to be successful with an exact Git version match. Only then may release publication jobs acquire write permissions or secrets and begin publishing.

## Tag and recovery semantics

A `v*` tag starts release-candidate qualification and is immutable. It is not itself evidence that a GitHub Release, archive, image, package, Homebrew update, or other channel has been published.

- A transient runner, network, or artifact-service failure may rerun the same tag only when tag and commit are unchanged.
- A source, test, compatibility, provisioning declaration, or documentation correction requires a new semantic version and tag.
- Failed tags are not moved, deleted, or reused for a different commit.
- No publication job may start before the completeness gate passes.
