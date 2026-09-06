# Contract: CLI / MCP Parity

**Requirements**: FR-017, FR-018 | **Artifacts**: `cmd/repokeeper/sync.go`, `internal/mcpserver/tools_mutation.go`, `DESIGN.md` §6.4

Where a CLI surface and an MCP tool expose the same information, they agree. This contract exists
because the agreement is load-bearing and easy to break silently.

## Why this is a contract and not an implementation detail

ADR-0006 states CLI and MCP contracts "do not need to be byte-for-byte identical" — and that remains
true at the envelope level, since the two transports differ. But `DESIGN.md` §6.4 records a
narrower, stronger commitment already made for the *shared records*:

> `reconcile -o json` ... emits a top-level JSON **array** of per-repo result objects ... Unlike the
> `get`/`status` collection it is unenveloped, matching the other action commands (`scan`,
> `repair-upstream`) and the MCP `plan_sync`/`execute_sync` result shape, **so CLI and MCP consumers
> parse identical fields.**

That parity is actively maintained: PR [#344](https://github.com/skaphos/repokeeper/pull/344)
("align sync plans with CLI dry-run results") exists precisely because it had drifted.

## The rule

| Level | Requirement |
| --- | --- |
| Envelope | CLI and MCP both carry `apiVersion`, from the same constant, with the same value. |
| Shared records | Identical field names, types, and semantics. A field meaning one thing in `reconcile -o json` and another in `plan_sync` is a defect. |
| Transport framing | May differ. MCP result framing and CLI stdout are not required to be byte-identical. |

## Consequence for this feature

The envelope must be applied to CLI and MCP **in the same change** (FR-018).

The bare-array shape of the CLI action commands is not incidental — §6.4 chose it *specifically* to
match the MCP result shape. Enveloping one side alone would reintroduce exactly the drift #344 fixed,
and would do so in the release that claims to stabilise the contract.

This is the accepted cost of the uniform-envelope decision, recorded rather than discovered during
implementation.

## Verification

- A test asserts the CLI and MCP `apiVersion` values are the same constant, not merely equal strings.
- For each information-sharing pair (`reconcile --dry-run` ↔ `plan_sync`, `reconcile` ↔ `execute_sync`,
  `scan` ↔ `scan_workspace`), a test asserts the shared record field sets match.
- The parity assertion runs in CI on every change, so drift fails the build rather than surfacing as
  a later bug report.
