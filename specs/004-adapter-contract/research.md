# Research: Stable Plugin Adapter Contract

**Feature**: `004-adapter-contract` | **Created**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

This document records what was already settled before this feature began, what the current
implementation actually does, and the alternatives weighed for the two open decisions. Per the
constitution's Specification and Decision Workflow, settled questions are cited, not re-researched.

---

## 1. Settled upstream: ADR-0006

[ADR-0006 "Adapter-Facing Contract Stability and Versioning"](../../docs/adr/0006-adapter-contract-stability.md)
was accepted 2026-04-03 and is immutable. It already decides:

- Machine-readable output (CLI JSON, MCP schemas) is an explicit contract surface; human-oriented
  table, colour and prose output is **not**, and adapters must not depend on it.
- Contract surfaces must be deterministic, documented, versioned, and clearly separated from human
  output.
- Breaking changes require documentation, schema/version notes, and release-note visibility.
- CLI and MCP contracts are *related but not identical* — they need not be byte-for-byte the same,
  but each must be independently well-defined and stable.
- Compatibility "must not depend on guesswork", though "the exact mechanism may vary by surface".

It explicitly rejects three alternatives, which are therefore **not** reopened here:

| Rejected alternative | ADR-0006's stated reason |
| --- | --- |
| Treat CLI JSON as best-effort only | Standalone adapters need a reliable machine-readable contract |
| Require adapters to import internal packages | Couples external repos to unstable internals, undermines the standalone model |
| Use MCP as the only stable contract | CLI JSON is a valid adapter surface some integrations prefer |

**Consequence for this feature**: the *policy* is fixed. What is missing is the concrete mechanism
and the enumerable inventory. This feature supplies exactly that, and must not contradict the ADR.

---

## 2. Current implementation state

### 2.1 What is already versioned

Exactly one surface. `cmd/repokeeper/status.go` defines:

```go
const statusJSONAPIVersion = "skaphos.io/repokeeper/v1beta1"

type statusJSONReport struct {
    APIVersion  string           `json:"apiVersion"`
    GeneratedAt time.Time        `json:"generated_at"`
    Repos       []statusJSONRepo `json:"repos"`
}
```

`DESIGN.md` §6.3 documents this shape and, below it, a "JSON output schema stability policy" that
already states the additive-vs-breaking rules this spec generalises. It also records a property
worth preserving verbatim: the value is sourced from a single constant, and
`TestDesignDocNamesStatusJSONAPIVersion` asserts the document names the current constant value, so
the emitted version and the documented version cannot silently diverge.

That drift test is the single most valuable existing pattern here. FR-011 generalises it from one
constant to the whole surface inventory.

### 2.2 What is not versioned

Everything else. `DESIGN.md` §6.4 documents `reconcile`/`sync -o json` as a bare top-level **array**
and states the reason explicitly: it is "unenveloped, matching the other action commands (`scan`,
`repair-upstream`) and the MCP `plan_sync`/`execute_sync` result shape, so CLI and MCP consumers
parse identical fields."

This is a deliberate design choice, not an oversight — which is why changing it is a contract
decision requiring a spec rather than a bugfix.

CLI commands offering a machine-readable format today (`addFormatFlag` / `--format,-o`):

| Command | Formats | Envelope today |
| --- | --- | --- |
| `get`, `get repos` | table, wide, json | **enveloped + versioned** |
| `describe`, `describe repo` | table, json | bare |
| `scan` | table, json | bare |
| `reconcile`, `reconcile repos` (alias `sync`) | table, wide, json | bare array |
| `repair upstream` | table, json | bare |
| `label` | table, json | bare |
| `version` | table, wide, json | bare |

### 2.3 MCP surface

Fourteen tools registered in `internal/mcpserver`. The read/mutation split is already real in the
code — mutating handlers live in `tools_mutation.go`, and `readonly.go` translates read-only-mount
write failures into a refusal naming cause and remedy:

**Read (9)**: `list_repositories`, `get_repository_context`, `get_workspace_config`,
`build_workspace_inventory`, `select_repositories`, `get_repo_metadata`,
`get_authoritative_paths`, `get_related_repositories`, `plan_sync`

**Mutation (5)**: `scan_workspace`, `execute_sync`, `set_labels`, `add_repository`,
`remove_repository`

Two are genuinely mis-guessable from their names alone, which is the concrete justification for
FR-014:

- **`plan_sync` is read.** Its description says "Always dry-run — never mutates", despite planning
  a mutation.
- **`scan_workspace` is mutation.** Its description says "Discover repos ... **and update the
  registry**", despite reading like a query.

`readonly.go` also records a deliberate stance worth lifting into the contract: mutating tools stay
advertised under a read-only mount, because "a tool that exists and explains why it cannot run is
honest, one that vanishes is not." That is Principle VII and VI applied together, and FR-015
makes it contractual rather than incidental.

---

## 3. Decision 1 — how adapters detect contract version

### Options weighed

**A. Uniform envelope on every adapter-facing surface** *(chosen)*

Wrap every adapter-facing JSON response, CLI and MCP, in `{apiVersion, ...}`.

- One detection mechanism for every surface; an adapter writes one compatibility check.
- Directly supplies what #286 needs to describe ("how adapters should detect unsupported core
  versions") and what #289 needs to harden against.
- Compatibility is determinable from the **first** response, with no extra handshake call (SC-002).
- **Cost**: breaking for every current consumer of the bare-array surfaces.
- **Cost**: §6.4's parity rationale means MCP results must be enveloped in the same change, or the
  CLI/MCP field parity that PR #344 recently fixed breaks again.

**B. Out-of-band discovery surface**

Leave every JSON shape byte-identical; add one contract-version discovery surface (extend
`version -o json`, or add `repokeeper contract`).

- Fully non-breaking; preserves CLI/MCP parity untouched; cheapest to land.
- **Rejected because**: it is coarse — one version for all surfaces means any single surface's break
  bumps everything — and it costs adapters an extra call before they can trust anything, which sits
  awkwardly against ADR-0006's "must not depend on guesswork" when the per-surface shape still
  carries no self-description.

**C. Per-surface mechanisms, as ADR-0006 permits**

Envelope where already enveloped, document a version per surface elsewhere, change no shapes.

- Stays literally within ADR-0006's "mechanism may vary by surface"; least churn.
- **Rejected because**: it is closest to the status quo that produced this issue. An adapter would
  still infer compatibility differently per surface — arguably the guesswork ADR-0006 set out to
  eliminate.

### Rationale for A

The 2.0.0 major is the only window in which the breaking change is cheap; after 2.0.0 the same
change costs a contract v2 or a long dual-shape deprecation. The milestone is named "the contract
release", and the two sibling issues (#289, #286) both consume whatever this decides. Uniformity is
worth one deliberate break taken at the major boundary.

The MCP coupling is a real cost and is accepted explicitly: FR-018 requires CLI and MCP be
enveloped in the same change rather than sequentially.

---

## 4. Decision 2 — promote `v1beta1` to `v1`

**Chosen: promote.** The 2.0.0 release ships `skaphos.io/repokeeper/v1`.

- A release named "the contract release" that ships a contract still labelled beta undercuts its own
  message, and Principle IX's honest-scope requirement cuts both ways: understating stability the
  project intends to hold is as inaccurate as overstating it.
- External adapter repositories are being asked to commit to this surface; `v1` is the signal that
  the commitment is mutual, and is what #286's policy will formalise.
- The promotion is safe under the assumption recorded in the spec: no external consumer of
  `v1beta1` exists today. If one is identified before this lands, the promotion is revisited.

**Alternative — stay on `v1beta1` through 2.0.0**, promoting once a real adapter has been built
against it. Rejected: it preserves freedom to break without a major bump, but asks adapter authors
to build against something labelled unstable, which is the adoption friction this milestone exists
to remove.

**Alternative — promote only the mature surface** (`get`/`status` to `v1`, newly-enveloped surfaces
stay `v1beta1`). Rejected: shipping mixed version labels in one release reintroduces per-surface
reasoning for adapters, which is the problem Decision 1 just removed. It is also incompatible with
FR-003's single shared constant.

---

## 5. Constraints carried into planning

- **Single constant** (FR-003). Two constants would let CLI and MCP drift; the existing pattern is
  already single-source and must not regress.
- **Drift test generalisation** (FR-011). The existing test proves the pattern works for one value.
  Whether the full inventory can be enumerated from code — rather than hand-listed — is the main
  open implementation risk and is the first thing planning should settle.
- **Independent versioning** (FR-004). Three `apiVersion` values already exist in this codebase:
  config (`internal/config`), repo metadata (`internal/repometa`), and output (`cmd/repokeeper`).
  They share a value today by coincidence, and §6.3 already warns they must not be coupled.
- **Empty collections** (FR-007). Go marshals a nil slice as `null`; an adapter parsing `[]` uniformly
  will break on it. This needs explicit handling wherever a collection can be empty.
- **Ginkgo v2 + Gomega** for all new tests, per the constitution's Engineering Constraints, with a
  meaningful test shipping in the same change as the behaviour.
