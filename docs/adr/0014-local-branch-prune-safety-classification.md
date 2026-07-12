# ADR-0014: Local Branch Prune-Safety Classification Model

**Status:** Proposed
**Date:** 2026-07-11
**Author:** Shawn Stratton

## Context

[ADR-0004](0004-prune-workflow-boundaries.md) established that local branch prune is a first-class, non-mechanical workflow that must classify branches before any deletion, using `safe_to_prune` / `probably_safe` / `keep` / `needs_review`, each with explicit reason codes. ADR-0004 named the categories but did not pin the per-branch data model, the signal-to-category precedence, or the reason-code contract. This ADR does.

Today RepoKeeper models only the **currently checked-out branch**. `model.RepoStatus` carries a single `Head` and a single `Tracking` (`internal/model/model.go:124-167`). `gitx.TrackingStatus` (`internal/gitx/gitx.go:157`) already runs `git for-each-ref refs/heads` but discards every non-HEAD branch. There is no `model.Branch` type, and **no merged-into-base, patch-equivalence, or branch-recency signal exists at all**.

Two git facts drive the safety design (both verified against git behavior during review):

- `git branch --merged <base>` is pure ancestor-reachability. It gives **zero credit for squash- and rebase-merges** (the GitHub/GitLab default), whose commits get new SHAs. So reachability alone marks most really-merged branches as unmerged.
- "Upstream ref is gone" is the *absence of a remote branch*. It is equally consistent with a squash-merge (work safely in base) and an abandoned branch whose remote was force-deleted **without** merging (work exists nowhere). It is therefore **not** integration evidence.

The recently shipped stale-remote-tracking-ref feature (`model.RemoteTrackingRefStatus` + `vcs.RemoteTrackingRefInspector` + `Engine.inspectRemoteTrackingRefs`) gives a proven layering to mirror.

## Decision

RepoKeeper adds a **per-branch prune-safety classifier** with a new per-branch model, a pure classification function, and read-only surfacing through inspection output.

### Enum placement (no import cycle)

`PruneCategory`, `PruneReason`, and the reason-hint table live in **`internal/model`** — the dependency-free leaf package where `TrackingStatus` already lives (`model.go:36`). The pure classifier lives in a new `internal/prune` package that imports `model` **one-way**, following the `ReconcileMode` idiom (`internal/remotemismatch/remotemismatch.go:15`) with snake_case values, a `ParsePruneCategory`, and a `map[PruneReason]string` hint table exposed via `HintForReason`. `prune` is re-aliased into `engine` the way `remotemismatch` is. (The enums do **not** live in `prune`: `model.LocalBranch` carries `Category`/`Reasons`, so `model → prune` plus `prune → model` would be a cycle.)

### Per-branch model

```go
type LocalBranch struct {
	Name                  string
	IsCurrent             bool
	CheckedOutElsewhere   bool           // checked out in another linked worktree
	Protected             bool
	Upstream              string
	UpstreamStatus        TrackingStatus // reuses gone/none/ahead/behind/diverged/equal
	Ahead, Behind         *int
	MergedIntoBase        *bool          // nil = merge check unavailable (ancestor reachability)
	PatchEquivalentToBase *bool          // nil = check unavailable (git cherry / patch-id)
	LastCommitAt          *time.Time     // committerdate; nil = unavailable
	Category              PruneCategory
	Reasons               []PruneReason
}

type LocalBranchStatus struct {
	Branches        []LocalBranch
	InspectionError string // whole-enumeration failure
}
```

Signals that can fail to compute are **tri-state** (`*bool` / `*time.Time`): `nil` means "unknown," never a silent `false`. `Category`/`Reasons` are the stable machine-readable contract; consumers must not re-derive safety from raw signals.

### Integration evidence

A branch is **integrated** into base when `MergedIntoBase == true` (branch tip is an ancestor of base — reachability) **or** `PatchEquivalentToBase == true` (every branch commit is patch-equivalent to a commit already in base, via `git cherry`). Reachability backs `safe_to_prune`; patch-equivalence exists specifically to recognize squash/rebase merges and backs only `probably_safe`. Base is resolved **per repo** and compared against its remote-tracking form (see [ADR-0015](0015-branch-retention-and-protection-policy.md)) so a stale local base cannot produce false "unmerged."

### Categories and decision precedence

`Classify(branch, policy, now) (PruneCategory, []PruneReason)` is pure and **first-match-wins** over an explicitly ordered guard set. No verdict beyond `keep`/`needs_review` is reachable without positive integration evidence:

1. **`keep`** — never a prune candidate: checked out here (`current_branch`), checked out in another worktree (`checked_out_elsewhere`), the base branch (`base_branch`), or matches a protected pattern (`protected_pattern`). A branch checked out in any worktree cannot be deleted by git, so it is protected structurally.
2. **`needs_review`** — safety cannot be established: both integration signals are unknown (`signal_unavailable`); the branch has no live upstream (`UpstreamStatus ∈ {none, gone}`) and is not integrated (`unmerged_local_work`); it is diverged (`UpstreamStatus == diverged`) and not integrated (`diverged_unmerged`); or it is not integrated and older than the policy stale window (`stale_unmerged`).
3. **`safe_to_prune`** — `MergedIntoBase == true`: the tip is an ancestor of base, so every commit is already in base and nothing is lost (`merged_into_base`).
4. **`probably_safe`** — `PatchEquivalentToBase == true` but not reachability-merged: the squash/rebase-merge case, with positive patch-equivalence evidence (`patch_equivalent_to_base`).
5. **`keep` (default)** — an ordinary active, unmerged branch with a live upstream falls through to `keep` (`active_unmerged`).

Predicates key on the `UpstreamStatus` enum, never on the nilable `Ahead`/`Behind` counts, so classification cannot panic on partial git output. The durable principles: protected/current/base/worktree-held always `keep`; an unknown integration signal is always `needs_review`; **a positive integration signal — reachability or patch-equivalence — is required for any `*_prune` verdict** ("upstream gone" alone never qualifies); stale-days may only escalate an unintegrated branch to `needs_review`; the conservative fallback is `keep`.

**Actionability.** Only `safe_to_prune` — backed by reachability, the strongest evidence — is eligible for non-interactive or batch prune. `probably_safe` carries real patch-equivalence evidence and so is surfaced distinctly from `needs_review`, but patch-equivalence is deliberately treated as **not sufficient to act on alone**: `probably_safe`, like `needs_review`, requires explicit per-branch human confirmation and is never batch-pruned. This bounds any future prune workflow (SKA-229) so that automated deletion touches only reachability-merged branches unless a human individually approves more. The category's job is to route the squash/rebase case to review *with evidence attached*, not to authorize its deletion.

### Surfacing

Classification is computed during inspection by an `Engine.inspectLocalBranches` method that type-asserts an optional `vcs.LocalBranchInspector` capability (Git-only, mirroring `RemoteTrackingRefInspector`), stored on `RepoStatus.LocalBranches`, and surfaced read-only through the status JSON output (a new additive `local_branches` object) and the describe/detail view. Adding the field is additive per DESIGN.md §6.3 and needs no `statusJSONAPIVersion` bump, but the **reason-code vocabulary becomes part of the `v1beta1` status contract** once emitted. This ADR adds **no** deletion path; prune planning and execution remain [ADR-0004](0004-prune-workflow-boundaries.md)'s separate workflow (SKA-229).

## Consequences

### Positive

- Branch hygiene views, prune plans, and MCP consumers share one classification engine with a stable category + reason-code contract.
- The pure `Classify` function is exhaustively table-testable independent of git; the reason-code contract is validated by a live consumer (status output) in the same PR rather than shipped unexercised.
- Conservative precedence makes destructive misclassification structurally hard: every prune verdict requires positive integration evidence, and worktree-held/protected branches are excluded before any prune path.

### Negative

- Real new surface area: a per-branch model, new gitx enumeration (including `%(worktreepath)`), and three brand-new git signals — reachability (`--merged`/`for-each-ref --merged=`), patch-equivalence (`git cherry`), and recency (`committerdate`) — none of which exist today. Patch-equivalence is an extra per-branch git invocation.
- Enumerating and patch-checking every local branch is materially more work than single-HEAD inspection; large repos with many branches pay for it whenever local branches are inspected.
- The reason-code set is now a `v1beta1` JSON-contract surface: renaming or re-semanticizing a code is a breaking change governed by the status schema version.

### Neutral

- `probably_safe` is retained but re-based on positive patch-equivalence rather than "upstream gone," and is treated as review-required evidence rather than prune authorization — honest evidence of *likely* integration, never acted on without per-branch human confirmation. Only reachability-merged (`safe_to_prune`) branches are auto-prune-eligible.
- The classifier consumes a `Policy` value defined in `internal/prune`; workspace configuration maps into it (see [ADR-0015](0015-branch-retention-and-protection-policy.md)), keeping the classifier free of a config dependency.

## Alternatives Considered

### 1. Treat "upstream gone" as `probably_safe` without patch-equivalence

**Rejected because:** a force-deleted-but-unmerged remote branch is indistinguishable from a squash-merge by that signal alone, so it is a data-loss path that violates the "positive integration evidence required" invariant. Patch-equivalence (`git cherry`) is the signal that actually distinguishes them.

### 2. Use `git branch --merged` (reachability) as the sole integration signal

**Rejected because:** it gives zero credit for squash/rebase merges, so `safe_to_prune` would almost never fire in the dominant merge workflows, and squash-merged branches would fall through to `needs_review` forever. Reachability backs `safe_to_prune`; patch-equivalence rescues the squash case into `probably_safe`. Patch-equivalence is deliberately **not** used for `safe_to_prune`, whose stronger guarantee must remain reachability.

### 3. Put the enums in a dependency-free `internal/prune` package

**Rejected because:** `model.LocalBranch` carries `Category`/`Reasons` and the classifier reads `model.TrackingStatus`, so enums-in-`prune` creates a `model ↔ prune` import cycle. `model` is the leaf where such vocabulary already lives (`TrackingStatus`); the classifier logic lives in `prune`.

### 4. Classify only the current branch, reusing existing state

**Rejected because:** prune hygiene is about the *other* branches — merged topic branches the user forgot. Classifying only HEAD misses the entire use case.

## Links

- Refines: [ADR-0004: Prune Workflow Boundaries and Safety Model](0004-prune-workflow-boundaries.md)
- Depends on: [ADR-0015: Branch Retention and Protection Policy](0015-branch-retention-and-protection-policy.md)
- Config home: [ADR-0005: Workspace Config vs Repo-Local Metadata Ownership](0005-config-vs-repo-metadata-ownership.md)
- Consumed by future work: SKA-229 (prune planning), SKA-224 (TUI branch hygiene view), SKA-571 (opt-in merged-branch pruning)
- Implements: SKA-226
