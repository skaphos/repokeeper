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

Every adapter-facing response is a JSON **object** carrying `apiVersion` and exactly one named
payload. No surface emits a bare top-level array.

| Field | Notes |
| --- | --- |
| `apiVersion` | Always present. `skaphos.io/repokeeper/v1` as of 2.0.0. |
| `generated_at` | Present only where the response reports observed state at a point in time. Omitted — not zero — when not applicable. |
| *payload* | Exactly one, named per surface (`repos`, `results`, `config`, …). |

An empty collection is `[]`, never `null` and never omitted.

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
| `get -o json`, `get repos -o json` | read | — | `repos` |
| `describe -o json`, `describe repo -o json` | read | — | `repo` |
| `scan -o json` | **mutation** | `--write-registry` (default `true`) | `repos` |
| `reconcile -o json` (alias `sync`) | **mutation** | `--dry-run` (default `false`) → read | `results` |
| `repair upstream -o json` | **read** | `--dry-run` (default `true`) → `=false` makes it mutation | `results` |
| `label -o json` | read | `--set` / `--remove` → mutation | `labels` |
| `version -o json` | read | — | `version` |

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
| `repokeeper://repo/{repo_id}` | read | `repository` |
| `repokeeper://repo/{repo_id}/metadata` | read | `metadata` |

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
https://alice:ghp_token@github.com/org/repo.git   →   https://***@github.com/org/repo.git
```

This is a guarantee, so you can log or display a payload without leaking a token. SSH remotes are
unchanged — they authenticate with keys, so there is nothing to strip.

The consequence: **a URL from a contract response is not usable for cloning.** If your adapter needs
to clone, use `add` or `reconcile --checkout-missing`, which read the registry directly.

## 7. Handle failure before you parse

```
exit code 0      → parse stdout as an envelope
exit code non-0  → read stderr; stdout carries no envelope
```

An intentional **skip** is a success, not a failure: a repository RepoKeeper declined to act on
arrives inside a normal envelope with `ok: true` and a machine-readable reason. Do not treat a skip
as an error just because no work happened.

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
