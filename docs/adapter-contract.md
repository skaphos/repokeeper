# RepoKeeper Adapter Contract

The stable, machine-readable surface external tools may build against — IDE plugins such as
`repokeeper-vscode` and `repokeeper-jetbrains`, CI glue, or any script that drives RepoKeeper from
outside.

Everything here is consumable without importing a single RepoKeeper Go package. That is deliberate:
[ADR-0006](adr/0006-adapter-contract-stability.md) rejects requiring adapters to import internals,
because it couples external repositories to unstable implementation details.

The mechanism is recorded in [ADR-0018](adr/0018-adapter-contract-envelope.md); the per-field
schemas live in [DESIGN.md](../DESIGN.md) §6.3–§6.4.

---

## 1. Check compatibility from the first response

Every adapter-facing response carries `apiVersion`. There is no handshake call.

```bash
repokeeper get -o json
```

```jsonc
{
  "apiVersion": "skaphos.io/repokeeper/v1",
  "generated_at": "2026-09-05T23:54:33Z",
  "repos": []
}
```

Your compatibility check:

1. Call any read surface.
2. Read `apiVersion`.
3. If the major does not match what you were built against, degrade — do not guess.

The value is identical across every surface, CLI and MCP alike, so this is one check rather than one
per command.

## 2. Ignore fields you do not recognise

Additive changes do not bump `apiVersion`, so new fields **will** appear within `v1`. An adapter that
errors on an unrecognised field turns every routine RepoKeeper release into an outage on your side.

Parse permissively.

## 3. The envelope

Every successful adapter-facing response is a JSON **object** carrying `apiVersion` and a named
primary payload. No surface emits a bare top-level array. Some reports include supplemental fields,
such as `get --only diverged`'s `diverged` advice array.

| Field | Notes |
| --- | --- |
| `apiVersion` | Always present. `skaphos.io/repokeeper/v1` as of 2.0.0. |
| `generated_at` | Present only where the response reports observed state at a point in time. Omitted — not zero — when not applicable. |
| *payload* | Named per surface (`repos`, `results`, `config`, …). |

Required collection payloads are `[]`, never `null` and never omitted. Optional nested collections
may be omitted when empty; their absence means no values. Object-valued optional metadata may be
`null` only where the inventory explicitly says so. Field names use snake_case except the schema
marker `apiVersion`. Map key order is not meaningful. Repository arrays retain the command's
selection order; consumers should match records by checkout ID or path rather than array position.

> **Migrating from 1.x:** the action commands previously emitted a bare top-level array.
> `reconcile -o json` now returns `{"apiVersion": …, "results": [...]}` — read `.results` instead of
> the top-level value. There is no compatibility flag; see ADR-0018.

## 4. Surface inventory

An adapter may depend on anything listed here. It may not depend on anything absent from it.

### CLI

Invoked with `-o json` / `--format json`. **Access is stated at the command's default flags**,
because three of these change class depending on a flag default that is easy to misread.

| Surface | Access (at defaults) | Flag that changes it | Payload |
| --- | --- | --- | --- |
| `get -o json`, `get repos -o json`, `get repo -o json` | read | `--reconcile-remote-mismatch` plus `--dry-run=false` → mutation | `repos` |
| `describe -o json`, `describe repo -o json` | read | — | `repo` |
| `scan -o json` | **mutation** | `--write-registry` (default `true`) | `repos` |
| `reconcile -o json`, `reconcile repos -o json`, `reconcile repo -o json` | **mutation** | `--dry-run` (default `false`) → read | `results` |
| `repair upstream -o json` | **read** | `--dry-run` (default `true`) → `=false` makes it mutation | `results` |
| `label -o json` | read | `--set` / `--remove` → mutation | `labels` |
| `version -o json` | read | — | `version` |

`install list --json` reports agent registration diagnostics and is explicitly outside this adapter
contract. Interactive setup, metadata editing, and bundle import/export are also outside it. For
metadata and validation status, use `describe` or the metadata MCP tools below. Future branch-prune
planning and agent-status commands are not promised by this inventory.

### MCP tools

| Tool | Access | Payload |
| --- | --- | --- |
| `list_repositories` | read | `repositories` |
| `get_repository_context` | read | `repository` |
| `get_workspace_config` | read | `config` |
| `build_workspace_inventory` | read | `inventory` |
| `select_repositories` | read | `repositories` |
| `get_repo_metadata` | read | `metadata` (`null` when the repo has none) |
| `get_authoritative_paths` | read | `paths` |
| `get_related_repositories` | read | `repositories` |
| `plan_sync` | **read** | `plan` |
| `scan_workspace` | **mutation** | `scan` |
| `execute_sync` | mutation | `results` |
| `set_labels` | mutation | `labels` |
| `add_repository` | mutation | `repository` |
| `remove_repository` | mutation | `repository` |

The envelope appears in both `structuredContent` and the text-content fallback, which carry the same
bytes so the two cannot disagree.

### MCP resources

| Resource URI | Access | Payload |
| --- | --- | --- |
| `repokeeper://config` | read | `config` |
| `repokeeper://registry` | read | `registry` |

### MCP resource templates

| Registered URI template | Access | Payload |
| --- | --- | --- |
| `repokeeper://repo/{+repo_id}` | read | `repository` or `metadata` |

The reserved expansion allows slashes in repository IDs. Read `repokeeper://repo/github.com/org/repo`
for the registry entry or append `/metadata` for repo-local metadata. Missing metadata is a resource
error (the `get_repo_metadata` tool instead returns `metadata: null`).

The inventory drift test compares these tables with the live Cobra tree, MCP tools, resources, and
resource templates. A new JSON command must be listed or explicitly excluded above.

### Resource payload fields

These schemas apply to MCP resources, including a registry embedded in the config response:

| Payload | Required fields | Optional fields |
| --- | --- | --- |
| `registry` | `repos` (entry array, empty as `[]`) | `updated_at` |
| registry entry / `repository` | `repo_id`, `path`, `remote_url`, `status` | `checkout_id`, `type`, `branch`, `labels`, `annotations`, `last_seen`, `repo_metadata_file`, `repo_metadata_error`, `repo_metadata_fingerprint`, `repo_metadata` |
| `config` | `apiVersion`, `kind`, `exclude`, `registry_stale_days`, `defaults`, `branch_policy` | `ignored_paths`, `registry_path`, `registry` |

Entry strings describe stored registry state; `status` is `present`, `missing`, or `moved`. Labels and
annotations are string maps. Metadata uses the repo-local schema described in DESIGN.md. Config
`defaults` contains `remote_name`, `main_branch`, `concurrency`, and `timeout_seconds`;
`branch_policy` contains `protected_patterns` (string array), optional `base_branch`, `stale_days`,
and `require_merged` (boolean). Numeric config values are integers. The inner `config.apiVersion`
describes the config schema; only the outer `apiVersion` identifies the adapter contract.

`updated_at` and `last_seen` are UTC RFC3339 timestamps with optional fractional seconds. When no
observation exists, they are omitted, including `list_repositories[].last_seen`. No year-one sentinel
is emitted for these fields. An absent config `registry` means no registry was stored; a present
registry with `repos: []` means a known empty registry.

**2.0.0 migration:** resource fields previously inherited Go names (`Entries`, `RemoteURL`,
`UpdatedAt`, `Exclude`). They now use `repos`, `remote_url`, `updated_at`, and `exclude`. Do not parse
the old names. Config-embedded registry URLs are redacted just like direct registry resources.

## 5. Know which calls are safe to make unprompted

- **`read`** — safe to call automatically: on a timer, on window focus, on file save.
- **`mutation`** — requires an explicit user gesture. These alter the registry, config, or a working
  tree.

Four are worth memorising, because the name or the flag default misleads:

| Surface | Looks like | Actually |
| --- | --- | --- |
| `plan_sync` | a mutation (it plans a sync) | **read** — always dry-run, never mutates |
| `scan_workspace` | a query | **mutation** — updates the registry |
| `scan -o json` | a query | **mutation** — `--write-registry` defaults to `true` |
| `repair upstream -o json` | a mutation ("repair") | **read** — `--dry-run` defaults to `true` |

Note the trap: **`reconcile` and `repair upstream` carry opposite `--dry-run` defaults.** Pass the
flag explicitly and you never have to remember which is which.

Under a read-only workspace mount, every `read` surface still succeeds, and every `mutation` surface
refuses with a message naming both the cause and the remedy — it stays advertised rather than
vanishing.

## 6. URLs are redacted — do not clone from them

Every URL in a contract response has embedded credentials stripped:

```
https://user:token@github.com/org/repo.git   →   https://***@github.com/org/repo.git
```

This is a guarantee, so you can log or display a payload without leaking a token. SSH remotes are
unchanged — they authenticate with keys, so there is nothing to strip.

The consequence: **a URL from a contract response is not usable for cloning.** If your adapter needs
to clone, use `add` or `reconcile --checkout-missing`, which read the registry directly.

## 7. Handle failure before you parse

Check both process status and output. A fatal invocation error (invalid arguments, unreadable
configuration) exits non-zero without a JSON envelope. However, a completed scan, status report, or
reconcile can exit non-zero **with a valid envelope** when individual repositories have warnings or
failures. Preserve those records and inspect `error`, `error_class`, `ok`, `outcome`, and `skip_reason`
where present. Empty stdout is not an empty result set. Stderr is diagnostic prose, not a source of
machine-readable fields.

An intentional **skip** is a success, not a failure: a repository RepoKeeper declined to act on
arrives inside a normal envelope with `ok: true` and a machine-readable reason. Some skipped outcomes
are failures (for example, a missing checkout); inspect `ok` and `error` rather than assuming every
`skipped_*` outcome succeeded.

For MCP, check protocol errors and `isError` first. A failed tool call has no success envelope in
`structuredContent`. Successful batch calls may still contain per-repository failures. The text
fallback on success contains the same JSON as `structuredContent`.

## 8. What is NOT contractual

Depending on any of these means depending on something free to change in a patch release:

| Not contractual | Why |
| --- | --- |
| `table` / `wide` output | Human-oriented — column order, widths, alignment and colour all evolve |
| Prose, hints, colour | Not a contract surface |
| Log and debug output on stderr | Diagnostic, not an interface |
| Everything under `internal/...` | ADR-0006 rejects internal imports as a consumption path |
| Specific non-zero exit values | Only the zero / non-zero distinction is guaranteed |

If you find yourself parsing a table to get something, that thing is missing from the
machine-readable contract. Please file an issue — the gap is the bug, not your workaround.

## 9. How the contract evolves

| Change | Breaking? | Bumps `apiVersion`? |
| --- | --- | --- |
| Add a top-level or per-record field | no | no |
| Add a new value to an existing enum | **yes** | yes |
| Remove or rename a field | yes | yes |
| Change a field's type | yes | yes |
| Change the meaning of an existing enum value | yes | yes |
| Change human-oriented table output | n/a — not contractual | no |

Adding an enum value counts as breaking because a consumer switching exhaustively over the value
space breaks when a new one appears.

Breaking changes bump `apiVersion`, are documented here and in `DESIGN.md`, and are called out in
release notes.

The contract version is independent of the config `apiVersion` and the repo-metadata `apiVersion`.
They are different schemas with different lifecycles; do not infer one from another.

## 10. CLI or MCP?

Both are supported. ADR-0006 explicitly declines to make MCP the only one.

- **CLI JSON** — simplest to drive from any language. Shell out, check the exit code, parse stdout.
- **MCP** — typed tool schemas, better suited to runtimes already speaking MCP.

Where both expose the same information the shared records use identical field names and semantics,
so you can move between them without relearning the data.

## 11. Declaring which core versions you support

The policy adapters use to declare and validate supported RepoKeeper core versions is tracked in
[#286](https://github.com/skaphos/repokeeper/issues/286). Until it lands, pin to a contract
`apiVersion` and state the minimum core version you tested against.
