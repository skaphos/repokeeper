# RepoKeeper Commands

This is the canonical command reference for RepoKeeper. Keep this file in sync with `--help` output.

## Top-level Commands

| Command | Description |
|---|---|
| `repokeeper init` | Bootstrap a new config file |
| `repokeeper scan` | Discover repos and update the registry |
| `repokeeper get` | Report repo health summary (path, branch, dirty, tracking) |
| `repokeeper get repos` | Explicit resource form for repo health |
| `repokeeper describe <repo-id-or-path>` | Show detailed status for one repository |
| `repokeeper describe repo <repo-id-or-path>` | Kubectl-style describe form |
| `repokeeper index <repo-id-or-path>` | Interactively preview or write repo-local metadata |
| `repokeeper index repos` | Preview or write repo-local metadata for selected repositories |
| `repokeeper install` | Register RepoKeeper as an MCP server in detected (or --claude/--codex/--opencode) runtimes |
| `repokeeper install list` | Show per-runtime MCP registration state (table or `--json`) |
| `repokeeper uninstall` | Remove the RepoKeeper MCP entry from each runtime (prompts unless `--yes`) |
| `repokeeper add <path> <git-repo-url>` | Clone and register a repository |
| `repokeeper delete <repo-id-or-path>` | Delete repo files and remove from registry |
| `repokeeper edit <repo-id-or-path>` | Open one repo entry in `$VISUAL`/`$EDITOR`, validate, save |
| `repokeeper label <repo-id-or-path>` | Show or mutate labels for one repository |
| `repokeeper repair upstream` | Repair missing/mismatched upstream tracking |
| `repokeeper reconcile` | Fetch and prune all repos safely |
| `repokeeper reconcile repos` | Explicit resource form for sync/reconciliation |
| `repokeeper export` | Export config and optional registry for migration |
| `repokeeper import` | Import a previously exported bundle |
| `repokeeper version` | Print version and build info |

## Command Notes

### `repokeeper get`

- Supports `--only`, `--field-selector`, and label selector `-l, --selector`.
- Label selector supports `key` and `key=value`, comma-separated AND.
- Use `-o wide` for additional `PRIMARY_REMOTE`, `UPSTREAM`, `AHEAD`, `BEHIND`, and `ERROR_CLASS`.
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
