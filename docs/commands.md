# RepoKeeper Commands

This is the canonical command reference for RepoKeeper. Keep this file in sync with `--help` output.

## Top-level Commands

| Command | Description |
|---|---|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper status` | Report repo health summary (path, branch, dirty, tracking) |
| `repokeeper get` | Kubectl-style alias for status/list view |
| `repokeeper get repos` | Backward-compatible alias for list view |
| `repokeeper describe <repo-id-or-path>` | Show detailed status for one repository |
| `repokeeper describe repo <repo-id-or-path>` | Kubectl-style describe form |
| `repokeeper add <path> <git-repo-url>` | Clone and register a repository |
| `repokeeper delete <repo-id-or-path>` | Delete repo files and remove from registry |
| `repokeeper edit <repo-id-or-path>` | Open one repo entry in `$VISUAL`/`$EDITOR`, validate, save |
| `repokeeper label <repo-id-or-path>` | Show or mutate labels for one repository |
| `repokeeper repair-upstream` | Repair missing/mismatched upstream tracking |
| `repokeeper repair upstream` | Kubectl-style alias for upstream repair |
| `repokeeper sync` | Fetch and prune all repos safely |
| `repokeeper reconcile` | Kubectl-style alias for sync/reconciliation |
| `repokeeper reconcile repos` | Backward-compatible alias for sync/reconciliation |
| `repokeeper export` | Export config and optional registry for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

## Command Notes

### `repokeeper status` / `repokeeper get`

- Supports `--only`, `--field-selector`, and label selector `-l, --selector`.
- Label selector supports `key` and `key=value`, comma-separated AND.
- Use `-o wide` for additional `PRIMARY_REMOTE`, `UPSTREAM`, `AHEAD`, `BEHIND`, and `ERROR_CLASS`.

### `repokeeper sync` / `repokeeper reconcile`

- Shows a preflight plan before execution.
- Prompts only when mutating actions are planned (rebase/stash/checkout-missing clone), unless `--yes`.
- Supports `--checkout-missing` to clone entries marked missing.

### `repokeeper edit`

- Opens a single entry YAML, not the whole registry file.
- Validates edited data before write:
- `repo_id` required and unique.
- `path` required and absolute.
- `status` required and must be `present`, `missing`, or `moved`.

### `repokeeper label`

- Focused label mutation command without opening an editor.
- `--set key=value` and `--remove key` are repeatable.
- Output: `-o table|json`.

### `repokeeper add`

- Supports `--branch <name>` or `--mirror` (mutually exclusive).
- Supports metadata on create:
- `--label key=value` (repeatable)
- `--annotation key=value` (repeatable)

## Global Flags

- `--verbose` / `-v` increase verbosity (repeatable)
- `--quiet` / `-q` suppress non-essential output
- `--config <path>` override config file location
- `--no-color` disable color output (also respects `NO_COLOR`)
- `--yes` accept mutating actions without interactive confirmation
