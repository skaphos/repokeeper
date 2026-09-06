# Feature Specification: Stable Plugin Adapter Contract

**Feature Branch**: `feat/004-adapter-contract` (spec directory `004-adapter-contract`)

**Created**: 2026-09-05

**Status**: Implemented (core) — see [tasks.md](./tasks.md) for what remains open

The envelope, the `v1` promotion, the shared constant and the supporting tests are implemented and
covered by [ADR-0018](../../docs/adr/0018-adapter-contract-envelope.md). Still open: publishing the
surface inventory to user-facing docs (T013), the code-derived inventory drift test (T014, whose
feasibility is recorded under T002), and the read/mutation enforcement tests (T017–T019). The
classification itself is documented; what is not yet in place is the test that proves a surface
classed `read` never writes.

**Input**: GitHub issue [#287](https://github.com/skaphos/repokeeper/issues/287) — "Define a stable plugin adapter contract for standalone IDE integrations", milestone `v2.0.0 — the contract release`.

---

## Context and Prior Art

This feature is **not greenfield**. Per the constitution's Specification and Decision Workflow ("specs MUST cite relevant upstream findings rather than re-researching settled questions"), the following are already settled and are inputs, not open questions:

| Source | Status | What it already settles |
| --- | --- | --- |
| [ADR-0006](../../docs/adr/0006-adapter-contract-stability.md) | Accepted 2026-04-03 | Machine-readable output is contractual; human table output is not. Breaking changes require documentation, schema/version notes and release-note visibility. CLI and MCP contracts are related but need not be byte-identical — each must be independently well-defined. Rejects best-effort JSON, rejects requiring adapters to import internal packages, rejects MCP-only. |
| `DESIGN.md` §6.3 | Shipped | `get`/`status -o json` is enveloped and versioned (`apiVersion: skaphos.io/repokeeper/v1beta1`), with additive-safe evolution rules, a single-source constant (`statusJSONAPIVersion`), and a drift test (`TestDesignDocNamesStatusJSONAPIVersion`) that fails if the document and the constant diverge. |
| `DESIGN.md` §6.4 | Shipped | `reconcile`/`sync -o json` emits a bare, unenveloped JSON **array**, deliberately matching `scan`, `repair-upstream` and the MCP `plan_sync`/`execute_sync` result shape so CLI and MCP consumers parse identical fields. |
| `internal/mcpserver/readonly.go` | Shipped | Mutating MCP tools remain advertised under a read-only workspace mount and fail with an explaining refusal rather than a bare `EROFS`. Principle VII made concrete. |

**The gap this feature closes.** ADR-0006 requires that adapter compatibility "must not depend on guesswork", but deliberately leaves the mechanism open ("the exact mechanism may vary by surface"). Today an adapter can detect a breaking change on exactly one surface — `get`/`status` — because it is the only one carrying an `apiVersion`. Every other adapter-facing surface (`scan`, `reconcile`/`sync`, `repair upstream`, `describe`, `label`, and all fourteen MCP tools) is unversioned. An adapter has no way to tell a v2 core from a v3 core on those surfaces except by parsing and hoping.

**Decisions taken for this feature** (resolved before drafting; see `research.md` for the alternatives weighed):

1. **Uniform envelope.** Every adapter-facing JSON surface — CLI *and* MCP — is wrapped in a versioned envelope carrying `apiVersion`. This is a breaking change to the currently-unenveloped surfaces, taken deliberately in the 2.0.0 major where it is cheap.
2. **Promote `v1beta1` → `v1`.** The 2.0.0 release ships the contract as `skaphos.io/repokeeper/v1`.

The envelope must be applied to CLI and MCP *together*. §6.4's bare-array shape exists specifically to preserve CLI/MCP field parity — a parity actively maintained (see [#344](https://github.com/skaphos/repokeeper/pull/344)). Enveloping only one side would break it.

---

## Clarifications

### Session 2026-09-05

- Q: How should adapters detect contract version on the currently unenveloped surfaces (`scan`, `sync`/`reconcile`, `repair upstream`)? → A: Uniform envelope everywhere — wrap every adapter-facing JSON output, CLI and MCP, in `{apiVersion, ...}`. Accepted as breaking for existing bare-array consumers, taken at the 2.0.0 major where it is cheap. Alternatives weighed and rejected in [research.md](./research.md) §3: an out-of-band discovery surface (non-breaking but coarse), and per-surface mechanisms (closest to the status quo that produced this issue).
- Q: The contract is currently marked `v1beta1`. Promote it for the 2.0.0 release? → A: Promote to `skaphos.io/repokeeper/v1`. A release named "the contract release" shipping a contract still marked beta undercuts its own message, and external adapter repos are being asked to commit to it. Conditional on the assumption, recorded below and gated as task T001, that no external consumer of `v1beta1` exists.
- Q: Does enveloping the CLI alone suffice? → A: No. `DESIGN.md` §6.4 chose the bare-array shape specifically to hold CLI/MCP field parity, and PR #344 recently restored that parity. CLI and MCP must be enveloped in the same change (FR-018).

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An adapter reads the contract version from any surface, uniformly (Priority: P1)

An author building `repokeeper-vscode` in a separate repository invokes any adapter-facing RepoKeeper surface — a CLI command with `-o json`, or an MCP tool — and finds the contract version in the same place, spelled the same way, every time. They write one compatibility check, not one per command.

**Why this priority**: This is the feature. Without a uniform, discoverable version marker, an adapter cannot safely detect an incompatible core, which is the precise thing ADR-0006 requires and the precise thing missing today. Every other story in this spec depends on the envelope existing.

**Independent Test**: Invoke every adapter-facing CLI command with `-o json` and every MCP tool against a fixture workspace; assert each response carries a top-level `apiVersion` equal to the single source constant. Fully testable without any adapter existing.

**Acceptance Scenarios**:

1. **Given** a registered workspace, **When** an adapter runs `repokeeper get -o json`, **Then** the response carries a top-level `apiVersion` of `skaphos.io/repokeeper/v1`.
2. **Given** the same workspace, **When** an adapter runs `repokeeper reconcile --dry-run -o json`, **Then** the response is an object carrying the same top-level `apiVersion` and the per-repo records under a named collection field, rather than a bare array.
3. **Given** the same workspace, **When** an adapter calls the MCP `plan_sync` tool, **Then** the structured result carries the same `apiVersion` value as the equivalent CLI surface.
4. **Given** any adapter-facing surface, **When** the emitted `apiVersion` is compared against the documented contract version in `DESIGN.md`, **Then** they match, enforced by an automated drift test rather than review.

---

### User Story 2 - An adapter author builds against a published contract without reading Go source (Priority: P2)

An author starting `repokeeper-jetbrains` reads one document that enumerates every surface they are allowed to depend on, the shape each returns, and which parts are guaranteed stable. They never open `internal/`, and they never infer behaviour from a table rendering.

**Why this priority**: ADR-0006 explicitly rejects requiring adapters to import internal packages, and issue #287's acceptance criteria require the contract be "explicit enough that standalone plugin repos can be built without importing internal core packages". The envelope (US1) is the mechanism; this is the documentation that makes it usable. It is P2 rather than P1 only because the mechanism must exist before it can be documented accurately.

**Independent Test**: A reviewer who has never seen the codebase can, using only the published contract document, enumerate every adapter-facing surface and predict the shape of each response. Verifiable by checking the document against the surface inventory produced by an automated test.

**Acceptance Scenarios**:

1. **Given** the published contract document, **When** an adapter author looks for the list of surfaces they may depend on, **Then** every adapter-facing CLI command and MCP tool is enumerated with its stability class.
2. **Given** the published contract, **When** a surface is added or removed in code, **Then** an automated drift test fails until the document is updated.
3. **Given** the published contract, **When** an adapter author looks for what is explicitly *not* contractual, **Then** human-oriented `table`/`wide` output, exit-code-adjacent prose, and all `internal/` packages are named as non-contractual (Principle IX: state plainly what this is not).

---

### User Story 3 - An adapter distinguishes read surfaces from mutation surfaces before calling them (Priority: P3)

An adapter offering a "refresh status" action needs to know which calls are safe to make automatically — on a timer, on window focus — and which alter the user's workspace and therefore require an explicit user gesture. It determines this from the contract, not from a tool's name.

**Why this priority**: Issue #287 requires the contract scope "distinguishes between read surfaces and mutation surfaces". The boundary already exists in code but is not stated as contract; the risk it mitigates is real but is a correctness-of-integration concern rather than a blocker to building an adapter at all.

Four surfaces are genuinely mis-guessable, which is what makes this more than bookkeeping. `plan_sync` is always dry-run and never mutates despite planning a mutation, while `scan_workspace` **does** write the registry despite reading like a query. On the CLI the trap is flag defaults rather than names: `scan --write-registry` defaults to `true`, so a bare `scan` writes; `repair upstream --dry-run` defaults to `true`, so a bare `repair upstream` does not. `reconcile` and `repair upstream` disagree on their `--dry-run` default, so an adapter cannot generalise from one to the other.

**Independent Test**: For every adapter-facing surface, the contract declares a class of `read` or `mutation`; assert by test that the declared class matches actual behaviour — no surface classed `read` writes to the registry, config, or a working tree.

**Acceptance Scenarios**:

1. **Given** the contract, **When** an adapter enumerates surfaces, **Then** each carries an explicit `read` or `mutation` classification.
2. **Given** a workspace mounted read-only, **When** an adapter invokes a `mutation` surface, **Then** it receives a refusal naming the cause and the remedy, not a bare filesystem error (existing behaviour, now contractual).
3. **Given** a workspace mounted read-only, **When** an adapter invokes any `read` surface, **Then** it succeeds — inspection degrades to read-only, never to blindness (Principle VII).

---

### Edge Cases

- **An adapter built for a newer core talks to an older core.** The older core emits an older `apiVersion`; the adapter must be able to detect this from the first response without a separate handshake call.
- **An adapter built for an older core talks to a newer core within the same contract major.** Additive fields appear that the adapter does not know. It must ignore unknown fields and continue — this is required of consumers, not optional.
- **A surface returns an empty result set.** The envelope is still present and still carries `apiVersion`; an empty collection is `[]`, never `null` and never an absent field, so an adapter's parse path is uniform.
- **A command fails.** Errors go to stderr with a non-zero exit (Principle XII) — the spec must state whether a failed invocation emits an envelope at all, so adapters do not attempt to parse stdout on failure.
- **A surface is invoked with a filter that changes shape** (e.g. `get --diverged` adds a `diverged` array). Per §6.3 the `apiVersion` is unchanged by filters; this must remain true and be stated.
- **Mercurial-backed repositories.** Per Principle XI, unsupported operations are skipped with a reason. The contract must carry that reason in a machine-readable field, not only in prose.
- **A mutating surface partially succeeds** across a multi-repo set. Per-record `ok` plus an overall envelope must let an adapter distinguish "all succeeded" from "some succeeded" without inspecting exit codes alone.

---

## Requirements *(mandatory)*

### Functional Requirements

**Envelope and versioning**

- **FR-001**: Every adapter-facing JSON surface MUST emit a top-level object carrying an `apiVersion` field. No adapter-facing surface may emit a bare array as its top-level value.
- **FR-002**: The `apiVersion` value MUST be `skaphos.io/repokeeper/v1` for the 2.0.0 release, promoted from the current `v1beta1`.
- **FR-003**: The `apiVersion` value MUST be sourced from a single constant shared by every surface, CLI and MCP alike, so the surfaces cannot drift from one another.
- **FR-004**: The contract `apiVersion` MUST remain versioned independently of the config `apiVersion` (`internal/config`) and the repo-metadata `apiVersion` (`internal/repometa`); a bump to one MUST NOT require a bump to another.
- **FR-005**: Additive changes — new top-level or per-record fields — MUST NOT bump `apiVersion`. Consumers MUST ignore unknown fields.
- **FR-006**: Removing or renaming a field, changing a field's type, or changing the meaning of an existing enum value MUST bump `apiVersion` and MUST be recorded in the contract document and release notes.
- **FR-007**: Collection-bearing envelopes MUST emit an empty collection as `[]`, never `null` and never an omitted field.

**Surface inventory and documentation**

- **FR-008**: RepoKeeper MUST publish a single document enumerating every adapter-facing surface — each adapter-facing CLI command and each MCP tool — that an external adapter may depend on.
- **FR-009**: For each enumerated surface the document MUST state its stability class, its read/mutation classification, and the shape of its response.
- **FR-010**: The document MUST state plainly what is **not** contractual: human-oriented `table`/`wide` output, prose messages, log output, and every package under `internal/`.
- **FR-011**: A drift test MUST fail when the set of adapter-facing surfaces in code diverges from the set enumerated in the document, extending the existing `TestDesignDocNamesStatusJSONAPIVersion` pattern from one constant to the full inventory.

**Read/mutation boundary**

- **FR-012**: Every enumerated surface MUST carry an explicit `read` or `mutation` classification in the contract.
- **FR-013**: A surface classified `read` MUST NOT write to the registry, the config, or any working tree. This MUST be asserted by test, not by review.
- **FR-014**: `plan_sync` and CLI `--dry-run` plan surfaces MUST be classified `read`; `scan_workspace` and CLI `scan` MUST be classified `mutation` because they write the registry.
- **FR-014a**: Where a surface's classification depends on a flag, the contract MUST state the classification **at the command's default flags** and name the flag that changes it. Verified cases: `scan --write-registry` defaults to `true` (mutation by default), while `repair upstream --dry-run` defaults to `true` (read by default) and `reconcile --dry-run` defaults to `false` (mutation by default). `reconcile` and `repair upstream` disagree on their `--dry-run` default, so the contract MUST NOT present a single generalised rule for plan-style commands.
- **FR-015**: Under a read-only workspace, `mutation` surfaces MUST refuse with an explanation naming cause and remedy, and MUST remain advertised rather than disappearing from the surface list.
- **FR-016**: Under a read-only workspace, every `read` surface MUST continue to succeed.

**CLI/MCP parity**

- **FR-017**: Where a CLI surface and an MCP tool expose the same information, they MUST carry the same `apiVersion` and the same field names and semantics for the shared records.
- **FR-018**: Adding the envelope MUST be applied to CLI and MCP in the same change, so the existing field parity is preserved rather than broken and re-fixed.

**Error behaviour**

- **FR-019**: The contract MUST state whether a failed invocation emits an envelope on stdout, so adapters have a defined parse path on failure.
- **FR-020**: Skipped work MUST carry a machine-readable reason (Principle VI), including backend-unsupported skips for non-Git backends (Principle XI).

### Key Entities

- **Contract envelope**: The top-level object wrapping every adapter-facing response. Carries the contract `apiVersion`, a generation timestamp where meaningful, and exactly one named collection or record payload.
- **Adapter-facing surface**: A CLI command invoked with a machine-readable format, or an MCP tool. Has an identity, a stability class, a read/mutation classification, and a response shape.
- **Stability class**: Whether a surface is contractual and stable, contractual but provisional, or explicitly non-contractual.
- **Read/mutation classification**: Whether invoking a surface can alter registry, config, or working-tree state.
- **Contract version**: The `apiVersion` identifying the schema, versioned independently of config and repo-metadata schemas.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of adapter-facing surfaces emit a top-level `apiVersion`; verified by a test that enumerates surfaces and asserts the field, so the figure cannot silently regress.
- **SC-002**: An adapter author can determine compatibility with a running core from a **single** response to any adapter-facing call — zero additional handshake calls required.
- **SC-003**: The count of adapter-facing surfaces documented equals the count present in code, enforced by a failing test on divergence rather than by review.
- **SC-004**: Zero surfaces classified `read` perform a write, asserted by test against registry, config and working-tree state.
- **SC-005**: An external adapter can be built against the published contract with zero imports from `github.com/skaphos/repokeeper/v2/internal/...`, demonstrated by the contract document containing no reference to an internal package as a consumption path.
- **SC-006**: Every read surface remains callable under a read-only workspace mount, and every mutation surface refuses there with a remedy-bearing message; both asserted by test.

---

## Assumptions

- **The 2.0.0 major is the window for the breaking envelope change.** Applying it later costs a v3 of the contract or a long dual-shape deprecation. This assumption is the basis for choosing the uniform envelope over a non-breaking out-of-band discovery surface.
- **`v1beta1` has no external consumers that a promotion to `v1` would strand.** No standalone adapter repository exists yet; `repokeeper-vscode` and `repokeeper-jetbrains` are prospective. If an external consumer of `v1beta1` is identified before this lands, the promotion needs revisiting.
- **The existing `TestDesignDocNamesStatusJSONAPIVersion` drift-test pattern generalises** from a single constant to a full surface inventory. If enumerating surfaces from code proves impractical, FR-011 degrades to a manually-maintained list with a weaker guarantee, which should be recorded as a deviation.
- **This spec defines the contract; it does not implement the JSON hardening.** Issue [#289](https://github.com/skaphos/repokeeper/issues/289) implements the per-command JSON work against this contract, and issue [#286](https://github.com/skaphos/repokeeper/issues/286) documents the version compatibility policy for adapter repositories. This spec is the shared dependency of both and should land first.
- **No IDE plugin is built in this repository.** Per #287, adapters live in standalone repositories; this feature produces the contract they consume and nothing more.
- **MCP structured results can carry an envelope** without breaking existing MCP clients beyond the intended contract break. If a client requires bare content, that constraint surfaces in planning and is recorded as a deviation.

---

## Out of Scope

- Implementing the per-command JSON hardening enumerated in #289.
- Writing the adapter version-compatibility policy document (#286) — this spec supplies the mechanism that policy will describe.
- Building `repokeeper-vscode`, `repokeeper-jetbrains`, or any adapter.
- A local service or daemon mode. ADR-0006 lists it as a *possible future* surface; nothing in this feature requires it and Principle IX forbids implying coverage that does not exist.
- Changing human-oriented `table`/`wide` output, which ADR-0006 explicitly holds non-contractual and free to evolve.
