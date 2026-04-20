// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexName(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	if a.Name() != "codex" {
		t.Fatalf("got %q want %q", a.Name(), "codex")
	}
}

// TestCodexDetectTrue is a sequential detection test because it uses
// t.Setenv("HOME", ...), which conflicts with t.Parallel on Go 1.26.2+.
func TestCodexDetectTrue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &codexAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Detect=true when ~/.codex exists")
	}
}

func TestCodexDetectFalse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &codexAdapter{}
	ok, err := a.Detect()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected Detect=false when ~/.codex does not exist")
	}
}

func TestCodexConfigPathUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := &codexAdapter{}
	path, err := a.ConfigPath(ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".codex", "config.toml")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestCodexConfigPathProjectUnsupported(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	_, err := a.ConfigPath(ScopeProject)
	if !errors.Is(err, ErrScopeUnsupported) {
		t.Fatalf("expected ErrScopeUnsupported, got %v", err)
	}
}

func TestCodexReadEntryEmpty(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/empty.toml")
	_, present, err := a.ReadEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected not present in empty config")
	}
}

func TestCodexReadEntryPresent(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/existing-match.toml")
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

func TestCodexReadEntryMalformed(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/malformed.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestCodexReadEntryRejectsNonTableMcpServers(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/mcp-servers-not-table.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-table mcp_servers")
	}
}

func TestCodexReadEntryRejectsNonTableEntry(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/entry-not-table.toml")
	if _, _, err := a.ReadEntry(path); err == nil {
		t.Fatal("expected error for non-table repokeeper entry")
	}
}

func TestCodexWriteEntryFreshFile(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := filepath.Join(t.TempDir(), "codex.toml")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err != nil {
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
}

func TestCodexWriteEntryPreservesOthers(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/other-servers.toml")
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if !strings.Contains(text, "model = 'gpt-4'") {
		t.Fatalf("lost top-level model key: %s", text)
	}
	if !strings.Contains(text, "[mcp_servers.other]") {
		t.Fatalf("lost sibling server: %s", text)
	}
	if !strings.Contains(text, "[mcp_servers.repokeeper]") {
		t.Fatalf("missing our entry: %s", text)
	}
}

func TestCodexWriteEntryRejectsNonTableMcpServers(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/mcp-servers-not-table.toml")
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}})
	if err == nil {
		t.Fatal("expected error for non-table mcp_servers on write")
	}
	// File must not be rewritten — original string value must remain.
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "not-a-table") {
		t.Fatalf("file was rewritten despite error: %s", raw)
	}
}

func TestCodexRemoveEntry(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/existing-match.toml")
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

func TestCodexRemoveEntryAbsent(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := copyFixture(t, "codex/empty.toml")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Fatal("expected removed=false on absent entry")
	}
}

func TestCodexRemoveEntryMissingFile(t *testing.T) {
	t.Parallel()
	a := &codexAdapter{}
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatalf("expected nil err for missing file, got: %v", err)
	}
	if removed {
		t.Fatal("expected removed=false for missing file")
	}
}
