# Requirements Quality Checklist: Stable Plugin Adapter Contract

**Purpose**: Requirements-quality review of the adapter contract specification before implementation begins.
**Created**: 2026-09-05
**Feature**: [spec.md](../spec.md)

**Review Ownership**: This checklist is a reviewer-owned requirements-quality review artifact. Mark an item `[x]` only when the reviewer determines the requirements-quality criterion is satisfied.
**Marker Semantics**: `[x]` means the criterion has been reviewed and satisfied for requirements quality. It does not mean implementation work is complete.

## Constitutional alignment

- [x] CHK001 Every requirement is traceable to a constitutional principle or to a stated acceptance criterion of issue #287, with no orphan requirements
- [x] CHK002 No requirement weakens or contradicts ADR-0006, which is accepted and immutable
- [x] CHK003 The spec cites settled prior art rather than re-deriving it, per the Specification and Decision Workflow
- [x] CHK004 Principle IX (honest scope) is satisfied: the spec states plainly what is *not* contractual and does not imply capability that will not ship
- [x] CHK005 The hard-to-reverse decisions (breaking envelope change, `v1beta1` → `v1`) are gated behind an ADR task rather than assumed

## Requirement completeness

- [x] CHK006 Every adapter-facing surface present in the code appears in the inventory — CLI commands, all fourteen MCP tools, **and the three MCP resources** (`config`, registry snapshot, repo template)
- [ ] CHK007 Each functional requirement is independently verifiable, with a stated means of verification
- [x] CHK008 Requirements cover the failure path (FR-019), not only the success path
- [x] CHK009 Requirements cover the empty-collection case (FR-007), which Go's nil-slice marshalling makes a real defect risk
- [x] CHK010 Requirements state the CLI/MCP parity obligation (FR-017, FR-018) rather than leaving it implicit

## Requirement clarity

- [x] CHK011 No requirement contains an unresolved `[NEEDS CLARIFICATION]` marker
- [x] CHK012 "Adapter-facing" is defined precisely enough that a reviewer can decide whether an arbitrary surface qualifies
- [x] CHK013 The distinction between a *skip* (success, `ok: true`, with reason) and a *failure* (non-zero exit) is unambiguous
- [x] CHK014 The read/mutation classification of every surface is stated, not inferable from naming
- [x] CHK015 Each success criterion is measurable and technology-agnostic, expressed as an outcome rather than an implementation

## Consistency

- [x] CHK016 `spec.md`, `data-model.md` and `contracts/` agree on the envelope field names and on the `apiVersion` value
- [x] CHK017 The surface inventory in `contracts/surface-inventory.md` agrees with the classification table in `data-model.md` §4
- [x] CHK018 The evolution rules are stated identically in `data-model.md` §5 and `contracts/envelope.md`
- [x] CHK019 The tightening of "add an enum value" to breaking is flagged as a deviation from the current §6.3 wording, not silently introduced

## Scope boundaries

- [x] CHK020 The boundary against #289 (JSON hardening) is explicit and leaves no work ambiguously owned
- [x] CHK021 The boundary against #286 (compatibility policy) is explicit
- [x] CHK022 The spec does not implement an IDE plugin, consistent with #287's standalone-repo direction
- [x] CHK023 A daemon or local service mode is excluded rather than implied, since ADR-0006 lists it only as a possible future surface

## Assumption risk

- [x] CHK024 The assumption that no external consumer depends on `v1beta1` is stated and gated by a verification task (T001)
- [x] CHK025 The assumption that the drift test generalises from one constant to a full inventory is stated, with a named fallback if it does not (T002)
- [x] CHK026 The assumption that MCP clients tolerate an enveloped result is stated and verified early rather than at integration time (T009)
- [x] CHK027 Each assumption names what happens if it proves false, rather than merely recording it

## Testability

- [x] CHK028 Every guarantee has a corresponding assertion in `tasks.md`; no guarantee relies on review alone
- [x] CHK029 Guarantees that could pass vacuously — a drift test over an empty inventory, a read-only assertion that never invokes a mutation surface — are written so they fail when the mechanism is absent
- [x] CHK030 Cross-platform parity is a tested requirement rather than an assumption (Principle XII)
