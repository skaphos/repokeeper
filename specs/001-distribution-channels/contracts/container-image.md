# Contract: `ghcr.io/skaphos/repokeeper`

**Requirements**: FR-021 – FR-029 | **Artifacts**: `Dockerfile`, `.dockerignore`, `dockers_v2` block

The container image contract. This is where RepoKeeper diverges most from sting, so the divergences
are stated before the interface.

## Divergences from the sting reference

| Aspect | sting | RepoKeeper | Why |
| --- | --- | --- | --- |
| Base | `gcr.io/distroless/static` | Alpine **with git** | `internal/gitx` shells out to `git`; distroless has none |
| Filesystem | None needed | Bind-mounted workspace | Tools read local working trees |
| Git ownership | n/a | `safe.directory=*` required | uid mismatch fails **every** git call |
| Workspace | n/a | One explicit root per entry | Container cwd is fixed; no upward crawl |
| Credentials | Provider tokens | Only for remote operations | Inspection is local |

## Image properties

| Property | Value |
| --- | --- |
| Registry | `ghcr.io/skaphos/repokeeper` |
| Tags | `{{ .Version }}`, plus `latest` on non-prereleases |
| Platforms | `linux/amd64`, `linux/arm64` |
| User | `65532:65532`, non-root (FR-023) |
| Entrypoint | `/usr/local/bin/repokeeper` |
| Default command | `mcp` (FR-022) |
| `WORKDIR` | **None** — deliberately unset (FR-026) |
| Contains | `repokeeper`, `git`, CA certificates, a system gitconfig |
| Does not contain | Credentials, user config, a registry file, Mercurial |

**No build stage.** The Dockerfile copies the binary GoReleaser already built, laid out per platform
as `linux/amd64/repokeeper` and `linux/arm64/repokeeper` and selected with `ARG TARGETPLATFORM`.
Compiling in the image would ship different bytes from the ones signed, notarized and attested.

## Invocation contract

### Read-only inspection — the documented default

```
docker run -i --rm \
  -v /home/me/work/skaphos:/home/me/work/skaphos:ro \
  ghcr.io/skaphos/repokeeper:0.8.0 \
  mcp --config /home/me/work/skaphos/.repokeeper.yaml
```

Mount path and container path are identical, so every absolute path in the registry resolves
unchanged. `--config` names the registry outright rather than relying on a search whose starting
point is invisible in an MCP client configuration.

### Read-write

```
docker run -i --rm \
  --user "$(id -u):$(id -g)" \
  -v /home/me/work/skaphos:/home/me/work/skaphos \
  ghcr.io/skaphos/repokeeper:0.8.0 \
  mcp --config /home/me/work/skaphos/.repokeeper.yaml
```

`--user` matching the host owner is **recommended** for read-write use so files git creates land with
correct ownership. It is not required for reads — the baked `safe.directory=*` covers those.

### Multiple workspace roots

One MCP server entry per root. There is no configuration that makes a single container serve several.

```jsonc
{
  "mcpServers": {
    "repokeeper-skaphos": { "command": "docker", "args": ["run","-i","--rm",
      "-v","/home/me/work/skaphos:/home/me/work/skaphos:ro",
      "ghcr.io/skaphos/repokeeper:0.8.0",
      "mcp","--config","/home/me/work/skaphos/.repokeeper.yaml"] },
    "repokeeper-alaska":  { "command": "docker", "args": ["run","-i","--rm",
      "-v","/home/me/work/alaska:/home/me/work/alaska:ro",
      "ghcr.io/skaphos/repokeeper:0.8.0",
      "mcp","--config","/home/me/work/alaska/.repokeeper.yaml"] }
  }
}
```

## Behavioral contract

| Given | When | Then |
| --- | --- | --- |
| RO mount, valid registry | read-only tool called | Identical result to native (FR-025) |
| RO mount, valid registry | mutating tool called | Refuses, names the read-only mount and the remedy; inspection unaffected |
| No workspace at the path | any tool called | Names the path searched, states discovery walks upward, states how to supply one (FR-027) |
| Registry outside the mount | discovery runs | Not adopted; treated as not found |
| Remote operation, no credentials | sync/fetch attempted | States the credential is missing — not that the remote is unreachable |
| `--vcs hg` | any tool | States Mercurial is unavailable in the container (FR-029) |
| Partial mount | inventory called | Each unresolved repository reported individually with its reason |

## The git ownership requirement

Measured, not assumed. Git ≥ 2.35.2 refuses a repository directory owned by a different uid:

```
fatal: detected dubious ownership in repository at '/home/me/work/skaphos/repokeeper'
```

Verified against git 2.55.0 with a real bind-mounted repository:

| Configuration | Result |
| --- | --- |
| uid 65532, no gitconfig | exit **128** on every git call |
| uid 65532, `safe.directory=*` | exit 0 |
| uid matching host owner | exit 0 |

The image therefore ships a system gitconfig setting `safe.directory=*`.

**Security note, to be stated in the Dockerfile and the docs.** `safe.directory=*` is discouraged on
a shared host because it lets git act on repositories owned by other users. Inside this container the
filesystem view *is* whatever the user chose to mount, so there is no foreign repository to be
tricked into trusting. The reasoning is recorded rather than left implicit (Principle IX).

## Supply chain (FR-028)

| Guarantee | Mechanism |
| --- | --- |
| Same bytes as archives | Binary copied, not compiled |
| SBOM | Attached to the image |
| Signature | cosign |
| Provenance | Build provenance attestation |
| Single invocation | Built and pushed inside the GoReleaser run (FR-031) |

## OCI labels

`org.opencontainers.image.{title,description,url,source,version,revision,licenses}`, with `version`
and `revision` templated from the release.

## Verification (FR-033)

`docker buildx imagetools inspect ghcr.io/skaphos/repokeeper:<version> --raw` must resolve and its
manifest must contain both `linux/amd64` and `linux/arm64`. Retried with backoff so a registry outage
is distinguishable from a channel that did not publish.
