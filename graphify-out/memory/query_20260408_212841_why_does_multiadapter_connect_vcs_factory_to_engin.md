---
type: "query"
date: "2026-04-08T21:28:41.675717+00:00"
question: "Why does MultiAdapter connect VCS Factory to Engine More Internal?"
contributor: "graphify"
source_nodes: ["MultiAdapter", "NewAdapterForSelection()", "planAdapter"]
---

# Q: Why does MultiAdapter connect VCS Factory to Engine More Internal?

## Answer

`MultiAdapter` bridges `VCS Factory` to `Engine More Internal` mostly because both areas sit on the same adapter abstraction, not because the engine internals directly depend on `MultiAdapter` at runtime.

The strongest direct evidence is in `internal/vcs/factory.go:41-63`, where `NewAdapterForSelection()` returns `MultiAdapter` when more than one backend is selected, and in `internal/vcs/factory.go:65-77`, where `MultiAdapter` is defined as the per-path delegating adapter for mixed VCS roots.

On the engine side, the relevant node is `planAdapter` in `internal/engine/engine_more_internal_test.go:17-27`. That test type implements the same adapter-shaped surface with methods like `Fetch`, `PullRebase`, `Push`, `Clone`, `NormalizeURL`, and `PrimaryRemote` in `internal/engine/engine_more_internal_test.go:29-92`. The graph links these areas because they both revolve around the same VCS adapter contract.

The shortest graph path from `MultiAdapter` to `planAdapter` is indirect: `MultiAdapter` -> `factory.go` -> `model.go` -> `engine_more_internal_test.go` -> `planAdapter`. That means the current bridge is primarily structural, driven by file/import containment plus shared adapter semantics, rather than a clean feature-level call path.

So the practical reading is: `MultiAdapter` matters to engine internals because the engine is designed against adapter behavior, and the engine tests model that behavior with `planAdapter`. The graph does not show a strong direct production-code dependency from `MultiAdapter` into the `Engine More Internal` cluster; it shows a contract-level connection.

## Source Nodes

- MultiAdapter
- NewAdapterForSelection()
- planAdapter