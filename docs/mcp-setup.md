# MCP Server Setup

RepoKeeper includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives agent runtimes typed, structured access to RepoKeeper inspection, planning, and currently shipped mutation workflows. When available, MCP is the preferred integration path over the bundled skill for inspection and planning, and it can also be used for explicit state-changing operations that carry their own safety gates.

## Prerequisites

- RepoKeeper installed and on your `PATH` (see [INSTALL.md](../INSTALL.md))
- A workspace initialized with `repokeeper init` and at least one `repokeeper scan`

## Quick Start

The fastest path is `repokeeper install`. It auto-detects Claude Code, Codex, and OpenCode and writes the MCP server entry into each detected runtime's config file:

```bash
# Register repokeeper with every detected runtime at user scope
repokeeper install

# Verify what was registered
repokeeper install list

# Limit to a single runtime
repokeeper install --claude
```

The command is idempotent — re-running it reports `unchanged` when config already matches, `updated` when it had to rewrite a stale entry, and `registered` when the entry was absent. After `brew upgrade` or any binary move, re-run `repokeeper install` to refresh the registered path.

### Supported runtimes

| Runtime | User scope config | Project scope | Format |
|---|---|---|---|
| Claude Code | `~/.claude.json` | `./.mcp.json` | JSON, key `mcpServers.repokeeper` |
| Codex | `~/.codex/config.toml` | not supported | TOML, key `[mcp_servers.repokeeper]` |
| OpenCode | `${OPENCODE_CONFIG_DIR:-${XDG_CONFIG_HOME:-~/.config}/opencode}/opencode.json` | `./opencode.json` | JSON, key `mcp.repokeeper` |

### Common flags

- `--claude` / `--codex` / `--opencode` — restrict the target set (otherwise auto-detect).
- `--scope user` (default) or `--scope project`. `--scope project --codex` is a hard error (Codex has no project scope).
- `--command PATH` — override the binary path written to config. Default is `os.Executable()`, which resolves to Homebrew's bin shim on macOS rather than a version-specific Cellar path.
- `--manual [=all|claude|codex|opencode]` — print the config snippet(s) to stdout instead of writing. Use this for runtimes RepoKeeper doesn't adapter (Cursor, Windsurf) or when you prefer to manage config by hand.

### Inspecting state

```bash
# Per-runtime table
repokeeper install list

# JSON output for scripting
repokeeper install list --json
```

The `STATE` column is one of `not registered`, `registered`, `registered (stale)` (command in config no longer matches the current binary), or `unsupported` (scope not supported by this runtime).

### Removing the entry

```bash
# Prompt, then remove repokeeper MCP entries from every detected runtime
repokeeper uninstall

# Skip the prompt
repokeeper uninstall --yes

# Limit to a single runtime
repokeeper uninstall --claude --yes
```

A declined prompt (or empty stdin in a non-interactive invocation without `--yes`) aborts without removing anything, which is the safe default for scripted contexts.

## Debugging

Use `--log-file` to capture MCP server logs (stdout is owned by the protocol in stdio mode). Pass it via `--command` or edit the config directly:

```bash
# Register repokeeper and capture debug output to a file.
repokeeper install --command "$(command -v repokeeper)"
# Then edit the runtime's config to append --log-file to the args array:
# "args": ["mcp", "--log-file", "/tmp/repokeeper-mcp.log"]
```

A future release may expose `--args` on `repokeeper install` to append MCP server flags directly.

## Available tools

The current MCP server exposes 14 tools organized by intent. Most tools are read-only or planning-oriented, and a smaller set are explicit mutations.

### Read tools (8)

| Tool | Description |
|---|---|
| `list_repositories` | List tracked repos from registry (fast, no git inspection) |
| `build_workspace_inventory` | Live health check across all repos |
| `select_repositories` | Query by label, field selector, or name match |
| `get_repository_context` | Deep single-repo context (git state, metadata, labels) |
| `get_repo_metadata` | Source-controlled repo-local metadata only |
| `get_authoritative_paths` | Path hints and entrypoints for a repo |
| `get_related_repositories` | Relationship graph from repo metadata |
| `get_workspace_config` | Current workspace configuration |

### Planning tools

| Tool | Description | Safety |
|---|---|---|
| `plan_sync` | Preview sync actions (always dry-run) | Read-only |

### Mutation tools (5)

| Tool | Description | Safety |
|---|---|---|
| `scan_workspace` | Discover repos and update registry | Writes registry |
| `execute_sync` | Execute sync (requires `confirm: true`) | Safety gate |
| `set_labels` | Set or remove labels on a repo | Writes registry |
| `add_repository` | Clone and register a repo | Clones to disk |
| `remove_repository` | Remove from registry (tracking-only default) | Optional disk delete |

CLI and TUI remain the preferred operator interfaces for execution-heavy workflows, but the current MCP server also ships explicit mutation tools. Treat those tools as state-changing operations and rely on their documented safety gates and confirmations before using them.

Argument notes:

- `plan_sync` remains side-effect-free even when it previews local update candidates.
- `scan_workspace.roots` is a string array of absolute or otherwise valid filesystem roots.
- If `scan_workspace.roots` is omitted, RepoKeeper falls back to the effective workspace root resolved from the active config path.
- `set_labels.set` is an object whose values must be strings.
- `set_labels.remove` is a string array of label keys to delete.
- `execute_sync.confirm` is a required safety gate. The call must include `"confirm": true`; omitting it or setting it to `false` is rejected.

Examples:

```json
{
  "roots": ["/work/repos", "/opt/mirrors"],
  "prune_stale": true
}
```

```json
{
  "repo": "github.com/example/alpha",
  "set": {"team": "platform"},
  "remove": ["env"]
}
```

```json
{
  "label_selector": "team=platform",
  "confirm": true,
  "update_local": true
}
```

### Resources

| URI | Description |
|---|---|
| `repokeeper://config` | Workspace configuration |
| `repokeeper://registry` | Full registry snapshot |
| `repokeeper://repo/{repo_id}` | Single registry entry |
| `repokeeper://repo/{repo_id}/metadata` | Repo-local metadata |

## Unsupported runtimes (manual configuration)

Some agent runtimes don't expose a canonical flat-file MCP config we can safely rewrite. For those, `repokeeper install --manual` prints the snippet you would paste into their configuration UI:

```bash
# Print snippets for all three adapter-backed runtimes (Claude, Codex, OpenCode)
repokeeper install --manual

# Print just the JSON shape a Cursor-style runtime expects
repokeeper install --manual=claude
```

### Cursor

Cursor's MCP configuration is managed through its Settings UI (Settings > MCP Servers). Use the JSON shape below:

```json
{
  "mcpServers": {
    "repokeeper": {
      "command": "repokeeper",
      "args": ["mcp"]
    }
  }
}
```

### Windsurf

Windsurf's MCP configuration is likewise UI-managed. Use the same JSON shape as Cursor. Replace `"repokeeper"` (the command value) with the absolute path from `command -v repokeeper` if the binary is not on the runtime's `PATH`.

### Custom config path

If your `.repokeeper.yaml` is not in a parent of your working directory, extend `args` with `--config`:

```json
{
  "mcpServers": {
    "repokeeper": {
      "command": "repokeeper",
      "args": ["mcp", "--config", "/path/to/.repokeeper.yaml"]
    }
  }
}
```

## CLI skill fallback

If your runtime does not support MCP, or you want a CLI-driven fallback instead of MCP, copy the canonical skill file into your runtime's skills directory:

```bash
# Example: Claude Code
mkdir -p ~/.claude/skills/repokeeper
cp docs/skills/repokeeper/SKILL.md ~/.claude/skills/repokeeper/SKILL.md
```

See [docs/skills/README.md](skills/README.md) for the full list of runtime skill paths and caveats. The previous `repokeeper skill install` CLI was removed in ADR-0008 — MCP via `repokeeper install` is the preferred integration surface for runtimes that support it.
