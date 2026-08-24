# MCP Tool Matrix Contract

## Coverage Gate

The table records the current canonical cases, but the runtime contract is not a hard-coded count. After real-stdio initialization, the harness MUST compare the exact set of names returned by `tools/list` with the keys in the case map.

Failure diagnostics list both sorted differences:

- `missing cases`: discovered tools with no E2E case;
- `unexpected cases`: declared cases no longer registered.

Every discovered tool receives a valid-input success call. Every tool discovered with `destructiveHint=true`, plus every tool with an explicit confirmation gate, also requires a refusal call and a confined success assertion.

## Canonical Fixtures

- **clean metadata repository**: clean tracked `main`, local bare origin addressed by a constructed `file://` URL, valid committed `.repokeeper-repo.yaml`, authoritative/low-value paths, entrypoints, labels, and relation to the dirty repository. Its remote is advanced from an unscanned source clone after the fixture checkout is established so fetch has an observable result without moving local HEAD.
- **dirty repository**: tracked `main`, local bare origin, one declared uncommitted file, and stable baseline HEAD.
- **missing entry**: seeded registry entry with an absent path beneath the workspace.
- **reserved add/remove remote**: populated local bare repository outside the scan root, with no target checkout until `add_repository` runs.

Repository IDs that incorporate local absolute remote paths are captured from actual scan output or normalization; they are never hard-coded.

## Current Canonical Cases

| Tool | Valid input | Response contract | State contract |
| --- | --- | --- | --- |
| `scan_workspace` | `roots: [workspace]`, `prune_stale: false` | Top-level `discovered`, `new`, `missing`, `pruned`, `repos`; two live repos become new and the seeded missing entry remains missing | Config/registry persists all three entries; clean and dirty HEAD/content unchanged |
| `list_repositories` | Empty or stable label/status selector | `repositories` array with required `repo_id`, `path`, `status`, `last_seen` | No state change |
| `get_repository_context` | Clean metadata repository absolute path or unambiguous ID | Top-level `repo_id`, `path`, `bare`, `head`, `tracking`, `submodules`, and expected optional metadata/worktree fields | No state change |
| `get_workspace_config` | Empty object | Top-level `config_path`, `registry_stale_days`, `defaults`, `repo_count` | Exact isolated config path and count; no state change |
| `build_workspace_inventory` | `filter: all`, `concurrency: 1` | Top-level `generated_at`, `repos`; entries carry live or missing health detail | No durable state beyond documented metadata refresh; Git state unchanged |
| `select_repositories` | `name_match` for known repository | `repositories` array with `repo_id`, `path`, `match_reason` and expected labels | No state change |
| `get_repo_metadata` | Clean metadata repository | Direct metadata object with `apiVersion`, `kind`, `name`, `entrypoints`, `paths`, `related_repos` | Must exercise populated metadata, not the valid-but-weaker `null` response |
| `get_authoritative_paths` | Clean metadata repository | Top-level `authoritative`, `low_value`, `entrypoints` matching committed metadata | No state change |
| `get_related_repositories` | Clean metadata repository | `repositories` array with dirty repo `repo_id`, relationship, local path, and status | No state change |
| `plan_sync` | `filter: clean`, `update_local: false`, `push_local: false` | `plan` array with `repo_id`, `path`, `action`, `outcome`, `planned`, `remote_tracking_refs` | Dry-run: config, registry, refs, HEAD, and files unchanged |
| `execute_sync` | Refusal: `filter: clean`, `confirm: false`; success: same with `confirm: true`, update/push false | Refusal is `isError` with safety-gate text. Success has `results`; clean repo reports successful fetch and missing entry reports an explicit skipped outcome rather than making the call an error | Refusal snapshot byte-for-byte unchanged. Confirmed fetch observes advanced remote refs but preserves local HEAD and all worktree files |
| `set_labels` | Known repo plus `set: {e2e: "true"}` | Top-level `repo_id`, `labels` | Label persists in isolated config/registry and is visible to a later read |
| `add_repository` | Reserved local bare URL, new target under scenario root, `mirror: false` | Top-level `repo_id`, target `path`, `status: cloned` | Target is a valid clone at expected commit and registry entry persists |
| `remove_repository` | Refusal: added target with `delete_files: true`, `confirm: false`; success: same with `confirm: true` | Refusal is `isError` with safety-gate text. Success has `repo_id`, `removed: true` | Refusal leaves directory and registry byte-for-byte unchanged. Confirmed call removes only the validated target and its registry entry |

## Deterministic Call Order

Use one fresh MCP materialization, seeded with the non-nil missing-only registry required for server startup:

1. initialize;
2. `tools/list` and exact coverage/annotation gate;
3. capture global baseline;
4. `scan_workspace`;
5. all read tools in stable name order, except `plan_sync` is retained with the sync phase;
6. `plan_sync`;
7. `execute_sync` refusal, then confirmed fetch-only success;
8. `set_labels`;
9. `add_repository`;
10. `remove_repository` refusal, then confirmed deletion last;
11. reload config/registry from disk and assert global containment, HEAD, and worktree invariants;
12. close the stdio session and validate recorded frames/readers/process termination.

`remove_repository` is last because every MCP handler reloads config from disk and its confirmed path deliberately consumes the clone created by `add_repository`.

## Result Shape Rules

- List-shaped handlers return their array in structured content under `repositories`, `plan`, or `results` as named above.
- Object handlers return their documented fields at the top level.
- Assertions validate required keys and representative values, not just JSON parseability.
- Raw structured content is decoded into case-specific typed test structs so numeric and nested shapes are not weakened by generic string matching.
- `isError=false` is required for success calls. A per-entry skipped result, such as the deliberate missing repository during sync, is asserted semantically and does not imply an MCP-level error.
