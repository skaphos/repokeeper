# Recipe Contract

## Purpose

The recipe is a version-controlled JSON artifact under `test/e2e/testdata/recipes/` used to produce real Git and RepoKeeper state. It is decoded into test-internal typed Go data and is not a user-facing RepoKeeper file format or compatibility guarantee.

## Canonical Topology

Each materialization owns one root with this shape:

```text
<scenario-root>/
├── home/                 # HOME/USERPROFILE and platform config dirs
├── cache/                # test-owned caches
├── tmp/                  # child process temp files
├── config/
│   ├── config.yaml       # exact REPOKEEPER_CONFIG target
│   └── registry.yaml     # when external registry mode is selected
├── remotes/              # local bare Git remotes, never scanned
├── sources/              # helper clones used to advance remotes, never scanned
└── workspace/            # sole RepoKeeper scan root
    ├── clean repo/
    ├── dirty-repo/
    └── missing-repo/     # deliberately absent
```

Paths with spaces are intentional in at least one canonical fixture.

## Preflight Contract

Before any fixture path is created, validation MUST:

1. Strictly decode one JSON object, reject trailing JSON values and unknown fields, and require `schema_version` to equal `1`.
2. Reject empty or duplicate recipe, repository, branch, and entry names.
3. Reject absolute logical paths.
4. Reject any cleaned path equal to `.` or beginning with a parent traversal.
5. Join each logical path to its declared namespace root, calculate absolute paths, and require `filepath.Rel(namespaceRoot, candidate)` to remain neither `..` nor prefixed by `..` plus a separator.
6. Reject relationships to undeclared repositories, branches, remotes, or upstreams.
7. Reject path collisions between live repositories, helper paths, and missing entries.
8. Reject an existing symlink in any destination parent component before writing.
9. Return an error naming the recipe field and offending value.

Validation MUST be pure with respect to the filesystem except for read-only checks needed to establish the supplied scenario root.

## Materialization Contract

- Run Git with argument arrays and a bounded context; never construct a shell command.
- Use only local `file://` remote URLs constructed with URL/path APIs, never string concatenation. This preserves a parseable remote identity on Windows and Unix.
- Set explicit author and committer name, email, and time.
- Disable commit/tag signing, credential helpers, terminal prompting, and system Git configuration.
- Create the base branch explicitly as `main`; do not depend on host `init.defaultBranch`.
- Commit repository metadata before recording baseline HEADs.
- Push declared upstream branches before applying uncommitted changes.
- Apply dirty files last and record their byte content and porcelain status.
- Write the missing registry entry without creating its checkout path.
- Persist a non-nil registry for MCP startup. For the MCP materialization it initially contains only the deliberate missing entry, leaving live discovery observable.
- Derive live repository IDs from actual scan results or the same normalized remote identity; do not hard-code temporary absolute prefixes.

## Ready-State Invariants

The materializer returns only after it has verified:

- all live checkout paths are Git working trees;
- all bare remote paths are bare Git repositories;
- expected branches and upstreams resolve;
- committed HEADs match the recorded baselines;
- the clean repository has empty porcelain status;
- the dirty repository has exactly the declared uncommitted content;
- the missing entry path is absent;
- every writable and mutation-capable path is contained beneath the scenario root;
- config and registry reload successfully from disk.

Partial setup errors include the logical operation, target path, Git arguments, exit status, stdout, and stderr.

## Reuse Contract

A new scenario may declare different recipe data and assertions, but MUST reuse the same path validation, environment construction, Git bootstrap, process runner, cleanup, and diagnostic formatting. Scenario-specific imperative bootstrap code is allowed only when the shared vocabulary is first shown to be unable to express a needed topology and the vocabulary is extended in the same change.
