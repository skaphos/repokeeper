# ADR-0015: Branch Retention and Protection Policy

**Status:** Proposed
**Date:** 2026-07-11
**Author:** Shawn Stratton

## Context

[ADR-0014](0014-local-branch-prune-safety-classification.md) defines a prune-safety classifier whose verdicts depend on which branches are protected, what the base branch is (so reachability and patch-equivalence can be computed), and how old an unintegrated branch may get before it is worth surfacing. Those inputs are policy, and ADR-0014 keeps the classifier free of any configuration dependency: it consumes a `Policy` value that something else must supply.

Today RepoKeeper has no branch-retention policy. The only related knob is the CLI `--protected-branches` flag (`cmd/repokeeper/sync.go:229`), a per-invocation CSV of globs threaded into `SyncOptions.ProtectedBranches` and matched by `matchesProtectedBranch` (`internal/engine/engine.go:1438`). Two facts about it matter (both verified in review):

- It is a **rebase**-protection knob: it gates auto-rebase during `--update-local` (`engine.go:1411`), nothing else.
- It already ships a per-invocation **narrowing** escape hatch, `--allow-protected-rebase` (`engine.go:1411`, `sync.go:230`).

`config.Config` (`internal/config/config.go:40`) has no branch policy of any kind. Crucially, **per-repo default-branch data already exists** — `registry.Entry.Branch` (`internal/registry/registry.go:37`) — and the resolution pattern is already written: `repairResolveTargetBranch` (`internal/engine/repair.go:125-142`) prefers `entry.Branch`, then the upstream-derived branch, then `Defaults.MainBranch`. `Load` also backfills persisted zero-values as "unset" (`config.go:261-273`), an idiom that interacts badly with boolean defaults.

[ADR-0005](0005-config-vs-repo-metadata-ownership.md) places machine-local execution and safety policy — explicitly "future machine-local prune / safety policy" — in `.repokeeper.yaml`.

## Decision

RepoKeeper adds a **branch retention and protection policy** as a new nested struct on `config.Config`, sourced from `.repokeeper.yaml`, with conservative defaults and **per-repo base resolution**.

### Schema

```yaml
branch_policy:
  protected_patterns: ["main", "master", "release/*"] # PRUNE protection; path.Match globs
  base_branch: ""      # optional GLOBAL override; empty => per-repo auto-resolution
  stale_days: 0        # 0 = disabled; N>0 escalates unintegrated branches older than N days to needs_review
  require_merged: true # forbid safe_to_prune without reachability evidence
```

### Per-repo base resolution

`base_branch` is **not** a workspace-global default. When empty, base is resolved **per repo**, mirroring `repairResolveTargetBranch`: `registry.Entry.Branch` → upstream-derived branch → `Defaults.MainBranch`. A non-empty `base_branch` is only a last-resort explicit global override. The classifier computes reachability and patch-equivalence against the **remote-tracking form** of the resolved base (e.g. `origin/main`), never the possibly-stale local base, and if the base ref cannot be resolved for a repo the branches classify as `needs_review` with a specific "base unresolved" reason — never as prunable. This prevents the systematic misclassification a single global `main` would cause across a workspace mixing `main`/`master`/`develop`.

### Protection is prune-scoped and independent of rebase protection

`branch_policy.protected_patterns` governs **prune** protection only. It is a **separate set** from the existing `--protected-branches` rebase knob; adding a pattern here does **not** change rebase behavior, and the two are reconciled only if a future ADR unifies them. Prune protection gets its own per-invocation narrowing escape hatch, `--allow-protected-prune`, mirroring the existing `--allow-protected-rebase` so the established "narrow for one run" UX is preserved rather than regressed. Protected patterns reuse `matchesProtectedBranch`'s `path.Match` semantics.

### Validation fails closed

RepoKeeper today has only GVK validation. This adds semantic validation at load, and **protection fails closed**: a config that cannot be validated is rejected rather than silently degraded. Specifically — reject a negative `stale_days`; reject any `protected_patterns` glob that does not compile under `path.Match` (rather than silently matching nothing, which would unprotect a branch); reject a `base_branch` override that is a glob or otherwise cannot be a ref; and reject an empty/`*`-only `protected_patterns` unless explicitly intended, since either disables or over-applies protection.

### Defaults must survive the zero-value backfill idiom

`require_merged` defaults to `true` and `protected_patterns` to a non-empty list. Because Go's zero values are `false` and `nil`, and `Load` treats persisted zero-values as "unset," defaulting **must** be done by seeding `DefaultConfig()` and letting YAML overwrite — **not** by a post-unmarshal backfill — or an explicit `require_merged: false` / `protected_patterns: []` would be silently reverted. This constraint is called out because the existing `Load` idiom (`config.go:261-273`) pushes implementers toward the broken version.

### Boundary

Policy affects **classification and planning only**, never execution ([ADR-0003](0003-sync-policy-and-execution-modes.md), [ADR-0004](0004-prune-workflow-boundaries.md)). Configuring a policy changes which branches are proposed and how, but never deletes a branch without the separate plan → confirm → execute path — and per ADR-0014, only `safe_to_prune` is auto-prune-eligible regardless of policy.

## Consequences

### Positive

- Base resolution is correct for heterogeneous workspaces, reusing per-repo data (`Entry.Branch`) and the existing resolution pattern instead of a global literal.
- Prune protection is persistent and reviewable, keeps its own escape hatch, and cannot silently alter rebase behavior.
- Fail-closed validation means a malformed protection pattern is an error, not a silent unprotection of a branch the operator believes is safe.

### Negative

- Two protected-branch sets now exist (prune vs rebase). They are intentionally independent, which is more surface to document and a future unification candidate.
- Semantic validation and the seed-not-backfill defaulting rule are net-new and must be implemented carefully against an existing idiom that pushes the other way.
- Per-repo base resolution adds a resolution step and a dependency on `Entry.Branch` being populated; repos with no registry branch and no upstream fall back to `Defaults.MainBranch`, which can still be wrong for an unusual trunk name (surfaced as `needs_review`, not a bad prune).

### Neutral

- `stale_days: 0` meaning "disabled" overloads the literal "0 days," consistent with the existing `RegistryStaleDays` precedent; documented rather than changed.
- `protected_patterns` stays machine-local per ADR-0005; a repo-contributed, source-controlled protected *floor* (unioned in) is a plausible future refinement but deliberately out of scope here.

## Alternatives Considered

### 1. Workspace-global `base_branch` (empty ⇒ `Defaults.MainBranch`)

**Rejected because:** RepoKeeper manages many repos and real workspaces mix default branches. A global base computes reachability against the wrong base for every off-default repo — inert at best, and destructive if a stale `main` ref exists in a `master` repo. Per-repo data already exists to do this correctly.

### 2. Reuse `--protected-branches` for prune and union it in

**Rejected because:** that flag is rebase-scoped and already has a narrowing counterpart. A union-only reuse both couples prune protection to rebase behavior and contradicts the existing `--allow-protected-rebase` UX. Prune gets its own set and its own `--allow-protected-prune`.

### 3. Store branch policy in `.repokeeper-repo.yaml`

**Rejected because:** ADR-0005 places execution/safety policy in machine-local config. A future source-controlled protected *floor* may union in, but base/stale/require-merged are operator-local decisions about a local checkout.

### 4. Define the policy type inside the classifier and have config embed it

**Rejected because:** it couples `internal/config` to `internal/prune`. The classifier owns a minimal `Policy` input type and the engine maps config into it (per ADR-0014's enum/layering decision), keeping both packages independently testable.

## Links

- Required by: [ADR-0014: Local Branch Prune-Safety Classification Model](0014-local-branch-prune-safety-classification.md)
- Config ownership: [ADR-0005: Workspace Config vs Repo-Local Metadata Ownership](0005-config-vs-repo-metadata-ownership.md)
- Execution/confirmation model: [ADR-0003](0003-sync-policy-and-execution-modes.md), [ADR-0004](0004-prune-workflow-boundaries.md)
- Reuses: `repairResolveTargetBranch` (`internal/engine/repair.go:125-142`), `matchesProtectedBranch` (`internal/engine/engine.go:1438`)
- Implements: SKA-222
