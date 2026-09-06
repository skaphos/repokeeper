# Data Model: Stable Plugin Adapter Contract

**Feature**: `004-adapter-contract` | **Created**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

This feature's data model is the shape of the contract itself. It defines no new persisted state —
nothing here is written to disk, and neither the registry nor the config schema changes.

---

## 1. Contract envelope

The single top-level object wrapping every adapter-facing JSON response.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `apiVersion` | string | yes | Contract schema identifier. `skaphos.io/repokeeper/v1` at 2.0.0. Sourced from one shared constant (FR-003). |
| `generated_at` | RFC 3339 timestamp | where meaningful | Present on surfaces reporting observed state; omitted where the response is not a point-in-time observation. |
| *payload* | object or array | yes | Exactly one named payload field per surface. Never anonymous, never a bare top-level array (FR-001). |

**Invariants**

- `apiVersion` is present on every adapter-facing response, without exception (FR-001).
- The payload field is **named per surface** — `repos`, `results`, and so on — so a surface can later
  add a sibling field additively without a breaking change.
- A collection payload is `[]` when empty, never `null` and never absent (FR-007).
- Filters that add data (e.g. `get --diverged`) add a sibling field and leave `apiVersion` unchanged,
  preserving the behaviour §6.3 already documents.

**Relationship to existing shapes**

`get`/`status` already matches this model (`apiVersion`, `generated_at`, `repos`). The model is that
existing envelope generalised, not a new invention — which is why `get`/`status` needs only the
version-value promotion, while the bare-array surfaces need wrapping.

---

## 2. Adapter-facing surface

A descriptor of one thing an external adapter may call. This is the unit the contract document
enumerates and the drift test counts.

| Attribute | Type | Notes |
| --- | --- | --- |
| `identity` | string | CLI: the command path plus its machine-readable format (`get -o json`). MCP: the tool name (`plan_sync`). |
| `kind` | enum | `cli` \| `mcp` |
| `stability` | enum | See §3 |
| `access` | enum | `read` \| `mutation`. See §4 |
| `payload_field` | string | The named payload field inside the envelope |
| `record_type` | reference | The per-record shape the payload carries |

**Invariants**

- Every surface present in code appears in the published inventory, and vice versa; divergence fails
  a test (FR-011).
- Where a CLI surface and an MCP tool expose the same information, their shared records use identical
  field names and semantics (FR-017).

---

## 3. Stability class

| Value | Meaning | Adapter guidance |
| --- | --- | --- |
| `stable` | Contractual. Governed by the evolution rules in §5. | Safe to depend on. |
| `provisional` | Contractual in shape but may change within the current `apiVersion` while it settles. | Depend on it only with a fallback. |
| `non-contractual` | Explicitly excluded from the contract. | Never depend on it. |

`non-contractual` covers, per ADR-0006 and FR-010: human-oriented `table`/`wide` output, colour and
prose, log output, and every package under `internal/`.

The class is stated rather than inferred. A surface with no stated class is a documentation defect,
not an implicitly stable surface.

---

## 4. Access classification

| Value | Meaning |
| --- | --- |
| `read` | Cannot alter registry, config, or any working tree. |
| `mutation` | Can alter at least one of those. |

**Invariants**

- No surface classed `read` performs a write; asserted by test, not review (FR-013).
- Under a read-only workspace, `read` surfaces succeed (FR-016) and `mutation` surfaces refuse with a
  message naming cause and remedy while remaining advertised (FR-015).

**Classification of the current MCP surface** (from `internal/mcpserver`):

| Access | Tools |
| --- | --- |
| `read` | `list_repositories`, `get_repository_context`, `get_workspace_config`, `build_workspace_inventory`, `select_repositories`, `get_repo_metadata`, `get_authoritative_paths`, `get_related_repositories`, `plan_sync` |
| `mutation` | `scan_workspace`, `execute_sync`, `set_labels`, `add_repository`, `remove_repository` |

**MCP resources** are adapter-facing too, and all are `read`: `repokeeper://config`,
`repokeeper://registry`, and the `repokeeper://repo/{repo_id}` template (plus its `/metadata`
form). The original spec enumerated only the tools and missed these; they emit `application/json`
and any MCP client can read them.

Two MCP classifications are counter-intuitive and are called out in the contract because an adapter
author would plausibly guess wrong:

- `plan_sync` is **read** — always dry-run, never mutates, despite planning a mutation.
- `scan_workspace` is **mutation** — it updates the registry, despite reading like a query.

**Flag-dependent classification.** On the CLI, three commands change class with a flag, and the
contract states the class **at default flags** plus the flag that changes it (FR-014a). Verified
against the built binary:

| Command | Default class | Flag | Default value |
| --- | --- | --- | --- |
| `scan` | mutation | `--write-registry` | `true` |
| `reconcile` (alias `sync`) | mutation | `--dry-run` | `false` |
| `repair upstream` | **read** | `--dry-run` | **`true`** |
| `label` | read | `--set` / `--remove` | unset |

`reconcile` and `repair upstream` carry **opposite** `--dry-run` defaults. This is existing
behaviour, not something this feature changes, but it means the contract cannot state a single
generalised rule for plan-style commands and must classify each one individually.

---

## 5. Contract version

| Attribute | Value |
| --- | --- |
| Format | `skaphos.io/repokeeper/vN[betaM]` |
| Value at 2.0.0 | `skaphos.io/repokeeper/v1` |
| Source | One shared constant, consumed by CLI and MCP alike |

**Evolution rules** (generalised from the §6.3 policy, unchanged in substance):

| Change | Breaking? | Bumps `apiVersion`? |
| --- | --- | --- |
| Add a top-level or per-record field | no | no |
| Add a new enum value to an existing field | **yes** — changes the value space a consumer must handle | yes |
| Remove or rename a field | yes | yes |
| Change a field's type | yes | yes |
| Change the meaning of an existing enum value | yes | yes |
| Change human-oriented table output | n/a — non-contractual | no |

> Note on new enum values: §6.3's existing wording makes "changing the meaning of an existing enum
> value" breaking but is silent on *adding* one. Adding a value can break a consumer that switches
> exhaustively, so it is classed breaking here. This is a tightening of the existing policy and is
> flagged in the plan's Complexity Tracking as a deliberate deviation to confirm.

**Independence** (FR-004): three `apiVersion` values coexist in this codebase — config
(`internal/config`), repo metadata (`internal/repometa`), and this output contract. They share a
value today by coincidence. A bump to one must never force a bump to another.

---

## 6. Per-record shapes

This feature does **not** redefine per-record fields. `statusJSONRepo`, `syncResultJSON` and their
peers keep their current fields and semantics; only the wrapper around them changes. Hardening the
individual record shapes is issue [#289](https://github.com/skaphos/repokeeper/issues/289), which
consumes this model.

Two cross-cutting record requirements are carried here, because they are contract properties rather
than per-command details:

- Skipped work carries a machine-readable reason, including backend-unsupported skips for non-Git
  backends (FR-020, Principles VI and XI). A skip without a reason is a defect.
- **URL fields are credential-redacted** (FR-021). Redaction happens at the output boundary and
  returns a copy, never mutating the value in place — the same `RepoStatus` and `registry.Entry`
  values feed the registry write path, and persisting a masked `remote_url` would break the user's
  ability to fetch. A consequence for consumers (FR-022): a URL read from a contract response cannot
  be used to clone.
