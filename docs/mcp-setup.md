# MCP Server Setup

RepoKeeper includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives agent runtimes typed, structured access to all RepoKeeper operations. When available, MCP is the preferred integration path over the bundled skill.

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

The MCP server exposes 14 tools organized by intent:

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

### Mutation Tools (6)

| Tool | Description | Safety |
|---|---|---|
| `scan_workspace` | Discover repos and update registry | Writes registry |
| `plan_sync` | Preview sync actions (always dry-run) | Read-only |
| `execute_sync` | Execute sync (requires `confirm: true`) | Safety gate |
| `set_labels` | Set or remove labels on a repo | Writes registry |
| `add_repository` | Clone and register a repo | Clones to disk |
| `remove_repository` | Remove from registry (tracking-only default) | Optional disk delete |

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

## Runtimes Without MCP Support

For runtimes that do not support MCP (such as Codex/OpenAI agents), use the bundled skill instead:

```bash
repokeeper skill install
```

The skill teaches agents to use RepoKeeper via CLI commands. See [docs/skills/README.md](skills/README.md) for details.
