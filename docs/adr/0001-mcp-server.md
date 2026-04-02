# ADR-0001: MCP Server for Agent-Native Repository Querying

**Status:** Proposed
**Date:** 2026-04-01
**Author:** Shawn Stratton

## Context

RepoKeeper integrates with agent runtimes (Claude Code, OpenCode, Codex) via a bundled skill (`SKILL.md`) that teaches agents to shell out to CLI commands. This works but has limitations:

- Agents must parse unstructured text output or request `-o json` and parse stdout.
- Agents construct CLI flags from natural language, introducing ambiguity.
- Every invocation is a cold start: load config, load registry, create engine, execute, exit.
- The skill document is ~250 lines of instruction the agent must internalize per session.
- There is no structured discovery of available operations or their parameters.

The [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) provides a standard for exposing tools, resources, and prompts to agent runtimes over a structured JSON-RPC transport. An MCP server would give agents typed tool schemas, structured JSON responses, and self-describing operations without requiring a skill document to explain CLI usage.

## Decision

Add an MCP server to RepoKeeper as a new frontend to the existing engine layer, exposed via a `repokeeper mcp` subcommand using stdio transport. The MCP tools will be **agent-intent-oriented** rather than 1:1 CLI mirrors, designed around how agents think about repository work.

The skill will be updated to reference MCP tools when available, with CLI fallback for runtimes or workflows where MCP is unavailable or not preferred.

## Tool Surface Design

### Principles

1. **Agent intent over CLI parity.** Tools map to what agents want to do ("list repositories", "get context for this repo", "which paths matter?"), not to CLI flags.
2. **Read/write separation.** Read tools have no side effects. Write tools are clearly marked and require explicit confirmation where destructive.
3. **Fast path and thorough path.** `list_repositories` reads the registry (fast). `build_workspace_inventory` runs live git inspection (thorough). Agents choose based on need.
4. **Safety at the protocol level.** `plan_sync` and `execute_sync` are separate tools (not a `dry_run` flag). `execute_sync` requires an explicit `confirm: true` parameter since MCP has no interactive prompts, and requests that omit `confirm` or set it to `false` are rejected.

### Discovery and Inventory (read-only)

#### `list_repositories`

List all tracked repos with summary info. Fast — reads registry only, no live git inspection.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `label_selector` | string | no | Label filter (e.g. `team=platform,role=service`) |
| `status` | string | no | Registry status filter: `present`, `missing`, `moved` |

**Returns:** Array of `{repo_id, checkout_id, path, remote_url, type, labels, annotations, status, last_seen}`

**Engine path:** `Registry().Entries` with filtering

#### `build_workspace_inventory`

Full live health check across all repos. Runs git inspect on each registered repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `filter` | string | no | Health filter: `all`, `errors`, `dirty`, `clean`, `gone`, `diverged`, `missing` |
| `label_selector` | string | no | Label filter |
| `concurrency` | integer | no | Max parallel inspections (default from config) |

**Returns:** `{generated_at, repos: [{repo_id, path, head, worktree, tracking, labels, repo_metadata, ...}]}`

**Engine path:** `Engine.Status()` + registry enrichment (labels, annotations)

#### `select_repositories`

Query repos by combining label selectors, field selectors, and free-text name matching. Returns matched repo IDs and paths without full status detail.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `label_selector` | string | no | Label filter |
| `field_selector` | string | no | Field filter (e.g. `tracking.status=behind`, `worktree.dirty=true`) |
| `name_match` | string | no | Substring or glob match on repo_id |

**Returns:** Array of `{repo_id, path, labels, match_reason}`

**Engine path:** `Engine.Status()` + label/field/name filtering

### Single-Repo Context (read-only)

#### `get_repository_context`

Deep context for a single repo — git state, labels, annotations, metadata, entrypoints, related repos. The "tell me everything about this repo" tool.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier (repo_id or absolute path) |

**Returns:** Full `RepoStatus` merged with registry entry (labels, annotations, repo_metadata)

**Engine path:** `Engine.InspectRepo()` + registry lookup and enrichment

#### `get_repo_metadata`

Source-controlled repo-local metadata only. Returns null if no metadata file exists in the repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier (repo_id or absolute path) |

**Returns:** `RepoMetadata` (`{labels, entrypoints, paths, provides, related_repos}`) or null

**Engine path:** Registry cached metadata or live read via `repometa` package

#### `get_authoritative_paths`

Returns the authoritative and low-value path hints for a repo. Quick way for an agent to know where to look first and what to avoid.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier (repo_id or absolute path) |

**Returns:** `{authoritative: [string], low_value: [string], entrypoints: {name: path}}`

**Engine path:** Registry metadata `Paths` + `Entrypoints` fields

#### `get_related_repositories`

Given a repo, returns its declared related repos with relationship types and optional cross-reference to registry for local paths.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier (repo_id or absolute path) |

**Returns:** Array of `{repo_id, relationship, path?, status?}`

**Engine path:** Registry metadata `RelatedRepos` cross-referenced with registry entries

### Mutation Tools

#### `scan_workspace`

Discover repos under configured or specified roots and update the registry.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `roots` | string[] | no | Override scan roots (default: effective workspace root). Each item must be a string path. |
| `prune_stale` | boolean | no | Remove entries marked missing beyond staleness threshold |

**Returns:** `{discovered, new, missing, pruned, repos: [{repo_id, path, status}]}`

**Engine path:** `Engine.Scan()` + config/registry save

#### `plan_sync`

Preview what a sync would do without executing. Always dry-run — never mutates.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `filter` | string | no | Health filter |
| `label_selector` | string | no | Label filter |
| `update_local` | boolean | no | Include local rebase in plan |
| `push_local` | boolean | no | Include push in plan |

**Returns:** Array of `{repo_id, path, action, outcome, planned: true}`

**Engine path:** `Engine.Sync(DryRun: true)`

#### `execute_sync`

Execute a sync operation. Requires explicit `confirm: true` as a safety gate.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `filter` | string | no | Health filter |
| `label_selector` | string | no | Label filter |
| `update_local` | boolean | no | Enable local rebase |
| `push_local` | boolean | no | Enable push |
| `force` | boolean | no | Force operations on diverged repos |
| `confirm` | boolean | **yes** | Must be present and `true` to execute. Safety gate; omitted or `false` requests are rejected. |

**Returns:** Array of `{repo_id, path, action, outcome, ok, error?}`

**Engine path:** `Engine.Sync()` + `ExecuteSyncPlanWithCallbacks()`

#### `set_labels`

Set or remove machine-local labels on a repository.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier |
| `set` | object | no | Labels to set (string-to-string key-value map) |
| `remove` | string[] | no | Label keys to remove. Each item must be a string label key. |

**Returns:** `{repo_id, labels}` (updated label set)

**Engine path:** Registry entry mutation + save

#### `add_repository`

Clone a repository and register it in the workspace.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `url` | string | yes | Remote URL to clone |
| `path` | string | yes | Local target path |
| `mirror` | boolean | no | Clone as bare mirror |

**Returns:** `{repo_id, path, status}`

**Engine path:** `Engine.CloneAndRegister()`

#### `remove_repository`

Remove a repository from the registry. Default is tracking-only removal (does not delete files).

| Parameter | Type | Required | Description |
|---|---|---|---|
| `repo` | string | yes | Repo identifier |
| `delete_files` | boolean | no | Also delete files on disk (default: false) |

**Returns:** `{repo_id, removed: true}`

**Engine path:** `Engine.DeleteRepo()`

### Configuration (read-only)

#### `get_workspace_config`

Returns current RepoKeeper workspace configuration.

| Parameter | Type | Required | Description |
|---|---|---|---|
| _(none)_ | | | |

**Returns:** `{roots, exclude, defaults: {remote_name, concurrency, timeout_seconds}, registry_path}`

**Engine path:** `Engine.Config()`

### Tool Count Summary

- **8 read-only tools:** `list_repositories`, `build_workspace_inventory`, `select_repositories`, `get_repository_context`, `get_repo_metadata`, `get_authoritative_paths`, `get_related_repositories`, `get_workspace_config`
- **6 mutation tools:** `scan_workspace`, `plan_sync`, `execute_sync`, `set_labels`, `add_repository`, `remove_repository`
- **14 total**

## MCP Resources

| URI Pattern | Description | Content Type |
|---|---|---|
| `repokeeper://config` | Current workspace configuration | `application/json` |
| `repokeeper://registry` | Full registry snapshot | `application/json` |
| `repokeeper://repo/{repo-id}` | Single registry entry by repo_id | `application/json` |
| `repokeeper://repo/{repo-id}/metadata` | Repo-local metadata for a repo | `application/json` |

Resources are read-only. The `registry` resource provides browsable context without tool invocation. The `repo/{repo-id}` template uses MCP resource templates for client-side discovery.

## Architecture

### Package Structure

```
internal/
  mcpserver/
    server.go            -- MCPServer struct, constructor, tool/resource registration
    engine.go            -- EngineAPI interface (extends TUI pattern with Scan)
    resolve.go           -- Shared repo resolution (repo_id or path -> registry entry)
    tools_discovery.go   -- list_repositories, build_workspace_inventory, select_repositories
    tools_context.go     -- get_repository_context, get_repo_metadata, get_authoritative_paths, get_related_repositories
    tools_mutation.go    -- scan_workspace, plan_sync, execute_sync, set_labels, add_repository, remove_repository
    tools_config.go      -- get_workspace_config
    resources.go         -- MCP resource handlers
    *_test.go
  selector/
    label.go             -- Label selector parsing and matching (extracted from cmd/)
    field.go             -- Field selector parsing (extracted from cmd/)
    *_test.go
cmd/
  repokeeper/
    mcp.go               -- `repokeeper mcp` cobra subcommand
```

### EngineAPI Interface

Follows the established pattern from `internal/tui/engine.go`. The MCP server defines its own interface satisfied by `*engine.Engine`:

```go
type EngineAPI interface {
    Status(ctx context.Context, opts engine.StatusOptions) (*model.StatusReport, error)
    Sync(ctx context.Context, opts engine.SyncOptions) ([]engine.SyncResult, error)
    ExecuteSyncPlanWithCallbacks(
        ctx context.Context,
        plan []engine.SyncResult,
        opts engine.SyncOptions,
        onStart engine.SyncStartCallback,
        onComplete engine.SyncResultCallback,
    ) ([]engine.SyncResult, error)
    InspectRepo(ctx context.Context, path string) (*model.RepoStatus, error)
    RepairUpstream(ctx context.Context, repoID, cfgPath string) (engine.RepairUpstreamResult, error)
    ResetRepo(ctx context.Context, repoID, cfgPath string) error
    DeleteRepo(ctx context.Context, repoID, cfgPath string, deleteFiles bool) error
    CloneAndRegister(ctx context.Context, remoteURL, targetPath, cfgPath string, mirror bool) error
    Scan(ctx context.Context, opts engine.ScanOptions) ([]model.RepoStatus, error)
    Registry() *registry.Registry
    Config() *config.Config
    Adapter() vcs.Adapter
}
```

### Selector Extraction

Label selector logic currently lives in `cmd/repokeeper/label_selector.go` and field selector logic in `cmd/repokeeper/selectors.go`. Both are ~65 lines each and will be extracted to `internal/selector/` for shared use by the CLI and MCP server. The CLI package will be updated to import from the new location.

### Transport

**Phase 1: stdio.** This is the standard transport for local agent runtimes (Claude Code, Cursor, Windsurf). The `mcp-go` SDK handles stdio framing natively.

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

**Phase 2 (future): Streamable HTTP.** Only if remote/multi-client access is needed. SSE transport is being deprecated in the MCP spec; skip it entirely.

### CLI Subcommand

```
repokeeper mcp [flags]

Flags:
  --transport string   Transport mode (default "stdio")
  --log-file string    Debug log file path (stdout is owned by MCP protocol in stdio mode)
```

Inherits global flags (`--config`, `--verbose`) from root command.

## Dependencies

**New:** `github.com/mark3labs/mcp-go` — the most mature Go MCP SDK. Provides server creation, stdio transport, tool/resource schema definitions, and handler registration.

**Alternative considered:** `github.com/modelcontextprotocol/go-sdk` (official SDK). Worth evaluating at implementation time if it has matured, but `mcp-go` has broader adoption and battle-testing as of this writing.

## Skill Update

The bundled `SKILL.md` will be updated with a new section mapping CLI workflows to MCP tool equivalents and instructing agents to prefer MCP tools when available:

- `repokeeper get -o json` -> `list_repositories` or `build_workspace_inventory`
- `repokeeper describe <repo> -o json` -> `get_repository_context`
- `repokeeper reconcile --dry-run` -> `plan_sync`
- `repokeeper reconcile --update-local` -> `execute_sync`
- `repokeeper label` -> `set_labels`
- `repokeeper scan` -> `scan_workspace`

The same safety rules apply in both modes. Agents fall back to CLI when MCP is not configured.

## Consequences

### Positive

- Agents get structured, typed tool interfaces with JSON Schema parameter validation.
- Self-describing tool list eliminates the need for agents to internalize a 250-line skill document.
- Safety rules (preview before mutate, confirm before execute) are enforced at the protocol level.
- Single binary — `repokeeper mcp` is just another subcommand.
- The engine layer is already cleanly separated; the MCP server is a thin adapter (same pattern as the TUI).

### Negative

- Adds a runtime dependency (`mcp-go`) and a new package to maintain.
- Two integration paths (skill + MCP) to document and test.
- MCP support varies across agent runtimes; some may not support it yet.
- Selector extraction touches existing CLI code (low risk, mechanical refactor).

### Neutral

- The CLI remains the primary human interface. MCP does not replace it.
- The skill remains useful for runtimes without MCP support and as documentation.
- GoReleaser and distribution are unaffected (same binary, same build).

## Alternatives Considered

### 1. Enhance CLI output only (no MCP)

Add more structured output flags (`-o json` everywhere, field selectors, etc.) and keep the skill as the sole agent integration.

**Rejected because:** Even with perfect JSON output, agents still construct CLI commands from natural language and parse stdout. MCP eliminates this indirection entirely.

### 2. gRPC server

Expose the engine as a gRPC service with protobuf definitions.

**Rejected because:** MCP is the emerging standard for agent-tool integration. gRPC would require agents to have a gRPC client, which none of the target runtimes support natively. MCP stdio transport works with all of them today.

### 3. CLI mirrors as MCP tools

Map each CLI command 1:1 to an MCP tool (`mcp_get`, `mcp_scan`, `mcp_reconcile`).

**Rejected because:** CLI commands are designed for human ergonomics (flags, filters, output modes). Agents think in terms of intent ("get context for this repo"), not CLI invocations. Agent-intent-oriented tools provide a better experience.
