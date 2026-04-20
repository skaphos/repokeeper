# ADR-0008: MCP Install Tooling for Supported Agent Runtimes

**Status:** Proposed
**Date:** 2026-04-20
**Author:** Shawn Stratton

## Context

RepoKeeper ships two integration surfaces for agent runtimes:

1. A bundled **skill** (Markdown prompt file), installed via `repokeeper skill install` into per-runtime skill directories (`~/.claude/skills/`, `~/.agents/skills/`, `~/.config/opencode/skills/`). ADR-0001 positions the skill as a *CLI fallback* for runtimes that do not support MCP.
2. A built-in **MCP server** (`repokeeper mcp`, stdio transport), which ADR-0001 designates as the *preferred* agent integration surface.

The skill installer already auto-detects present runtimes, writes the correct file to the correct directory, and supports a target flag for scoping. The MCP server has no equivalent installer. Users must hand-edit three different config files in two different formats to register the MCP server, as currently documented in `docs/mcp-setup.md`. This produces two concrete problems:

- **Parity gap.** `repokeeper skill install` is a one-liner; MCP setup is multi-step manual JSON/TOML editing per runtime. The preferred surface is harder to install than the fallback.
- **Doc drift and incorrect paths.** `docs/mcp-setup.md:21` currently instructs users to write the MCP config to `~/.claude/settings.json`. Anthropic's own docs (<https://github.com/ericbuess/claude-code-docs/blob/main/docs/mcp.md>) state that user-scope MCP servers are stored in `~/.claude.json`, with project-scope in `.mcp.json`. The repo's manual instructions are wrong in a way that silently prevents the server from loading on a fresh install.

The broader direction in ADR-0001 is *MCP-first, skill as narrow fallback*. If the skill is the fallback, its dedicated install command is a parity wart: the primary path should be MCP, installed with a single command, and the skill should be demoted from a first-class CLI verb.

## Decision

Add `repokeeper install`, `repokeeper install list`, and `repokeeper uninstall` as the unified integration path for all supported runtimes. Remove `repokeeper skill install` / `repokeeper skill uninstall`. The bundled skill content (`docs/skills/repokeeper/SKILL.md`) remains in the repo for anyone who needs a manual skill drop, but it is no longer a CLI-installable product.

### Supported runtimes

The installer supports the same three runtimes the skill installer currently targets: **Claude Code**, **Codex** (OpenAI's `codex` CLI, formerly referred to as `openai` in the skill installer), and **OpenCode**. Cursor and Windsurf are intentionally out of scope — their MCP configuration is UI-driven with no canonical file on disk we can safely rewrite. `--manual` (Section 3) covers those runtimes by printing the config snippets verbatim.

Per-runtime config file conventions, sourced from the upstream docs and verified for this ADR:

| Runtime | User scope | Project scope | Format | Key | Entry shape |
|---|---|---|---|---|---|
| Claude Code | `~/.claude.json` | `<cwd>/.mcp.json` | JSON | `mcpServers.<name>` | `{"command": "...", "args": [...]}` |
| Codex | `~/.codex/config.toml` | **not supported** | TOML | `[mcp_servers.<name>]` | `command = "..."`, `args = [...]` |
| OpenCode | `${OPENCODE_CONFIG_DIR:-${XDG_CONFIG_HOME:-~/.config}/opencode}/opencode.json` | `<cwd>/opencode.json` | JSON | `mcp.<name>` | `{"type": "local", "command": [argv...], "enabled": true}` |

Two runtime quirks the adapter layer must absorb:

- Codex has no project-scope MCP config. `--scope project --codex` hard-errors with a message directing the user to split the invocation.
- OpenCode uses a different top-level key (`mcp`, not `mcpServers`) and a different command representation (`command` is a single argv array, not a command string plus separate args list). The adapter serializes accordingly.

### 1. CLI surface

Three top-level commands:

```
repokeeper install      [--claude] [--codex] [--opencode] [--scope user|project] [--command PATH] [--manual [=claude|codex|opencode|all]]
repokeeper install list [--scope user|project] [--json]
repokeeper uninstall    [--claude] [--codex] [--opencode] [--scope user|project] [-y]
```

`install list` is a subcommand of `install` (cobra supports a command with both `RunE` and children); `uninstall` is a sibling top-level verb so that it reads symmetrically with `install` rather than as `install remove`.

**Runtime flag resolution:**

- **No runtime flags set** → auto-detect runtimes present on disk, operate on those. If none detected, exit non-zero with a message that points at `--manual`.
- **One or more runtime flags set** → operate *only* on the named runtimes, and proceed even if the runtime's config directory does not yet exist (we create it). The flag is explicit user intent, not a detection filter. This is the escape hatch for "I am about to install Codex, configure its MCP first."
- Auto-detection heuristic (per runtime):
  - Claude: `~/.claude.json` exists **or** `~/.claude/` directory exists.
  - Codex: `~/.codex/` directory exists.
  - OpenCode: `$OPENCODE_CONFIG_DIR` is set **or** `${XDG_CONFIG_HOME:-~/.config}/opencode/` directory exists.

**Scope handling:**

- `--scope user` (default) writes to the user-scope path in the table above.
- `--scope project` writes to a project-scope file in `os.Getwd()` (no upward repo-root search — simple, predictable, matches Claude Code's own `claude mcp add --scope project` behavior).
- `--scope project --codex` → hard error: `codex does not support project-scoped MCP config; re-run without --scope project or install codex at user scope separately`.
- `--scope project` without an ancestor `.git` directory → warn but proceed (a bare `.mcp.json` in a non-repo directory is still valid for anyone who points their tooling at that cwd).

**Binary path (`command` field):**

- Default: `os.Executable()` — the raw path the current process was started from, without symlink resolution. This is the path the user's shell used, which is usually the stable entry point maintained by their package manager: Homebrew's `/opt/homebrew/bin/repokeeper` symlink (swung atomically by `brew upgrade`), `mise`/`asdf` shims, or the raw binary from `go install`. Resolving through symlinks would write version-specific Cellar paths that go stale on the next upgrade — the opposite of what we want.
- Override: `--command /custom/path` — written verbatim. No existence check (user may be staging the config ahead of a binary install; we warn but do not fail).
- `args` is always `["mcp"]` in v1. Users who want to append `--config` or `--log-file` hand-edit; a flag for this is out of scope.

**`--manual`:**

Prints the MCP config snippet(s) to stdout instead of writing any file. Accepts an optional target (`--manual=claude`, `--manual=codex`, `--manual=opencode`, `--manual=all`); bare `--manual` is equivalent to `--manual=all`. Detection is skipped in this mode. This covers Cursor, Windsurf, and any future runtime we have not written an adapter for.

**Overwrite / idempotency:**

- Our key absent → write.
- Our key present and byte-identical to what we'd write → no-op, status `unchanged`.
- Our key present and different → overwrite without prompt, status `updated`, print a one-line diff of the `command` and `args` fields. The common cases are a user migrating between install methods (`go install` → Homebrew or vice versa), relocating the binary, or re-running with an explicit `--command` override; prompting would create friction without safety value. `-y` is a no-op for `install` (already non-interactive) and only affects `uninstall`, which prompts once per invocation by default.
- Non-`repokeeper` entries and top-level keys are preserved verbatim.

**`install list` output:**

Table form by default:

```
AGENT      SCOPE      PATH                                  STATUS              COMMAND
claude     user       ~/.claude.json                        registered          /opt/homebrew/bin/repokeeper mcp
codex      user       ~/.codex/config.toml                  registered (stale)  /old/path/repokeeper mcp
opencode   user       ~/.config/opencode/opencode.json      not registered      —
```

`stale` indicates the registered `command` differs from the current `os.Executable()` — a hint that re-running `install` will fix it. `install list` always iterates over every supported adapter (including ones that are neither auto-detected nor registered) so the table is a complete view of integration state, not a filtered subset. `--scope` filters which scope's config file is inspected (default user; `--scope project` inspects cwd project-scope files). `--json` emits a structured shape for scripting; other commands do not accept `--json` in v1.

**Exit codes:**

- `0` — every targeted runtime succeeded (including `unchanged`).
- `1` — one or more targeted runtimes failed; others were still attempted.
- `2` — usage error (e.g., `--scope project --codex`, unknown target in `--manual`).

### 2. Architecture: per-runtime adapter interface

A new package `internal/mcpinstall/` with a `Runtime` interface:

```go
type Runtime interface {
    Name() string                                      // "claude", "codex", "opencode"
    Detect() (bool, error)                             // path-existence heuristic
    ConfigPath(scope Scope) (string, error)            // returns err for unsupported scopes (codex project)
    ReadEntry(path string) (Entry, bool, error)        // bool=present, error=parse/io
    WriteEntry(path string, e Entry) error             // merge + atomic write
    RemoveEntry(path string) (bool, error)             // bool=was-present
}

type Entry struct {
    Command string
    Args    []string
}

type Scope int
const (
    ScopeUser Scope = iota
    ScopeProject
)
```

One adapter per runtime (`claude.go`, `codex.go`, `opencode.go`), each responsible for:

- parsing its specific format (`encoding/json` for Claude/OpenCode, `github.com/pelletier/go-toml/v2` for Codex),
- serializing its specific shape (`mcpServers.<name>` vs `[mcp_servers.<name>]` vs `mcp.<name>` with argv-array command),
- writing atomically (`<path>.tmp.<pid>` → `os.Rename`, symlink-aware, preserving file mode if the file pre-existed; otherwise `0600` for user-scope, `0644` for project-scope).

The same `runtime.go` file also exposes the slice of all adapters (`All()`) and lookup helpers (`ByName`, `BySelection(flags)`) — they are thin enough not to warrant a separate file and keep the package's Go-source count within the project's per-phase cap.

**Rejected alternatives for the architecture:**

- A data-table mapping runtime → `{path, format, key}` plus generic `jsonMcpMerge`/`tomlMcpMerge` helpers. Tempting for brevity, but OpenCode's `mcp`-not-`mcpServers` key and argv-array `command` force special-case branches into the primitive from day one. The adapter boundary stays cleaner with one file per runtime.
- A canonical descriptor plus per-runtime serializer pipeline. Over-abstracted for three runtimes.

**Refused edge cases (explicit, v1):**

- **Malformed JSON/TOML** → hard error with file path and parse message. We refuse to overwrite a config we cannot round-trip, because silent rewrites would destroy unrelated user edits.
- **`.jsonc` with comments** (OpenCode) → refuse with a message asking the user to rename to `.json` or edit manually. Round-tripping comments is a large rabbit hole; defer until demanded.
- **Concurrent `install` invocations** → last writer wins. Not worth a file lock in v1.

### 3. Documentation changes

- Rewrite `docs/mcp-setup.md` so `repokeeper install` is the headline path. The manual snippets become a "Manual configuration (unsupported runtimes)" appendix, covering Cursor/Windsurf and anything else a user might want to configure by hand.
- Fix the incorrect `~/.claude/settings.json` reference in `docs/mcp-setup.md:21`. Replace with `~/.claude.json` (user scope) and `./.mcp.json` (project scope).
- Update `README.md` to reference `repokeeper install` instead of `repokeeper skill install`.
- Deprecation note in CHANGELOG.md via the release-please flow: `repokeeper skill install` and `repokeeper skill uninstall` removed; the bundled skill file is still present at `docs/skills/repokeeper/SKILL.md` for manual copying.

## Consequences

### Positive

- The preferred integration surface (MCP) is finally easier to install than the fallback (skill). Parity gap closed.
- `docs/mcp-setup.md`'s incorrect Claude Code config path (`~/.claude/settings.json`) is fixed; users who follow the docs actually get a working MCP server.
- One command that works across all supported runtimes replaces three different hand-editing workflows.
- A `Runtime` interface is the natural extension point for Cursor / Windsurf once their MCP config surfaces stabilize.
- `install list` becomes a diagnostic: users can see where RepoKeeper is registered, whether any registrations are stale, and the exact binary path in use.

### Negative / risks

- **Breaking change:** `repokeeper skill install` and `repokeeper skill uninstall` are removed. RepoKeeper is pre-1.0 (last release 0.7.1), so this is acceptable under conventional semver, but anyone scripting against those commands has to update. Documented in the release notes and CHANGELOG.
- Writing to shared config files (`~/.claude.json`, `~/.codex/config.toml`, `~/.config/opencode/opencode.json`) is inherently risky — a bug in the merge path could damage unrelated entries. Mitigated by: strict parse-first policy, atomic rename, fixture-based golden tests for the merge output, and an explicit `install list` command users can run before/after to verify state.
- The ADR does not cover Cursor or Windsurf. Users on those runtimes are no better served than today; they continue to hand-edit, now via `--manual` instead of the `docs/mcp-setup.md` appendix. This is an accepted gap; the `Runtime` interface is ready for their adapters later.
- Implementation adds a new TOML dependency (`github.com/pelletier/go-toml/v2`) if not already in the module graph. Low risk, maintained library, wide use.

### Neutral

- The skill content (`docs/skills/repokeeper/SKILL.md`) is preserved in the repo. Nothing forces a user off the skill-only workflow; they just copy the file themselves.
- The `repokeeper mcp` command (server start, stdio transport) is unchanged. Its behavior is orthogonal to install tooling.

## Alternatives Considered

### 1. Keep `skill install/uninstall`, add parallel `mcp install/uninstall/list`

Two parallel CLI groups (`skill` and `mcp`), both with install/uninstall/list. Maximum surface area and parallelism.

**Rejected because:** it entrenches the skill command as a first-class product at exactly the moment ADR-0001 is demoting the skill to "fallback only." Two install paths also force users to decide which to run. Today, nearly every supported runtime has MCP support — the skill installer is solving a problem that barely exists, and doubling the CLI surface to support it indefinitely is the wrong trade.

### 2. `repokeeper install` as a leaf verb, with a separate top-level `repokeeper status` for state

Flat tree: `install`, `uninstall`, `status`. No `install list` subcommand.

**Rejected because:** the user preference is `install list`, and nesting the state query under `install` (as a subcommand) reads as "list what install touches," which is accurate. `status` at the top level either collides with a future "workspace status" verb or forces a longer name like `integrations`/`registrations` that adds no clarity. The one awkwardness in the chosen tree is `install remove` (which is why we split to a separate top-level `uninstall` verb); otherwise `install list` is the cleanest phrasing of the query.

### 3. Data-table architecture instead of per-runtime adapters

Already discussed in the Decision. Rejected because OpenCode's divergent shape (`mcp` key, argv-array `command`) forces case branches into the "generic" merge primitive on day one, and the adapter pattern is the natural unit of test isolation.

### 4. Silent fallback for `--scope project --codex` to user scope

Instead of erroring, silently write the Codex entry at user scope when `--scope project` is requested.

**Rejected because:** it violates principle of least surprise. A user who asks for project scope and gets user scope without notice would reasonably assume their project-scope install is clean. An explicit error keeps the user in the loop about where their config landed.

### 5. Prompt before every overwrite

Rather than overwriting byte-different entries in place, prompt the user.

**Rejected because:** the dominant reason to re-run `install` is fixing a stale `command` path after a `brew upgrade` or a `go install` into a new `$GOPATH/bin`. Prompting in that flow would be pure friction. `install list`'s `stale` marker already tells the user when a re-install will change something, so they can opt into running with eyes open.

## Verification plan

1. On a clean Linux host with Codex and OpenCode installed and no prior `repokeeper` MCP registration:
   - `repokeeper install` → both configs registered, `install list` shows both as `registered`.
   - Re-run `repokeeper install` → both show `unchanged`, no disk writes.
   - Run `install` from a differently-pathed copy of the binary (e.g., `/tmp/repokeeper-alt`), re-run `install` → both show `updated` with the new `command`, previous entries replaced.
   - `repokeeper uninstall -y` → both keys removed, `install list` shows `not registered`, non-`repokeeper` entries (if any) still present.
2. On a macOS host with Claude Code installed via Homebrew:
   - `repokeeper install --claude` → `~/.claude.json` gets the `mcpServers.repokeeper` entry with the stable `/opt/homebrew/bin/repokeeper` shim path (not a version-specific Cellar path).
   - Run `brew upgrade repokeeper` → the registered `command` path is still valid (shim swung, no re-install needed).
   - Start Claude Code, confirm the MCP server loads.
3. `repokeeper install --scope project --codex` returns exit code `2` with the documented error message.
4. `repokeeper install --manual` prints all three format blocks; `--manual=codex` prints only the TOML block.
5. Hand-corrupt a target config with a truncated JSON brace and run `install` → refusal with parse error, original file unmodified.
6. `go test ./internal/mcpinstall/...` and `go test ./cmd/repokeeper/...` pass with coverage over the fixture matrix (empty, other-servers, existing-match, existing-stale, malformed, symlinked, `.jsonc`).

## Implementation plan

Phased to stay within the project's 5-files-per-phase guardrail and keep each PR reviewable.

**Phase 1 — Adapter layer (no CLI wiring).**
- `internal/mcpinstall/runtime.go` — `Runtime` interface, `Entry` struct, `Scope` enum, adapter registry (`All()`, `ByName`, `BySelection`), and shared helpers.
- `internal/mcpinstall/atomic.go` — atomic-write helper (tmp + rename, symlink-aware, mode preservation).
- `internal/mcpinstall/claude.go` — Claude Code adapter (JSON, `mcpServers.<name>`, user + project scopes).
- `internal/mcpinstall/codex.go` — Codex adapter (TOML, `[mcp_servers.<name>]`, user scope only; project path returns error).
- `internal/mcpinstall/opencode.go` — OpenCode adapter (JSON, `mcp.<name>`, argv-array command, user + project scopes, `OPENCODE_CONFIG_DIR` support, `.jsonc` refusal).
- Fixture-based unit tests alongside each adapter, fixtures under `internal/mcpinstall/testdata/` (per-adapter: empty, other-servers, existing-match, existing-stale, malformed, symlinked, with-comments).

Five Go source files, respecting the project's phase cap. Tests and `testdata/` fixtures are additive and do not count against the cap. Phase 1 ships as a library with no user-visible CLI change.

**Phase 2 — CLI surface.**
- `cmd/repokeeper/install.go` — `install` command + `install list` subcommand; flag parsing; auto-detect and selection; `--manual` output; `--json` for `install list`.
- `cmd/repokeeper/uninstall.go` — `uninstall` command; shared selection helpers with `install`.
- Remove `cmd/repokeeper/skill.go` and delete `internal/skillbundle/` if fully unreferenced (the bundled skill file stays at `docs/skills/repokeeper/SKILL.md` as pure content; the embed and CLI go away).
- Integration tests under `cmd/repokeeper/install_test.go` using `t.TempDir()` as a fake HOME and `OPENCODE_CONFIG_DIR` override.

Phase 2 is the breaking change: `repokeeper skill install/uninstall` disappears.

**Phase 3 — Documentation.**
- Rewrite `docs/mcp-setup.md`: `repokeeper install` headline path; manual snippets demoted to "Unsupported runtimes" appendix; fix the `~/.claude/settings.json` → `~/.claude.json` reference.
- Update `README.md` to reference the new command.
- Update `docs/skills/README.md` to note that the bundled skill is no longer CLI-installed; point at `docs/mcp-setup.md` and the manual drop location.
- CHANGELOG entry captured via release-please from the conventional-commit messages; no direct edit.

**Phase 4 — Manual verification on both platforms (per the verification plan above).** No code; this is the pre-release sanity pass.

**Out of scope for this ADR (future work):**

- Cursor / Windsurf adapters (no canonical flat-file MCP config today).
- Project-root auto-detection for `--scope project`.
- JSONC round-trip for OpenCode's `opencode.jsonc`.
- `--args` override to append flags to `repokeeper mcp` invocation.
- A `doctor`-style health check that actually starts the server and verifies it responds.
