# Contributing Guidelines

Thanks for contributing to RepoKeeper.

## Development Setup

- Go version: see `go.mod` (`go 1.26.1`).
- Run task targets without installing tools globally:
  - `go -C tools tool task --list`

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

- `feat(reconcile): add checkout-missing support`
- `fix(get): align colored table output`

## Coding Standards

- Follow Go conventions and keep code readable.
- Keep REUSE metadata valid:
  - Source files should include SPDX headers (for example an `SPDX-License-Identifier` header with value `MIT`).
  - Use `reuse lint` to validate licensing metadata.
- Format code:
  - `go -C tools tool task fmt`
- Lint code:
  - `go -C tools tool task lint`

## Testing

Run before opening a PR:

- `go -C tools tool task test`
- `go -C tools tool task test-cover`
- `go -C tools tool task test-integration`
- `go -C tools tool task staticcheck`
- `go -C tools tool task vuln`

Or run full local CI:

- `go -C tools tool task ci`

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
