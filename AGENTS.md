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
- `go tool ginkgo ./...`: run the Ginkgo test suite.
- `go test -coverprofile=coverage.out ./...`: run tests with coverage output.
- `go tool golangci-lint run ./...`: run linting (gofmt/goimports and static checks; v2 config).
- `goreleaser build --snapshot --clean`: snapshot build for all platforms.

## Coding Style & Naming Conventions
- Go version: `go 1.25` (see `go.mod`).
- Formatting: `gofmt` and `goimports` are enforced via `golangci-lint`.
- Naming: follow Go conventions (exported `PascalCase`, unexported `camelCase`).
- Tests: filename suffix `_test.go`; suite files follow `*_suite_test.go`.

## Testing Guidelines
- Frameworks: Ginkgo v2 + Gomega (see `go.mod`).
- Prefer small, focused specs; keep fixtures in the same package when possible.
- New functionality must include meaningful tests in the same change; avoid shipping new behavior without direct coverage.
- Run locally with `go tool ginkgo ./...`; coverage with `go test -coverprofile=coverage.out ./...`.

## Engineering Guardrails
- Keep cognitive load low: prefer small functions, clear names, early returns, and simple control flow over clever abstractions.
- Write comments where intent is not obvious, especially around invariants, edge cases, and non-obvious tradeoffs.
- Avoid noise comments that restate the code; comments should explain why, not just what.

## Commit & Pull Request Guidelines
- This checkout does not include Git history, so no project-specific commit convention is detectable.
- All commits must be signed (for example, use `git commit -S` with a configured GPG or SSH signing key).
- Use concise, imperative subjects (example: “Add registry staleness check”) and include context in the body if needed.
- For release automation, prefer Conventional Commits so `svu` can infer semantic version bumps:
- `feat:` -> minor bump
- `fix:` -> patch bump
- `perf:` -> patch bump
- `refactor:`, `chore:`, `task:`, `docs:`, `test:`, `build:`, `ci:` -> no release bump by default unless configured otherwise
- Any `!` in the type/scope or a `BREAKING CHANGE:` footer -> major bump
- Example subjects: `feat(sync): add opt-in pull --rebase`, `fix(ci): make coverage command shell-safe`
- PRs should include: summary, testing performed, and doc updates when behavior changes (`README.md` or `DESIGN.md`).

## Configuration & Safety Notes
- Config lives in platform config dirs (example: `%APPDATA%\\repokeeper\\config.yaml` on Windows).
- The tool is designed to avoid modifying working trees; do not add commands that checkout/pull/reset repositories.

## Repository Docs & Agent Notes
- This guide is also available as `CLAUDE.md`, which is a symlink to `AGENTS.md`.
- Update `README.md` for user-facing behavior changes and `DESIGN.md` for architectural changes.
- Git operations should prefer the `git` CLI for correctness and parity with user environments; use libraries only when the CLI is a poor fit (performance, missing capability, or brittle parsing).
- Maintain the Git compatibility matrix in `DESIGN.md` when changing git invocation logic.
