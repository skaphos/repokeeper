# RepoKeeper Commands

This is the canonical command reference for RepoKeeper. Keep this file in sync with `--help` output.

## Top-level Commands

| Command | Description |
|---|---|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper get` | Report repo health summary (path, branch, dirty, tracking) |
| `repokeeper get repos` | Explicit resource form for repo health |
| `repokeeper describe <selector>` | Show detailed status for one repository |
| `repokeeper describe repo <selector>` | Kubectl-style describe form |
| `repokeeper index <selector>` | Interactively preview or write repo-local metadata |
| `repokeeper index repos` | Preview or write repo-local metadata for selected repositories |
| `repokeeper install` | Register RepoKeeper as an MCP server in detected (or --claude/--codex/--opencode) runtimes |
| `repokeeper install list` | Show per-runtime MCP registration state (table or `--json`) |
| `repokeeper uninstall` | Remove the RepoKeeper MCP entry from each runtime (prompts unless `--yes`) |
| `repokeeper add <path> <git-repo-url>` | Clone and register a repository |
| `repokeeper delete <selector>` | Delete repo files and remove from registry |
| `repokeeper move <selector> <new-path>` | Move a tracked checkout and update its registry path |
| `repokeeper edit <selector>` | Open one repo entry in `$VISUAL`/`$EDITOR`, validate, save |
| `repokeeper label <selector>` | Show or mutate labels for one repository |
| `repokeeper repair upstream` | Repair missing/mismatched upstream tracking |
| `repokeeper reconcile` | Fetch and prune all repos safely |
| `repokeeper reconcile repos` | Explicit resource form for sync/reconciliation |
| `repokeeper export` | Export config and optional registry for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

## Repository selectors

`describe`, `label`, `edit`, `delete`, `move`, and `index` share these selectors:

- Absolute checkout path (normalized, with symlinks resolved when available).
- Bare `checkout_id`; IDs shared by multiple repositories are ambiguous.
- `repo_id`, when only one checkout matches, or explicit `repo_id@checkout_id`.
- Relative path from the current directory or config root.

Absolute paths take precedence over IDs, and checkout IDs take precedence over repo IDs. Ambiguity errors list qualified IDs and absolute paths for the matching checkouts. Missing checkout paths can still be selected by their stored path.

## Command Notes

### `repokeeper get`

- Supports `--only`, `--field-selector`, and label selector `-l, --selector`.
- Label selector supports `key` and `key=value`, comma-separated AND.
- Repository errors exit with status `2`, name each affected path and error on stderr, and add an error count to the completion summary. The default table adds `ERROR_CLASS` when any displayed repository has an error. Filters apply to diagnostics and counts.
- `--quiet` retains error diagnostics but suppresses the summary. JSON and custom-column stdout remain machine-readable.
- Use `-o wide` for additional `PRIMARY_REMOTE`, `UPSTREAM`, `AHEAD`, `BEHIND`, and `ERROR_CLASS`.
- Table output includes `STALE_REFS`, the number of remote-tracking refs a prune would remove. JSON and `describe` include the ref names and any non-fatal remote inspection error.
- JSON output includes repo-local metadata when `.repokeeper-repo.yaml` or `repokeeper.yaml` is present.

### `repokeeper describe`

- Table and JSON output include repo-local metadata details when present.
- Invalid repo-local metadata is reported per repo instead of aborting the whole command.

### `repokeeper index`

- Interactive by default; proposes metadata from the tracked repo and prints a YAML preview.
- Writes only when `--write` is passed.
- `--force` replaces an existing repo-local metadata file.
- `--promote-local-labels` merges machine-local labels into shared repo metadata labels before preview/write.
- `--yes` skips the final write confirmation, but still requires `--write`.
- The command writes `.repokeeper-repo.yaml` by default and updates `repokeeper.yaml` when that legacy filename already exists.

### `repokeeper index repos`

- Explicit bulk metadata workflow; does not run unless you ask for it.
- Requires `--promote-local-labels` and at least one of `--selector` or `--local-selector`.
- Uses `--selector` for shared repo metadata labels and `--local-selector` for machine-local labels.
- Prints a preview for every selected repo and writes only with `--write`.

### `repokeeper install` / `repokeeper install list` / `repokeeper uninstall`

- Auto-detects Claude Code, Codex, and OpenCode; `--claude`/`--codex`/`--opencode` restrict the target set.
- `--scope user` (default) or `--scope project`. `--scope project --codex` is a hard error (Codex has no project scope).
- `--command PATH` overrides the binary path written to config; default is `os.Executable()` so Homebrew's bin shim is used instead of a version-specific Cellar path.
- `--manual [=all|claude|codex|opencode]` prints config snippets to stdout instead of writing; use for Cursor, Windsurf, or any runtime RepoKeeper doesn't adapter.
- `install list` reports `not registered`, `registered`, `registered (stale)`, or `unsupported` for each runtime at the chosen scope; `--json` emits `{scope, runtimes[]}`.
- `uninstall` prompts once before removing unless `--yes` is passed; empty stdin aborts as a safe default.
- Replaces the removed `repokeeper skill install/uninstall`. The canonical skill file is still at `docs/skills/repokeeper/SKILL.md` — copy it into your runtime's skills directory manually if you need the CLI fallback. See [docs/mcp-setup.md](mcp-setup.md) for per-runtime config paths and tool reference.

### `repokeeper reconcile`

- Shows a preflight plan before execution.
- Dry-run plans include stale remote-tracking ref count/list data for the fetch/prune step.
- Sync is fetch/prune-first; `--update-local` is the explicit path for local branch update behavior.
- Prompts only when mutating actions are planned (rebase/stash/checkout-missing clone), unless `--yes`.
- Supports `--checkout-missing` to clone entries marked missing.
- Does not act as a general branch-switch workflow.

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
