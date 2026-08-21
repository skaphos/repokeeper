# Contributing Guidelines

Thanks for contributing to RepoKeeper.

## Development Setup

- Go version: see the `go` directive in `go.mod`.
- Run task targets without installing tools globally:
  - `go -C tools tool task --list`

## Graphify

The canonical code graph is **committed and shared** so contributors and agents get it on clone without a rebuild. Three artifacts under `graphify-out/` are tracked:

- `graph.json` — the graph itself (what `graphify query`/`path`/`explain`/`affected` and cross-repo `merge-graphs` consume).
- `GRAPH_REPORT.md` — the human-readable report.
- `.graphify_labels.json` — community labels.

Everything else under `graphify-out/` is local build state and stays untracked (`.gitignore`): the extraction `cache/`, the incremental `manifest.json` (per-machine file mtimes), the regenerable `graph.html` viz, and dated snapshot directories.

Local setup (per machine, not tracked):

- Install `graphify` into a Python environment you control.
- Record the interpreter that can import `graphify`:
  - `python3 -c "import sys; open('.graphify_python', 'w').write(sys.executable)"`
- Install or refresh the local git hooks **and the `graphify` merge driver**:
  - `graphify hook install`
- The hooks live under `.git/hooks` and are not tracked. They read `.graphify_python` first, then fall back to `python3`.
- `graphify hook install` also registers the `graphify` merge driver referenced by `.gitattributes`, which union-merges `graph.json` so parallel rebuilds do not conflict. Without it, git falls back to a normal merge for that file.

Notes:

- The post-commit hook rebuilds the graph after each commit, which stamps `graph.json` with the new `built_at_commit` and leaves it modified in your working tree; commit that refresh (or let the next change carry it). The committed graph therefore trails HEAD by one commit.
- The checkout hook only rebuilds after `graphify-out/` already exists — which it now does on a fresh clone, since the graph is committed.

## Branching and Commits

- Create focused branches from `main`.
- Keep commits small and scoped.
- Use DCO sign-offs on every commit:
  - `git commit --signoff ...`
  - Required trailer format: `Signed-off-by: Your Name <you@example.com>`
- Use Conventional Commits for the commits that land on `main`. Release Please uses them to infer the next version and release notes:
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

See `RELEASE.md` for the release flow (Release Please PR gate + `goreleaser` artifact publish) and downstream release automation.
