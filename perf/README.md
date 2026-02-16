# Performance Tracking

- Run `go tool task perf-bench` to execute benchmarks and append a timestamped history record.
- The default benchmark mode uses `-benchtime 1x` for predictable runtime in large repo sets; increase benchtime manually when you want lower-noise measurements.
- Structured history is stored in `perf/history.jsonl` (JSON Lines format).
- Raw `go test -bench` output is stored in `perf/runs/` (ignored by git).

The benchmark runner also prints a delta versus the previous history entry for quick regression checks.
