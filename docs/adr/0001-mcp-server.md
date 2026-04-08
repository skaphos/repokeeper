# ADR-0001: MCP Server for Agent-Native Repository Querying and Planning

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper integrates with agent runtimes (Claude Code, OpenCode, Codex, Cursor, Windsurf) through a bundled skill and a built-in MCP server.

The original MCP direction improved on the skill-only workflow:

- agents no longer need to parse human-oriented CLI output
- tools are self-describing and typed
- structured JSON responses reduce ambiguity

That direction still stands, but the product boundary is now stricter than the original proposal.

RepoKeeper needs a clean separation between:

- **MCP** as the primary information and planning surface for agents
- **CLI/TUI** as the primary operator interfaces for repo mutations

This matches the broader RepoKeeper operating model:

- dry-run first
- plan -> confirm -> execute
- explainability over magic
- no non-trivial mutation without explicit operator intent

## Decision

RepoKeeper will keep an MCP server, and MCP is defined as **primarily** a read-and-plan surface.

MCP may expose:

- inventory and inspection
- repository context
- metadata status
- relationship and ownership queries
- validation output
- side-effect-free plans and previews

The current shipped MCP surface also includes some explicit mutation tools while the product boundary converges.

Those mutation tools:

- must be intentionally invoked as mutation surfaces
- must preserve explicit safety gates and confirmation requirements where applicable
- must not be hidden behind read-only or preview operations

CLI and TUI remain the primary operator interfaces for execution, and narrowing the MCP surface remains the long-term direction.

## Tool Surface Design

### Principles

1. **Agent intent over CLI parity.** Tools should map to what an agent wants to know, not to Cobra command shapes.
2. **Read and plan first.** MCP should prioritize inspection, explanation, validation, and side-effect-free previews.
3. **Deterministic structured output.** MCP responses must be machine-readable and explain non-trivial decisions with stable fields and reason codes where needed.
4. **Mutation must stay explicit.** Any action that changes repo state or local state must be clearly marked and safety-bounded regardless of interface.
5. **Fast path and thorough path.** MCP may provide both registry-backed and live-inspection views when both are useful.

### Read Tools

Read tools are side-effect-free and may include surfaces such as:

- `list_repositories`
- `build_workspace_inventory`
- `select_repositories`
- `get_repository_context`
- `get_repo_metadata`
- `get_authoritative_paths`
- `get_related_repositories`
- `get_workspace_config`

These tools may read registry state, repo-local metadata, or live VCS state, but they must not write anything.

### Planning Tools

Planning tools are also side-effect-free. They may evaluate current state and return a preview of what RepoKeeper would do if an operator later chooses execution through CLI, TUI, or an explicit mutation-capable MCP tool.

Examples include:

- sync preview / `plan_sync`
- future prune preview
- future branch-switch preview

Planning tools:

- must never mutate repository, registry, or metadata state
- must return explicit classifications and reason codes for skips, blockers, or proposed actions where practical
- must not blur into hidden execution just because a plan is precise

### Resources

MCP resources remain read-only.

Resources may expose:

- workspace config
- registry snapshots
- repo context
- repo metadata

Resources must not be used as a write path.

## Architecture

The MCP server remains a thin adapter over the existing engine layer, following the same general pattern as the TUI.

The architectural change in this ADR is about **surface area**, not transport or packaging:

- `repokeeper mcp` remains the stdio-based MCP entrypoint
- the server continues to use typed schemas and structured JSON
- read and planning paths remain valid MCP responsibilities
- narrowing execution-oriented engine entrypoints remains the long-term MCP contract direction

## Skill Guidance

The bundled RepoKeeper skill should continue to prefer MCP when available, but only for inspection and planning flows.

Examples:

- `repokeeper get -o json` -> MCP read tools
- `repokeeper describe <repo>` -> MCP context tools
- `repokeeper reconcile --dry-run` -> MCP planning tool

Examples that remain CLI/TUI-only:

- executing sync
- writing labels or metadata
- adding or removing repositories
- branch switching or checkout

## Consequences

### Positive

- MCP becomes a simpler and more stable adapter surface for agents.
- The product boundary is clearer: agents can inspect and preview, while operators execute through CLI/TUI.
- Safety rules become easier to explain and document.
- Future standalone adapters can depend on a smaller, more deterministic contract.

### Negative

- Some currently implemented MCP mutation paths no longer fit the intended long-term design.
- Deprecation and migration work is required for existing docs and tool surfaces.
- Some workflows will require a CLI/TUI handoff rather than end-to-end execution in MCP.

### Neutral

- The CLI remains the primary human interface.
- The TUI remains the primary interactive operations dashboard.
- MCP remains valuable even without execution because structured inspection and planning are still high-leverage agent workflows.

## Alternatives Considered

### 1. Keep MCP as a mixed read/write surface

Continue exposing both read and mutation tools through MCP, relying on explicit confirmation parameters for safety.

**Rejected because:** it weakens the product boundary between information and execution, increases the blast radius of agent-initiated actions, and makes RepoKeeper's safety model harder to reason about.

### 2. Remove MCP entirely and rely on CLI JSON only

Keep the skill as the only agent integration path and invest only in structured CLI output.

**Rejected because:** MCP still provides typed discovery, better tool ergonomics, and a cleaner agent integration path for inspection and planning workflows.

### 3. Allow execution only for "safe" mutations

Permit a subset of MCP mutations such as label writes or registry updates while forbidding riskier repo-state changes.

**Rejected because:** the resulting boundary is inconsistent and invites repeated case-by-case exceptions. RepoKeeper needs a simpler rule: if it mutates state, it belongs in CLI/TUI.
