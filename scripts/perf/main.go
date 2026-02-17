// SPDX-License-Identifier: MIT
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type benchmarkMetric struct {
	NsPerOp     float64 `json:"ns_per_op"`
	BPerOp      float64 `json:"b_per_op,omitempty"`
	AllocsPerOp float64 `json:"allocs_per_op,omitempty"`
}

type benchmarkRunRecord struct {
	Timestamp  string                     `json:"timestamp"`
	Commit     string                     `json:"commit"`
	GoVersion  string                     `json:"go_version"`
	Packages   []string                   `json:"packages"`
	Bench      string                     `json:"bench"`
	Benchtime  string                     `json:"benchtime"`
	Count      int                        `json:"count"`
	Benchmarks map[string]benchmarkMetric `json:"benchmarks"`
}

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op(?:\s+([0-9.]+)\s+B/op\s+([0-9.]+)\s+allocs/op)?`)

func main() {
	historyPath := flag.String("history", "perf/history.jsonl", "path to benchmark history jsonl")
	rawDir := flag.String("raw-dir", "perf/runs", "directory for raw benchmark logs")
	packageCSV := flag.String("packages", "./internal/engine", "comma-separated benchmark packages")
	benchPattern := flag.String("bench", ".", "go test -bench pattern")
	benchtime := flag.String("benchtime", "1x", "go test benchmark time (for example: 1x, 500ms, 2s)")
	count := flag.Int("count", 5, "go test benchmark count")
	flag.Parse()

	packages := splitCSV(*packageCSV)
	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "no benchmark packages provided")
		os.Exit(2)
	}

	rawOutput, err := runBenchmarks(packages, *benchPattern, *benchtime, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	metrics, err := parseBenchmarkMetrics(rawOutput)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	record := benchmarkRunRecord{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Commit:     gitShortCommit(),
		GoVersion:  runtimeGoVersion(),
		Packages:   packages,
		Bench:      *benchPattern,
		Benchtime:  *benchtime,
		Count:      *count,
		Benchmarks: metrics,
	}

	if err := os.MkdirAll(*rawDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create raw dir: %v\n", err)
		os.Exit(1)
	}
	rawFile := filepath.Join(*rawDir, time.Now().UTC().Format("20060102T150405Z")+".txt")
	if err := os.WriteFile(rawFile, []byte(rawOutput), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write raw log: %v\n", err)
		os.Exit(1)
	}

	previous, _ := loadLastRecord(*historyPath)
	if err := appendRecord(*historyPath, record); err != nil {
		fmt.Fprintf(os.Stderr, "append history: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("saved raw benchmark log: %s\n", rawFile)
	fmt.Printf("updated benchmark history: %s\n", *historyPath)
	printSummary(record, previous)
}

func runBenchmarks(packages []string, bench, benchtime string, count int) (string, error) {
	args := []string{
		"test",
		"-run=^$",
		"-bench=" + bench,
		"-benchmem",
		"-benchtime=" + benchtime,
		fmt.Sprintf("-count=%d", count),
	}
	args = append(args, packages...)
	cmd := exec.Command("go", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("benchmark run failed: %w\n%s", err, output.String())
	}
	return output.String(), nil
}

func parseBenchmarkMetrics(raw string) (map[string]benchmarkMetric, error) {
	metrics := make(map[string]benchmarkMetric)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		match := benchmarkLinePattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		entry := benchmarkMetric{NsPerOp: parseFloat(match[2])}
		if match[3] != "" {
			entry.BPerOp = parseFloat(match[3])
		}
		if match[4] != "" {
			entry.AllocsPerOp = parseFloat(match[4])
		}
		metrics[match[1]] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no benchmark metrics found in output")
	}
	return metrics, nil
}

func parseFloat(v string) float64 {
	var out float64
	_, _ = fmt.Sscanf(v, "%f", &out)
	return out
}

func gitShortCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func runtimeGoVersion() string {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func splitCSV(in string) []string {
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func appendRecord(path string, record benchmarkRunRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	return nil
}

func loadLastRecord(path string) (*benchmarkRunRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	var last string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			last = line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if last == "" {
		return nil, fmt.Errorf("history file is empty")
	}
	var record benchmarkRunRecord
	if err := json.Unmarshal([]byte(last), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func printSummary(current benchmarkRunRecord, previous *benchmarkRunRecord) {
	fmt.Println("benchmark summary (ns/op):")
	for name, metric := range current.Benchmarks {
		if previous == nil {
			fmt.Printf("  %-40s %.2f\n", name, metric.NsPerOp)
			continue
		}
		prev, ok := previous.Benchmarks[name]
		if !ok || prev.NsPerOp == 0 {
			fmt.Printf("  %-40s %.2f\n", name, metric.NsPerOp)
			continue
		}
		deltaPct := ((metric.NsPerOp - prev.NsPerOp) / prev.NsPerOp) * 100
		fmt.Printf("  %-40s %.2f (%+.2f%% vs previous)\n", name, metric.NsPerOp, deltaPct)
	}
}
