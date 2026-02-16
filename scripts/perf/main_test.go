package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBenchmarkMetrics(t *testing.T) {
	raw := `
goos: linux
goarch: amd64
BenchmarkSyncDryRunPlan-8   	    1000	   12345 ns/op	    512 B/op	      10 allocs/op
BenchmarkStatusReport-8      	    2000	    6789 ns/op	    256 B/op	       4 allocs/op
PASS
`
	metrics, err := parseBenchmarkMetrics(raw)
	if err != nil {
		t.Fatalf("parseBenchmarkMetrics failed: %v", err)
	}
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}
	if metrics["BenchmarkSyncDryRunPlan-8"].NsPerOp != 12345 {
		t.Fatalf("unexpected ns/op for sync benchmark: %+v", metrics["BenchmarkSyncDryRunPlan-8"])
	}
	if metrics["BenchmarkStatusReport-8"].AllocsPerOp != 4 {
		t.Fatalf("unexpected allocs/op for status benchmark: %+v", metrics["BenchmarkStatusReport-8"])
	}
}

func TestParseBenchmarkMetricsNoBenchmarks(t *testing.T) {
	if _, err := parseBenchmarkMetrics("PASS\n"); err == nil {
		t.Fatal("expected parse failure when no benchmark lines exist")
	}
}

func TestAppendAndLoadLastRecord(t *testing.T) {
	tmp := t.TempDir()
	history := filepath.Join(tmp, "history.jsonl")

	first := benchmarkRunRecord{
		Timestamp: "2026-02-16T00:00:00Z",
		Commit:    "abc123",
		Benchmarks: map[string]benchmarkMetric{
			"BenchmarkOne-8": {NsPerOp: 100},
		},
	}
	second := benchmarkRunRecord{
		Timestamp: "2026-02-16T00:01:00Z",
		Commit:    "def456",
		Benchmarks: map[string]benchmarkMetric{
			"BenchmarkOne-8": {NsPerOp: 90},
		},
	}
	if err := appendRecord(history, first); err != nil {
		t.Fatalf("append first record: %v", err)
	}
	if err := appendRecord(history, second); err != nil {
		t.Fatalf("append second record: %v", err)
	}

	last, err := loadLastRecord(history)
	if err != nil {
		t.Fatalf("loadLastRecord failed: %v", err)
	}
	if last.Commit != "def456" {
		t.Fatalf("unexpected last commit: got=%s want=def456", last.Commit)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" ./internal/engine, ,./internal/discovery ")
	if len(got) != 2 {
		t.Fatalf("unexpected split length: %#v", got)
	}
	if got[0] != "./internal/engine" || got[1] != "./internal/discovery" {
		t.Fatalf("unexpected split values: %#v", got)
	}
}

func TestLoadLastRecordErrorsOnEmpty(t *testing.T) {
	tmp := t.TempDir()
	history := filepath.Join(tmp, "history.jsonl")
	if err := os.WriteFile(history, []byte(""), 0o644); err != nil {
		t.Fatalf("seed history file: %v", err)
	}
	if _, err := loadLastRecord(history); err == nil {
		t.Fatal("expected error for empty history file")
	}
}
