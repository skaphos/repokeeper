# Contributing Guidelines

Thanks for contributing to RepoKeeper.

## Development Setup

- Go version: see `go.mod` (`go 1.25`).
- Install task runner and tools used by `Taskfile.yml`:
  - `go tool task --list`

## Branching and Commits

- Create focused branches from `main`.
- Keep commits small and scoped.
- Use DCO sign-offs on every commit:
  - `git commit --signoff ...`
  - Required trailer format: `Signed-off-by: Your Name <you@example.com>`
- Prefer Conventional Commits for release automation:
  - `feat:` -> minor
  - `fix:` / `perf:` -> patch
  - `docs:`, `test:`, `ci:`, `chore:`, `refactor:` -> no bump by default

Examples:

- `feat(sync): add checkout-missing support`
- `fix(status): align colored table output`

## Coding Standards

- Follow Go conventions and keep code readable.
- Keep REUSE metadata valid:
  - Source files should include SPDX headers (for example an `SPDX-License-Identifier` header with value `MIT`).
  - Use `reuse lint` to validate licensing metadata.
- Format code:
  - `go tool task fmt`
- Lint code:
  - `go tool task lint`

## Testing

Run before opening a PR:

- `go tool task test`
- `go tool task test-cover`
- `go tool task test-integration`
- `go tool task staticcheck`
- `go tool task vuln`

Or run full local CI:

- `go tool task ci`

## Pull Requests

PRs should include:

- Summary of what changed
- Why the change is needed
- Testing performed (commands and results)
- Docs updates when behavior changes (`README.md`, `DESIGN.md`, `RELEASE.md`)

## Safety Expectations

- Do not introduce git operations that mutate user working trees unexpectedly.
- Keep sync actions safe by default (fetch/prune-first behavior).
- Add explicit opt-in flags for destructive behavior.

## Release Process

See `RELEASE.md` for tagging and release automation.
