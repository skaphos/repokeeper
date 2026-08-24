<!-- SPDX-License-Identifier: MIT -->

# Recipe-driven end-to-end tests

This integration-only package builds the real RepoKeeper executable once, creates disposable local Git remotes and checkouts from JSON recipes, and exercises both the CLI and MCP stdio process boundaries. Every scenario uses an allowlisted environment with test-owned home, config, cache, temporary, Git identity, credential, and signing state. Fixture and product processes are bounded to 30 seconds and the suite never requires hosted remotes or credentials.

Run the suite with:

```sh
go test -v -count=1 -timeout 10m -tags integration ./test/e2e
go test -race -v -count=1 -timeout 10m -tags integration ./test/e2e
```

Focused stories use Ginkgo labels in the description:

```sh
go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus 'Real CLI workflows'
go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus 'Real MCP stdio'
go test -v -count=1 -tags integration ./test/e2e -ginkgo.focus 'Recipe extensibility'
```

Recipes live in `testdata/recipes/`. They contain portable slash paths and declarative repository state only; validation rejects absolute paths, traversal, collisions, unknown relationships, and unknown JSON fields before materialization writes anything. Add a scenario by adding a recipe and assertions that reuse `loadRecipe`, `materializeRecipe`, `buildChildEnvironment`, and `runCommand`.

Failures print the operation, argument vector, working directory, scenario root, config path, exit or launch error, timeout state, duration, stdout, and stderr. Workspaces are normally removed after the spec; use the diagnostic paths while reproducing a failing operation locally.

## Git compatibility command

`testdata/git-compatibility.json` is the executable source for the closed Git `2.53`/`2.54`/`2.55` claim across Linux, macOS, native Windows, and WSL. Routine pull request CI does not run the matrix. All twelve cells run on the release pull request as a merge gate, and a tagged full release must produce one successful evidence document for all twelve. The `routine` scope selects one declared cell per environment and is kept for local and on-demand runs.

```sh
go run -tags integration ./test/e2e/cmd/compatibility matrix --scope routine
go run -tags integration ./test/e2e/cmd/compatibility matrix --scope release
go run -tags integration ./test/e2e/cmd/compatibility verify-docs
go run -tags integration ./test/e2e/cmd/compatibility provision --cell linux-git-2.55 --prefix /tmp/repokeeper-git
```

Provisioning downloads only the declared archive, verifies its SHA-256, installs beneath the supplied prefix, and requires exact `git --version` output. WSL imports the pinned Canonical Noble `20240423` rootfs as WSL1, executes Linux binaries inside it, and unregisters the job-owned distribution during always-run cleanup. A cell never falls back to runner Git or another environment.

Release qualification uploads bounded per-cell JSON even on failure. The evidence gate rejects missing, duplicate, unexpected, skipped, failed, version-mismatched, digest-mismatched, tag-mismatched, or commit-mismatched results. Rerun an unchanged tag only for transient infrastructure failure; source or declaration corrections require a new tag.
