# MCP Server Setup

RepoKeeper includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives agent runtimes typed, structured access to RepoKeeper inspection, planning, and currently shipped mutation workflows. When available, MCP is the preferred integration path over the bundled skill for inspection and planning, and it can also be used for explicit state-changing operations that carry their own safety gates.

## Prerequisites

- RepoKeeper installed and on your `PATH` (see [INSTALL.md](../INSTALL.md))
- A workspace initialized with `repokeeper init` and at least one `repokeeper scan`

## Quick Start

```bash
# Verify the MCP server starts
repokeeper mcp --help
```

## Per-Runtime Configuration

### Claude Code

Add to `~/.claude/settings.json` (user scope) or `.claude/settings.json` (project scope):

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

### Cursor

Add to your Cursor MCP configuration (Settings > MCP Servers):

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

Add to your Windsurf MCP configuration:

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

### OpenAI Codex

Codex supports MCP server definitions in `~/.codex/config.toml`. For a local RepoKeeper MCP server, add:

```toml
[mcp_servers.repokeeper]
command = "repokeeper"
args = ["mcp"]
```

If your workspace config is not discoverable from the working directory, pass `--config` here the same way you would in other MCP clients.

### Custom Config Path

If your `.repokeeper.yaml` is not in a parent of your working directory, pass `--config`:

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

## Available Tools

The current MCP server exposes 14 tools organized by intent. Most tools are read-only or planning-oriented, and a smaller set are explicit mutations.

### Read Tools (8)

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

### Planning Tools

| Tool | Description | Safety |
|---|---|---|
| `plan_sync` | Preview sync actions (always dry-run) | Read-only |

### Mutation Tools (5)

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

## Debugging

Use `--log-file` to capture MCP server logs (stdout is owned by the protocol in stdio mode):

```json
{
  "mcpServers": {
    "repokeeper": {
      "command": "repokeeper",
      "args": ["mcp", "--log-file", "/tmp/repokeeper-mcp.log"]
    }
  }
}
```

## CLI Skill Fallback

If your runtime does not support MCP, or you want a CLI-driven fallback instead of MCP, use the bundled skill:

```bash
repokeeper skill install
```

The skill teaches agents to use RepoKeeper via CLI commands. See [docs/skills/README.md](skills/README.md) for details.
