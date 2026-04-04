# Third-Party Notices

RepoKeeper ships and uses open source software from third-party projects.

The canonical notice artifacts live under [`third_party_licenses/`](third_party_licenses):

- `runtime/` and `runtime-report.csv` cover the dependencies included in the distributed `repokeeper` binary.

## Regenerating notices

Regenerate the notice inventory and license texts whenever `go.mod` or `go.sum` changes:

```bash
go -C tools tool task notices
```

The task uses `github.com/google/go-licenses/v2@v2.0.1` to:

1. generate a CSV inventory of the applicable dependencies and detected licenses
2. copy the upstream license texts required for attribution into `third_party_licenses/`

## Crediting libraries correctly

When dependency changes are part of a pull request, make sure the change also:

1. regenerates `third_party_licenses/`
2. keeps the runtime CSV inventory in sync with the committed dependency graph
3. preserves the shipped notice files in release archives

If you add non-runtime development tooling, review its upstream license before adoption even though it is not part of the shipped binary notice set.

If `go-licenses` reports a missing or unclear runtime license, stop and review that dependency manually before shipping the change.
