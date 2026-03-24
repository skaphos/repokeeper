# Repository Guidelines

## Project Structure & Module Organization
- `cmd/repokeeper/`: CLI command wiring (Cobra commands like `init`, `scan`, `status`, `sync`, `version`).
- `internal/`: core packages
- `internal/config`, `internal/registry`, `internal/discovery`, `internal/engine`, `internal/gitx`, `internal/model`, `internal/tui`.
- `main.go`: entrypoint that boots the CLI.
- Docs: `README.md` (usage), `DESIGN.md` (architecture), `TASKS.md` (milestones).

## Build, Test, and Development Commands
- `go build -o repokeeper .`: build the local binary.
- `go install .`: install binary to `$GOPATH/bin` from local source.
- `go clean -i github.com/skaphos/repokeeper`: uninstall the binary.
- `go run github.com/onsi/ginkgo/v2/ginkgo@v2.28.1 ./...`: run the Ginkgo test suite.
- `go test -coverprofile=coverage.out ./...`: run tests with coverage output.
- `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...`: run linting (gofmt/goimports and static checks; v2 config).
- `goreleaser build --snapshot --clean`: snapshot build for all platforms (goreleaser managed via `.tool-versions`, not `go tool`).

## Coding Style & Naming Conventions
- Go version: `go 1.26` (see `go.mod`).
- Formatting: `gofmt` and `goimports` are enforced via `golangci-lint`.
- Naming: follow Go conventions (exported `PascalCase`, unexported `camelCase`).
- Tests: filename suffix `_test.go`; suite files follow `*_suite_test.go`.

## Testing Guidelines
- Frameworks: Ginkgo v2 + Gomega (see `go.mod`).
- Prefer small, focused specs; keep fixtures in the same package when possible.
- New functionality must include meaningful tests in the same change; avoid shipping new behavior without direct coverage.
- Run locally with `go run github.com/onsi/ginkgo/v2/ginkgo@v2.28.1 ./...`; coverage with `go test -coverprofile=coverage.out ./...`.

## Engineering Guardrails
- Keep cognitive load low: prefer small functions, clear names, early returns, and simple control flow over clever abstractions.
- Write comments where intent is not obvious, especially around invariants, edge cases, and non-obvious tradeoffs.
- Avoid noise comments that restate the code; comments should explain why, not just what.

## Commit & Pull Request Guidelines
- **All changes must be delivered via a pull request. Never commit directly to `main`.**
- Branch naming: use a prefix that matches the change type:
  - `feature/<short-description>` — new functionality
  - `bug/<short-description>` — bug fixes
  - `chore/<short-description>` — maintenance, deps, tooling
  - `docs/<short-description>` — documentation only
  - `ci/<short-description>` — CI/CD pipeline changes
  - `refactor/<short-description>` — internal restructuring without behaviour change
- Keep branches focused: one logical change per PR. Split unrelated concerns into separate PRs.
- All commits must be signed (global SSH signing via 1Password is configured; do not pass `-S` manually).
- Use concise, imperative subjects (example: "Add registry staleness check") and include context in the body if needed.
- **All commits MUST be cryptographically signed AND carry a DCO sign-off.** Always pass both `-S` and `-s` to `git commit` (e.g. `git commit -S -s -m "..."`). The repo uses SSH signing; the key and `commit.gpgsign = true` are set in git config. Never omit `-S`, never use `--no-gpg-sign`.
- For release automation, prefer Conventional Commits so `svu` can infer semantic version bumps:
  - `feat:` -> minor bump
  - `fix:` -> patch bump
  - `perf:` -> patch bump
  - `refactor:`, `chore:`, `task:`, `docs:`, `test:`, `build:`, `ci:` -> no release bump by default unless configured otherwise
  - Any `!` in the type/scope or a `BREAKING CHANGE:` footer -> major bump
  - Example subjects: `feat(sync): add opt-in pull --rebase`, `fix(ci): make coverage command shell-safe`
- PRs should include: summary, testing performed, and doc updates when behaviour changes (`README.md` or `DESIGN.md`).

## Configuration & Safety Notes
- Config lives in platform config dirs (example: `%APPDATA%\\repokeeper\\config.yaml` on Windows).
- The tool is designed to avoid modifying working trees; do not add commands that checkout/pull/reset repositories.

## Repository Docs & Agent Notes
- This guide is also available as `CLAUDE.md`, which is a symlink to `AGENTS.md`.
- Update `README.md` for user-facing behavior changes and `DESIGN.md` for architectural changes.
- Git operations should prefer the `git` CLI for correctness and parity with user environments; use libraries only when the CLI is a poor fit (performance, missing capability, or brittle parsing).
- Maintain the Git compatibility matrix in `DESIGN.md` when changing git invocation logic.
