// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrokName(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	if a.Name() != "grok" {
		t.Fatalf("got %q want %q", a.Name(), "grok")
	}
}

// TestGrokDetectTrue is a sequential detection test because it uses
// t.Setenv("HOME", ...), which conflicts with t.Parallel on Go 1.26.2+.
func TestGrokDetectTrue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &grokAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.grok exists")
	}
}

func TestGrokDetectTrueWithConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	grokDir := filepath.Join(home, ".grok")
	if err := os.MkdirAll(grokDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(grokDir, "config.toml"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	a := &grokAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.grok/config.toml exists")
	}
}

func TestGrokDetectFalse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &grokAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected Detect=false when ~/.grok does not exist")
	}
}

func TestGrokConfigPathUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &grokAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".grok", "config.toml")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestGrokConfigPathProject(t *testing.T) {
	// Use a temp cwd for project scope. Cannot use t.Parallel because of os.Chdir.
	cwd := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	a := &grokAdapter{}
	path, err := a.ConfigPath(ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cwd, ".grok", "config.toml")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestGrokConfigPathGROK_CONFIG_DIR(t *testing.T) {
	// Cannot use t.Parallel because of t.Setenv.
	custom := t.TempDir()
	t.Setenv("GROK_CONFIG_DIR", custom)
	a := &grokAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(custom, "config.toml")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestGrokReadEntryEmpty(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/empty.toml")
	_, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected not present in empty config")
	}
}

func TestGrokReadEntryPresent(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/existing-match.toml")
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

func TestGrokReadEntryMalformed(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/malformed.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGrokReadEntryRejectsNonTableMcpServers(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/mcp-servers-not-table.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-table mcp_servers")
	}
}

func TestGrokReadEntryRejectsNonTableEntry(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/repokeeper-not-table.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-table repokeeper entry")
	}
}

func TestGrokReadEntryRejectsBadEnabledType(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/bad-enabled-type.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for bad enabled type in repokeeper entry")
	}
}

func TestGrokWriteEntryFreshFile(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := filepath.Join(t.TempDir(), "grok.toml")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if !strings.Contains(text, "[mcp_servers.repokeeper]") {
		t.Fatalf("missing our table: %s", text)
	}
	if !strings.Contains(text, "command = '/bin/repokeeper'") {
		t.Fatalf("missing command: %s", text)
	}
	if !strings.Contains(text, "enabled = true") {
		t.Fatalf("missing enabled: %s", text)
	}
}

func TestGrokWriteEntryPreservesOthers(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/other-servers.toml")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if !strings.Contains(text, "model = 'grok-3'") {
		t.Fatalf("lost top-level model key: %s", text)
	}
	if !strings.Contains(text, "[mcp_servers.other]") {
		t.Fatalf("lost sibling server: %s", text)
	}
	if !strings.Contains(text, "[mcp_servers.repokeeper]") {
		t.Fatalf("missing our entry: %s", text)
	}
}

func TestGrokWriteEntryRejectsNonTableMcpServers(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/mcp-servers-not-table.toml")
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true})
	if err == nil {
		t.Fatal("expected error for non-table mcp_servers on write")
	}
	// File must not be rewritten — original string value must remain.
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "not-a-table") {
		t.Fatalf("file was rewritten despite error: %s", raw)
	}
}

func TestGrokRemoveEntry(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/existing-match.toml")
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

func TestGrokRemoveEntryAbsent(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := copyFixture(t, "grok/empty.toml")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Fatal("expected removed=false on absent entry")
	}
}

func TestGrokRemoveEntryMissingFile(t *testing.T) {
	t.Parallel()
	a := &grokAdapter{}
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatalf("expected nil err for missing file, got: %v", err)
	}
	if removed {
		t.Fatal("expected removed=false for missing file")
	}
}
