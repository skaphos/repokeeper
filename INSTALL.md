# Installation

RepoKeeper can be installed with Homebrew cask, from release binaries, or from source.

## Homebrew (cask)

```bash
brew tap skaphos/tools
brew install --cask skaphos/tools/repokeeper
```

Upgrade:

```bash
brew update
brew upgrade --cask skaphos/tools/repokeeper
```

If you recently pushed a tap update and Homebrew has not refreshed yet:

```bash
brew tap skaphos/tools
HOMEBREW_NO_INSTALL_FROM_API=1 brew upgrade --cask skaphos/tools/repokeeper
```

## From release binaries

Download the latest archive from:

- <https://github.com/skaphos/repokeeper/releases>

Supported build targets:

- Linux: `amd64`, `arm64`
- macOS: `amd64`, `arm64`
- Windows: `amd64`, `arm64`

After extracting, place `repokeeper` (or `repokeeper.exe` on Windows) somewhere on your `PATH`.

Windows archives are **not** Authenticode-signed, so SmartScreen will warn on them. This is a known,
accepted gap with its own follow-up, not an oversight — see
[DECISIONS/0001](https://github.com/skaphos/skaphos-resources/blob/main/DECISIONS/0001-distribution-channels-by-artifact-shape.md).

## Linux packages

Each release publishes `.deb` and `.rpm` packages for `amd64` and `arm64`, built from the same
binaries as the archives and covered by the same checksum manifest, SBOM, signature and build
provenance.

```bash
# Debian / Ubuntu
sudo dpkg -i repokeeper_<version>_amd64.deb

# Fedora / RHEL / openSUSE
sudo rpm -i repokeeper-<version>-1.x86_64.rpm
```

The package installs:

| Path | Contents |
| --- | --- |
| `/usr/bin/repokeeper` | the binary |
| `/usr/share/doc/repokeeper/LICENSE` | project license |
| `/usr/share/doc/repokeeper/THIRD_PARTY_NOTICES.md` | dependency notices |
| `/usr/share/doc/repokeeper/third_party_licenses/` | per-dependency licenses |

**There is no hosted apt or yum repository.** Shipping a `.deb` on a release page is not the same as
running an apt repo, and it is worth being explicit because users reasonably read it that way.
Hosting signed repositories is a materially larger commitment — GPG signing keys with their own
rotation story, repository metadata, hosting — and is deliberately out of scope. Upgrading means
downloading the next release's package:

```bash
sudo dpkg -i repokeeper_<newversion>_amd64.deb   # dpkg upgrades in place
sudo rpm -U repokeeper-<newversion>-1.x86_64.rpm
```

To remove:

```bash
sudo dpkg -r repokeeper
sudo rpm -e repokeeper
```

## Container image

```bash
docker pull ghcr.io/skaphos/repokeeper:latest
```

A multi-arch (`linux/amd64`, `linux/arm64`) image that runs `repokeeper mcp` over stdio, for
`docker`-based MCP client configurations. It exists to serve the MCP server use case without a local
toolchain; it is not positioned as a general-purpose way to run the CLI, whose natural home is the
host filesystem it inspects.

The image runs as a non-root user, contains no credentials, and carries the same SBOM, signature and
provenance guarantees as the release archives. It supports **Git only** — Mercurial is not installed.

See the [README](README.md#running-the-mcp-server-from-a-container) for the client configuration and
its workspace contract, which has real constraints worth reading before you rely on it.

## From source

```bash
go install github.com/skaphos/repokeeper@latest
```

Installed builds include the bundled MCP server (`repokeeper mcp`) used by Claude Code, Cursor, Windsurf, OpenAI Codex, and similar runtimes. See [docs/mcp-setup.md](docs/mcp-setup.md) for runtime configuration and the current MCP tool boundary, including currently shipped state-changing operations.

## From local source checkout

Install from a cloned repository:

```bash
cd /path/to/repokeeper
go install .
```

Uninstall:

```bash
go clean -i github.com/skaphos/repokeeper
```

Or manually remove the binary:

```bash
rm "$(go env GOPATH)/bin/repokeeper"
```

## Migration from old Homebrew formula install

RepoKeeper previously shipped as a Homebrew formula. If you installed that older package, switch to the cask:

```bash
brew uninstall repokeeper
brew install --cask skaphos/tools/repokeeper
```

Then verify:

```bash
repokeeper version
```
