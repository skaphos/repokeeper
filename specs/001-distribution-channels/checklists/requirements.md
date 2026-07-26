# Specification Quality Checklist: Distribution Channel Conformance

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-07-26
**Feature**: [spec.md](../spec.md)

## Content Quality

> Two of the template's generic items are **reworded** rather than ticked as written, because ticking
> them would have been false. See Notes for why the standard phrasing does not fit a feature whose
> subject matter *is* distribution artifacts.

- [x] No incidental implementation detail — every artifact and tool named is the feature's own
      subject matter, not a design choice leaking in *(reworded from "No implementation details
      (languages, frameworks, APIs)", which this spec does not satisfy as written)*
- [x] Focused on user value and business needs
- [x] Written for the audience that must act on it — maintainers and release engineers — and
      readable by a non-specialist stakeholder at the Problem, Success Criteria and Out of Scope
      level *(reworded from "Written for non-technical stakeholders", which is not true of the
      Requirements or Prior Art sections)*
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

Two clarifications were resolved in session 2026-07-26 and are recorded in the spec's
Clarifications section; both originated as `[NEEDS CLARIFICATION]` markers and neither remains.

**Revalidated after `/speckit-plan` (2026-07-26).** Phase 0 research produced three changes to the
spec, all of which keep every checklist item passing:

1. **A third clarification** — how a containerized server selects its workspace root — was raised and
   resolved during planning rather than specification. RepoKeeper's registry discovery walks upward
   from the working directory, which lets several purpose-specific roots self-select by position; a
   container cannot reproduce that, so it serves one explicitly named root per configured entry.
   Added as FR-026 – FR-029, a Key Entity, three edge cases, and SC-006.
2. **FR-037 corrected.** It previously asserted version identity was "the one change to compiled
   code". The read-only-mount refusal (FR-025) is a second. Both are stdlib-only, so the normative
   requirement — no new dependencies — was never affected; the parenthetical was inaccurate.
3. **Requirements renumbered** to FR-001 – FR-037 and SC-001 – SC-012. All cross-references were
   re-audited mechanically; none dangle.

Two Phase 0 findings are worth carrying into review because they have no counterpart in the sting
implementation this spec adopts from: `internal/gitx` shells out to the `git` binary (so the
container cannot use a distroless base), and a bind-mounted workspace triggers git's
dubious-ownership refusal on every call unless `safe.directory` is set. The second was measured, not
assumed — see `research.md` R3.

**On the two reworded Content Quality items.** Raised in review (Copilot, PR #308): the spec names
`cmd/repokeeper/version.go:13`, `.goreleaser.yaml` flags, `go.mod`/`go.sum` and `sigstore-go`
package names, so ticking "No implementation details" and "Written for non-technical stakeholders"
as written was not honest. Both items are reworded above to state the bar this spec actually meets,
rather than left ticked against a bar it does not. The reasoning follows.

This specification names more concrete artifacts than a typical feature spec — `.deb`/`.rpm`, a container image, `server.json`, the `io.skaphos` namespace,
a DNS TXT record. These are not implementation choices leaking into the spec; they *are* the
user-facing subject matter. The feature is "which distribution channels does RepoKeeper ship on",
and a channel is identified by name in the upstream standard being adopted. Naming them is
equivalent to naming a screen in a UI spec. The spec consistently avoids stating *how* they are
produced — no GoReleaser block names, no workflow YAML, no Go package or symbol names — which is
the boundary that matters.

Two exceptions are deliberate and justified:

1. **`cmd/repokeeper/version.go:13`** is cited in the Problem statement. A file-and-line citation
   is implementation detail, but the problem being described is precisely that a specific default
   value ships today. Removing the citation would make the claim unverifiable.
2. **The Prior Art measurement** names `sigstore-go` packages and reports binary sizes and module
   counts. This is evidence for a scope decision, not a design instruction. It is confined to a
   clearly labelled subsection, and the requirement it supports (FR-007) is stated without
   reference to any library.

**On testability of the deviation requirements.** FR-007 is a MUST NOT, which is verified by
absence — no `update` command in the command surface, and `go.mod`/`go.sum` unchanged (SC-010).
FR-008 and FR-009 are verified by the existence and content of `docs/adr/0016-*` and the
`README.md` upgrade table. All three are checkable without ambiguity.

**Scope note carried into planning.** Three items are named in the spec but belong elsewhere and
must not be pulled into this feature's task list: authoring `DECISIONS/0002` in
`skaphos-resources`, promoting the `MACOS_*` secrets to org scope, and Windows Authenticode
signing with the `winget`/`scoop` channels that depend on it.
