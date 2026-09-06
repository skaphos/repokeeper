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
3. Require an exact match against the adapter's supported contract identifiers. For example,
   `skaphos.io/repokeeper/v1beta1` does not match `skaphos.io/repokeeper/v1`.
4. Check the core-version range and required capabilities declared by the adapter, as described in
   [§11](#11-declaring-which-core-versions-you-support), before enabling its features.

The value is identical across CLI JSON responses and MCP tool/resource JSON results. MCP
initialization does not carry this marker. Choose compatibility once per binary or MCP session,
then validate each result's marker before interpreting its payload. A marker that changes
mid-session invalidates that compatibility decision.

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
| Stop providing a required response field, or require a previously optional request argument | yes | yes |
| Remove or rename a supported command, flag, MCP tool, or resource | yes | yes |
| Change a read surface to mutate state, including through a flag default | yes | yes |
| Add an optional command, tool, resource, or request argument without changing existing behavior | no | no |
| Change human-oriented table output | n/a — not contractual | no |

Adding an enum value counts as breaking because a consumer switching exhaustively over the value
space breaks when a new one appears.

Breaking adapter-contract changes require a new core major release and a new `apiVersion`. They are
documented here and in `DESIGN.md`, with migration instructions in release notes. Additive minor or
patch releases preserve the existing contract identifier. A new identifier does not promise that
the old shape remains available: 2.0.0, for example, has no legacy-output mode.

The contract version is independent of the config `apiVersion` and the repo-metadata `apiVersion`.
They are different schemas with different lifecycles; do not infer one from another.

## 10. CLI or MCP?

Both are supported. ADR-0006 explicitly declines to make MCP the only one.

- **CLI JSON** — simplest to drive from any language. Shell out, check the exit code, parse stdout.
- **MCP** — typed tool schemas, better suited to runtimes already speaking MCP.

Where both expose the same information the shared records use identical field names and semantics,
so you can move between them without relearning the data.

## 11. Declaring which core versions you support

Each released adapter must publish its support declaration in its own README or release metadata
and enforce the same declaration at runtime. RepoKeeper does not load this declaration or prescribe
a manifest format. Adapter authors own their compatibility checks and tests; they consume the
documented CLI/MCP interfaces without importing RepoKeeper's internal packages.

### Keep the version identifiers separate

| Identifier | What it controls | Where the adapter reads it |
| --- | --- | --- |
| Core release, such as `2.0.0` | Available features and fixes in the running binary | CLI `version -o json` → `version.version`; MCP initialization → `serverInfo.version` |
| Output contract, `skaphos.io/repokeeper/v1` for core 2.0.0 | Response shapes, meanings, and supported surface behavior | The outer `apiVersion` on successful CLI/MCP JSON responses |
| MCP protocol version | Transport negotiation between the MCP SDKs | MCP initialization's negotiated `protocolVersion` |
| Adapter release | The adapter's own features and support declaration | That adapter's release metadata |

Core 2.0.0 uses output contract **v1**, not v2. The Go module's `/v2` suffix is not a response schema.
Config and repo-metadata versions are independent too: a nested `config.apiVersion` must never
substitute for the envelope's marker. An adapter release does not have to share the core's version
number. Neither successful MCP protocol negotiation nor a matching core major proves output
compatibility by itself.

### Publish an explicit support declaration

For each adapter release, state:

- The supported stable core-version range, including a minimum and an exclusive upper bound.
- The exact output contract identifiers the adapter can decode.
- The transport used and the commands, tools, resources, and arguments required for each feature.
- Optional capabilities and the feature behavior when they are unavailable.
- The core versions and platforms actually tested, plus any known excluded versions.

For example, an adapter targeting the 2.0.0 contract could publish the following **after testing the
released binaries**:

| Declaration | Example |
| --- | --- |
| Stable core range | `>=2.0.0 <3.0.0` |
| Output contract allow-list | `skaphos.io/repokeeper/v1` |
| Required MCP tools | `list_repositories`, `get_repository_context` |
| Optional feature | Sync preview uses `plan_sync`; hide preview if that tool is unavailable |
| Tested core/platform combinations | Record the actual release versions and operating systems in the adapter's CI results |

This example is a template, not a claim that either IDE adapter already exists or has been tested.
The range accepts later stable 2.x releases only when they also pass the contract and capability
checks. If an adapter begins relying on a field, tool, or fix introduced after its old minimum, it
must raise that minimum or make the dependency optional. Matching `apiVersion` alone does not mean
an earlier release contains a later additive feature.

Use semantic-version comparison, not string ordering; normalize a single leading `v` when parsing
core release versions. Do not normalize or prefix-match output contract identifiers. Exclude
prereleases, snapshots, and Go pseudo-versions from ordinary stable-release ranges unless the adapter
explicitly names and tests them. An empty version, revision hash, `unavailable`, or known modified
development build is not evidence of a supported release. Development use requires a separately
documented opt-in with an exact tested build; it does not expand the stable support claim.

### Validate before enabling adapter operations

**CLI adapters:** invoke `repokeeper version -o json` using the same resolved executable that will
handle subsequent commands. This command requires no workspace configuration. Validate the outer
`apiVersion` and parse `version.version`; do not parse the human version headline. The JSON also
reports build provenance and a `modified` flag, which can identify unsupported local builds. Recheck
after an executable change or adapter restart. Never run a mutation as a compatibility probe.
The declared minimum core version establishes baseline CLI command/flag availability. Gate an optional
newer command or flag on the release that introduced it, backed by adapter tests; no machine-readable
CLI capability-discovery endpoint is promised, and help/table output is not a substitute.

**MCP adapters:** negotiate the protocol through the MCP client, read the core version from
`serverInfo.version`, and enumerate tools/resources required by enabled features. Use a successful
read call such as `list_repositories` to obtain the output contract marker. Initialization has no
RepoKeeper output-contract marker of its own. Read the outer marker from `structuredContent`, or
from the JSON text fallback when the client does not expose structured content. If required tool
input schemas or capabilities are incompatible, the matching output marker does not override that
failure. Recheck on reconnection and refresh capability availability when the server reports a list
change. MCP still requires a configured workspace; a startup/configuration error is not a version
mismatch and does not justify falling back to unversioned output.

Evaluate all of these conditions before enabling a feature: supported core range, exact supported
contract identifier, compatible MCP protocol where used, and available required capabilities.
Then validate required fields, their types, and known enum values on each response while ignoring unknown additive
fields. For optional data, distinguish absent/not-supported from empty; do not synthesize a result
by parsing prose. A valid envelope containing per-repository errors follows [§7](#7-handle-failure-before-you-parse),
not the unsupported-version path.

### Unsupported versions and graceful degradation

| Observation | Adapter behavior |
| --- | --- |
| Core below the minimum, at/above the upper bound, or explicitly excluded | Disable dependent operations; show the detected core version and supported range with an upgrade/downgrade instruction |
| Missing, malformed, or unknown outer `apiVersion` | Do not interpret that payload or enable mutations; show the detected marker and supported identifiers |
| Required capability missing or required field/type invalid | Disable the affected feature and report the specific missing capability or invalid response |
| Optional capability missing | Hide or disable only that feature; keep independently validated features available |
| Unsupported MCP protocol | Report the transport incompatibility; use CLI only if the adapter explicitly implements and separately validates that fallback |
| Development/unknown core identity | Explain that stable support cannot be established; require the adapter's explicit development opt-in, if it offers one |

Do not silently retry using the old bare-array shape, scrape tables, guess a contract from a version
suffix, or execute an operation to discover whether it is supported. Previously cached data may
remain visible only when clearly marked stale; it must not be presented as a successful fresh read
or used to authorize a mutation. If an adapter supports multiple contracts, each needs its own
explicit decoder and tests; the current core does not negotiate an older contract on request.

### Maintain the support claim

Before publishing an adapter release, test its minimum supported core and the newest published
stable core within its range on every claimed platform. Add fixtures covering missing/unknown
contract markers, unsupported core versions, prereleases, additive fields, missing required and
optional capabilities, and both fatal and per-repository failures. Exercise supported read, plan,
and explicitly authorized execution flows against real binaries or an MCP server, not only mocked
JSON. Record the tested versions separately from the broader declared range.

Dropping an advertised core range or contract is a breaking change for that adapter and must be
versioned and release-noted in its repository. Extending support requires tests and a declaration
update; a new core major is not automatically supported even if it retains a familiar contract.

Release-please creates the core tag when the release PR is merged. Tags are release candidates until
the qualification and publication workflow succeeds; see [RELEASE.md](../RELEASE.md). Main-branch
builds, an open release PR, and a candidate tag alone are not published-release compatibility
evidence. Qualify the adapter against the final published version before advertising it as tested.
