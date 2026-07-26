# Tasks: Distribution Channel Conformance

**Input**: Design documents from `/specs/001-distribution-channels/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/)

**Tests**: **REQUIRED, not optional.** The Spec Kit scaffold marks test tasks optional; the RepoKeeper
constitution overrides it — *"New behavior MUST ship with meaningful tests in the same change"* — and
the constitution's own Sync Impact Report directs that this gate be enforced at `/speckit-tasks`
time. Test tasks below are therefore mandatory.

**Organization**: Grouped by user story so each is independently implementable and testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1–US4, mapping to the spec's prioritized user stories
- Exact file paths included in every task

## Path Conventions

Single Go module at the repository root: `cmd/repokeeper/`, `internal/`, `docs/adr/`,
`.github/workflows/`. Paths below are repository-relative.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Pre-flight checks only. This feature adds no scaffolding — it adds channels to an
existing pipeline.

- [X] T001 Verify toolchain matches `.tool-versions` (Go 1.26.5, GoReleaser 2.17.0) and that `docker buildx` is available, per quickstart.md prerequisites
- [X] T002 [P] Add `*.out` and `vendor/` to `.gitignore` (Go convention; currently absent)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The deviation record. This blocks nothing technically, but DECISIONS/0001 requires the
record to exist in the repo that drops a required channel — landing it first keeps the feature honest
if later phases are interrupted.

- [X] T003 Write `docs/adr/0016-no-self-update-subcommand.md` (Status: Accepted) recording the dropped self-update channel: the measurement table from spec.md Prior Art (+128.7% binary, 47→106 direct requirements, 95→406 modules), the one-of-six-channels reachability analysis, the container incoherence, what users get instead, and the reversal condition — a verification stack of materially lower cost, or DECISIONS/0002 rescoping the requirement (FR-008)
- [X] T004 Add the ADR-0016 index entry to `docs/adr/README.md` (FR-008)

**Checkpoint**: The deviation is recorded before any channel work lands.

---

## Phase 3: User Story 1 — Know which version you are running (Priority: P1) 🎯 MVP

**Goal**: Every install path that produces a usable binary reports a real version and revision; no
placeholder reaches a user.

**Independent Test**: Build without release stamping and confirm the version command reports module
and revision metadata rather than a placeholder; then build *with* stamping and confirm output is
unchanged from today.

### Tests for User Story 1

- [X] T005 [P] [US1] Write `internal/buildinfo/buildinfo_test.go` covering all five resolution tiers against a synthetic `*debug.BuildInfo`: ldflags stamped wins; `"dev"` sentinel falls through; `"none"`/`"unknown"` sentinels map to empty; `Main.Version == "(devel)"` yields empty Version with revision reported; no build info yields `SourceUnknown`; `vcs.modified == "true"` surfaces (FR-001 – FR-004)
- [X] T006 [P] [US1] Write `cmd/repokeeper/version_test.go` asserting the human-readable output for each tier and the JSON shape from contracts/cli-version.md — unknown string fields are `""` not omitted, `source` is one of three literals, exit code is 0 even when fully unknown (FR-004, FR-006)

### Implementation for User Story 1

- [X] T007 [US1] Create `internal/buildinfo/buildinfo.go` with `Info`, `Source` (`SourceUnknown`/`SourceBuildInfo`/`SourceLDFlags`), `Known()`, exported `Resolve(ldVersion, ldCommit, ldDate string) Info`, and an unexported injectable core taking `(*debug.BuildInfo, bool)` so tests can supply fixtures — `debug.ReadBuildInfo()` reads the running test binary and cannot be faked. Stdlib only: `runtime/debug` (FR-001 – FR-005)
- [X] T008 [US1] Rewrite `cmd/repokeeper/version.go` to delegate to `buildinfo.Resolve`, add `--output json` (FR-006), and render each tier per contracts/cli-version.md. **Keep `Version`, `Commit`, `Date` exported at `github.com/skaphos/repokeeper/cmd/repokeeper`** — `.goreleaser.yaml` references them by full path in `-X` flags and moving them breaks release stamping silently
- [X] T009 [US1] Resolve the version once and pass the resolved value into `mcpserver.New(eng, cfgPath, version, logger)` in `cmd/repokeeper/mcp.go`, so the MCP handshake and the version command cannot disagree (FR-005)

**Checkpoint**: `go test ./internal/buildinfo/... ./cmd/repokeeper/... -race` passes; quickstart
check 1 shows no placeholder on any of the four build paths.

---

## Phase 4: User Story 2 — Install on Linux with the system package manager (Priority: P2)

**Goal**: `.deb` and `.rpm` on the release page for both architectures, covered by the same checksum
manifest, signature and provenance as every other artifact.

**Independent Test**: Install each package on a matching container image, confirm the binary runs and
reports the release version, then remove it and confirm nothing is left behind.

### Implementation for User Story 2

- [X] T010 [US2] Add the `nfpms` block to `.goreleaser.yaml`: formats `deb` + `rpm`, `bindir: /usr/bin`, `file_name_template: "{{ .ConventionalFileName }}"`, metadata (maintainer `Skaphos <shawn@skaphos.io>`, homepage, license MIT, vendor, section `utils`), and `contents` installing `LICENSE`, `THIRD_PARTY_NOTICES.md` and `third_party_licenses/` under `/usr/share/doc/repokeeper/` (FR-011 – FR-013)
- [X] T011 [US2] Add a **second** `sboms` entry with `artifacts: package` to `.goreleaser.yaml`. Packages do not inherit the archive SBOM — without this the `.deb`/`.rpm` ship unattested and the omission is silent (FR-014)
- [X] T012 [US2] Run `goreleaser check` and a `goreleaser release --snapshot --clean --skip=publish`, confirming `dist/` contains both package formats for both architectures **and** a matching `*.sbom.json` per package (quickstart check 4)

**Checkpoint**: Packages build locally with their own SBOMs.

---

## Phase 5: User Story 3 — Find RepoKeeper from an MCP client (Priority: P3)

**Goal**: A checked-in, schema-valid `server.json` published to the MCP registry as
`io.skaphos/repokeeper`, which cannot drift from the server's real tool surface.

**Independent Test**: Validate the entry against the published schema in CI, then confirm a release
publishes an entry whose version matches the release and whose outcome appears in the run's output.

### Tests for User Story 3

- [X] T013 [US3] Write `internal/mcpserver/serverjson_test.go` asserting: `name` is exactly `io.skaphos/repokeeper`; `transport.type` is `stdio`; `packages[].identifier` is `ghcr.io/skaphos/repokeeper`; both `version` fields are equal; the described tool surface agrees with `ReadOnlyToolNames()`; and a `packages` array that is empty or absent fails cleanly rather than panicking the test binary (FR-016, FR-020)

### Implementation for User Story 3

- [X] T014 [US3] Create `server.json` at the repository root per contracts/server-json.md, with `0.0.0` placeholders in both version fields. **Describe the mixed tool surface honestly** — 9 read-only and 5 mutating tools; unlike sting, RepoKeeper must not claim a read-only server — and state the container's one-explicit-root-per-entry contract (FR-016, FR-017, FR-020, FR-026)
- [X] T015 [US3] Add a `manifests` job to `.github/workflows/ci.yml` running `goreleaser check` and `check-jsonschema` (pinned) against the MCP registry schema; wire it into the `build` and `summary` job `needs` lists and increment the summary's job count from 8 to 9 (FR-017)
- [X] T016 [P] [US3] Add `check-jsonschema` to `.tool-versions` so a local run validates against the same version CI uses

**Checkpoint**: `server.json` validates against the schema and the drift tests pass.

---

## Phase 6: User Story 4 — Run RepoKeeper as a container (Priority: P4)

**Goal**: A multi-arch image serving `repokeeper mcp` over stdio against an identical-path,
read-only-by-default workspace mount.

**Independent Test**: Pull the image on both architectures, drive it with an MCP client over stdio,
confirm an inventory query returns the mounted workspace's repositories and that a mutating tool
refuses while naming the read-only mount.

### Tests for User Story 4

- [X] T017 [P] [US4] Write `internal/mcpserver/readonly_test.go` covering both the refusal message for a read-only-filesystem write failure and the unaffected inspection path (FR-025)

### Implementation for User Story 4

- [X] T018 [US4] Create `internal/mcpserver/readonly.go` translating a read-only-filesystem write failure into a refusal that names the mount as the cause and states the remedy (a read-write mount plus supplied credentials). Detect with `errors.Is(err, syscall.EROFS)` behind `//go:build unix`, with a no-op fallback elsewhere — no string matching. Stdlib only (FR-025)
- [X] T019 [US4] Apply the translation at the mutating MCP tool handlers so refusals are explained rather than surfacing a raw `*fs.PathError` or raw git error. **Keep every mutating tool advertised** — hiding them would be a silently reduced surface, which FR-025 forbids (Principles VI, VII)
- [X] T020 [US4] Create `Dockerfile` with **no build stage**, copying GoReleaser's pre-built binary via `ARG TARGETPLATFORM`. Base: Alpine pinned by digest plus `apk add --no-cache git ca-certificates` — **not** distroless, because `internal/gitx` shells out to the `git` binary (research.md R2). `USER 65532:65532`, `ENTRYPOINT` the binary, `CMD ["mcp"]`, and **no `WORKDIR`** (FR-022, FR-023, FR-026)
- [X] T021 [US4] Bake a system gitconfig setting `safe.directory=*` into the image, with a comment recording why it is safe here — the container's filesystem view is exactly what the user mounted, so there is no foreign repository to be tricked into trusting. **Without this every git call fails** with `detected dubious ownership` (measured: exit 128 at uid 65532; research.md R3)
- [X] T022 [P] [US4] Create `.dockerignore` excluding `.git/`, `dist/`, `specs/`, `graphify-out/`, `coverage*`, `Dockerfile*` and local build output
- [X] T023 [US4] Add the `dockers_v2` block to `.goreleaser.yaml`: images `ghcr.io/skaphos/repokeeper`, platforms `linux/amd64` + `linux/arm64`, tags `{{ .Version }}` and `{{ if not .Prerelease }}latest{{ end }}`, and the full OCI label set with `version`/`revision` templated (FR-021, FR-028)
- [X] T024 [US4] Verify the ownership guard end-to-end per quickstart check 5a: build a snapshot image and run `git status` inside it as uid 65532 against a bind-mounted repository, asserting exit 0 rather than 128

**Checkpoint**: Snapshot image starts as an MCP server, inspection works against a read-only
identical-path mount, and mutating tools refuse with an explanation.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: The release pipeline that publishes and confirms every channel, plus the documentation
that makes the deviations legible.

### Release pipeline

- [X] T025 **Replace** the credential-driven skip in `.github/workflows/release.yml` with a pre-flight failure. The current `Determine GoReleaser args` step emits `--skip=homebrew` and a `::warning::` when the tap token is missing or cannot reach the tap, then completes green — the ADR-0007 stale-cask incident encoded as workflow logic. The replacement must `::error::` and exit 1, naming `RELEASE_BOT_CLIENT_ID`/`RELEASE_BOT_PRIVATE_KEY` or the App installation (FR-032)
- [X] T026 Add QEMU setup, Buildx setup, GHCR login and the `packages: write` permission to `.github/workflows/release.yml` so `dockers_v2` publishes inside the single GoReleaser invocation (FR-021, FR-031)
- [X] T027 Add the `server.json` version stamping step (single `jq` expression setting both fields from the tag), a pinned `mcp-publisher` install, and the DNS-authenticated publish — all `continue-on-error`, with an `always()` report naming the `skaphos.io` TXT record and `MCP_REGISTRY_KEY` as the things to check (FR-018, FR-019)
- [X] T028 Add the `verify` job to `.github/workflows/release.yml` (`needs: release`) querying each channel for the version it actually **serves**: release assets, the Homebrew cask, the container manifest on both architectures, and the registry entry. Retry with increasing backoff so an outage is distinguishable from a channel that did not publish. Blocking for every channel except the MCP registry, which is `continue-on-error`; final summary step runs `if: always()` (FR-033 – FR-035)

### Documentation

- [X] T029 [P] Add the per-channel upgrade table to `README.md` covering all six channels, and the container MCP client configuration showing the identical-path mount and explicit `--config` (FR-009, FR-029)
- [X] T030 [P] Update `INSTALL.md`: Linux package install/removal, the explicit statement that **no hosted apt or yum repository exists**, and the container section (FR-015, FR-029)
- [X] T031 [P] Update `RELEASE.md` describing the new channels, the credential pre-flight, and the verify job (FR-036)
- [X] T032 Document the container's honest limits wherever the container is described: one configured entry serves one workspace root, a multi-root workflow needs one entry per root, the native binary remains better for position-sensitive multi-root use, and the container supports Git only — Mercurial is unavailable (FR-026, FR-029)

### Validation

- [X] T033 Run the full local gate `go -C tools tool task ci` and fix everything it reports
- [X] T034 Assert zero dependency growth: `go mod tidy` then `git diff --exit-code go.mod go.sum`. The entire self-update deviation rests on dependency cost, so this feature must not add any by accident (FR-037, SC-010)
- [X] T035 Walk quickstart.md checks 1–5 and 8, confirming each acceptance criterion

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependencies
- **Phase 2 (Foundational)**: independent of all stories; landed first so the deviation record exists
- **Phase 3 (US1)**: independent — the MVP
- **Phase 4 (US2)**: independent of US1
- **Phase 5 (US3)**: independent of US1/US2
- **Phase 6 (US4)**: T020 benefits from US1 (the image reports a real version), but is not blocked by it
- **Phase 7 (Polish)**: T026 needs T023; T027 needs T014; T028 needs every channel to exist

### Story independence

| Story | Depends on | Deliverable alone |
| --- | --- | --- |
| US1 | — | Every install path reports a real version |
| US2 | — | `.deb`/`.rpm` on the release page |
| US3 | — | Discoverable in the MCP registry |
| US4 | — (soft: US1) | Runnable as a container |

All four are independently shippable. The plan's original ordering put the container before the
registry; either order works because `server.json`'s image identifier is asserted against a constant
in T013 rather than against the GoReleaser config.

### Parallel opportunities

- **T005 + T006** — different test files, same story
- **T016 + T015** — `.tool-versions` and `ci.yml` are separate files
- **T017 + T022** — test file and `.dockerignore`
- **T029 + T030 + T031** — three separate documentation files
- **Whole stories**: US1, US2 and US3 touch disjoint files and can proceed concurrently. US4 and US2 both edit `.goreleaser.yaml`, so they must not run in parallel.

### Sequential constraint on `.goreleaser.yaml`

T010, T011 and T023 all edit the same file and must run in order. This is the one file-level
serialization point in the feature.

---

## Implementation Strategy

### MVP

**User Story 1 alone** is a shippable increment: it closes the widest-blast-radius defect (every
source install indistinguishable from every other, making bug reports unactionable) with three files
and no release-pipeline risk.

### Incremental delivery

1. Phase 1 + 2 → deviation recorded
2. Phase 3 (US1) → **MVP**, independently releasable
3. Phase 4 (US2) → Linux users served
4. Phase 5 (US3) → discoverable
5. Phase 6 (US4) → container channel
6. Phase 7 → pipeline publishes and confirms all of it

### Highest-risk tasks

| Task | Risk |
| --- | --- |
| **T021** | Omitting `safe.directory` makes **every** container git call fail; measured, and the likeliest defect in the feature |
| **T011** | Omitting the package `sboms` entry ships unattested packages, silently |
| **T008** | Moving the ldflags variables breaks release stamping with no test to catch it |
| **T025** | The behavior being replaced currently reports success while dropping a channel |

### Format validation

All 35 tasks follow `- [ ] [TaskID] [P?] [Story?] Description with file path`. Setup, Foundational
and Polish tasks carry no story label by design; every task in Phases 3–6 carries US1–US4.

---

## Execution Notes

All 35 tasks completed. Four diverged from the task text as written; each is recorded rather than
silently absorbed.

- **T004 — no ADR index exists.** The task assumed `docs/adr/README.md` from sting's layout.
  RepoKeeper has no ADR index file. Creating one would be scope beyond this feature, so ADR-0016 is
  linked from `README.md`'s upgrade section and `RELEASE.md` instead, where a reader actually
  encounters the decision.
- **T008 — flag name.** The contract specified `--output json`. RepoKeeper's convention is
  `--format`/`-o` with `table|wide|json` (`addFormatFlag` in `cmd/repokeeper/flags.go`); `--output`
  already means a *file path* on `export`. Implemented as `--format` to match the surrounding code;
  `contracts/cli-version.md` describes the intent, not the flag spelling.
- **T016 — widened.** Beyond `check-jsonschema`, `.tool-versions` now also pins `syft`, `cosign` and
  `pipx:reuse`, matching sting's file exactly so every tool this repo's gate needs resolves from the
  repo rather than a developer's global mise config.
- **T024 — verified without a bind mount.** Docker bind-mount propagation degraded partway through
  this environment's session (a path that mounted successfully earlier stopped resolving, for stock
  `alpine/git` as well as this image), so the guard was proven with an in-container A/B instead:
  a repository owned by uid 1000 baked into a fixture image, read as uid 65532.

  | Condition | Result |
  | --- | --- |
  | `GIT_CONFIG_NOSYSTEM=1` (guard disabled) | `fatal: detected dubious ownership` — exit 128 |
  | Shipped image default | exit 0; `git log` returns the commit |
  | `git config --system --get safe.directory` | `*` |

  This is the same conclusion as research.md R3, reached without depending on the host mount.

### Correction made during implementation

The tool surface is **9 read-only / 5 mutating**, not the 8/6 recorded during planning. `plan_sync`
carries `ReadOnlyHint: true` — it plans without executing — despite sitting under the
"Phase 3: mutation tools" comment in `registerTools`, which is what the earlier count mis-read. The
annotation is authoritative, and the drift test reads `ReadOnlyToolNames()` precisely so the
registration comments cannot mislead again. `spec.md`, `data-model.md` and
`contracts/server-json.md` were corrected.

### Verification performed

| Check | Result |
| --- | --- |
| `go -C tools tool task ci` | exit 0 |
| Per-package coverage gate | passed; `buildinfo` and `readonly` at 100% |
| `reuse lint` | compliant, 357/357 files |
| `go mod tidy` + `git diff go.mod go.sum` | clean — zero dependency growth (FR-037, SC-010) |
| `goreleaser check` | valid |
| Snapshot build | 4 packages + 6 archive SBOMs + 4 **package** SBOMs (FR-014) |
| `.deb` install/remove in `debian:stable-slim` | binary on `PATH`, docs installed, clean removal |
| `server.json` vs MCP schema | valid (description 98/100 chars) |
| Container image | builds, runs non-root, git 2.49.1 present, Mercurial absent |
| Windows cross-compile | `GOOS=windows go build ./...` OK (non-unix `isReadOnlyFS` path) |

`.deb` doc files initially appeared missing on install; `dpkg -c` showed them present. Cause was
`path-exclude /usr/share/doc/*` in the `debian:stable-slim` image, not a packaging defect —
confirmed by re-installing with that config removed.
