---
name: repokeeper
description: Use RepoKeeper to initialize workspace tracking, discover repositories, inspect health, label registry entries, navigate safely, and run safe update workflows. Use this when a task involves multiple local repositories, registry hygiene, repo status checks, or safe synchronization.
license: MIT
compatibility: opencode
metadata:
  author: skaphos
  source: repokeeper
---

# RepoKeeper

> **Prefer MCP when available for inspection and planning.** If your agent runtime supports the Model Context Protocol, configure the RepoKeeper MCP server instead of using this skill for read-only and side-effect-free preview workflows. Use CLI or TUI for execution and mutation workflows. MCP provides typed tool schemas, structured JSON responses, and automatic tool discovery without parsing CLI output. See [docs/mcp-setup.md](https://github.com/skaphos/repokeeper/blob/main/docs/mcp-setup.md) for setup instructions.

Use RepoKeeper as the first stop for multi-repository work. It tells you what repositories exist, where they live, how healthy they are, and which ones can be updated safely.

## Scope and context

RepoKeeper operates within the current RepoKeeper context, typically defined by the active workspace configuration. Do not assume repositories outside the current RepoKeeper context are visible or relevant unless the user explicitly asks to change scope.

## When not to use

Do not use this skill when:

- the task only concerns a single already-known repository and ordinary git commands are sufficient
- the task only requires reading files in the current repository without repository discovery
- the user explicitly wants raw git behavior rather than RepoKeeper-managed workflows
- the user has not explicitly asked to create or update repo-local metadata files via `index --write`

## When to use

Use this skill when you need to:

- initialize RepoKeeper for a workspace
- discover repositories under one or more roots
- inspect repository health before editing or syncing
- label or annotate repositories in the machine-local registry
- find the right repository before browsing files
- update repositories safely without guessing which ones are behind, dirty, missing, or diverged

## Core rules

1. Prefer RepoKeeper over ad hoc filesystem crawling when the task spans more than one repository.
2. Treat `scan`, `get`, `describe`, `add`, `import`, and the TUI as read-only with respect to repo contents unless the user explicitly requests repo-local metadata writes.
3. Treat `label` and `edit` as machine-local registry changes, not source-controlled repo changes.
4. Treat `index --write` and the TUI metadata editor as explicit repo-local metadata write flows.
5. Always inspect health with `get` before attempting reconcile or update workflows.
6. Prefer preview-first flows: use `--dry-run` where available before executing mutating operations.

## Initialization workflow

When a workspace is not yet managed by RepoKeeper:

```bash
repokeeper init
```

Expected outcome:

- creates `.repokeeper.yaml` in the current directory by default
- sets that directory as the default workspace root
- performs an initial scan

After initialization, use:

```bash
repokeeper get
```

to confirm the registry is populated and the repos are visible.

## Discovery workflow

Use scan whenever repositories may have been added, moved, or removed:

```bash
repokeeper scan
```

For explicit roots:

```bash
repokeeper scan --roots /path/one,/path/two
```

If you only need a current view of tracked repositories and health, use:

```bash
repokeeper get
```

Use JSON when another agent step needs structured output:

```bash
repokeeper get -o json
```

## Labeling workflow

Use labels for machine-local classification that helps routing and filtering:

```bash
repokeeper label <repo-id-or-path> --set team=platform --set role=service
```

Remove labels with:

```bash
repokeeper label <repo-id-or-path> --remove role
```

Filter by label with:

```bash
repokeeper get -l team=platform
```

Important distinction:

- `label` edits the local registry only
- `index --write` creates or updates a repo-root metadata file meant to live in the repository

## Safe navigation workflow

When you need to choose the right repository before opening files:

1. Run `repokeeper get -o json`.
2. Look at:
   - `repo_id`
   - `path`
   - `labels`
   - `repo_metadata`
   - `repo_metadata.entrypoints`
   - `repo_metadata.paths.authoritative`
   - `repo_metadata.paths.low_value`
3. Prefer files under `repo_metadata.paths.authoritative` when present.
4. Avoid `low_value` paths unless the task explicitly needs generated or archival content.
5. Use `repokeeper describe <repo-id-or-path>` before editing when you need a single-repo deep view.

For interactive browsing, use:

```bash
repokeeper tui
```

The TUI detail view surfaces labels, annotations, and repo-local metadata when available.

## Safe update workflow

When asked whether repos are up to date, start with get, not reconcile.

### Check health first

```bash
repokeeper get -o json
```

Read each repo's tracking state from the JSON output:

- `tracking.status == "behind"` means local branch is behind upstream
- `tracking.status == "ahead"` means local commits exist
- `tracking.status == "diverged"` means local and upstream both moved
- `worktree.dirty == true` means local edits are present

### Preview updates safely

To preview fetch plus local update behavior:

```bash
repokeeper reconcile --update-local --dry-run
```

This is the preferred planning step before making changes.

### Execute safe updates

If the user wants updates applied after reviewing the plan:

```bash
repokeeper reconcile --update-local
```

Behavior to understand:

- clean repos behind upstream can be rebased safely
- dirty repos are skipped unless `--rebase-dirty` is explicitly requested
- ahead repos are skipped unless `--push-local` is explicitly requested
- diverged repos are skipped unless `--force` is explicitly requested

### Good agent response pattern

If asked:

> Are my repos up to date?

Use RepoKeeper `get` first and summarize the result in plain language.

If asked:

> Update them for me.

Use this sequence:

1. `repokeeper reconcile --update-local --dry-run`
2. summarize what will be updated and what will be skipped
3. run `repokeeper reconcile --update-local` if the user still wants execution
4. report the outcome clearly, for example:
   - safely updated these repositories
   - skipped these repositories because they were dirty
   - skipped these repositories because they were diverged or ahead

Do not claim a repo was safely updated if RepoKeeper reported it as skipped or failed.

## Repo-local metadata workflow

Use repo-local metadata when source-controlled repository hints are helpful.

Preview a metadata file:

```bash
repokeeper index <repo-id-or-path>
```

Write it only when explicitly intended:

```bash
repokeeper index <repo-id-or-path> --write
```

To bridge machine-local labels into shared metadata intentionally:

```bash
repokeeper index <repo-id-or-path> --promote-local-labels --write
```

For explicit bulk promotion by selectors:

```bash
repokeeper index repos --local-selector team=platform --promote-local-labels --write
```

Existing shared metadata labels win on key conflicts; promoted local labels only fill missing keys.

Use `--force` only when you intentionally want to replace or reconcile an existing repo metadata file.

## Avoid these mistakes

- do not use RepoKeeper labels as if they are committed to the repository
- do not assume `get` updates repositories; it only reports health
- do not skip the preview step before mutating reconcile flows
- do not treat dirty repos as safe to update unless the user explicitly accepts `--rebase-dirty`
- do not assume `tracking.status=behind` is currently selectable with `--field-selector`; inspect JSON output from `get` instead

## MCP tool equivalents

If the RepoKeeper MCP server is configured, prefer these tools over CLI commands for inspection and planning workflows:

| CLI workflow | MCP tool |
|---|---|
| `repokeeper get -o json` | `list_repositories` (fast) or `build_workspace_inventory` (live) |
| `repokeeper describe <repo> -o json` | `get_repository_context` |
| `repokeeper reconcile --dry-run` | `plan_sync` |

Use CLI or TUI for execution workflows such as sync execution, label mutation, add/remove flows, metadata writes, prune execution, or branch switching.

MCP tools return structured JSON directly for inspection and side-effect-free planning workflows.
