# Contract: Adapter Response Envelope

**Requirements**: FR-001 – FR-007, FR-019 | **Artifacts**: `cmd/repokeeper/*.go`, `internal/mcpserver/tool_results.go`, `DESIGN.md` §6.3

The single top-level object every adapter-facing RepoKeeper response is wrapped in. It exists so an
adapter can determine contract compatibility from the first response it receives, with no separate
handshake call.

## Identity

| Field | Value |
| --- | --- |
| Contract version at 2.0.0 | `skaphos.io/repokeeper/v1` |
| Promoted from | `skaphos.io/repokeeper/v1beta1` |
| Source of truth | one shared constant, consumed by CLI and MCP |
| Versioned independently of | config `apiVersion` (`internal/config`), repo-metadata `apiVersion` (`internal/repometa`) |

## Shape

```jsonc
{
  // Always present, on every adapter-facing response, without exception.
  "apiVersion": "skaphos.io/repokeeper/v1",

  // Present where the response reports observed state at a point in time.
  // Omitted where it does not (e.g. a pure configuration echo).
  "generated_at": "2026-09-05T23:54:33Z",

  // Exactly one named payload per surface. Never anonymous; never a bare
  // top-level array. The name varies by surface: "repos", "results", ...
  "repos": []
}
```

## Rules

- **Never a bare array.** A top-level JSON array carries nowhere to put `apiVersion`, which is why
  the current `reconcile`/`scan`/`repair upstream` shapes cannot satisfy this contract unchanged.
  This is the breaking change taken deliberately at the 2.0.0 boundary.
- **Named payload.** Naming the payload field leaves room to add a sibling additively later. An
  anonymous payload would force a breaking change the first time a surface needs to report anything
  alongside its results.
- **Empty is `[]`.** A nil Go slice marshals to `null`; an adapter parsing `[]` uniformly breaks on
  it. Every collection payload emits `[]` when empty, never `null`, never omitted.
- **Filters do not bump the version.** A filter that adds data (`get --diverged` adding a `diverged`
  array) adds a sibling field and leaves `apiVersion` unchanged. This preserves behaviour §6.3
  already documents and adapters may already rely on.

## Evolution

| Change | Breaking? | Bumps `apiVersion`? |
| --- | --- | --- |
| Add a top-level or per-record field | no | no |
| Add a new value to an existing enum | yes | yes |
| Remove or rename a field | yes | yes |
| Change a field's type | yes | yes |
| Change the meaning of an existing enum value | yes | yes |

Consumers **must** ignore unknown fields. An adapter that rejects an unrecognised field converts
every additive, non-breaking change into an outage on its own side.

## Failure behaviour

Fatal invocation errors (invalid arguments or unavailable configuration) exit non-zero without an
envelope. Completed batch reports can exit non-zero **with** a valid envelope when repositories have
warnings or failures. Adapters check both process status and stdout, preserve completed records, and
inspect per-record errors. Stderr remains diagnostic prose (FR-019).

Benign skips carry `ok: true` and a machine-readable reason (FR-020). A skipped missing checkout can
instead carry `ok: false` and an error; outcome prefixes alone do not establish success. MCP tool
failures carry `isError: true` and no success structured content, while successful batch calls may
contain per-record failures. Real-process regression tests pin both cases.

## Verification

- Every adapter-facing surface emits `apiVersion`, asserted by enumerating surfaces in test rather
  than by a hand-maintained list where practical.
- The emitted value and the value documented in `DESIGN.md` cannot diverge, extending the existing
  `TestDesignDocNamesStatusJSONAPIVersion` guarantee.
- CLI and MCP emit the same value, asserted by comparing both against the shared constant.
