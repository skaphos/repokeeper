// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withInstallEnv isolates HOME and clears OPENCODE_CONFIG_DIR /
// XDG_CONFIG_HOME so the adapters resolve paths beneath the tempdir.
// Returns the tempdir for assertions.
func withInstallEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	return home
}

// resetInstallFlags restores the default install-command flag values.
// Cobra stores flag state on the shared command singleton, so tests
// that mutate flags must reset them in Cleanup.
func resetInstallFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_ = installCmd.Flags().Set("claude", "false")
		_ = installCmd.Flags().Set("codex", "false")
		_ = installCmd.Flags().Set("opencode", "false")
		_ = installCmd.Flags().Set("scope", "user")
		_ = installCmd.Flags().Set("command", "")
		_ = installCmd.Flags().Set("manual", "")
		installCmd.Flags().Lookup("manual").Changed = false
	})
}

func runInstallWithFlags(t *testing.T, flags map[string]string) (stdout, stderr *bytes.Buffer, err error) {
	t.Helper()
	resetInstallFlags(t)
	for k, v := range flags {
		if err := installCmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set flag %s=%s: %v", k, v, err)
		}
	}
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	installCmd.SetOut(stdout)
	installCmd.SetErr(stderr)
	t.Cleanup(func() {
		installCmd.SetOut(os.Stdout)
		installCmd.SetErr(os.Stderr)
	})
	err = installCmd.RunE(installCmd, nil)
	return
}

func TestInstallClaudeExplicitWritesConfig(t *testing.T) {
	home := withInstallEnv(t)
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"claude":  "true",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --claude: %v", err)
	}
	path := filepath.Join(home, ".claude.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("config not valid JSON: %v", err)
	}
	servers, ok := doc["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing: %v", doc)
	}
	entry, ok := servers["repokeeper"].(map[string]any)
	if !ok {
		t.Fatal("repokeeper entry missing")
	}
	if entry["command"] != "/fake/repokeeper" {
		t.Fatalf("command: got %v", entry["command"])
	}
	if !strings.Contains(stdout.String(), "registered claude at "+path) {
		t.Fatalf("stdout missing registered line: %q", stdout.String())
	}
}

func TestInstallNoRuntimeDetected(t *testing.T) {
	withInstallEnv(t)
	_, _, err := runInstallWithFlags(t, map[string]string{
		"command": "/fake/repokeeper",
	})
	if err == nil {
		t.Fatal("expected error when no runtime is detected")
	}
	if !strings.Contains(err.Error(), "no MCP-capable runtime detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallCodexProjectScopeIsError(t *testing.T) {
	withInstallEnv(t)
	_, _, err := runInstallWithFlags(t, map[string]string{
		"codex":   "true",
		"scope":   "project",
		"command": "/fake/repokeeper",
	})
	if err == nil {
		t.Fatal("expected error for --codex --scope project")
	}
	if !strings.Contains(err.Error(), "codex") || !strings.Contains(err.Error(), "project") {
		t.Fatalf("error should name codex and project scope: %v", err)
	}
}

func TestInstallInvalidScope(t *testing.T) {
	withInstallEnv(t)
	_, _, err := runInstallWithFlags(t, map[string]string{
		"claude":  "true",
		"scope":   "global",
		"command": "/fake/repokeeper",
	})
	if err == nil {
		t.Fatal("expected invalid-scope error")
	}
	if !strings.Contains(err.Error(), "invalid --scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallIdempotentNoRewrite(t *testing.T) {
	home := withInstallEnv(t)
	// First install
	if _, _, err := runInstallWithFlags(t, map[string]string{
		"claude":  "true",
		"command": "/fake/repokeeper",
	}); err != nil {
		t.Fatalf("first install: %v", err)
	}
	path := filepath.Join(home, ".claude.json")
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure mtime granularity does not hide a rewrite.
	time.Sleep(10 * time.Millisecond)

	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"claude":  "true",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatalf("expected mtime unchanged on idempotent install, got %v → %v", info1.ModTime(), info2.ModTime())
	}
	if !strings.Contains(stdout.String(), "unchanged claude") {
		t.Fatalf("expected 'unchanged' status on idempotent install: %q", stdout.String())
	}
}

func TestInstallUpdatesStaleEntry(t *testing.T) {
	home := withInstallEnv(t)
	// Seed a stale entry.
	path := filepath.Join(home, ".claude.json")
	seed := `{"mcpServers":{"repokeeper":{"command":"/old/repokeeper","args":["mcp","--old-flag"]}}}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"claude":  "true",
		"command": "/new/repokeeper",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated claude") {
		t.Fatalf("expected 'updated' status: %q", stdout.String())
	}
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	entry := doc["mcpServers"].(map[string]any)["repokeeper"].(map[string]any)
	if entry["command"] != "/new/repokeeper" {
		t.Fatalf("command not updated: %v", entry["command"])
	}
	args := entry["args"].([]any)
	if len(args) != 1 || args[0] != "mcp" {
		t.Fatalf("args not reset: %v", args)
	}
}

func TestInstallAutoDetectSkipsCodexForProjectScope(t *testing.T) {
	home := withInstallEnv(t)
	// Seed .claude.json and ~/.codex/ so both adapters would auto-detect.
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Use a project cwd so project-scope writes land in a tempdir, not the real repo.
	cwd := t.TempDir()
	prevCwd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevCwd) })

	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"scope":   "project",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("auto-detect project install: %v", err)
	}
	// Claude should have been written to <cwd>/.mcp.json.
	if _, err := os.Stat(filepath.Join(cwd, ".mcp.json")); err != nil {
		t.Fatalf("expected project .mcp.json, got %v", err)
	}
	// Codex must not have been written to.
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); err == nil {
		t.Fatal("codex config was written under project scope; expected silent skip")
	}
	if strings.Contains(stdout.String(), "codex") {
		t.Fatalf("stdout should not mention codex on auto-skip: %q", stdout.String())
	}
}

func TestInstallManualAllEmitsAllSnippets(t *testing.T) {
	home := withInstallEnv(t)
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "all",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --manual: %v", err)
	}
	out := stdout.String()
	for _, name := range []string{"claude", "codex", "opencode"} {
		if !strings.Contains(out, "# "+name) {
			t.Fatalf("missing %s section in: %q", name, out)
		}
	}
	if !strings.Contains(out, "\"mcpServers\"") {
		t.Fatal("expected claude JSON mcpServers key")
	}
	if !strings.Contains(out, "[mcp_servers.repokeeper]") {
		t.Fatal("expected codex TOML header")
	}
	if !strings.Contains(out, "\"mcp\"") {
		t.Fatal("expected opencode JSON mcp key")
	}
	// Must not write any files under HOME.
	entries, _ := os.ReadDir(home)
	if len(entries) != 0 {
		t.Fatalf("expected clean HOME, found %d entries", len(entries))
	}
}

func TestInstallManualBareIsEquivalentToAll(t *testing.T) {
	withInstallEnv(t)
	// Simulate bare `--manual` by honoring NoOptDefVal of "all".
	installCmd.Flags().Lookup("manual").NoOptDefVal = "all"
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "all",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --manual: %v", err)
	}
	for _, name := range []string{"claude", "codex", "opencode"} {
		if !strings.Contains(stdout.String(), "# "+name) {
			t.Fatalf("missing %s section: %q", name, stdout.String())
		}
	}
}

func TestInstallManualSingleTarget(t *testing.T) {
	home := withInstallEnv(t)
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "codex",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --manual=codex: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "[mcp_servers.repokeeper]") {
		t.Fatalf("missing codex TOML header: %q", out)
	}
	if strings.Contains(out, "\"mcpServers\"") || strings.Contains(out, "\"mcp\"") {
		t.Fatalf("should not include other runtimes: %q", out)
	}
	entries, _ := os.ReadDir(home)
	if len(entries) != 0 {
		t.Fatalf("expected clean HOME, found %d entries", len(entries))
	}
}

func TestInstallManualRejectsUnknownTarget(t *testing.T) {
	withInstallEnv(t)
	_, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "nope",
		"command": "/fake/repokeeper",
	})
	if err == nil {
		t.Fatal("expected rejection of --manual=nope")
	}
	if !strings.Contains(err.Error(), "invalid --manual") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallCommandDefaultsToOsExecutable(t *testing.T) {
	home := withInstallEnv(t)
	if _, _, err := runInstallWithFlags(t, map[string]string{
		"claude": "true",
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	entry := doc["mcpServers"].(map[string]any)["repokeeper"].(map[string]any)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if entry["command"] != exe {
		t.Fatalf("expected command to default to os.Executable() %q, got %v", exe, entry["command"])
	}
}
