# Quickstart: Validating Distribution Channel Conformance

**Feature**: `specs/001-distribution-channels` | **Date**: 2026-07-26

Runnable checks that prove each channel works. Grouped by what they need, so the parts requiring a
published release are clearly separated from the parts that run on any working copy.

Details live in the contracts: [`cli-version.md`](contracts/cli-version.md),
[`container-image.md`](contracts/container-image.md),
[`release-artifacts.md`](contracts/release-artifacts.md),
[`server-json.md`](contracts/server-json.md).

## Prerequisites

| Check | Needs |
| --- | --- |
| 1–3 | Go toolchain, working copy |
| 4–5 | Docker with buildx |
| 6–7 | A published release |

```bash
go version          # 1.26.5 per go.mod
docker buildx version
goreleaser --version   # 2.17.0 per .tool-versions
```

---

## 1. Version identity across install paths (FR-001 – FR-006)

The core assertion: **no install path that produces a usable binary reports a placeholder.**

```bash
# Local build — expect a revision, no invented version
go build -o /tmp/rk-local . && /tmp/rk-local version

# Release-style stamping — expect output identical to today's releases
go build -ldflags "-X github.com/skaphos/repokeeper/cmd/repokeeper.Version=v9.9.9 \
  -X github.com/skaphos/repokeeper/cmd/repokeeper.Commit=abc123 \
  -X github.com/skaphos/repokeeper/cmd/repokeeper.Date=2026-07-26T00:00:00Z" \
  -o /tmp/rk-stamped . && /tmp/rk-stamped version

# Module install — expect a real version from the proxy
go install github.com/skaphos/repokeeper/cmd/repokeeper@latest && repokeeper version

# Modified tree — expect the dirty marker
touch README.md && go build -o /tmp/rk-dirty . && /tmp/rk-dirty version
```

**Pass**: none of the four prints `dev`, a bare `(devel)`, `none`, or `unknown` as a version. The
stamped build prints exactly what it prints today. The dirty build says so.

```bash
/tmp/rk-local version --output json | jq -e '.version != "dev" and .source != null'
```

## 2. Unit tests

```bash
go test ./internal/buildinfo/... ./internal/mcpserver/... -race
go -C tools tool task -d .. ci        # full local gate before opening a PR
```

## 3. Manifests validate (FR-017)

```bash
goreleaser check
pipx run check-jsonschema \
  --schemafile https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json \
  server.json
```

**Pass**: both exit 0. `goreleaser check` also confirms the `nfpms` and `dockers_v2` blocks parse.

## 4. Build the image locally

```bash
goreleaser release --snapshot --clean --skip=publish
docker images | grep repokeeper
```

**Pass**: `.deb`, `.rpm`, both archives and a local image appear under `dist/`. Confirm packages have
their own SBOM — the easiest requirement to miss:

```bash
ls dist/*.deb dist/*.rpm
ls dist/*.deb.sbom.json dist/*.rpm.sbom.json   # must exist (FR-014)
```

## 5. Container behavior (FR-024 – FR-029)

The highest-risk area. Set up a throwaway workspace first:

```bash
WS=/tmp/rk-ws && rm -rf $WS && mkdir -p $WS/demo
git init -q $WS/demo && (cd $WS/demo && git commit -qm init --allow-empty)
# create $WS/.repokeeper.yaml registering $WS/demo, e.g. via: repokeeper scan $WS
IMG=ghcr.io/skaphos/repokeeper:snapshot
```

### 5a. Read-only inspection works

```bash
docker run -i --rm -v $WS:$WS:ro $IMG mcp --config $WS/.repokeeper.yaml
```

**Pass**: server starts on stdio; an inventory call lists `demo`. **Fail** if any tool reports
`detected dubious ownership` — that means the `safe.directory` gitconfig is missing from the image,
and it is the single most likely defect in this feature.

Direct check of the ownership guard:

```bash
docker run --rm -u 65532:65532 -v $WS/demo:$WS/demo:ro -w $WS/demo $IMG \
  --entrypoint git status --porcelain
```

**Pass**: exit 0. **Fail**: exit 128 with `dubious ownership`.

### 5b. Mutating tools refuse, and explain

Call `execute_sync` against the read-only mount.

**Pass**: refuses, names the read-only mount as the cause, states that a read-write mount and
credentials would enable it. **Fail**: a raw `read-only file system` path error, a raw git error, or
the tool missing from the advertised surface entirely.

### 5c. Read-only tools are unaffected by 5b

Call an inventory tool after the refusal. **Pass**: still works. Degradation is read-only, not blind
(Principle VII).

### 5d. No workspace

```bash
docker run -i --rm $IMG mcp
```

**Pass**: names the path searched, states discovery walks upward, states how to supply a workspace.
**Fail**: an empty inventory reported as a successful scan.

### 5e. Multi-root pinning

Create a second root and configure two entries per
[`container-image.md`](contracts/container-image.md).

**Pass**: each entry answers from its own root, and neither silently serves the other's. This is the
container's most dangerous failure — a confident, complete answer about the wrong workspace.

### 5f. Native multi-root is unchanged

```bash
cd $WS/demo && repokeeper list          # finds $WS/.repokeeper.yaml by walking up
cd /some/other/root/sub && repokeeper list   # finds that root's registry
```

**Pass**: position-sensitive discovery behaves exactly as before. This feature must not change it.

## 6. Installed packages (needs a release)

```bash
docker run --rm -v $PWD/dist:/pkgs debian:stable-slim \
  bash -c 'dpkg -i /pkgs/repokeeper_*_amd64.deb && repokeeper version && \
           test -f /usr/share/doc/repokeeper/LICENSE && \
           dpkg -r repokeeper && ! command -v repokeeper'

docker run --rm -v $PWD/dist:/pkgs fedora:latest \
  bash -c 'rpm -i /pkgs/repokeeper-*.x86_64.rpm && repokeeper version && \
           rpm -V repokeeper && rpm -e repokeeper'
```

**Pass**: binary on `PATH`, reports the release version, license and notice files present, clean
removal leaving the package database consistent.

## 7. Published channels (needs a release)

Mirrors the `verify` job — run it by hand when diagnosing a release.

```bash
V=0.8.0
gh release view "v$V" --json assets --jq '.assets[].name' | sort
curl -sf https://raw.githubusercontent.com/skaphos/homebrew-tools/main/Casks/repokeeper.rb | grep "version \"$V\""
docker buildx imagetools inspect ghcr.io/skaphos/repokeeper:$V --raw | \
  jq -e '[.manifests[].platform | select(.os=="linux") | .architecture] | inside(["amd64","arm64"]) | not' >/dev/null || echo "both arches present"
curl -sf "https://registry.modelcontextprotocol.io/v0/servers?search=io.skaphos/repokeeper" | jq '.servers[].version'
```

**Pass**: every channel serves `$V`. A channel that responds with the *previous* version is the
failure this feature exists to catch — reachability is not the test.

## 8. Zero dependency growth (FR-037)

```bash
git diff --exit-code go.mod go.sum && echo "no dependency change"
go mod tidy && git diff --exit-code go.mod go.sum && echo "tidy clean"
```

**Pass**: both clean. The entire self-update decision rests on dependency cost; a feature justified
by avoiding +59 requirements must not add any by accident.

---

## Acceptance summary

| # | Validates | Requirements |
| --- | --- | --- |
| 1, 2 | Version identity on every install path | FR-001 – FR-006 |
| 3 | Manifests valid before merge | FR-017 |
| 4 | Packages built, each with its own SBOM | FR-011 – FR-014 |
| 5a–5d | Container read/refuse/explain behavior | FR-024 – FR-027 |
| 5e, 5f | One root per entry; native crawl unchanged | FR-026, FR-029 |
| 6 | Install, verify, remove cleanly | FR-012, FR-013 |
| 7 | Every channel serves the released version | FR-033 – FR-035 |
| 8 | No dependency growth | FR-037 |
