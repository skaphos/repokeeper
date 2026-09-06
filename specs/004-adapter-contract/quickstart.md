# Quickstart: Building an Adapter Against the RepoKeeper Contract

**Feature**: `004-adapter-contract` | **Created**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

For an author building a standalone adapter repository — `repokeeper-vscode`, `repokeeper-jetbrains`,
or anything else that drives RepoKeeper from outside. Everything here is consumable without importing
a single RepoKeeper Go package.

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
  "repos": [ /* ... */ ]
}
```

Your compatibility check, in whatever language your adapter is written in:

1. Run any read surface.
2. Read `apiVersion`.
3. If the major does not match what you were built against, degrade — do not guess.

Because the value is identical across every surface, this is one check, not one per command.

## 2. Ignore fields you do not recognise

Additive changes do not bump `apiVersion`. New fields **will** appear within `v1`.

An adapter that rejects or errors on an unrecognised field turns every routine, non-breaking
RepoKeeper release into an outage on your side. Parse permissively; ignore what you do not know.

## 3. Know which calls are safe to make unprompted

Consult the [surface inventory](./contracts/surface-inventory.md). Each surface is classified `read`
or `mutation`.

- **`read`** — safe to call automatically. On a timer, on window focus, on file save.
- **`mutation`** — requires an explicit user gesture. These alter the registry, config, or a working
  tree.

Four are worth memorising because the name or the flag default misleads:

| Surface | Looks like | Actually |
| --- | --- | --- |
| `plan_sync` | a mutation (it plans a sync) | **read** — always dry-run, never mutates |
| `scan_workspace` | a query (it "scans") | **mutation** — updates the registry |
| `scan -o json` | a query | **mutation** — `--write-registry` defaults to `true` |
| `repair upstream -o json` | a mutation ("repair") | **read** — `--dry-run` defaults to `true` |

Note the trap in the last two: **`reconcile` and `repair upstream` disagree on their `--dry-run`
default.** `reconcile` defaults to `--dry-run=false` (it writes); `repair upstream` defaults to
`--dry-run=true` (it does not). Do not generalise the default from one command to the other — pass
the flag explicitly and you never have to remember which is which.

## 4. Handle failure before you parse

```
exit code 0      → parse stdout as an envelope
exit code non-0  → read stderr; stdout carries no envelope
```

A failed invocation does not emit an envelope. Check the exit code first.

Note that an intentional **skip** is a success, not a failure: a repository RepoKeeper declined to
act on arrives inside a normal envelope with `ok: true` and a machine-readable reason. Do not treat
a skip as an error just because work did not happen.

## 5. Do not depend on these

| Do not depend on | Why |
| --- | --- |
| `table` / `wide` output | Explicitly non-contractual; free to change in a patch release |
| Prose, hints, colour | Human-oriented, not a contract |
| `internal/...` packages | ADR-0006 rejects internal imports as a consumption path |
| Specific non-zero exit values | Only the zero / non-zero distinction is guaranteed |

If you find yourself parsing a table to get something, that thing is missing from the machine-readable
contract. File an issue — the gap is the bug, not your workaround.

## 6. CLI or MCP?

Both are supported adapter surfaces; ADR-0006 explicitly declines to make MCP the only one.

- **CLI JSON** — simplest to drive from any language. Shell out, check the exit code, parse stdout.
- **MCP** — typed tool schemas, better suited to agent runtimes already speaking MCP.

Where both expose the same information, the shared records use identical field names and semantics,
so you can move between them without relearning the data. See
[CLI/MCP parity](./contracts/cli-mcp-parity.md).

## 7. Declaring which core versions you support

The compatibility policy adapters use to declare and validate supported RepoKeeper core versions is
issue [#286](https://github.com/skaphos/repokeeper/issues/286), which builds on this contract. Until
it lands, pin to a contract `apiVersion` and state the minimum core version you were tested against.
