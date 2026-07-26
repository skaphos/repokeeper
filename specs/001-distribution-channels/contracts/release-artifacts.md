# Contract: Release Artifacts and Channel Verification

**Requirements**: FR-011 – FR-015, FR-028, FR-031 – FR-035
**Artifacts**: `.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/ci.yml`

What a release publishes, and how each channel is confirmed to have actually served it.

## Published artifacts

| Artifact | Naming | Platforms |
| --- | --- | --- |
| `tar.gz` | existing template | linux, darwin × amd64, arm64 |
| `zip` | existing template | windows × amd64, arm64 |
| `.deb` | `repokeeper_0.8.0_amd64.deb` | linux × amd64, arm64 |
| `.rpm` | `repokeeper-0.8.0-1.x86_64.rpm` | linux × amd64, arm64 |
| Container image | `ghcr.io/skaphos/repokeeper:0.8.0`, `:latest` | linux × amd64, arm64 |
| `checksums.txt` | — | — |
| `*.sbom.json` | one per archive **and** one per package | — |
| `checksums.txt.sigstore.json` | — | — |

Package names use `{{ .ConventionalFileName }}` so each format follows its own convention rather than
one shape forced onto both.

## Linux package contents (FR-012, FR-013)

| Path | Source |
| --- | --- |
| `/usr/bin/repokeeper` | the release binary |
| `/usr/share/doc/repokeeper/LICENSE` | `LICENSE` |
| `/usr/share/doc/repokeeper/THIRD_PARTY_NOTICES.md` | `THIRD_PARTY_NOTICES.md` |
| `/usr/share/doc/repokeeper/third_party_licenses` | `third_party_licenses/` |

Metadata: maintainer `Skaphos <shawn@skaphos.io>`, homepage the repository URL, license `MIT`, vendor
`Skaphos`, section `utils` — consistent with the repository's REUSE/SPDX attribution.

**The easily-missed rule (FR-014).** Packages do not inherit the archive SBOM. The `sboms` block
needs a second entry with `artifacts: package`, or `.deb` and `.rpm` ship without one. An
archives-only configuration looks complete and is not.

## Documentation obligation (FR-015)

Installation docs must state that **no hosted apt or yum repository exists**. Users reasonably read
"we ship a `.deb`" as "there is an apt repo". Upgrading means downloading the next release's package.

## Credential pre-flight (FR-032) — a behavior change

RepoKeeper's release workflow today contains the failure mode this requirement forbids. The
`Determine GoReleaser args` step emits `--skip=homebrew` with a `::warning::` when the tap token is
missing or cannot reach the tap, then completes **green**. A release that silently drops the cask and
reports success is the ADR-0007 incident encoded as workflow logic.

**Replaced by**: a pre-flight step that fails the release before GoReleaser runs.

| Credential | Checked by | On failure |
| --- | --- | --- |
| Homebrew tap token | Minted, then a GitHub API call against `skaphos/homebrew-tools` | **`::error::` and exit 1**, naming `RELEASE_BOT_CLIENT_ID` / `RELEASE_BOT_PRIVATE_KEY` or the App installation |
| GHCR | `docker login` in the workflow | Fails the job |
| `MCP_REGISTRY_KEY` | Presence check at publish | Warning only — registry is the exempt channel |

`MACOS_*` remain auto-skipping inside GoReleaser's `notarize` block, which is pre-existing behavior
and out of scope; it is listed in the spec's edge cases as separate follow-up.

## Verification job (FR-033 – FR-035)

A separate job, `needs: release`, that queries each channel rather than trusting the publish step.

**The proposition being tested is not "did the release job succeed"** — that is already visible. It
is "does each channel now *serve* the released version". Reachability is not the test: a cask pinned
to the previous release responds perfectly well, and that is precisely the failure being guarded
against.

| Channel | Query | Assertion | On failure |
| --- | --- | --- | --- |
| Release assets | GitHub API asset list | Archives, packages, checksums, SBOMs, signature all present | **fail** |
| Homebrew cask | `raw.githubusercontent.com/.../Casks/repokeeper.rb` | Contains `version "<v>"` | **fail** |
| Container image | `docker buildx imagetools inspect --raw` | Resolves; manifest has both arches | **fail** |
| MCP registry | Registry search API | Entry present at the version | **warn**, `continue-on-error` |

Every check retries with increasing backoff (FR-035) so a third-party outage is distinguishable from
a channel that genuinely did not publish. A final summary step runs with `if: always()` and reports
per-channel outcomes.

**The registry is the only non-blocking channel** (FR-034), per DECISIONS/0001's own statement that a
break there is a smaller emergency than a broken cask. Its publish step is `continue-on-error` with
an `always()` report naming the `skaphos.io` DNS TXT record and `MCP_REGISTRY_KEY` — the two things
that fail quietly.

## CI additions

A `manifests` job, wired into the `build` and `summary` job dependencies (and the summary's job
count):

| Step | Tool | Catches |
| --- | --- | --- |
| `goreleaser check` | GoReleaser, pinned to `.tool-versions` | Invalid release config before release time |
| `check-jsonschema` against the MCP schema | pinned version | A malformed `server.json` before merge |

Both validate artifacts consumed outside this repository, where a mistake surfaces at release time or,
worse, publishes something wrong.

## Single-invocation invariant (FR-031)

Archives, Linux packages, the Homebrew cask and the container image are all produced by one
GoReleaser run and fail as a unit. The MCP registry is the sole exception, and no further channel may
be carved out without a recorded justification — each carve-out reinstates the half-landed release
this requirement exists to prevent.
