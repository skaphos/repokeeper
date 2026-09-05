<!--
SYNC IMPACT REPORT
==================
Version change: (none) → 1.0.0
  Initial ratification of the RepoKeeper constitution, derived from the
  canonical Skaphos Constitution v1.1.0
  (../skaphos-resources/standards/constitution.md).

Principles:
  Inherited (I–IX) verbatim in intent from the upstream Skaphos Constitution.
  Added RepoKeeper-specific principles (X–XII) that sharpen — never weaken —
  the upstream:
    X.   Safe-by-Default VCS Operations
    XI.  Git-First, Multi-VCS Explicitly Scoped
    XII. CLI-First With Machine-Readable Output

Added sections:
  - Engineering Constraints (RepoKeeper overlay on upstream constraints)
  - Specification and Decision Workflow (inherited, references upstream)
  - Governance (records the derivation relationship; upstream authoritative)

Removed sections: none (template placeholders fully replaced).

Templates requiring updates:
  ✅ .specify/templates/plan-template.md   — Constitution Check gate reviewed; generic, compatible
  ✅ .specify/templates/spec-template.md   — reviewed; no principle-mandated sections missing
  ⚠ .specify/templates/tasks-template.md  — generic scaffold marks tests OPTIONAL; the
       constitution's Engineering Constraints require meaningful tests per change. Left
       unedited (upstream Spec Kit scaffold); enforce the testing gate per feature at
       /speckit-tasks / /speckit-plan time.
  ✅ .specify/templates/checklist-template.md — reviewed; no constitution coupling

Follow-up TODOs: none. All placeholders resolved. (tasks-template divergence noted above
  is intentional and handled at planning time, not a deferred edit to this file.)
-->

# RepoKeeper Constitution

The non-negotiable principles every change to RepoKeeper is specified, planned,
and reviewed against. This document is **derived** from the canonical,
organization-level [Skaphos Constitution](../../skaphos-resources/standards/constitution.md)
(v1.1.0), which remains the authoritative upstream. It inherits every upstream
principle and constraint without weakening or contradiction, and adds
RepoKeeper-specific principles where they sharpen the operating model for a
cross-platform multi-repo hygiene CLI.

Where this document and the upstream conflict, the upstream governs and this
document is the bug (see "Governance").

---

## Core Principles

### I. Explicit State Over Implicit Behavior

Operational concepts MUST be first-class declared primitives — durable objects
with lifecycle, status, and history — not tribal knowledge. RepoKeeper's
registry (`.repokeeper.yaml`) and repo-local metadata
(`.repokeeper-repo.yaml` / `repokeeper.yaml`) are the explicit state; behavior
that depends on undocumented assumptions is a defect.

*Rationale: intent that is not explicit cannot be enforced, explained, or
recovered.*

### II. Git Is the Durable Desired-State Boundary

Every byte of RepoKeeper's intended configuration MUST be derivable from a
committed file the operator controls. Registry and repo-local metadata
round-trip through files on disk under version control; invisible mutations and
un-reconstructable state are forbidden. RepoKeeper MUST NOT become an invisible
second source of truth about a repository's health.

*Rationale: recovery stays simple, audit stays grounded, behavior stays
explainable with Git and a CLI.*

### III. Deterministic, Reconstructible Operation

RepoKeeper MUST behave predictably and reconstructibly from declared
configuration. The same inventory and repository state MUST produce the same
report and the same sync plan. Output formats are stable contracts; discovery
and classification are deterministic given identical inputs.

*Rationale: determinism is what makes drift detectable and results
trustworthy.*

### IV. Kubernetes-Native, Never Obscured

Skaphos tools MUST integrate with Kubernetes primitives directly where
appropriate, and MUST NOT obscure Kubernetes behavior when they touch it.
RepoKeeper is a local developer CLI and does not currently interact with
Kubernetes; per Principle IX it states this scope plainly rather than implying
coverage. Should RepoKeeper ever gain control-plane surface area, it MUST honor
this principle in full — clarifying, never hiding, the control-plane model.

*Rationale: hiding API machinery discards the most useful part of the system;
declaring non-applicability honestly is preferable to silent scope creep.*

### V. Compose, Don't Trap

RepoKeeper MUST do one important operational job well — multi-repo hygiene:
inventory, drift/tracking reporting, and safe sync — expose its state clearly
(table/wide/JSON, MCP), and compose with other tools through files, exit codes,
and machine-readable output. It MUST be independently adoptable and provide
concrete value standalone, with no hard dependency on other Skaphos tools.

*Rationale: Skaphos is an ecosystem of focused tools, not a monolith and not a
trap.*

### VI. Explainable Reconciliation, Evidence-Grade Audit

For every reported status and every sync action, RepoKeeper MUST be able to show
the observed state, the decision, the action taken, and the reason. "Failed",
"skipped", or "stale" without a reason and a next safe action is a defect —
this applies to skipped VCS backends, unsupported flows, and stale
remote-tracking refs alike.

*Rationale: a tool that cannot explain its decisions is not trustworthy.*

### VII. Read-Only Degradation Over Blindness

When mutation paths are degraded or disabled, RepoKeeper MUST still inspect and
report inventory, per-repo health, tracking status, ahead/behind, and stale
upstreams. Inspection is always available; mutation is opt-in. Designs MUST fail
toward read-only, never toward blindness.

*Rationale: read-only degradation is a feature; blindness during failure is an
architectural bug.*

### VIII. Topology Is Deployment State

Tools that model delivery, policy, health, or audit MUST treat topology as part
of the data model, not reconstructed from convention. RepoKeeper's analogue is
its inventory model: root directories, per-repo path, VCS backend, remote, and
tracking relationships are encoded state, not inferred ad hoc from the
filesystem at report time.

*Rationale: a tool that cannot model where something lives cannot safely answer
basic operational questions.*

### IX. Technical Precision, Honest Scope

Documentation and specifications MUST describe actual, verified behavior — not
intent or aspiration — and MUST state plainly what RepoKeeper is *not* and its
known limitations (e.g., experimental Mercurial support, no submodule recursion,
no hidden working-tree mutation). Marketing language and exaggerated claims are
forbidden in all repository content.

*Rationale: operational credibility is the product; a tool that overclaims is
worse than a tool that does less.*

### X. Safe-by-Default VCS Operations

RepoKeeper MUST NOT mutate a user's working tree unexpectedly. Sync is
fetch/prune-first by default; any operation that can alter local state
(e.g., `--update-local` rebase/pull flows) MUST be explicit opt-in, gated by
documented conditions, and MUST NOT recurse into submodules implicitly.
Destructive or history-affecting actions require an explicit flag and MUST
report exactly what they will change before doing it. Stale remote-tracking
refs are *reported* during inspection, never pruned as a side effect.

*Rationale: a hygiene tool that surprises the working tree is more dangerous
than the drift it reports. This sharpens Principles III and VII for the specific
risk RepoKeeper carries.*

### XI. Git-First, Multi-VCS Explicitly Scoped

Git is the default and fully supported backend. Additional VCS backends
(currently Mercurial) are **experimental**, opt-in per command (`--vcs`), and
their supported and unsupported flows MUST be stated plainly. Unsupported
operations for a backend MUST be skipped with a reason (Principle VI), never
silently degraded or misreported as Git behavior.

*Rationale: honest scope (Principle IX) applied to backend coverage — users
must know exactly what a non-Git backend does and does not do.*

### XII. CLI-First With Machine-Readable Output

RepoKeeper MUST expose its functionality via a CLI following the text I/O
protocol: results to stdout, errors to stderr, non-zero exit on failure. Every
inspection surface MUST offer both human-readable (`table`/`wide`) and
machine-readable (JSON, MCP) output so RepoKeeper composes cleanly in scripts
and agents. Cross-platform parity (macOS, Windows, Linux incl. WSL) is a
requirement, not a courtesy.

*Rationale: composability (Principle V) and explainability (Principle VI) depend
on stable, parseable output that behaves identically across platforms.*

---

## Engineering Constraints

These bind the *how* for RepoKeeper. They layer on top of the upstream Skaphos
engineering constraints and the referenced standards, which remain normative.

- **Stack**: Go where practical; CLI uses Cobra; configuration is declarative;
  external dependencies are minimized. Go version is pinned in `go.mod`
  (see the `go` directive there).
- **Go engineering**: per the upstream `go-engineering-standard.md`. A
  regression test accompanies every bugfix; race-enabled CI; hard coverage
  gates; generated artifacts are drift-gated. New behavior MUST ship with
  meaningful tests in the same change.
- **Testing**: Ginkgo v2 + Gomega. Prefer small, focused specs; keep fixtures
  in-package where possible. Run the full local gate
  (`go -C tools tool task ci`) before opening a PR.
- **Documentation**: per the upstream `documentation-standard.md`. Behavior
  changes update `README.md`, `DESIGN.md`, and `RELEASE.md` as applicable;
  significant hard-to-reverse decisions get immutable ADRs.
- **Repository governance**: per the upstream `repository-governance.md`.
  **All changes land via pull request; never commit directly to `main`.** Every
  commit MUST be cryptographically signed AND carry a DCO sign-off
  (`git commit -S -s`). Branches use a Conventional Commit type prefix
  (`feat/`, `fix/`, `chore/`, …), never a username or tracker-suggested name.
- **Conventional Commits**: commit and squash-merge messages follow
  Conventional Commits so Release Please can infer version and notes.
- **Attribution**: keep REUSE/SPDX metadata valid; regenerate
  `third_party_licenses/` and review `THIRD_PARTY_NOTICES.md` whenever `go.mod`
  or `go.sum` changes.

---

## Specification and Decision Workflow

- `skaphos-resources` is the canonical upstream for suite-level context and for
  this constitution. Specs MUST cite relevant upstream findings rather than
  re-researching settled questions, and MUST NOT contradict an accepted ADR
  without proposing its supersession.
- Feature work SHOULD follow the spec-driven flow (specify → plan → tasks)
  checked against this constitution.
- **Adopt before build**: where upstream `ECOSYSTEM.md` records mature prior
  art, a plan that builds instead of adopting MUST document why the verdict does
  not apply.
- Decisions that are hard to reverse get an ADR; ADRs are immutable and
  superseded, never rewritten.

---

## Governance

This constitution is **derived** from the Skaphos Constitution, which is the
authoritative upstream. This document MAY add RepoKeeper-specific principles and
constraints, and MUST NOT weaken or contradict anything upstream. When the
upstream changes, this file is re-synced — the same propose-upstream-first,
mirror-second flow the standards use.

**Amendment**: amendments land by pull request against this file, with the
rationale in the PR description. Version semantics for *this* document: MAJOR for
removing or redefining a principle, MINOR for adding a principle or section,
PATCH for clarifications that change no requirement. A re-sync that only mirrors
an upstream change carries the bump type of the underlying upstream change.

**Compliance**: specs and plans are gated against this constitution. A deviation
is either (a) justified in writing in the plan's complexity/deviation tracking,
or (b) a proposed amendment — silent divergence is not an option. The upstream
Skaphos Constitution, `MANIFESTO.md`, `AGENTS.md`, and the standards remain the
richer narrative; if this derivation drifts from them, this document is the bug
and gets fixed first.

**Version**: 1.0.0 | **Ratified**: 2026-07-25 | **Last Amended**: 2026-07-25
