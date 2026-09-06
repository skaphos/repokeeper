# Contract: Adapter-Facing Surface Inventory

**Requirements**: FR-008 – FR-016 | **Artifacts**: `cmd/repokeeper/`, `internal/mcpserver/`, `DESIGN.md`

The enumerated set of surfaces an external adapter may depend on. This is the document an author of
`repokeeper-vscode` or `repokeeper-jetbrains` reads instead of the Go source.

An adapter may depend on a surface listed here as `stable`. It may not depend on anything absent
from this list.

## CLI surfaces

Invoked with a machine-readable format (`-o json` / `--format json`). Access is stated **for the
command's default flags**, because three of these commands change class depending on a flag whose
default is easy to misread. Payload names are the proposed envelope field for each surface; the last
column records what ships today.

| Surface | Access (at defaults) | Flag that changes it | Stability | Payload | Envelope today |
| --- | --- | --- | --- | --- | --- |
| `get -o json`, `get repos -o json` | read | — | stable | `repos` | **already enveloped** |
| `describe -o json`, `describe repo -o json` | read | — | stable | `repos` | needs wrapping |
| `scan -o json` | **mutation** | `--write-registry` (**default `true`**) | stable | `results` | needs wrapping |
| `reconcile -o json` (alias `sync`) | **mutation** | `--dry-run` (default `false`) → read | stable | `results` | needs wrapping |
| `repair upstream -o json` | **read** | `--dry-run` (**default `true`**) → `=false` makes it mutation | stable | `results` | needs wrapping |
| `label -o json` | read | `--set` / `--remove` → mutation | stable | `labels` | needs wrapping |
| `version -o json` | read | — | stable | `version` | needs wrapping |

### Flag defaults that invert the obvious reading

Verified against the built binary's `--help`, because guessing these wrong is a live risk for an
adapter deciding what to call unprompted:

- **`scan` writes by default.** `--write-registry` defaults to `true`, so a bare `repokeeper scan`
  is a mutation. Passing `--write-registry=false` makes it read-only.
- **`repair upstream` does *not* write by default.** `--dry-run` defaults to `true`, so a bare
  `repokeeper repair upstream` is read-only. It becomes a mutation only at `--dry-run=false`.
- **`reconcile` writes by default.** `--dry-run` defaults to `false`, the opposite of
  `repair upstream`. The two commands disagree on this default, so an adapter cannot generalise
  from one to the other.

## MCP surfaces

Fourteen registered tools.

| Tool | Access | Stability |
| --- | --- | --- |
| `list_repositories` | read | stable |
| `get_repository_context` | read | stable |
| `get_workspace_config` | read | stable |
| `build_workspace_inventory` | read | stable |
| `select_repositories` | read | stable |
| `get_repo_metadata` | read | stable |
| `get_authoritative_paths` | read | stable |
| `get_related_repositories` | read | stable |
| `plan_sync` | **read** | stable |
| `scan_workspace` | **mutation** | stable |
| `execute_sync` | mutation | stable |
| `set_labels` | mutation | stable |
| `add_repository` | mutation | stable |
| `remove_repository` | mutation | stable |

### Two classifications adapters get wrong

Called out because the tool name misleads in both directions:

- **`plan_sync` is `read`.** It plans a mutation but is always dry-run and never mutates. Safe to
  call automatically — on a timer, on window focus — without a user gesture.
- **`scan_workspace` is `mutation`.** It reads like a query but **updates the registry**. It requires
  the same explicit user gesture as `execute_sync`.

## Explicitly NOT contractual

Per ADR-0006 and FR-010. An adapter depending on any of these is depending on something free to
change in a patch release:

- Human-oriented `table` and `wide` output, including column order, widths, alignment and colour.
- Prose messages, hints, and any human-readable text not carried in a documented machine-readable
  field.
- Log and debug output on stderr.
- **Every package under `github.com/skaphos/repokeeper/v2/internal/...`.** ADR-0006 rejects requiring
  adapters to import internals; correspondingly, nothing under `internal/` is a supported consumption
  path, and the `/v2` module's internal package layout may change without notice.
- Exit-code *values* beyond the zero/non-zero distinction Principle XII guarantees.

## Read/mutation guarantees

- No surface classed `read` writes to the registry, the config, or any working tree (FR-013).
- Under a read-only workspace mount, every `read` surface succeeds (FR-016).
- Under a read-only workspace mount, every `mutation` surface refuses with a message naming both the
  cause and the remedy, and **remains advertised** rather than disappearing from the surface list
  (FR-015). A tool that vanishes under a read-only mount is a silently reduced surface; one that
  explains why it cannot run is honest. This is existing behaviour in `internal/mcpserver/readonly.go`,
  made contractual here.

## Drift

The inventory above and the surfaces present in code must not diverge. A test fails when they do
(FR-011), extending the pattern `TestDesignDocNamesStatusJSONAPIVersion` already establishes for the
single version constant. A surface added to the code without a corresponding entry here is a build
failure, not a documentation backlog item.
