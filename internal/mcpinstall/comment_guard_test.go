// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- tomlHasComments unit tests (finding 6) ---

func TestTomlHasComments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"no comment", "command = '/bin/repokeeper'\nargs = ['mcp']\n", false},
		{"leading comment", "# hand-maintained\ncommand = 'x'\n", true},
		{"trailing comment", "command = 'x' # inline note\n", true},
		{"hash inside basic string", `command = "a#b"` + "\n", false},
		{"hash inside literal string", `command = 'a#b'` + "\n", false},
		{"hash after literal string closes", `command = 'x' #c` + "\n", true},
		{"escaped quote then hash in string", `command = "a\"#b"` + "\n", false},
		{"multiline basic string with hash", "x = \"\"\"\nline # not a comment\n\"\"\"\n", false},
		{"multiline literal string with hash", "x = '''\nline # not a comment\n'''\ny = 1\n", false},
		{"comment after multiline string", "x = \"\"\"a\"\"\" # real\n", true},
		{"hash at end of file no newline", "command = 'x'\n#tail", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tomlHasComments([]byte(tc.in)); got != tc.want {
				t.Fatalf("tomlHasComments(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// writeTemp writes content to a fresh temp file and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}

// mustReadFile reads path and fails the test if the read errors, so a failed
// read can't masquerade as an unexpected file content in the assertions below.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

// --- codex/grok WriteEntry refuses to clobber comments (finding 6) ---

func TestCodexWriteEntryRefusesCommentedFile(t *testing.T) {
	t.Parallel()
	original := "# user notes, please keep\nmodel = 'gpt-4'\n"
	path := writeTemp(t, "config.toml", original)
	a := &codexAdapter{}
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true})
	if err == nil {
		t.Fatal("expected WriteEntry to refuse a commented file")
	}
	if !strings.Contains(err.Error(), "comments") {
		t.Fatalf("expected a comment-preservation error, got: %v", err)
	}
	raw := mustReadFile(t, path)
	if string(raw) != original {
		t.Fatalf("commented file was modified despite refusal:\n%s", raw)
	}
}

func TestGrokWriteEntryRefusesCommentedFile(t *testing.T) {
	t.Parallel()
	original := "# grok config, hand tuned\napi_key = 'secret'\n"
	path := writeTemp(t, "config.toml", original)
	a := &grokAdapter{}
	err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true})
	if err == nil {
		t.Fatal("expected WriteEntry to refuse a commented file")
	}
	raw := mustReadFile(t, path)
	if string(raw) != original {
		t.Fatalf("commented file was modified despite refusal:\n%s", raw)
	}
}

// A file with no comments is still written normally.
func TestCodexWriteEntryAllowsUncommentedFile(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, "config.toml", "model = 'gpt-4'\n")
	a := &codexAdapter{}
	if err := a.WriteEntry(path, Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true}); err != nil {
		t.Fatalf("unexpected error writing uncommented file: %v", err)
	}
	raw := mustReadFile(t, path)
	if !strings.Contains(string(raw), "[mcp_servers.repokeeper]") {
		t.Fatalf("entry not written: %s", raw)
	}
}

// --- RemoveEntry comment handling (finding 6) ---

// A no-op remove (entry absent) must NOT refuse just because the file has
// comments, since no rewrite happens.
func TestCodexRemoveEntryAbsentCommentedFileSucceeds(t *testing.T) {
	t.Parallel()
	original := "# just comments and unrelated config\nmodel = 'gpt-4'\n"
	path := writeTemp(t, "config.toml", original)
	a := &codexAdapter{}
	removed, err := a.RemoveEntry(path)
	if err != nil {
		t.Fatalf("no-op remove on commented file should succeed, got: %v", err)
	}
	if removed {
		t.Fatal("expected removed=false when entry absent")
	}
	raw := mustReadFile(t, path)
	if string(raw) != original {
		t.Fatalf("file changed on no-op remove:\n%s", raw)
	}
}

// Removing a present entry from a commented file must refuse rather than
// silently drop the comments.
func TestCodexRemoveEntryPresentCommentedFileRefuses(t *testing.T) {
	t.Parallel()
	original := "# keep me\n[mcp_servers.repokeeper]\ncommand = '/bin/repokeeper'\nargs = ['mcp']\n"
	path := writeTemp(t, "config.toml", original)
	a := &codexAdapter{}
	_, err := a.RemoveEntry(path)
	if err == nil {
		t.Fatal("expected RemoveEntry to refuse a commented file with a present entry")
	}
	raw := mustReadFile(t, path)
	if string(raw) != original {
		t.Fatalf("commented file was modified despite refusal:\n%s", raw)
	}
}

// --- new config files are created 0o600 (finding 7) ---

func TestNewConfigFilesAreOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes not meaningful on windows")
	}
	entry := Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}, Enabled: true}
	cases := []struct {
		name    string
		file    string
		writeFn func(path string, e Entry) error
	}{
		{"codex", "config.toml", (&codexAdapter{}).WriteEntry},
		{"grok", "config.toml", (&grokAdapter{}).WriteEntry},
		{"claude", ".claude.json", (&claudeAdapter{}).WriteEntry},
		{"opencode", "opencode.json", (&opencodeAdapter{}).WriteEntry},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), tc.file)
			if err := tc.writeFn(path, entry); err != nil {
				t.Fatalf("WriteEntry: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("new %s config mode = %o, want 0600", tc.name, got)
			}
		})
	}
}
