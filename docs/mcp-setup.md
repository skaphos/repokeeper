# MCP Server Setup

RepoKeeper includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that gives agent runtimes typed, structured access to RepoKeeper inspection and planning workflows. When available, MCP is the preferred integration path over the bundled skill for read-only and side-effect-free preview operations.

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

The MCP server is intended as a read-and-plan surface organized by intent.

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
 
Execution remains in the CLI and TUI. State-changing workflows such as sync execution, label mutation, add/remove flows, metadata writes, prune execution, and branch switching are intentionally out of scope for MCP.

Planning-tool argument notes:

- `plan_sync` remains side-effect-free even when it previews local update candidates.
- Future planning tools such as prune or branch-switch preview should follow the same side-effect-free contract.

Examples:

```json
{
  "label_selector": "team=platform",
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
