# ADR-0014: Local Branch Prune-Safety Classification Model

**Status:** Proposed
**Date:** 2026-07-11
**Author:** Shawn Stratton

## Context

[ADR-0004](0004-prune-workflow-boundaries.md) established that local branch prune is a first-class, non-mechanical workflow that must classify branches before any deletion, using the reusable set `safe_to_prune` / `probably_safe` / `keep` / `needs_review`, each with explicit reason codes. ADR-0004 named the categories but did not pin the per-branch data model, the signal-to-category precedence, or the reason-code contract. This ADR does.

Today RepoKeeper models only the **currently checked-out branch**. `model.RepoStatus` carries a single `Head` and a single `Tracking` (`internal/model/model.go:124-167`). The git layer (`gitx.TrackingStatus`, `internal/gitx/gitx.go:157-218`) already runs `git for-each-ref refs/heads` but filters every non-HEAD branch out and discards it. There is no `model.Branch` type, no per-branch enumeration surfaced anywhere, and **no merged-into-base or branch-recency signal exists at all**. A prune-safety classifier therefore cannot be a thin read over existing state — it needs a new per-branch model, new git plumbing, and a defined decision procedure.

The recently shipped stale-remote-tracking-ref feature (ADR-0004's remote-tracking half; `model.RemoteTrackingRefStatus` + `vcs.RemoteTrackingRefInspector` + `Engine.inspectRemoteTrackingRefs`) gives a proven five-layer pattern to mirror. `matchesProtectedBranch` (`internal/engine/engine.go:1438`) already provides glob matching, and `pullRebaseSkipReason` (`engine.go:1399`) already classifies a single current branch for rebase safety at the altitude we want.

## Decision

RepoKeeper adds a **per-branch prune-safety classifier** built from three parts: a new per-branch model, a pure classification function, and read-only surfacing through the existing inspection channels.

### Per-branch model

Introduce `model.LocalBranch` carrying the raw signals plus the computed verdict, and a `model.LocalBranchStatus` container embedded on `RepoStatus` (mirroring `RemoteTrackingRefStatus`):

```go
type LocalBranch struct {
	Name               string
	IsCurrent          bool
	Protected          bool
	Upstream           string
	UpstreamStatus     TrackingStatus // reuses existing gone/none/ahead/behind/diverged/equal
	Ahead, Behind      *int
	MergedIntoBase     bool
	MergedIntoUpstream bool
	LastCommitAt       *time.Time
	Category           PruneCategory
	Reasons            []PruneReason
}

type LocalBranchStatus struct {
	Branches        []LocalBranch
	InspectionError string // distinguishes "signal unavailable" from "no prunable branches"
}
```

`Category` and `Reasons` are the stable machine-readable contract; consumers (CLI, TUI, prune planning, MCP) must not re-derive safety from raw signals.

### Categories and decision precedence

`PruneCategory` and `PruneReason` are `type ... string` enums following the `ReconcileMode` idiom (`internal/remotemismatch/remotemismatch.go:15-46`) with snake_case values, a `ParsePruneCategory`, and a separate `map[PruneReason]string` hint table (the `internal/engine/hints.go` idiom). The classifier is a pure function `Classify(branch rawSignals, policy Policy) (PruneCategory, []PruneReason)` living in a new dependency-free package (`internal/prune`), re-aliased into `engine` the way `remotemismatch` is.

Classification is **first-match-wins** over an explicitly ordered set of guards, chosen so that no weak signal alone justifies deletion:

1. **`keep`** — the branch must never be a prune candidate: it is checked out (`current_branch`), it is the base branch (`base_branch`), or it matches a protected pattern (`protected_pattern`).
2. **`needs_review`** — safety cannot be established: a required signal is unavailable (`signal_unavailable`); the branch holds local commits that exist nowhere else — no upstream and not merged into base (`unmerged_local_work`); it has diverged from its upstream and is not merged into base (`diverged_unmerged`); or it is unmerged and older than the policy stale window (`stale_unmerged`).
3. **`safe_to_prune`** — integration is corroborated and nothing is lost: merged into base with no unpushed divergence (`merged_into_base`), optionally strengthened by `merged_into_upstream` / `upstream_gone_merged`.
4. **`probably_safe`** — one strong signal without full corroboration: the upstream ref is gone but merge-base cannot confirm integration, the common squash-merge case (`upstream_gone_unmerged`).
5. **`keep` (default)** — an ordinary active, unmerged branch with a live upstream falls through to `keep` (`active_unmerged`), never to a prune category.

The ordering principles are the durable decision: protected/current/base always `keep`; an unresolvable signal is always `needs_review`, never a prune category; a positive integration signal is **required** for any `*_to_prune` verdict; the stale-days policy may only escalate an unmerged branch to `needs_review`, never demote a branch into a prune category; and the conservative fallback is `keep`.

### Surfacing

Classification is read-only. The verdict is produced by classifying enumerated branches (via an optional `vcs.LocalBranchInspector` capability, Git-only, mirroring `RemoteTrackingRefInspector`) and is carried on the `LocalBranch` model, which embeds on `RepoStatus` so any consumer can read it without re-derivation. The first consumers are prune planning (SKA-229) and the TUI branch-hygiene view (SKA-224); wiring classification into the default inspection output and any user-facing rendering is deferred to those consumers. This ADR adds **no** deletion path; prune planning and execution remain [ADR-0004](0004-prune-workflow-boundaries.md)'s separate workflow.

## Consequences

### Positive

- Branch hygiene views, prune plans, and MCP consumers share one classification engine with a stable category + reason-code contract, as ADR-0004 intended.
- The pure `Classify` function is exhaustively table-testable independent of git.
- Conservative precedence makes destructive misclassification structurally hard: prune verdicts require positive integration evidence.

### Negative

- Real new surface area: a per-branch model, new gitx enumeration, and two brand-new git signals (merged-into-base via `git branch --merged`/merge-base, recency via `for-each-ref committerdate`) that do not exist today.
- Enumerating and inspecting every local branch is more work per repo than the current single-HEAD inspection; large repos with many branches pay for it whenever local branches are inspected.
- The reason-code set becomes a compatibility surface: adding or renaming codes is a contract change for JSON consumers, governed by the status JSON schema version.

### Neutral

- `probably_safe` exists specifically to model the squash-merge case honestly rather than forcing it into `safe_to_prune` or `keep`.
- The classifier consumes a `Policy` value defined in the `internal/prune` package; workspace configuration maps into it (see [ADR-0015](0015-branch-retention-and-protection-policy.md)), keeping the classifier free of a config dependency.

## Alternatives Considered

### 1. Classify only the current branch, reusing existing state

**Rejected because:** prune hygiene is fundamentally about the *other* branches — merged topic branches the user forgot. Classifying only HEAD would miss the entire use case.

### 2. Binary merged / not-merged decision instead of four categories

**Rejected because:** ADR-0004 already rejected this. Squash-merges (upstream gone, not merge-base reachable) and unavailable signals are neither cleanly "merged" nor "unmerged"; collapsing them loses the safety distinction and forces either unsafe deletion or uninformative retention.

### 3. Put classification logic in the engine package directly

**Rejected because:** a pure, dependency-free `internal/prune` package (re-aliased into `engine` like `remotemismatch`) is table-testable in isolation and reusable by CLI/TUI/MCP without pulling in engine wiring. The marginal cost is one small package and a set of type aliases.

## Links

- Refines: [ADR-0004: Prune Workflow Boundaries and Safety Model](0004-prune-workflow-boundaries.md)
- Depends on: [ADR-0015: Branch Retention and Protection Policy](0015-branch-retention-and-protection-policy.md)
- Config home: [ADR-0005: Workspace Config vs Repo-Local Metadata Ownership](0005-config-vs-repo-metadata-ownership.md)
- Consumed by future work: SKA-229 (prune planning), SKA-224 (TUI branch hygiene view), SKA-571 (opt-in merged-branch pruning)
- Implements: SKA-226
