# RepoKeeper Agent Skill

This repository ships an OpenCode-compatible agent skill for RepoKeeper at:

`docs/skills/repokeeper/SKILL.md`

> **Prefer MCP when your runtime supports it.** `repokeeper install` registers the MCP server with Claude Code, Codex, and OpenCode — the skill file described here is a CLI fallback for runtimes without MCP support, and it remains the preferred guide for CLI/TUI-driven execution workflows. The CLI installer for the skill (`repokeeper skill install/uninstall`) was removed in [ADR-0008](../adr/0008-mcp-install-tooling.md); installation is now a manual copy step.

The current MCP server also exposes some execution and mutation tools, so treat any scan, sync, label, add/remove, or other write-capable MCP call as an explicit state-changing operation. See [docs/mcp-setup.md](../mcp-setup.md) for the `repokeeper install` reference and the full MCP tool catalog.

## Manual installation

Copy the skill folder into one of the user-scope skill directories your runtime expects:

- OpenCode: `~/.config/opencode/skills/repokeeper/` (or `${OPENCODE_CONFIG_DIR}/skills/repokeeper/`)
- Claude Code: `~/.claude/skills/repokeeper/`
- Agent Skills interoperable path (OpenAI, Codex): `~/.agents/skills/repokeeper/`

Example:

```bash
mkdir -p ~/.config/opencode/skills
cp -R docs/skills/repokeeper ~/.config/opencode/skills/
```

After installation, restart the agent runtime so it can discover the new skill. Refreshing the skill on a new release is a simple re-copy — there is no embedded bundle to resync against the binary anymore.
