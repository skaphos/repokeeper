# ADR-0017: Retire the TUI

**Status:** Accepted
**Date:** 2026-09-02
**Author:** Shawn Stratton

## Context

RepoKeeper has shipped three operator surfaces:

- the **CLI** (`repokeeper <subcommand>`), the original and complete surface;
- the **MCP server** ([ADR-0001][adr0001]), the typed surface agents use for inspection, planning,
  and a bounded set of explicit mutations;
- the **TUI**, a Bubble Tea application launched when `repokeeper` is invoked with no subcommand on
  an interactive terminal. `DESIGN.md` §5.2 describes it as a k9s-style operations dashboard and
  `TASKS.md` Milestone 9 tracks it as "phase 2" work.

ADR-0001 through ADR-0005 draw their boundaries in terms of "CLI and TUI" as the execution
surfaces and MCP as the information-and-planning surface. That framing assumed the TUI would become
the "primary interactive operations dashboard" (ADR-0001, Consequences).

That did not happen. Since the MCP server and `repokeeper install` landed, day-to-day operation has
moved to agent-driven MCP calls for inspection and planning, with the CLI for execution. The TUI's
remaining backlog (Milestone 9, the #288 branch hygiene view, #295 sync history, batch progress
streaming, keyboard parity with every CLI flag) is a branch of work that is not going to be
finished, and an unfinished interactive surface is worse than none: it is the first thing a new
user sees when they type `repokeeper`, and it is the least complete thing in the binary.

What the TUI costs today, measured on this repository at the head of `chore/deps-go1.27`
(go 1.27.1, release flags from `.goreleaser.yaml`: `CGO_ENABLED=0 -trimpath -ldflags '-s -w'`,
`linux/amd64`):

| Measure | With TUI | Without TUI | Delta |
| --- | --- | --- | --- |
| Binary size | 10,580,128 B (10.1 MiB) | 8,933,536 B (8.5 MiB) | **-1,646,592 B (-15.6%)** |
| Modules linked into the binary (`go version -m`) | 33 | 17 | **-16** |
| Direct dependencies in `go.mod` | 14 | 12 | -2 (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`) |
| Lines in `internal/tui/` | 6,718 (3,632 source + 3,086 test) | 0 | **-6,718** |

The sixteen modules that leave with it: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
`charmbracelet/colorprofile`, `charmbracelet/ultraviolet`, `charmbracelet/x/{ansi,term,termios,windows}`,
`clipperhouse/displaywidth`, `clipperhouse/uax29/v2`, `lucasb-eyer/go-colorful`,
`mattn/go-runewidth`, `muesli/cancelreader`, `rivo/uniseg`, `xo/terminfo`, `golang.org/x/exp`.
Every one of them is TUI-only; nothing in the CLI or MCP paths imports them.

That is one sixth of the binary, half of the supply chain, and a fifth of the non-test source
tree carrying a surface with no users and an open-ended backlog.

## Decision

**RepoKeeper ships no TUI.**

- `internal/tui/` is deleted in full. No file is kept "in case".
- `repokeeper` with no subcommand prints help on every terminal, interactive or not. There is no
  interactive fallback, no prompt, and no `tui` subcommand stub that says "removed".
- `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2` and their transitive closure leave
  `go.mod`. `internal/termstyle` (ANSI colour for CLI tables) and `golang.org/x/term` (terminal
  detection for table width and in-place sync progress) stay; they serve the CLI.
- Documentation that describes the TUI as a surface (`README.md`, `DESIGN.md` §5.2,
  `docs/skills/repokeeper/SKILL.md`, `docs/mcp-setup.md`, `AGENTS.md`,
  `.github/copilot-instructions.md`) is corrected in the same change. `TASKS.md` Milestone 9 is
  closed as withdrawn, not completed.
- Existing ADRs are immutable and are not edited. Where ADR-0001 through ADR-0005, ADR-0010 and
  ADR-0014 say "CLI and TUI" or "CLI/TUI", read **CLI**. The boundary they draw is unchanged:
  execution belongs to the operator surface, MCP inspects and plans. There is now one operator
  surface instead of two.

The surfaces going forward are exactly two: **CLI** for execution by a human, **MCP** for
inspection and planning by an agent (plus the explicit mutation tools ADR-0001 already bounds).

## Rationale

### The interaction model the TUI was built for has been replaced, not deferred

The TUI's design goal (`DESIGN.md` §5.2) was a keyboard-first dashboard so an operator could
filter, inspect and act on many repos without composing CLI invocations. The MCP server delivers
that outcome through a different mechanism: the operator states intent in prose, the agent
composes `list_repositories`, `select_repositories`, `plan_sync`, `get_repository_context`, and
the operator confirms execution. Filtering by label, path, branch and tracking state, the core
of the TUI's value, is already exposed as `select_repositories` and the CLI label selectors
(`get repos --selector`), both of which the TUI was consuming rather than owning.

Every TUI action routed through `internal/engine` by design (DESIGN.md §5.2, "no independent
business logic"). Removing the TUI removes presentation only. No engine capability is lost.

### Half-built surfaces are a liability, not an option value

An interactive surface either earns trust by being complete or loses it by being almost complete.
Milestone 9 has been open since the TUI first shipped; the sync progress view, delete and reset
confirmations, and metadata editor each exist but none has the parity with CLI flags (`--yes`,
`--dry-run`, `--selector`, JSON output) that ADR-0003 and ADR-0004 require of an execution
surface. Keeping it means either finishing it, which no one is going to do, or documenting which
half works, which is worse than the honest answer.

### The cost is real and concentrated

The table above is the whole argument in numbers: 16 of 33 linked modules, 15.6% of the binary,
and 6,718 lines exist for one code path reachable only by typing `repokeeper` with no
arguments on a TTY. ADR-0016 declined a self-updater over a 128% size increase; the same
discipline says do not carry 15.6% for a surface nobody drives. Fewer modules is also fewer
Dependabot PRs, a smaller `govulncheck` surface, and a shorter `third_party_licenses/` inventory.

### Deleting beats deprecating

A deprecation window would mean shipping a "this will be removed" banner in the TUI for a release
or two. That protects users who depend on it. There is no evidence of any. Every open issue that
names the TUI (#282, #284, #288, #290, #291, #295, #299) is a feature request from the
maintainer's own backlog, two of them already labelled `parked`; none is a defect report or a
usage question from a user. The only agent-facing reference is one line in `SKILL.md`, which this
change removes. A window would delay the dependency and size wins for no one's benefit.

## Consequences

**Positive**

- One operator surface to keep at parity with the engine instead of two. ADR-0003's and
  ADR-0004's "CLI and TUI own execution" contracts collapse to a single implementation.
- Binary shrinks by ~1.6 MB; module graph halves; every dropped module is one fewer
  Dependabot, licence-inventory and vulnerability-scan entry.
- `cmd/repokeeper/root.go` loses its config-loading and registry-presence branch; `repokeeper`
  with no arguments becomes a pure Cobra help invocation with no filesystem access.
- The "inherently low testability" allowance `TASKS.md` grants `internal/tui/` goes away with the
  package; every remaining non-`cmd/` package holds the normal coverage threshold.

**Negative / accepted**

- **User-visible removal.** Anyone who typed `repokeeper` expecting the dashboard gets help text.
  This is a breaking change in Conventional Commit terms and is committed as one so release
  tooling cuts the next major version. That is the correct signal; the alternative is a silent
  removal in a minor release.
- The TUI's label editor (`l`) and repo-local metadata editor (`i`) go with it. Every write path
  they fronted still exists on the CLI: `label` (and MCP `set_labels`) for labels, `index`
  (interactive proposal) and `index --write` for repo-local metadata, and `edit` for registry
  entries. No write path is lost, only the modal-form presentation of three that remain.
- The seven open TUI feature issues (#282, #284, #288, #290, #291, #295, #299) are closed as
  not planned with a pointer to this ADR. Where one of them names an engine capability rather
  than a TUI view (#282 and #288 both stem from ADR-0002 and ADR-0014 branch-hygiene work), the
  engine half of the request stays tracked by the CLI-side issue (#280) or a new one.
- `DESIGN.md` §5.2 and `TASKS.md` Milestone 9 become historical. They are rewritten to say the
  surface was withdrawn under this ADR rather than deleted outright, so the design intent is still
  discoverable.

## Reversal conditions

This decision is revisited only if **both** hold:

1. A concrete operator workflow is identified that neither `CLI + shell` nor `MCP + agent` can
   serve, and the gap is the interaction model rather than a missing engine capability (a missing
   capability is an engine change, not a TUI).
2. Someone commits to shipping and maintaining that surface to full CLI-flag parity as defined by
   ADR-0003 and ADR-0004 before it becomes the default no-argument entry point.

A reinstated TUI would be a new ADR, a new package, and would not resurrect `internal/tui/` from
history as a starting point; the engine API has moved on and the old code encodes the old
boundaries.

## Alternatives Considered

### Keep the TUI and finish Milestone 9

**Rejected because:** the remaining work (branch hygiene view, batch progress, full flag parity)
is a multi-release effort for a surface whose intended user now drives the tool through an agent.
The opportunity cost is engine and MCP work that the actual workflow needs.

### Keep the TUI as-is, move it behind an explicit `repokeeper tui` subcommand

**Rejected because:** it removes the TUI from the default path but keeps every cost in the table
above. It also normalises shipping an unfinished surface as long as it is opt-in, which is the
outcome this ADR exists to prevent.

### Deprecate for one release cycle, then remove

**Rejected because:** a deprecation window protects existing users, and there are none to
protect. See "Deleting beats deprecating".

### Replace the TUI with a thinner interactive picker (fzf-style) for repo selection

**Rejected because:** repo selection is already solved by label selectors on the CLI and
`select_repositories` on MCP, and a shell user has `fzf` itself. RepoKeeper should not re-implement
the shell's interactive primitives.

## Links

- [ADR-0001: MCP server][adr0001] — defines MCP as inspection/planning and "CLI/TUI" as execution;
  read "CLI".
- [ADR-0003: Sync policy and execution modes](0003-sync-policy-and-execution-modes.md),
  [ADR-0004: Prune workflow boundaries](0004-prune-workflow-boundaries.md) — execution-surface
  parity requirements the TUI never met.
- [ADR-0016: No self-update subcommand](0016-no-self-update-subcommand.md) — the precedent for
  declining binary weight that does not pay for itself.
- `DESIGN.md` §5.2 and `TASKS.md` Milestone 9 — the withdrawn design.

[adr0001]: 0001-mcp-server.md
