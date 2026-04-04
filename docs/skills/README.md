# RepoKeeper Agent Skill

This repository ships an OpenCode-compatible agent skill for RepoKeeper at:

`docs/skills/repokeeper/SKILL.md`

Use MCP when available for inspection and side-effect-free planning. Use the skill as the CLI fallback and for execution-oriented workflows that intentionally stay outside the MCP surface.

## User-scope installation

The easiest path is to let RepoKeeper install or update the bundled skill for you:

```bash
repokeeper skill install
```

Behavior:

- with no target, installs into every existing supported user-scope skill directory it can find
- `repokeeper skill install claude` installs into `~/.claude/skills/`
- `repokeeper skill install openai` and `repokeeper skill install codex` install into `~/.agents/skills/`
- `repokeeper skill install opencode` installs into `~/.config/opencode/skills/`
- the skill content is bundled into the compiled RepoKeeper binary, so installs and updates do not depend on a checked-out source tree

To remove the bundled skill again:

```bash
repokeeper skill uninstall
```

Manual installation is also possible by copying the bundled skill folder.

Install the skill by copying the `repokeeper/` folder into one of these user-scope skill directories:

- OpenCode: `~/.config/opencode/skills/repokeeper/`
- Claude Code compatible: `~/.claude/skills/repokeeper/`
- Agent Skills interoperable path: `~/.agents/skills/repokeeper/`

Example:

```bash
mkdir -p ~/.config/opencode/skills
cp -R docs/skills/repokeeper ~/.config/opencode/skills/
```

After installation, restart the agent runtime so it can discover the new skill.
