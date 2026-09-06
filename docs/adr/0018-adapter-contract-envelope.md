# ADR-0018: Uniform Adapter Contract Envelope and Promotion to v1

**Status:** Accepted
**Date:** 2026-09-05
**Author:** Shawn Stratton

## Context

[ADR-0006](0006-adapter-contract-stability.md) established that RepoKeeper's machine-readable output is an explicit contract surface: deterministic, documented, versioned, and separate from human-oriented table output. It requires that adapters be able to determine compatibility without guesswork, and deliberately left the mechanism open — "the exact mechanism may vary by surface".

That openness was never closed. By the 2.0.0 milestone the situation was:

- `get` / `status -o json` carried a top-level `apiVersion` (`DESIGN.md` §6.3), with additive-vs-breaking rules, a single-source constant, and a drift test.
- Every other adapter-facing surface carried no version at all: `scan`, `reconcile`/`sync`, `repair upstream`, `describe`, `label`, `version`, and all fourteen MCP tools.

The action commands emitted **bare top-level JSON arrays**. `DESIGN.md` §6.4 records that this was deliberate — the array shape matched the MCP `plan_sync`/`execute_sync` results so CLI and MCP consumers parsed identical fields, a parity actively maintained (PR #344 restored it after drift).

The consequence: an adapter could detect a breaking change on exactly one surface. A top-level array has nowhere to put a version, so no amount of documentation could make those surfaces self-describing.

This ADR records the mechanism chosen under ADR-0006. It does **not** supersede ADR-0006, which remains accepted and immutable; it implements it.

## Decision

### 1. Every adapter-facing response is wrapped in a versioned envelope

Every adapter-facing JSON surface — CLI **and** MCP — emits a top-level object carrying `apiVersion` and exactly one named payload:

```json
{
  "apiVersion": "skaphos.io/repokeeper/v1",
  "generated_at": "2026-09-05T23:54:33Z",
  "results": []
}
```

No adapter-facing surface emits a bare top-level array.

The payload is **named**, not anonymous, so a surface can add a sibling field later as an additive change rather than a breaking restructure.

`generated_at` appears only where the response reports point-in-time observed state. Emitting a zero time elsewhere would make "not applicable" indistinguishable from "the epoch".

### 2. The version is promoted from `v1beta1` to `v1`

The 2.0.0 release ships `skaphos.io/repokeeper/v1`.

### 3. CLI and MCP are enveloped in the same change

Not sequentially. §6.4's array shape existed to hold CLI/MCP record parity; enveloping one side alone would break the parity it was protecting.

### 4. One shared constant

`contract.APIVersion` in `internal/contract`, imported by both the CLI and the MCP server. A single constant is what makes "CLI and MCP cannot drift" mechanically true rather than a convention reviewers must police. It lives in `internal/` because `internal/mcpserver` cannot import `cmd/repokeeper` — the command package already imports the server.

### 5. MCP resources are inside the contract, not just MCP tools

The three MCP resources — `repokeeper://config`, `repokeeper://registry`, and the
`repokeeper://repo/{repo_id}` template (plus its `/metadata` form) — are advertised with MIME type
`application/json` and readable by any MCP client. They emitted raw domain objects with no version
marker.

Excluding them would have left three unversioned JSON surfaces inside a contract that claims to
cover every one. Adapters would have found and used them regardless of what the contract said.

### 6. URL fields are credential-redacted

Every URL emitted on an adapter-facing surface has embedded credentials stripped, using the existing
`urlutil.RedactCredentials`. This is a contract guarantee asserted per surface by test.

The helper predates this change but was wired only into `export` (as a hazard *detector*) and
debug-argument logging, so `remotes[].url` reached adapter JSON verbatim from git. A remote
configured as `https://user:token@host/repo.git` would have travelled into any IDE plugin that logs
or displays a payload. The registry resource is the sharpest case, since it serialises every entry's
`remote_url`.

Redaction happens at the output boundary and returns a **copy**. The same `RepoStatus` and
`registry.Entry` values feed the registry write path, and persisting a masked `remote_url` would
break the user's ability to fetch. SSH remotes are unchanged: they authenticate with keys, so there
is no embedded secret to strip.

Consequence for consumers: a URL read from a contract response cannot be used to clone. Adapters
needing to clone use `add` or `reconcile --checkout-missing`, which read the registry directly.

### 7. No compatibility mode for the old shape

2.0.0 is a clean break. There is no `--legacy-output` flag. A second output path would have to be
maintained and tested on every surface and then deprecated in turn, which is the cost the
major-version boundary exists to avoid. The migration is one line — read `.results` instead of the
top-level array — and belongs in release notes.

### 8. Empty collections are `[]`, never `null`

A nil Go slice marshals as `null`. The envelope constructors normalise, so the guarantee is automatic rather than per-call-site vigilance.

### 9. Adding an enum value is a breaking change

`DESIGN.md` §6.3 previously made *changing* the meaning of an enum value breaking but was silent on *adding* one. A consumer switching exhaustively over a value space breaks when a new value appears, so this is now classed breaking. This tightens the existing policy.

### 10. The contract version stays independent of the config and repo-metadata schemas

Three `apiVersion` values exist in this codebase:

| Schema | Value | Why it cannot follow the others |
| --- | --- | --- |
| Output contract (`internal/contract`) | `skaphos.io/repokeeper/v1` | This ADR |
| Config (`internal/config`) | `skaphos.io/repokeeper/v1beta1` | Written into user `.repokeeper.yaml` files and validated on load; bumping it invalidates every existing config |
| Repo metadata (`internal/repometa`) | `repokeeper/v1` | Own unprefixed scheme; written into `.repokeeper-repo.yaml` |

The output contract and the config schema shared a value before this change. They no longer do, and must never be coupled.

## Consequences

### Positive

- An adapter determines compatibility from the **first response to any call** — no handshake, no per-surface reasoning.
- The read/mutation boundary and the surface inventory become documentable as one contract rather than seven conventions.
- Issues #289 (JSON hardening) and #286 (adapter compatibility policy) have a concrete mechanism to build on.
- The `[]`-not-`null` guarantee removes a whole class of adapter parse bug.

### Negative

- **This is a breaking change** for every consumer of the previously-unenveloped surfaces. Scripts parsing `reconcile -o json` as a top-level array must read `.results` instead.
- It is taken at the 2.0.0 major, which is the only point where it is cheap. Deferring it would have cost a contract v2 or a long dual-shape deprecation.
- MCP results are enveloped too, so MCP clients see the same restructuring.

### Neutral

- Human-oriented `table`/`wide` output is unchanged and remains explicitly non-contractual.

## Alternatives Considered

### 1. Out-of-band version discovery

Leave every JSON shape byte-identical and add one discovery surface (extend `version -o json`, or add `repokeeper contract`).

**Rejected because:** it is coarse — a single version for all surfaces means any one surface's break bumps everything — and it costs adapters an extra call before they can trust anything, while the per-surface payloads remain self-undescribing. That sits poorly against ADR-0006's "must not depend on guesswork".

Its genuine advantage was being non-breaking. At a major version boundary that advantage is at its cheapest to give up, which is precisely why the change belongs here and not later.

### 2. Per-surface mechanisms

Envelope what is already enveloped, document a version per surface elsewhere, change no shapes.

**Rejected because:** it stays literally within ADR-0006's "mechanism may vary by surface" but is closest to the status quo that produced the problem. Adapters would still infer compatibility differently per surface.

### 3. Inline `apiVersion` into existing objects rather than nesting a named payload

For the object-shaped responses, add `apiVersion` as a sibling of the existing fields — additive, and non-breaking for those surfaces.

**Rejected because:** it cannot work for the array-shaped surfaces at all, so it would leave the contract half-uniform. It also makes the payload anonymous, so the first time a surface needs a sibling field it forces the breaking change this design avoids.

### 4. Stay on `v1beta1` through 2.0.0

Promote once a real external adapter has been built against the contract.

**Rejected because:** it preserves freedom to break without a major bump, but asks adapter authors to build against something labelled unstable — the adoption friction the "contract release" milestone exists to remove. No external consumer of `v1beta1` was identified.

## References

- [ADR-0006](0006-adapter-contract-stability.md) — the policy this implements
- `DESIGN.md` §6.3, §6.4 — the shipped contract documentation
- `specs/004-adapter-contract/` — specification, research, and contracts
- Issue #287 — Define a stable plugin adapter contract for standalone IDE integrations
