# Contract: `server.json` — MCP Registry Entry

**Requirements**: FR-016 – FR-020 | **Artifacts**: `server.json`, `internal/mcpserver/serverjson_test.go`

The checked-in description of RepoKeeper's MCP server, published to the registry MCP clients read.

## Identity

| Field | Value |
| --- | --- |
| Registry name | `io.skaphos/repokeeper` |
| Namespace proof | DNS TXT record on `skaphos.io` (`v=MCPv1; k=ed25519`) |
| Publisher credential | `MCP_REGISTRY_KEY`, organization-scoped |
| Package identifier | `ghcr.io/skaphos/repokeeper` |
| Transport | `stdio` |

The namespace is **inherited, not re-decided** — sting established `io.skaphos` and the DNS proof.
The name is not cosmetic: it derives from the domain whose TXT record proves ownership, so changing
it either breaks publishing or publishes under a name nobody controls.

## Shape

```jsonc
{
  "$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
  "name": "io.skaphos/repokeeper",
  "description": "Multi-repo workspace inventory, tracking status and sync planning over a local Git workspace.",
  "repository": { "url": "https://github.com/skaphos/repokeeper", "source": "github" },
  "version": "0.0.0",                        // stamped from the tag at release
  "packages": [{
    "registryType": "oci",
    "identifier": "ghcr.io/skaphos/repokeeper",
    "version": "0.0.0",                      // stamped from the same tag
    "transport": { "type": "stdio" }
  }]
}
```

`0.0.0` is a checked-in placeholder. The release workflow stamps both `version` fields from the tag
with a single `jq` expression, so they cannot diverge — a mismatch would publish an entry naming an
image tag that does not exist.

## Describing a mixed tool surface — the RepoKeeper divergence

sting's entry asserts every tool is read-only, and its tests enforce that. **That assertion is false
for RepoKeeper**, which exposes both kinds:

| Class | Count | Examples |
| --- | --- | --- |
| Read-only (`ReadOnlyHint: true`) | 8 | `list_repositories`, `get_repository_context`, `build_workspace_inventory` |
| Mutating | 6 | `scan_workspace`, `plan_sync`, `execute_sync`, `set_labels`, `add_repository`, `remove_repository` |

The description must represent this honestly. Claiming a read-only server would misrepresent the
mutation surface to every client that reads the registry — the same class of defect as an entry
describing tools that do not exist.

The description must also state the container's workspace contract (FR-026): one explicitly named
workspace root per configured entry. The registry is where a prospective user first learns how the
server is configured, and the constraint is surprising enough that omitting it is misleading.

## Validation: two layers

Neither is sufficient alone.

### Layer 1 — schema, in CI

`check-jsonschema` against the published MCP registry schema, pinned to the version in
`.tool-versions`. Catches a malformed entry before merge rather than at publish time.

### Layer 2 — drift, in Go tests

Catches what schema validation cannot: an entry that is well-formed and *wrong*.

| Test | Asserts | Guards against |
| --- | --- | --- |
| Identity | `name` is exactly `io.skaphos/repokeeper` | Publishing under an uncontrolled namespace |
| Transport | `transport.type` is `stdio` | A client configured from the registry cannot connect |
| Package identity | `identifier` matches the image `dockers_v2` publishes | An entry pointing at a nonexistent image |
| Version consistency | Both `version` fields are equal | Half-stamped entries |
| **Tool surface** | The described surface agrees with `mcpserver.ReadOnlyToolNames()` | The registry describing a surface RepoKeeper does not have |
| Malformed input | Fails cleanly on a `packages` array that is empty or absent | A panicking test binary on the exact input the tests exist to report on |

The tool-surface test is the important one and is why this is checked in Go rather than in CI
tooling: `ReadOnlyToolNames()` derives from live `ReadOnlyHint` annotations on registered tools, so
it cannot drift from what the server actually exposes. FR-020 requires the entry move in the same
change as the tool surface; this test is what enforces it.

## Publishing (FR-018, FR-019)

```
mcp-publisher login dns --domain skaphos.io --private-key "$MCP_REGISTRY_KEY"
mcp-publisher publish
```

`mcp-publisher` is **pinned**, not `@latest`: a floating version makes releases non-reproducible and
lets an upstream change break a release.

| Property | Behavior |
| --- | --- |
| On failure | `::warning::`, release remains valid and complete |
| Blocking | No — the sole exempt channel (FR-034) |
| Retryable | Yes, without cutting a new release |
| Failure message names | The `skaphos.io` TXT record and `MCP_REGISTRY_KEY` |

Both named things fail quietly — an expired credential and a removed DNS record produce the same
generic authentication error unless the message says what to check.
