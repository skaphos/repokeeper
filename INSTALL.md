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

## From source

```bash
go install github.com/skaphos/repokeeper@latest
```

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
