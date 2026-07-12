# ADR-0015: Branch Retention and Protection Policy

**Status:** Proposed
**Date:** 2026-07-11
**Author:** Shawn Stratton

## Context

[ADR-0014](0014-local-branch-prune-safety-classification.md) defines a prune-safety classifier whose verdicts depend on which branches are protected, what counts as the base branch, and how old an unmerged branch may get before it is worth surfacing. Those inputs are policy, and ADR-0014 deliberately keeps the classifier free of any configuration dependency: it consumes a `Policy` value that something else must supply.

Today RepoKeeper has no branch-retention policy of any kind. The only related knob is the CLI-only `--protected-branches` flag (`cmd/repokeeper/sync.go:61`, `kubectl_aliases.go:81`), a per-invocation CSV of globs threaded into `SyncOptions.ProtectedBranches` and matched by `matchesProtectedBranch` (`internal/engine/engine.go:1438`). There is **no** `protected_branches` field in `config.Config`; the persistent baseline is empty. The only branch-ish default is `Defaults.MainBranch: "main"` (`internal/config/config.go:60`).

[ADR-0003](0003-sync-policy-and-execution-modes.md) and [ADR-0005](0005-config-vs-repo-metadata-ownership.md) already decided that machine-local execution and safety policy — explicitly including "future machine-local sync / prune / safety policy" — belongs in `.repokeeper.yaml`, not in source-controlled `.repokeeper-repo.yaml`. This ADR decides the concrete shape of that policy and how it reaches the classifier.

## Decision

RepoKeeper adds a **branch retention and protection policy** as a new nested struct on `config.Config`, sourced from `.repokeeper.yaml`, with conservative defaults.

### Schema

```yaml
branch_policy:
  protected_patterns: ["main", "master", "release/*"] # globs, path.Match semantics
  base_branch: ""      # merge-into-base reference; empty => defaults.main_branch
  stale_days: 0        # 0 = disabled; N>0 escalates unmerged branches older than N days to needs_review
  require_merged: true # forbid a safe_to_prune verdict without positive merge evidence
```

- `protected_patterns` reuses `matchesProtectedBranch`'s `path.Match` glob semantics verbatim.
- `base_branch` defaults to `Defaults.MainBranch` when empty, so existing configs need no edit.
- Defaults are conservative: `require_merged: true`, `stale_days: 0` (staleness escalation off until opted in), and `protected_patterns` seeded with the base branch and common protected names.

The engine maps `config.BranchPolicy` into the classifier's `prune.Policy` input (defined in the `internal/prune` package per ADR-0014), keeping the classifier config-free and avoiding a config→prune import.

### Relationship to the CLI flag

`branch_policy.protected_patterns` becomes the persistent protection baseline. When `--protected-branches` is also supplied, the effective protected set is the **union** of config patterns and flag patterns for that invocation. A flag can widen protection but cannot narrow what config protects.

### Boundary

Policy affects **classification and planning only**, never execution, echoing [ADR-0003](0003-sync-policy-and-execution-modes.md) and [ADR-0004](0004-prune-workflow-boundaries.md): configuring a policy changes which branches are proposed and how, but never causes a branch to be deleted without the separate plan → confirm → execute path.

## Consequences

### Positive

- The classifier's protection, base, and staleness inputs are configurable per workspace rather than hard-coded, satisfying SKA-222's "policy-driven, not hard-coded" requirement.
- Protected-branch intent becomes persistent and reviewable in `.repokeeper.yaml` instead of living only in shell history and aliases.
- Conservative defaults mean an un-migrated workspace behaves safely: nothing is newly proposed for deletion by default.

### Negative

- Two sources of protection (config baseline + CLI flag) exist. The union rule is safe but means a flag cannot un-protect a config-protected branch; narrowing requires editing config.
- A new config surface must be validated and documented; RepoKeeper currently has only GVK validation, so semantic validation (e.g. rejecting a negative `stale_days`) is net-new.
- Per-repo policy overrides are intentionally out of scope here, so workspaces needing divergent per-repo rules are not yet served.

### Neutral

- `stale_days: 0` defaulting to "disabled" means the recency signal is computed but does not affect verdicts until a workspace opts in.
- `base_branch` is workspace-global; multi-base repos (e.g. maintenance branches) are a future refinement, not this decision.

## Alternatives Considered

### 1. Keep protected branches CLI-only

**Rejected because:** SKA-222 requires policy-driven classification, and a per-invocation flag cannot express a persistent, reviewable protection baseline. It also leaves the classifier with no principled default protected set.

### 2. Store branch policy in `.repokeeper-repo.yaml`

**Rejected because:** ADR-0005 places execution and safety policy in machine-local workspace config. Prune safety is an operator decision about a local checkout, not portable source-controlled repository metadata.

### 3. Make the CLI flag override (replace) config rather than union

**Rejected because:** replacement lets a narrow flag silently unprotect branches the workspace deliberately protected. Union fails safe — the direction that matters for a destructive workflow.

### 4. Define the policy type inside the classifier and have config embed it

**Rejected because:** it couples `internal/config` to `internal/prune`. Instead the classifier owns a minimal `Policy` input type and the engine maps config into it, keeping both packages independently testable.

## Links

- Required by: [ADR-0014: Local Branch Prune-Safety Classification Model](0014-local-branch-prune-safety-classification.md)
- Config ownership: [ADR-0005: Workspace Config vs Repo-Local Metadata Ownership](0005-config-vs-repo-metadata-ownership.md)
- Execution/confirmation model: [ADR-0003: Sync Policy and Execution Modes](0003-sync-policy-and-execution-modes.md), [ADR-0004: Prune Workflow Boundaries and Safety Model](0004-prune-workflow-boundaries.md)
- Implements: SKA-222
