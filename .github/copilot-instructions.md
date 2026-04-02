# GitHub Copilot Instructions for RepoKeeper

RepoKeeper is a cross-platform Go CLI and TUI for multi-repo hygiene. It inventories repositories, reports drift, and performs safe sync actions without modifying user working trees by default.

## What Good Changes Look Like

- Prefer small, focused pull requests with one logical change.
- Keep code straightforward: small functions, clear names, early returns, simple control flow.
- Follow normal Go naming and layout conventions used in this repository.
- Preserve cross-platform behavior across macOS, Linux, Windows, and WSL.
- Prefer the `git` CLI for repository operations unless there is a strong reason to use a library.

## Safety Rules

- Do not introduce git behavior that unexpectedly mutates user working trees.
- Do not add default behavior that performs `checkout`, `reset`, `merge`, `rebase`, `stash`, or submodule updates.
- Never update submodules as part of normal repo maintenance flows.
- Keep destructive or state-changing behavior explicit and opt-in behind flags or dedicated commands.
- Preserve the distinction between machine-local registry state and repo-local metadata files.

## Codebase Shape

- CLI command wiring lives in `cmd/repokeeper/`.
- Core logic lives under `internal/`, especially `internal/config`, `internal/discovery`, `internal/engine`, `internal/gitx`, `internal/mcpserver`, `internal/model`, `internal/registry`, `internal/tui`, and `internal/vcs`.
- `main.go` is the entrypoint.
- User-facing documentation lives in `README.md`, `INSTALL.md`, `docs/`, and command behavior/design notes live in `DESIGN.md`.

## Testing Expectations

Before proposing a PR, prefer running the repo’s task-based checks from `tools/`:

- `go -C tools tool task fmt`
- `go -C tools tool task lint`
- `go -C tools tool task test`
- `go -C tools tool task test-cover`
- `go -C tools tool task test-integration`
- `go -C tools tool task staticcheck`
- `go -C tools tool task vuln`

If a change is small, run the narrowest relevant tests at minimum. New behavior should include direct test coverage.

## Documentation Expectations

- Update `README.md` for user-visible behavior changes.
- Update `DESIGN.md` for architectural changes or git invocation logic changes.
- Update `RELEASE.md` when release or packaging behavior changes.
- Keep the git compatibility matrix in `DESIGN.md` accurate when changing git behavior.

## Go and Repository Conventions

- Use the Go version declared in `go.mod`.
- Keep files `gofmt` and `goimports` clean.
- Maintain REUSE/SPDX metadata. New source files should include the repository’s expected SPDX license header.
- Tests should use the existing Ginkgo v2 and Gomega conventions where applicable.

## Pull Request Instructions

When drafting a pull request for this repository:

- Explain what changed and why.
- Summarize user-visible or behavior changes clearly.
- List the exact tests and checks that were run, with outcomes.
- Call out doc updates when behavior changed.
- Mention any residual risks, limitations, or follow-up work if relevant.

## Commit and Branch Guidance

- Never target direct commits to `main`; changes should land through pull requests.
- Prefer focused branch names such as `feature/...`, `bug/...`, `chore/...`, `docs/...`, `ci/...`, or `refactor/...`.
- Prefer Conventional Commit subjects such as `feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`, or `refactor:`.
- Commits in this repository are expected to be signed and include a DCO sign-off.

## When Unsure

- Choose the safer behavior.
- Avoid expanding scope beyond the requested change.
- Match existing command patterns, test style, and output conventions instead of inventing new ones.
