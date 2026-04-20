// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOpenCodeName(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	if a.Name() != "opencode" {
		t.Fatalf("got %q want %q", a.Name(), "opencode")
	}
}

// Detect/ConfigPath tests mutate env; they cannot run with t.Parallel
// alongside other tests that also set OPENCODE_CONFIG_DIR / HOME.

func TestOpenCodeDetectTrueViaEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "/nowhere/opencode")
	a := &opencodeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when OPENCODE_CONFIG_DIR is set")
	}
}

func TestOpenCodeDetectTrueViaDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &opencodeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.config/opencode exists")
	}
}

func TestOpenCodeDetectFalse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	a := &opencodeAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected Detect=false on a clean home")
	}
}

func TestOpenCodeConfigPathDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	a := &opencodeAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "opencode", "opencode.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestOpenCodeConfigPathXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	a := &opencodeAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "opencode", "opencode.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestOpenCodeConfigPathEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	a := &opencodeAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "opencode.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestOpenCodeConfigPathProject(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path, err := a.ConfigPath(ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	want := filepath.Join(cwd, "opencode.json")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestOpenCodeReadEntryEmpty(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/empty.json")
	_, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected not present in empty config")
	}
}

func TestOpenCodeReadEntryPresent(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/existing-match.json")
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
	if !reflect.DeepEqual(e.Args, []string{"mcp"}) {
		t.Fatalf("args: got %v", e.Args)
	}
}

func TestOpenCodeReadEntryMalformed(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/malformed.json")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
}

func TestOpenCodeReadEntryRejectsNonObjectMcp(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/mcp-not-object.json")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-object mcp")
	}
}

func TestOpenCodeReadEntryRejectsNonObjectEntry(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/entry-not-object.json")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-object repokeeper entry")
	}
}

func TestOpenCodeWriteEntryFreshFile(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := filepath.Join(t.TempDir(), "opencode.json")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp", "-v"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("wrote invalid JSON: %v", err)
	}
	servers, ok := doc["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("mcp missing or wrong type: %v", doc)
	}
	ours, ok := servers["repokeeper"].(map[string]any)
	if !ok {
		t.Fatal("repokeeper entry missing")
	}
	if ours["type"] != "local" {
		t.Fatalf("type: got %v want local", ours["type"])
	}
	if ours["enabled"] != true {
		t.Fatalf("enabled: got %v want true", ours["enabled"])
	}
	cmd, ok := ours["command"].([]any)
	if !ok {
		t.Fatalf("command not a JSON array: %T", ours["command"])
	}
	if len(cmd) != 3 || cmd[0] != "/bin/repokeeper" || cmd[1] != "mcp" || cmd[2] != "-v" {
		t.Fatalf("argv: got %v", cmd)
	}
}

func TestOpenCodeWriteEntryPreservesOtherKeys(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/other-servers.json")
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
	servers := doc["mcp"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("lost sibling server entry")
	}
	if _, ok := servers["repokeeper"]; !ok {
		t.Fatal("missing our entry")
	}
}

func TestOpenCodeWriteEntryOverwritesStale(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/existing-stale.json")
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
	if !reflect.DeepEqual(e.Args, []string{"mcp"}) {
		t.Fatalf("args not updated: got %v", e.Args)
	}
}

func TestOpenCodeWriteEntryRejectsNonObjectMcp(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/mcp-not-object.json")
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}})
	if err == nil {
		t.Fatal("expected error for non-object mcp on write")
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "\"not\"") {
		t.Fatalf("file was rewritten despite error: %s", raw)
	}
}

func TestOpenCodeRemoveEntry(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/existing-match.json")
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

func TestOpenCodeRemoveEntryAbsent(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := copyFixture(t, "opencode/empty.json")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Fatal("expected removed=false on absent entry")
	}
}

func TestOpenCodeRemoveEntryMissingFile(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatalf("expected nil err for missing file, got: %v", err)
	}
	if removed {
		t.Fatal("expected removed=false for missing file")
	}
}

func TestOpenCodeRefusesJsoncPath(t *testing.T) {
	t.Parallel()
	a := &opencodeAdapter{}
	path := filepath.Join(t.TempDir(), "opencode.jsonc")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected ReadEntry to refuse .jsonc path")
	}
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err == nil {
		t.Fatal("expected WriteEntry to refuse .jsonc path")
	}
}

func TestOpenCodeRefusesJsoncSibling(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "opencode.json")
	jsoncPath := filepath.Join(dir, "opencode.jsonc")
	if err := os.WriteFile(jsoncPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &opencodeAdapter{}
	if _, _, err := a.ReadEntry(jsonPath); err == nil {
		t.Fatal("expected ReadEntry to refuse when .jsonc sibling exists")
	}
	if err := a.WriteEntry(jsonPath, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err == nil {
		t.Fatal("expected WriteEntry to refuse when .jsonc sibling exists")
	}
	if _, err := a.RemoveEntry(jsonPath); err == nil {
		t.Fatal("expected RemoveEntry to refuse when .jsonc sibling exists")
	}
	if _, err := os.Stat(jsonPath); err == nil {
		t.Fatal(".json file should not have been created")
	}
}
