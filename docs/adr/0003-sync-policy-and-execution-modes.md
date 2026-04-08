# ADR-0003: Sync Policy and Execution Modes

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper uses `reconcile` as its sync workflow. Historically, the core definition of sync has been safe remote refresh:

- fetch
- prune stale remote-tracking refs
- avoid touching the working tree by default

RepoKeeper also supports a stronger local update path through `--update-local`, with additional optional behavior such as push and dirty-worktree handling under explicit flags.

The product direction now needs a clearer contract for what sync means, how configuration influences it, and how the plan -> confirm -> execute model applies when local updates are part of the workflow.

## Decision

RepoKeeper defines sync as a repository maintenance workflow with two policy modes:

1. **`remote_only`**
   - fetch
   - remote-tracking prune
   - no local branch mutation

2. **`update_local`**
   - includes `remote_only`
   - may also apply local branch update actions such as `pull --rebase` when policy and repo state allow

The default sync policy may be configured in `.repokeeper.yaml`, but configuration changes the **default planned behavior**, not the confirmation model.

RepoKeeper must still:

- preview the plan before execution
- explain why each repo will be updated, skipped, or blocked
- require explicit confirmation or an explicit non-interactive execute path for mutation

## Policy Model

### `remote_only`

Use when the operator wants safe remote refresh without local branch mutation.

Expected outcomes include:

- fetched
- pruned remote-tracking refs
- skipped due to repo state or unsupported backend conditions

### `update_local`

Use when the operator explicitly wants sync planning to include local branch updates.

Local update remains conditional. A repo is not eligible just because the policy mode is `update_local`.

The planner must still consider:

- dirty worktree state
- detached HEAD
- upstream presence
- ahead / behind / diverged state
- protected-branch rules
- backend support (for example, non-Git adapters may skip local update)

## Configuration Boundary

Sync policy belongs in `.repokeeper.yaml`, not `.repokeeper-repo.yaml`.

Reason:

- sync behavior is an operator/workspace policy decision
- it affects machine-local execution behavior
- it is not source-controlled repository metadata

Future per-repo overrides are allowed, but only if they remain clearly policy-oriented and do not blur the distinction between machine-local execution policy and repo-local source-controlled metadata.

## Execution Model

Sync follows:

- plan first
- confirm next
- execute last

Config may make `update_local` the default sync mode for planning, but config must not create silent execution.

That means:

- config may cause the default plan to include local update actions
- config must not bypass confirmation requirements for mutation
- `--yes` or equivalent explicit execution controls still govern non-interactive mutation

## Interface Boundaries

### CLI and TUI

CLI and TUI own sync execution.

They may:

- show the sync plan
- collect confirmation
- execute remote-only or update-local sync
- preserve structured outcomes for later inspection

### MCP

MCP primarily exposes sync planning.

The current shipped server also includes an explicit `execute_sync` mutation tool. Any MCP sync execution must remain opt-in and safety-gated rather than implicit.

## Consequences

### Positive

- Sync has a stable meaning even when stronger local update behavior is configured.
- Workspace config can express operator intent without weakening the safety model.
- CLI, TUI, and MCP share a cleaner separation of responsibilities.

### Negative

- Users may initially assume config-driven defaults imply implicit execution.
- The planner and result model need to stay explicit as the sync surface grows.

## Alternatives Considered

### 1. Keep sync permanently fetch/prune only

**Rejected because:** RepoKeeper already has a stronger local update path, and the product needs a coherent way to describe and configure it.

### 2. Treat configured `update_local` as silent execution permission

**Rejected because:** it breaks the plan -> confirm -> execute model and weakens mutation safety.

### 3. Store sync policy in repo-local metadata

**Rejected because:** sync policy is a machine-local execution concern, not source-controlled repository metadata.
