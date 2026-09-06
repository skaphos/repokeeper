---

description: "Task list for the stable plugin adapter contract"
---

# Tasks: Stable Plugin Adapter Contract

**Input**: Design documents from `/specs/004-adapter-contract/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: **Mandatory, not optional.** The upstream Spec Kit scaffold marks test tasks optional; the
RepoKeeper constitution's Engineering Constraints require meaningful tests in the same change as the
behaviour, and the constitution's own sync-impact report directs that this gate be enforced at
`/speckit-tasks` time. Test tasks below are therefore required work, not suggestions.

**Organization**: Grouped by user story so each ships independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3, or blank for shared work

---

## Phase 0: Gates (blocking)

**Purpose**: Settle the decisions that change the shape of everything downstream. Do these first.

- [x] **T001** Confirm no external consumer depends on `skaphos.io/repokeeper/v1beta1`.
      **Outcome:** no external consumer found. Every remaining occurrence of the beta value is either
      the **config** schema (`internal/config`, plus the config examples in `README.md` and
      `DESIGN.md` §"config"), which is a different schema that keeps its value, or documentation of
      the output contract, which this change updates. Repo metadata was found to use an entirely
      separate scheme (`repokeeper/v1`) and never shared the value at all — the spec's claim that
      "all three shared" was inaccurate and has been corrected in `DESIGN.md` and the independence
      test. No adapter repository exists yet. Promotion proceeds.
- [x] **T002** Resolve the open risk in plan.md: can the surface inventory be **derived** from the
      Cobra command tree and MCP registries, or must it be hand-maintained?
      **Outcome: derivable.** The blocker was that nothing distinguished an adapter-facing command
      from an interactive one. The clarify session settled the rule — a command is adapter-facing
      exactly when it accepts a JSON format flag — which is readable straight from the Cobra tree.
      The MCP side already enumerates its tools in a test asserting every registered tool has a
      contract case. T014 is therefore a real drift guarantee, not a count, and the hand-maintained
      fallback is dropped.
      *(Resolved by clarify session 2026-09-05.)*
- [x] **T003** Write and merge an ADR recording the envelope mechanism and the `v1beta1` → `v1`
      promotion (`docs/adr/0018-adapter-contract-envelope.md`). Required by the constitution for
      hard-to-reverse decisions, and gated before implementation. Must cite ADR-0006 as the policy it
      implements — **not** supersede it; ADR-0006 remains accepted and immutable.
- [x] **T004** Confirm `version -o json` has no downstream parser that enveloping would break
      (check `internal/mcpinstall`, install tooling, `.goreleaser.yaml`, CI). If it does, decide
      whether to exempt it and record the exemption in the inventory.

---

## Phase 1: Foundational (blocking prerequisites)

**Purpose**: The shared envelope. No surface consumes it yet, so this phase ships safely on its own.

- [x] **T005** Create `cmd/repokeeper/contract.go` with the single `apiVersion` constant
      (`skaphos.io/repokeeper/v1`) and a reusable generic envelope type carrying `apiVersion`,
      optional `generated_at`, and a named payload.
- [x] **T006** Ensure the constant is reachable from `internal/mcpserver` without an import cycle,
      so CLI and MCP share one source (FR-003). If `cmd/repokeeper` cannot be imported by
      `internal/`, hoist the constant to a small `internal/contract` package — decide here, not
      later.
- [x] **T007** [P] Add `cmd/repokeeper/contract_test.go`: envelope always carries `apiVersion`;
      empty collections marshal as `[]` and never `null` (FR-007); `generated_at` omitted rather than
      zero-valued when not meaningful.
- [x] **T007a** [P] Test: the contract `apiVersion` is independent of the config and repo-metadata
      `apiVersion` values (FR-004). They share a string today, so assert they are *distinct
      constants* — a test comparing values would pass while the coupling this forbids went unnoticed.
      *(Added by [analysis.md](./analysis.md) A3.)*

---

## Phase 2: User Story 1 — uniform version detection (P1) 🎯 MVP

**Goal**: Every adapter-facing surface carries `apiVersion` in the same place, CLI and MCP.

**Independent test**: Invoke every surface against a fixture workspace; assert each response carries
`apiVersion` equal to the shared constant.

> **Ordering constraint**: T008 and T009 must land in the **same change** (FR-018). Enveloping CLI
> alone reintroduces the CLI/MCP drift that PR #344 fixed.

- [x] **T008** [US1] Wrap the CLI action surfaces in the envelope: `scan.go`, `sync.go`
      (reconcile/sync), `describe.go`, `label.go`, `repair_upstream.go`, `version.go`. Each gains a
      named payload field per [surface-inventory.md](./contracts/surface-inventory.md).
- [x] **T009** [US1] Envelope MCP structured results in `internal/mcpserver/tool_results.go`, and
      verify against a real MCP client (plan.md open risk) that an enveloped result is accepted.
- [x] **T010** [US1] Promote `status.go` from `v1beta1` to `v1` and switch it to the shared constant,
      removing the now-duplicated local `statusJSONAPIVersion`.
- [x] **T011** [US1] Test: enumerate every adapter-facing surface, assert each emits `apiVersion`
      equal to the shared constant (SC-001), and assert CLI and MCP resolve to the *same constant*
      rather than merely equal strings.
- [ ] **T011a** [US1] Test: a failed invocation emits **no** envelope on stdout and exits non-zero
      (FR-019); an intentional skip is a success carrying `ok: true` plus a reason. Adapters branch on
      this, so it must be verified rather than assumed.
      *(Added by [analysis.md](./analysis.md) A1.)*
- [ ] **T011b** [US1] Test: shared-record **field-set** parity between each CLI/MCP pair —
      `reconcile --dry-run` ↔ `plan_sync`, `reconcile` ↔ `execute_sync`, `scan` ↔ `scan_workspace`
      (FR-017). T011 only covers the `apiVersion`; this covers the parity `cli-mcp-parity.md` actually
      promises and that #344 had to restore.
      *(Added by [analysis.md](./analysis.md) A2.)*

- [x] **T009a** [US1] Envelope the three MCP **resources** in `internal/mcpserver/resources.go` —
      `config`, the registry snapshot, and the per-repo template. All three are advertised as
      `application/json` and currently emit `json.Marshal` of raw domain objects, so they sit outside
      the contract despite being reachable by any MCP client (FR-001a). The original spec enumerated
      only the 14 tools and missed them.
      *(Added by clarify session 2026-09-05.)*
- [x] **T009b** [US1] Add the three resources to the surface inventory with stability and
      read/mutation classes, and confirm the registry snapshot is covered by the FR-021 redaction —
      it serialises `remote_url` and is the surface most likely to carry credentials today.
      *(Added by clarify session 2026-09-05.)*

**Checkpoint**: Adapters can detect contract version from any single response. Shippable.

---

## Phase 3: User Story 2 — published contract (P2)

**Goal**: An adapter author builds against a document, never against Go source.

**Independent test**: A reviewer who has not seen the code can enumerate every surface and predict
each response shape from the published document alone.

- [x] **T012** [US2] Revise `DESIGN.md` §6.3 and §6.4: generalise the single-surface stability policy
      to the uniform envelope; update §6.4's bare-array description, preserving its CLI/MCP parity
      *rationale* while correcting the shape.
- [ ] **T013** [US2] Publish the surface inventory in the repository's user-facing docs (not only in
      `specs/`), since adapter authors are external and will not read a feature spec directory.
- [ ] **T014** [US2] Implement the drift test per T002's outcome — now a **derived** check, not a
      count. Walk the Cobra command tree for commands accepting a JSON format flag, plus the MCP tool
      and resource registries, and fail when that set diverges from the documented inventory
      (FR-011, SC-003). Extend the existing `TestDesignDocNamesStatusJSONAPIVersion` pattern rather
      than duplicating it.
      *(Unblocked by clarify session 2026-09-05.)*
- [ ] **T015** [P] [US2] Document what is explicitly non-contractual — table/wide output, prose,
      logs, `internal/...`, specific exit values (FR-010).

**Checkpoint**: An external repo can be built against the contract. Shippable.

---

## Phase 4: User Story 3 — read/mutation boundary (P3)

**Goal**: Every surface declares `read` or `mutation`, and the declaration is enforced.

**Independent test**: For every surface, assert the declared class matches actual behaviour.

- [x] **T016** [US3] Annotate every surface with its `read`/`mutation` classification in the
      inventory, including the two counter-intuitive MCP cases (`plan_sync` is read,
      `scan_workspace` is mutation, FR-014) **and** the CLI flag-default cases, classified at default
      flags with the flag that changes them named (FR-014a): `scan --write-registry=true`,
      `reconcile --dry-run=false`, `repair upstream --dry-run=true`.
      *(Amended by [analysis.md](./analysis.md) A5.)*
- [ ] **T017** [US3] Test: no surface classified `read` writes to registry, config, or working tree
      (FR-013, SC-004). Assert by snapshotting state before and after each read surface.
- [ ] **T018** [US3] Test: under a read-only workspace, every `read` surface succeeds (FR-016) and
      every `mutation` surface refuses with a message naming cause and remedy while remaining
      advertised (FR-015, SC-006). Extend the existing `readonly_*_test.go` coverage.
- [ ] **T019** [P] [US3] Test: skipped work carries a machine-readable reason, including
      backend-unsupported skips for Mercurial (FR-020, Principles VI and XI).
- [x] **T019a** [US3] Apply `urlutil.RedactCredentials` to every URL field emitted on an
      adapter-facing surface (FR-021). Today the helper is only wired into `export` (as a hazard
      detector) and debug-arg logging, so `remotes[].url` reaches adapter JSON verbatim from git.
      *(Added by clarify session 2026-09-05.)*
- [x] **T019b** [US3] Test: feed a credential-bearing remote (`https://user:token@host/repo.git`)
      through **each** adapter-facing surface and assert no response contains the credential
      (SC-007). Assert end-to-end per surface rather than unit-testing the helper — a helper test
      passes while a surface that never calls it leaks.
      *(Added by clarify session 2026-09-05.)*
- [x] **T019c** [US3] Document in the contract that a URL from an adapter-facing response is not
      usable for cloning, and name `add` / `--checkout-missing` as the supported path (FR-022).
      *(Added by clarify session 2026-09-05.)*

**Checkpoint**: The boundary is contractual and enforced. Shippable.

---

## Phase 5: Polish & cross-cutting

- [ ] **T020** Update `README.md` where it describes machine-readable output, per the constitution's
      documentation constraint.
- [ ] **T021** Release-note the breaking change explicitly. ADR-0006 requires breaking changes carry
      release-note visibility; a contract break discovered by adapter authors at runtime is a defect
      in this feature, not in their code.
- [ ] **T022** [P] Verify cross-platform parity of the enveloped output on macOS, Linux and Windows
      (Principle XII — parity is a requirement, not a courtesy).
- [ ] **T023** [P] Confirm no measurable performance regression on `list_repositories`, documented as
      "fast — reads registry only".
- [ ] **T024** Run the full local gate `go -C tools tool task ci` before opening the PR.
- [ ] **T025** Confirm the follow-on issues can now proceed: [#289](https://github.com/skaphos/repokeeper/issues/289)
      (JSON hardening against this contract) and [#286](https://github.com/skaphos/repokeeper/issues/286)
      (adapter version compatibility policy).

---

## Dependencies & Execution Order

### Phase dependencies

```text
Phase 0 (gates)  ──►  Phase 1 (envelope)  ──►  Phase 2 (US1)  ──►  Phase 3 (US2)
                                                      │
                                                      └──────────►  Phase 4 (US3)  ──►  Phase 5
```

- **Phase 0 blocks everything.** T002's outcome changes T012–T014; T003 is a constitutional gate.
- **Phase 3 depends on Phase 2.** The inventory cannot accurately document an envelope that does not
  exist yet.
- **Phase 4 is independent of Phase 3.** The read/mutation boundary already exists in code; this
  phase classifies and tests it. It could ship before US2 if priorities changed.

### Within Phase 2

T008 and T009 are a single atomic change (FR-018). T010 follows. T011 last, since it asserts across
all of them.

### Parallel opportunities

- T007 runs alongside T005/T006 once the type signature is agreed.
- T015, T019, T022, T023 are marked `[P]` — independent files, no shared state.
- T012 and T013 touch different documents and can proceed together.

---

## Implementation Strategy

**MVP is Phase 0 + 1 + 2.** That delivers the thing #287 exists for: an adapter can detect contract
compatibility from any single response. Phases 3–5 make it usable and enforced, but the mechanism is
the irreducible core.

**Do not start Phase 2 before T002 and T003.** T002 determines whether the drift guarantee is real or
nominal, and T003 is a constitution gate on a hard-to-reverse decision. Both are cheap; discovering
either late is not.
