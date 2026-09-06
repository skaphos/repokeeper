# Implementation Plan: Stable Plugin Adapter Contract

**Branch**: `feat/004-adapter-contract` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-adapter-contract/spec.md`

## Summary

Give external adapters one uniform, discoverable way to detect RepoKeeper's machine-readable contract
version, and publish the enumerated surface inventory they may build against.

Concretely: wrap every adapter-facing JSON response — CLI and MCP alike — in a versioned envelope
carrying `apiVersion`, promote that version from `skaphos.io/repokeeper/v1beta1` to
`skaphos.io/repokeeper/v1`, classify every surface `read` or `mutation`, and back the whole thing with
drift tests so the document and the code cannot diverge.

ADR-0006 already sets the policy and is not reopened. `get`/`status` already implements the target
shape; this feature generalises it to the remaining surfaces and makes the inventory explicit.

## Technical Context

**Language/Version**: Go, version pinned in `go.mod` (currently `go 1.27.1`)

**Primary Dependencies**: Cobra (CLI), `mark3labs/mcp-go` (MCP server). No new dependencies required —
this feature reshapes existing output rather than adding capability.

**Storage**: N/A. No persisted state changes. Registry (`.repokeeper.yaml`) and repo-local metadata
schemas are untouched; only output shape changes.

**Testing**: Ginkgo v2 + Gomega, per the constitution's Engineering Constraints. Full local gate is
`go -C tools tool task ci`.

**Target Platform**: macOS, Linux (incl. WSL), Windows — cross-platform parity is a requirement, not a
courtesy (Principle XII).

**Project Type**: Single Go module, CLI + bundled MCP server.

**Performance Goals**: No regression. Enveloping is a marshalling change; it must not add a
measurable cost to `list_repositories`, documented as "fast — reads registry only".

**Constraints**:
- Breaking output change, permissible only because it lands in the 2.0.0 major.
- CLI and MCP must be enveloped in the same change (FR-018) or the §6.4 parity breaks.
- `apiVersion` from a single shared constant (FR-003) — two constants would permit drift.
- Contract version stays independent of config and repo-metadata `apiVersion` (FR-004).

**Scale/Scope**: ~10 CLI surfaces, 14 MCP tools, one shared envelope type, three contract documents,
plus `DESIGN.md` §6.3/§6.4 revision.

## Constitution Check

*GATE: evaluated against `.specify/memory/constitution.md` v1.0.0.*

| Principle | Assessment |
| --- | --- |
| **III. Deterministic, Reconstructible Operation** — "Output formats are stable contracts" | **Directly served.** This feature is that principle made enforceable rather than aspirational. |
| **V. Compose, Don't Trap** | **Served.** Machine-readable output is the composition surface; a uniform envelope makes it usable without RepoKeeper-specific parsing knowledge per command. |
| **VI. Explainable Reconciliation** | **Served.** FR-020 requires a machine-readable reason on every skip; FR-015 requires read-only refusals to name cause and remedy. |
| **VII. Read-Only Degradation Over Blindness** | **Served.** FR-016 makes "every read surface works under a read-only mount" a tested guarantee rather than incidental behaviour. |
| **IX. Technical Precision, Honest Scope** | **Served.** FR-010 requires naming what is explicitly *not* contractual. Out of Scope declines to imply a daemon mode ADR-0006 merely lists as possible. |
| **XI. Git-First, Multi-VCS Explicitly Scoped** | **Served.** FR-020 requires backend-unsupported skips carry a machine-readable reason rather than being misreported as Git behaviour. |
| **XII. CLI-First With Machine-Readable Output** | **Directly served.** "Every inspection surface MUST offer both human-readable and machine-readable output" — this makes the machine-readable half contractual. |
| **Engineering: tests with behaviour** | Satisfied by design — every FR with a guarantee has a corresponding assertion in `tasks.md`. |
| **Engineering: docs updated** | `DESIGN.md` §6.3/§6.4 revision is in scope and tracked as a task. |
| **Engineering: PR-only, signed + DCO, Conventional branch prefix** | Branch is `feat/004-adapter-contract`. Commits signed and signed-off. |
| **Workflow: cite settled upstream** | ADR-0006, `DESIGN.md` §6.3/§6.4 cited in `research.md` rather than re-derived. |
| **Workflow: hard-to-reverse decisions get an ADR** | **Action required — see below.** |

**Result: PASS.** The one required follow-up is satisfied.

Promoting `v1beta1` → `v1` and taking a deliberate breaking change to the output shape is
hard-to-reverse and therefore warrants an ADR per the constitution's Specification and Decision
Workflow. ADR-0006 set the *policy* and remains immutable; a new ADR records the *mechanism* chosen
under it. Tracked as task T003 and delivered as
[ADR-0018](../../docs/adr/0018-adapter-contract-envelope.md), which cites ADR-0006 as the policy it
implements rather than superseding it.

## Project Structure

### Documentation (this feature)

```text
specs/004-adapter-contract/
├── spec.md                         # Feature specification
├── research.md                     # Prior art + the two decisions weighed
├── data-model.md                   # Envelope, surface descriptor, classifications
├── quickstart.md                   # Adapter author's entry point
├── plan.md                         # This file
├── tasks.md                        # Task breakdown
├── contracts/
│   ├── envelope.md                 # The response envelope
│   ├── surface-inventory.md        # Enumerated adapter-facing surfaces
│   └── cli-mcp-parity.md           # Shared-record parity rule
└── checklists/
    └── requirements.md             # Requirements-quality review
```

### Source Code (repository root)

```text
cmd/repokeeper/
├── contract.go          # NEW: shared envelope type + the single apiVersion constant
├── contract_test.go     # NEW: envelope invariants, empty-collection, drift
├── status.go            # MODIFIED: consume shared constant; v1beta1 -> v1
├── scan.go              # MODIFIED: wrap in envelope
├── sync.go              # MODIFIED: wrap in envelope (reconcile/sync)
├── describe.go          # MODIFIED: wrap in envelope
├── label.go             # MODIFIED: wrap in envelope
├── repair_upstream.go   # MODIFIED: wrap in envelope
└── version.go           # MODIFIED: wrap in envelope

internal/mcpserver/
├── tool_results.go      # MODIFIED: envelope MCP structured results
├── readonly.go          # UNCHANGED behaviour; now contractual
└── contract_test.go     # NEW: CLI/MCP parity + read/mutation assertions

DESIGN.md                # MODIFIED: §6.3/§6.4 generalised to the uniform envelope
docs/adr/0018-*.md       # NEW: ADR recording the envelope mechanism + v1 promotion
```

**Structure Decision**: Single Go module, unchanged. The one structural addition is
`cmd/repokeeper/contract.go` as the single home for the envelope type and the `apiVersion` constant.
Centralising it is what makes FR-003 (one constant, no CLI/MCP drift) mechanically true rather than a
convention reviewers must police.

## Phasing

The user stories are ordered so each is independently shippable, per the spec-template requirement.

| Phase | Story | Delivers |
| --- | --- | --- |
| 1 | Foundational | Shared envelope type + constant. No surface uses it yet. |
| 2 | **US1 (P1)** | Every surface enveloped, CLI + MCP together, `v1` promoted. This alone is a viable MVP: adapters can detect version. |
| 3 | **US2 (P2)** | Published inventory + drift tests. Adapters can build without reading Go. |
| 4 | **US3 (P3)** | Read/mutation classification asserted by test; read-only guarantees made contractual. |
| 5 | Polish | `DESIGN.md`, ADR, release notes, cross-platform verification. |

US1 must precede US2 — the inventory cannot accurately document an envelope that does not yet exist.
US3 is genuinely independent and could ship first if prioritisation changed, since the read/mutation
boundary already exists in code and needs classification and tests rather than new behaviour.

## Complexity Tracking

| Violation / deviation | Why needed | Simpler alternative rejected because |
| --- | --- | --- |
| Deliberate breaking change to output shape | A bare top-level array has nowhere to carry `apiVersion`; uniform detection is impossible without it | Out-of-band discovery (option B in `research.md`) is non-breaking but coarse and leaves per-surface shapes self-undescribing — see §3 there |
| Tightening "add an enum value" to breaking | A consumer switching exhaustively breaks on a new value; §6.3 is currently silent on this | Leaving it non-breaking preserves the existing wording but ships a known adapter-breaking hole in a release that claims contract stability |
| New ADR alongside immutable ADR-0006 | Constitution requires hard-to-reverse decisions get an ADR; ADR-0006 is immutable and set policy, not mechanism | Amending ADR-0006 is forbidden — ADRs are superseded, never rewritten |

## Clarify outcomes folded into this plan (session 2026-09-05)

Four clarifications changed the plan after it was first written. Recorded here so the plan is not
read as predating them.

| Clarification | Effect on this plan |
| --- | --- |
| **"Adapter-facing" = accepts a JSON format flag, plus every MCP tool and resource** | Resolves the first Open Risk below. The surface set is now derivable from the Cobra command tree and the MCP registries, so the drift test (T014) can be a real guarantee rather than a hand-maintained list. T002 moves from "spike both" to "implement the mechanical rule". |
| **URL fields must be credential-redacted (FR-021/FR-022)** | New scope. `urlutil.RedactCredentials` exists but is wired only into `export` (as a hazard detector) and debug-arg logging, so adapter JSON emits `remotes[].url` verbatim from git. Adds T019a–T019c and success criterion SC-007. |
| **MCP resources are in the contract (FR-001a)** | New scope, and a gap in the original spec: it enumerated only the 14 tools. The three resources (`config`, registry snapshot, repo template) are `application/json` and unenveloped. Adds T009a–T009b. The registry snapshot intersects the redaction decision, since it serialises `remote_url`. |
| **Clean break, no legacy output mode** | Bounds scope: no second output path, no compatibility flag, no later removal task. The migration lives in release notes (T021), not in code. |

The two scope additions both enlarge Phase 2 and Phase 4. Neither changes the phasing or the
dependency order.

## Open Risks

- ~~**Enumerating surfaces from code may not be practical.**~~ **Resolved by the clarify session.**
  The blocker was that nothing distinguished an adapter-facing command from an interactive one. The
  agreed rule — a command is adapter-facing exactly when it accepts a JSON format flag — is
  mechanical and readable from the Cobra command tree, and the MCP side already enumerates its tools.
  FR-011 can therefore be a real drift guarantee. The hand-maintained fallback is no longer needed.
- **MCP clients may not tolerate an enveloped structured result.** If a client requires bare content,
  the envelope may need to live in a defined field of the MCP result rather than replacing it.
  Verify against a real client early (task T009) rather than at the end.
- **`version -o json` is arguably not adapter-facing.** It is included in the inventory for
  uniformity, but if enveloping it breaks install tooling that parses it, that is a real
  dependency to discover before shipping, not after.
