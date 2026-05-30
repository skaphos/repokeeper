# MCP Manual Verification Results (SKA-201)

**Date**: 2026-05-30  
**Performed by**: Grok (on behalf of user) + prepared for local Claude Code execution  
**Related Tickets**: SKA-201, SKA-128 (parent), SKA-470

## Purpose
This document captures the preparation and evidence for the manual end-to-end verification of all 14 RepoKeeper MCP tools from a real Claude Code session, as required by SKA-201.

## Verification Environment Setup

The environment was prepared using:

```bash
./scripts/verify-mcp.sh
```

**Resulting persistent workspace** (as of this run):

> `<WORKSPACE>` is a placeholder. `scripts/verify-mcp.sh` creates the workspace
> with `mktemp`, so the real path is unique per run. Substitute the absolute
> path that the script prints (e.g. `/tmp/repokeeper-mcp-verify-XXXXXX`) into
> every snippet below.

- **Path**: `<WORKSPACE>`
- **Config**: `<WORKSPACE>/.repokeeper.yaml`
- **Claude Code MCP config** (copy this):

```json
{
  "mcpServers": {
    "repokeeper-verify": {
      "command": "repokeeper",
      "args": ["mcp", "--config", "<WORKSPACE>/.repokeeper.yaml", "--log-file", "<WORKSPACE>/mcp.log"]
    }
  }
}
```

**Test repositories created**:
- `alpha` (clean)
- `beta` (clean)
- `gamma` (has uncommitted "dirty change" for status testing)

**Log file** (will be written when the MCP server runs): `<WORKSPACE>/mcp.log`

## How to Run the Verification (for the user)

1. Copy the `.mcp.json` content above into your Claude Code configuration (either workspace `.mcp.json` or global settings).
2. Restart/reload Claude Code so it picks up the new MCP server.
3. Start a fresh conversation.
4. Work through the checklist below with the specific test data in the workspace.
5. Record actual pass/fail + any observations or bugs found.

## Verification Checklist (with Evidence Placeholders)

### 1. Tool Discovery
- [ ] Claude lists exactly 14 tools when asked.
- [ ] Annotations (`readOnlyHint`, `destructiveHint`) are visible and correct on the tools.

**Evidence from automated side (for comparison)**: The in-process client tests in `internal/mcpserver/mcpserver_test.go` already confirm discovery of all 14 tools + correct annotations.

### 2. Read-only Tools
- [ ] `list_repositories`
- [ ] `get_repository_context` (including unknown repo error case)
- [ ] `get_workspace_config`
- [ ] `build_workspace_inventory`
- [ ] `select_repositories`
- [ ] `get_repo_metadata`
- [ ] `get_authoritative_paths`
- [ ] `get_related_repositories`

### 3. Planning Tools
- [ ] `plan_sync` (with filters)

### 4. Mutation Tools + Safety Gates
- [ ] `scan_workspace`
- [ ] `execute_sync` (rejects without `confirm: true`; succeeds with it)
- [ ] `set_labels`
- [ ] `add_repository`
- [ ] `remove_repository`

### 5. Structured Content & Error Quality
- [ ] List tools return proper `structuredContent` objects (not bare arrays).
- [ ] Errors are clear and actionable.

### 6. Overall Claude Experience
- [ ] No "expected record / received array" regressions.
- [ ] Claude can reason effectively over the tool outputs.
- [ ] Permissions behavior works as expected (read-only tools can be auto-approved; mutations prompt).

## Run Log / Evidence

**Setup command executed**:
```bash
./scripts/verify-mcp.sh
```

**Workspace location** (preserved by default; pass `--cleanup` to remove on exit): `<WORKSPACE>`

**MCP server invocation** (what Claude Code will use):
```
repokeeper mcp --config <WORKSPACE>/.repokeeper.yaml --log-file <WORKSPACE>/mcp.log
```

**User action required**: Point your local Claude Code at the config above, then execute the checklist in a real session and append your actual results below.

---

**Automated Test Coverage Achieved for SKA-200** (as of 2026-05-30):

**COMPLETE** — We now have **15 dedicated in-process MCP client protocol tests** using the official `client.NewInProcessClient` pattern.

Every one of the 14 tools has at least one dedicated test exercising the full MCP protocol path (Initialize → CallTool):

All 15 new + supporting specs pass (`ginkgo`).

See `internal/mcpserver/mcpserver_test.go` (the "InProcess MCP Client" describe block) for the implementation.

This fully satisfies the core "one dedicated test per tool" requirement for SKA-200.

---

**Actual Claude Code Run Results** (to be filled by user after running the prepared environment at `<WORKSPACE>`):

- Date of Claude session: 
- Claude Code version:
- Tools tested:
- Issues found:
- Overall verdict:

---

**Next steps after this document**:
- User runs the verification in their local Claude Code using the prepared workspace.
- Results posted as comment on SKA-128 / SKA-201.
- Ticket can then be closed.

This completes the preparation and evidence capture for SKA-201.