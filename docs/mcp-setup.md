# MCP Server Setup

RepoKeeper includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives agent runtimes typed, structured access to RepoKeeper inspection, planning, and currently shipped mutation workflows. When available, MCP is the preferred integration path over the bundled skill for inspection and planning, and it can also be used for explicit state-changing operations that carry their own safety gates.

## Prerequisites

- RepoKeeper installed and on your `PATH` (see [INSTALL.md](../INSTALL.md))
- A workspace initialized with `repokeeper init` and at least one `repokeeper scan`

## Quick Start

The fastest path is `repokeeper install`. It auto-detects Claude Code, Codex, OpenCode, and Grok and writes the MCP server entry into each detected runtime's config file:

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
| Grok | `~/.grok/config.toml` | `./.grok/config.toml` | TOML, key `[mcp_servers.repokeeper]` (with `enabled = true`) |

### Common flags

- `--claude` / `--codex` / `--opencode` / `--grok` — restrict the target set (otherwise auto-detect).
- `--scope user` (default) or `--scope project`. `--scope project --codex` is a hard error (Codex has no project scope).
- `--command PATH` — override the binary path written to config. Default is `os.Executable()`, which resolves to Homebrew's bin shim on macOS rather than a version-specific Cellar path.
- `--manual [=all|claude|codex|opencode|grok]` — print the config snippet(s) to stdout instead of writing. Use this for runtimes RepoKeeper doesn't adapt (Cursor, Windsurf) or when you prefer to manage config by hand.

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

## Recommended Claude Code permissions

Claude Code prompts for confirmation on every MCP tool call by default. To skip the prompt for RepoKeeper's safe, read-only tools — while keeping the prompt for anything that changes state — add the following to your Claude Code settings (`~/.claude/settings.json` for user scope, or `.claude/settings.json` for project scope):

```json
{
  "permissions": {
    "allow": [
      "mcp__repokeeper__build_workspace_inventory",
      "mcp__repokeeper__get_authoritative_paths",
      "mcp__repokeeper__get_related_repositories",
      "mcp__repokeeper__get_repo_metadata",
      "mcp__repokeeper__get_repository_context",
      "mcp__repokeeper__get_workspace_config",
      "mcp__repokeeper__list_repositories",
      "mcp__repokeeper__plan_sync",
      "mcp__repokeeper__select_repositories"
    ]
  }
}
```

> **Warning:** Pasting this grants Claude Code blanket auto-approval for the listed tools — it will run them without asking. Every entry is read-only or dry-run (`plan_sync`), so they cannot change your repositories or registry.

The five mutation tools — `scan_workspace`, `execute_sync`, `set_labels`, `add_repository`, `remove_repository` — are deliberately **not** listed, so they keep prompting on every call (RepoKeeper's read-and-plan safety model, [ADR-0001](adr/0001-mcp-server.md)). Avoid the broad `"mcp__repokeeper__*"` wildcard: it would also auto-approve those mutations.

`repokeeper install --manual=claude` prints this same block (derived from the live server's tool annotations) alongside the registration snippet.

## Runtimes without a RepoKeeper adapter

`repokeeper install` only writes config for runtimes it has an adapter for (Claude Code, Codex, OpenCode, Grok). For other MCP-capable runtimes, edit the runtime's config file by hand using the shape documented below. `repokeeper install --manual` prints the supported runtime snippets to stdout as a convenience, but the sections here are authoritative for each runtime.

Tip: if `repokeeper` is not on the runtime's `PATH`, replace `"repokeeper"` in the `command` field with the absolute path from `command -v repokeeper`.

### Cursor

Cursor reads MCP servers from a flat JSON file. The key is `mcpServers`, the same shape Claude Code uses, so `repokeeper install --manual=claude` emits a snippet you can paste verbatim.

- **User scope:** `~/.cursor/mcp.json`
- **Project scope:** `<repo-root>/.cursor/mcp.json`

Project config overrides user config when both define the same server. See [Cursor's MCP docs](https://cursor.com/docs/context/mcp) for the full option list.

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

### VS Code + GitHub Copilot

VS Code uses its own JSON shape for MCP servers — the root key is `servers` (not `mcpServers`), and each entry carries a `type` discriminator (`"stdio"` for local commands like RepoKeeper).

- **User scope:** open the Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`) and run **MCP: Open User Configuration**. This opens the correct file under your VS Code user profile.
- **Project scope:** `<repo-root>/.vscode/mcp.json`

```json
{
  "servers": {
    "repokeeper": {
      "type": "stdio",
      "command": "repokeeper",
      "args": ["mcp"]
    }
  }
}
```

VS Code supports sandboxing for stdio MCP servers, restricting filesystem and network access to explicitly allowed paths/domains. If you enable sandboxing for RepoKeeper, allow the directories your workspace config and registry live under (typically the directory containing `.repokeeper.yaml` plus every repo path it references). See [VS Code's MCP configuration reference](https://code.visualstudio.com/docs/copilot/reference/mcp-configuration) for the sandbox schema.

### Windsurf

Windsurf's MCP configuration is managed through its UI. Use the same JSON shape Cursor uses (`mcpServers` root key), then paste it into Windsurf's MCP settings panel.

### Other runtimes

Any runtime that speaks stdio MCP and accepts `{"command": "...", "args": [...]}` in some form can run RepoKeeper. Start from `repokeeper install --manual=claude` (the most widely adopted shape) and translate into the runtime's format if the root key or entry shape differs.

### Custom config path

If your `.repokeeper.yaml` is not in a parent of your working directory, extend the `args` array with `--config`:

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

VS Code's equivalent:

```json
{
  "servers": {
    "repokeeper": {
      "type": "stdio",
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
