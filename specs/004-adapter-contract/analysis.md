# Cross-Artifact Analysis: Stable Plugin Adapter Contract

**Feature**: `004-adapter-contract` | **Run**: 2026-09-05 | **Artifacts**: [spec.md](./spec.md), [plan.md](./plan.md), [tasks.md](./tasks.md)

Consistency pass across the three core artifacts before implementation, per the speckit analyze
workflow. Constitution conflicts are automatically CRITICAL and are resolved by adjusting the
artifacts, never by reinterpreting a principle.

## Findings

| ID | Category | Severity | Location | Summary | Resolution |
| --- | --- | --- | --- | --- | --- |
| A1 | Coverage gap | HIGH | spec.md FR-019 | "Failed invocation does not emit an envelope on stdout" had **zero** task coverage. An adapter's parse path on failure would ship unverified. | Added task **T011a**. |
| A2 | Coverage gap | HIGH | spec.md FR-017 | Shared-record field parity had only partial coverage: T011 asserted CLI and MCP share the `apiVersion` constant, but nothing asserted the *record field sets* match — which is the parity `cli-mcp-parity.md` actually promises and #344 had to restore. | Added task **T011b**. |
| A3 | Coverage gap | MEDIUM | spec.md FR-004 | Independent versioning (contract vs config vs repo-metadata `apiVersion`) was asserted in prose but had no test. Three constants sharing a value today makes accidental coupling easy and silent. | Added task **T007a**. |
| A4 | Underspecification | MEDIUM | spec.md FR-005 | "Additive changes MUST NOT bump `apiVersion`" is a policy humans apply at review time; it is documented (T012) but inherently untestable as written. | Accepted as documentation-only. Recorded here so it is a known limit rather than an assumed guarantee. |
| A5 | Inconsistency | MEDIUM | spec.md FR-014a vs tasks.md T016 | FR-014a requires classifying flag-dependent surfaces at their *default* flags; T016 named only the two MCP cases and did not mention the CLI flag defaults. | T016 amended to name the flag-default cases explicitly. |
| A6 | Ambiguity | LOW | spec.md FR-018 | "Applied in the same change" is an ordering constraint, not an independently verifiable requirement. It is enforced by the Phase 2 note rather than a task. | Accepted. Enforced at review; noted so it is not mistaken for a tested guarantee. |

**Constitution alignment**: no violations. The one hard-to-reverse decision (breaking envelope +
`v1` promotion) is already gated behind an ADR task (T003) per the Specification and Decision
Workflow, and the testing gate is enforced despite the upstream template marking tests optional.

**Duplication**: none found. The evolution rules appear in both `data-model.md` §5 and
`contracts/envelope.md`, which is intentional restatement for two audiences (implementer vs adapter
author); checklist item CHK018 exists to keep them in sync.

## Coverage summary

| Metric | Before | After |
| --- | --- | --- |
| Requirements with ≥1 task | 18 / 21 (86%) | 21 / 21 (100%) |
| Requirements documented-only by design | — | 2 (FR-005, FR-018, recorded above) |
| Critical issues | 0 | 0 |
| High issues | 2 | 0 |

## Verdict

**Ready to implement** after the three added tasks. The gaps found were all in the same direction —
guarantees stated in the spec that no task would have verified — which is the failure mode this pass
exists to catch, and the reason it ran before implementation rather than after.
