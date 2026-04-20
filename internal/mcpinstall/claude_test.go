// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeNameAndDetect(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	if a.Name() != "claude" {
		t.Fatalf("Name(): got %q want %q", a.Name(), "claude")
	}
	// Detect's return depends on HOME. Just exercise it and assert
	// no error — presence/absence is tested via ConfigPath with an
	// overridden HOME below.
	if _, err := a.Detect(); err != nil {
		t.Fatalf("Detect(): %v", err)
	}
}

func TestClaudeDetectTrueWhenDotClaudeJson(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &claudeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.claude.json exists")
	}
}

func TestClaudeDetectTrueWhenDotClaudeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &claudeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.claude dir exists")
	}
}

func TestClaudeDetectFalseWhenNeither(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &claudeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected Detect=false when neither exists")
	}
}

func TestClaudeConfigPathUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &claudeAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestClaudeConfigPathProject(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path, err := a.ConfigPath(ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	want := filepath.Join(cwd, ".mcp.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestClaudeReadEntryEmpty(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/empty.json")
	_, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected not present in empty config")
	}
}

func TestClaudeReadEntryPresent(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/existing-match.json")
	e, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("expected present")
	}
	if e.Command != "/usr/local/bin/repokeeper" {
		t.Fatalf("command: got %q", e.Command)
	}
	if len(e.Args) != 1 || e.Args[0] != "mcp" {
		t.Fatalf("args: got %v", e.Args)
	}
}

func TestClaudeReadEntryMalformed(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/malformed.json")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
}

func TestClaudeWriteEntryFreshFile(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := filepath.Join(t.TempDir(), "claude.json")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("wrote invalid JSON: %v", err)
	}
	servers, ok := doc["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing or wrong type: %v", doc)
	}
	ours, ok := servers["repokeeper"].(map[string]any)
	if !ok {
		t.Fatal("repokeeper entry missing")
	}
	if ours["command"] != "/bin/repokeeper" {
		t.Fatalf("command: got %v", ours["command"])
	}
}

func TestClaudeWriteEntryPreservesOtherKeys(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/other-servers.json")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["someOtherKey"] != "value" {
		t.Fatalf("lost top-level key: %v", doc)
	}
	servers := doc["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("lost sibling server entry")
	}
	if _, ok := servers["repokeeper"]; !ok {
		t.Fatal("missing our entry")
	}
}

func TestClaudeWriteEntryOverwritesStale(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/existing-stale.json")
	if err := a.WriteEntry(path, Entry{Command: "/new/repokeeper", Args: []string{"mcp"}}); err != nil {
		t.Fatal(err)
	}
	e, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("expected present after write")
	}
	if e.Command != "/new/repokeeper" {
		t.Fatalf("command not updated: got %q", e.Command)
	}
	if len(e.Args) != 1 || e.Args[0] != "mcp" {
		t.Fatalf("args not updated: got %v", e.Args)
	}
}

func TestClaudeRemoveEntry(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/existing-match.json")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}
	_, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("entry still present after Remove")
	}
}

func TestClaudeRemoveEntryAbsent(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/empty.json")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Fatal("expected removed=false on absent entry")
	}
}

func TestClaudeReadEntryRejectsNonObjectMcpServers(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/mcpservers-not-object.json")
	_, _, err := a.ReadEntry(path)
	if err == nil {
		t.Fatal("expected error for non-object mcpServers")
	}
}

func TestClaudeWriteEntryRejectsNonObjectMcpServers(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/mcpservers-not-object.json")
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}})
	if err == nil {
		t.Fatal("expected error for non-object mcpServers on write")
	}
	// Verify the file wasn't rewritten — it should still contain the original array.
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "\"not\"") {
		t.Fatalf("file was rewritten despite error: %s", raw)
	}
}

func TestClaudeReadEntryRejectsNonObjectEntry(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	path := copyFixture(t, "claude/entry-not-object.json")
	_, _, err := a.ReadEntry(path)
	if err == nil {
		t.Fatal("expected error for non-object repokeeper entry")
	}
}

func TestClaudeRemoveEntryMissingFile(t *testing.T) {
	t.Parallel()
	a := &claudeAdapter{}
	// Path that does not exist — readJSONDoc normalizes this to empty map, and
	// RemoveEntry should return (false, nil) via the "servers is nil" path.
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatalf("expected (false, nil) for missing file, got err: %v", err)
	}
	if removed {
		t.Fatal("expected removed=false for missing file")
	}
}

// copyFixture copies a testdata file into a tempdir so tests can mutate
// it without leaking state between runs. Shared across adapter tests.
func copyFixture(t *testing.T, rel string) string {
	t.Helper()
	src := filepath.Join("testdata", rel)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dst := filepath.Join(t.TempDir(), filepath.Base(rel))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture copy: %v", err)
	}
	return dst
}
