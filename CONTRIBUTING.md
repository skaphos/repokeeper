# Contributing Guidelines

Thanks for contributing to RepoKeeper.

## Development Setup

- Go version: see `go.mod` (`go 1.26.2`).
- Run task targets without installing tools globally:
  - `go -C tools tool task --list`

## Graphify

- Graphify hooks in this repository are local-only. They live under `.git/hooks` and are not tracked in git.
- Install `graphify` into a Python environment you control.
- Record the interpreter that can import `graphify`:
  - `python3 -c "import sys; open('.graphify_python', 'w').write(sys.executable)"`
- Install or refresh the local hooks:
  - `graphify hook install`
- The hooks read `.graphify_python` first, then fall back to `python3`.
- The checkout hook only rebuilds after `graphify-out/` already exists, so create an initial graph once with your normal graphify workflow before relying on automatic rebuilds.

## Branching and Commits

- Create focused branches from `main`.
- Keep commits small and scoped.
- Use DCO sign-offs on every commit:
  - `git commit --signoff ...`
  - Required trailer format: `Signed-off-by: Your Name <you@example.com>`
- Use Conventional Commits for the commits that land on `main`. `skaphos/actions/release-pr` infers the next version from these via [`svu`](https://github.com/caarlos0/svu):
  - `feat:` -> minor
  - `fix:` / `perf:` -> patch
  - `docs:`, `test:`, `ci:`, `chore:`, `refactor:` -> no bump by default
  - `!` in the type/scope or a `BREAKING CHANGE:` footer -> major
- If you use squash merges, the final squash commit message must also follow Conventional Commit format.

Examples:

- `feat(reconcile): add checkout-missing support`
- `fix(get): align colored table output`

## Coding Standards

- Follow Go conventions and keep code readable.
- Keep REUSE metadata valid:
  - Source files should include SPDX headers (for example an `SPDX-License-Identifier` header with value `MIT`).
  - Use `reuse lint` to validate licensing metadata.
- Properly credit every library we ship or use for repo automation:
  - Regenerate `third_party_licenses/` with `go -C tools tool task notices` whenever `go.mod` or `go.sum` changes.
  - Review `THIRD_PARTY_NOTICES.md` and the generated runtime CSV inventory before merging dependency updates.
  - Review new development or CI tooling licenses before adoption, even when they are not part of the shipped binary notice set.
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

See `RELEASE.md` for the release flow (`skaphos/actions` PR gate + `goreleaser` publish) and downstream release automation.
