# ADR-0002: Branch Switch and Checkout Workflow Boundaries

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper currently treats sync as a safe repository maintenance workflow centered on inspection, fetch, prune, and optional local update under explicit conditions.

RepoKeeper safety language has also historically treated checkout as out of scope by default. That boundary is no longer sufficient because branch switching is becoming a first-class operator workflow, especially in the TUI and in local shell-driven flows.

Branch switching is a different mutation class than sync or prune. It introduces its own risks:

- dirty worktree handling
- detached HEAD handling
- local-vs-remote target resolution
- branch creation and upstream tracking
- ambiguous branch names across remotes
- blocked or partial results across selected repositories

Without an explicit ADR, checkout behavior is likely to drift into sync or be implemented inconsistently across CLI and TUI.

## Decision

RepoKeeper will define branch switching / checkout as a **separate workflow** from sync and prune.

The branch-switch workflow follows the same high-level safety model as other mutation workflows:

- plan first
- explain blockers and proposed actions
- require explicit confirmation before execution
- report structured outcomes after execution

Branch-switch execution belongs to **CLI and TUI only**. MCP may expose branch-switch planning in the future, but it must not execute branch changes.

## Workflow Boundary

### Sync

Sync is for repository refresh and reconciliation work.

It may include operations such as:

- fetch
- remote-tracking prune
- optional local update when explicitly enabled by policy or command mode

Sync does **not** own general branch navigation.

### Prune

Prune is for branch hygiene and cleanup planning/execution.

It is distinct from sync and distinct from branch switching.

### Branch switch / checkout

Branch switch is the workflow for changing the current branch of a repository checkout.

It is not folded into sync, and it is not implied by prune.

## Initial Scope

The first-class design target is **single-repo branch switching**.

RepoKeeper may later add explicit batch workflows, especially for deliberate operations such as returning multiple selected repositories to their default branch when they are safe to switch.

What is intentionally not in the initial contract:

- automatic multi-repo switching based on identically named branches
- silent branch changes as a side effect of sync
- implicit cross-repo branch coordination

## Planning Model

Branch switching requires a side-effect-free plan before execution.

The planner should classify at least:

- switch-ready
- blocked-dirty
- blocked-detached
- blocked-missing-target
- blocked-untracked-conflict
- needs-create-tracking
- needs-review

Every non-trivial classification should include explicit reason codes and operator-readable rationale.

## Target Resolution Policy

Branch target resolution must be deterministic.

The workflow should distinguish cases such as:

- local branch exists
- only a remote-tracking branch exists
- multiple remotes expose the same branch name
- no matching branch exists

Default resolution rule:

- if the operator explicitly asked to switch to a named branch
- and no local branch exists
- and exactly one matching remote-tracking branch exists

then RepoKeeper may plan creation of a local tracking branch from that remote branch, mirroring normal Git expectations.

If target resolution is ambiguous, RepoKeeper must not guess. It should return an explicit error or `needs-review` style classification.

## Safety Defaults

By default, branch-switch execution should be conservative.

Expected default behavior:

- do not switch when the worktree is dirty unless an explicit policy or option allows it
- do not silently stash to make a switch succeed
- do not guess among multiple candidate remotes
- do not hide blocked repos behind partial success messaging

Future batch workflows such as "return selected clean repos to their default branch" are allowed, but they must still use plan -> confirm -> execute and preserve per-repo explanations.

## Interface Boundaries

### TUI

The TUI is an appropriate execution surface for branch switching because it can:

- present candidate branches
- show a switch plan
- request confirmation
- retain post-run results for inspection

### CLI

The CLI is also an appropriate execution surface for branch switching.

It may support:

- explicit branch arguments
- interactive local selection workflows
- structured output for plans and results

### MCP

MCP must not execute branch switching.

MCP may expose future read-only context such as:

- current branch
- candidate targets
- blockers
- branch-switch plan output

But execution remains outside MCP.

## Consequences

### Positive

- Sync remains conceptually clean.
- Branch switching gets its own safety model instead of becoming hidden command behavior.
- CLI and TUI can share a common planner and result model.
- Future batch workflows can be added without redefining sync.

### Negative

- RepoKeeper grows another explicit workflow area to document and test.
- Some users may initially expect branch switching to be part of sync.
- Planning and classification logic must be implemented before execution UX can be shipped safely.

## Alternatives Considered

### 1. Fold checkout into sync

Allow sync to switch branches as part of repository reconciliation.

**Rejected because:** sync and branch navigation solve different problems and have different operator expectations and risk profiles.

### 2. Keep checkout permanently out of scope

Continue treating branch switching as something users should always do outside RepoKeeper.

**Rejected because:** branch navigation is a real operator workflow, especially in the TUI, and RepoKeeper is better positioned to make it explainable and safe than ad hoc shell loops.

### 3. Allow branch switching through MCP

Expose checkout execution directly to agent runtimes.

**Rejected because:** branch switching is an execution workflow with meaningful mutation risk. It belongs to CLI/TUI, not the MCP information-and-planning surface.
