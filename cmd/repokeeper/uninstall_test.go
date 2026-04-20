// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetUninstallFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_ = uninstallCmd.Flags().Set("claude", "false")
		_ = uninstallCmd.Flags().Set("codex", "false")
		_ = uninstallCmd.Flags().Set("opencode", "false")
		_ = uninstallCmd.Flags().Set("scope", "user")
	})
}

func runUninstallWithFlags(t *testing.T, stdin io.Reader, flags map[string]string) (stdout, stderr *bytes.Buffer, err error) {
	t.Helper()
	resetUninstallFlags(t)
	for k, v := range flags {
		if err := uninstallCmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set flag %s=%s: %v", k, v, err)
		}
	}
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	uninstallCmd.SetOut(stdout)
	uninstallCmd.SetErr(stderr)
	if stdin != nil {
		uninstallCmd.SetIn(stdin)
	}
	t.Cleanup(func() {
		uninstallCmd.SetOut(os.Stdout)
		uninstallCmd.SetErr(os.Stderr)
		uninstallCmd.SetIn(os.Stdin)
	})
	err = uninstallCmd.RunE(uninstallCmd, nil)
	return
}

func seedClaudeEntry(t *testing.T, home string) string {
	t.Helper()
	path := filepath.Join(home, ".claude.json")
	seed := `{"mcpServers":{"repokeeper":{"command":"/fake/repokeeper","args":["mcp"]}}}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUninstallRemovesEntry(t *testing.T) {
	home := withInstallEnv(t)
	path := seedClaudeEntry(t, home)
	restore := withAssumeYes(t, true)
	defer restore()

	stdout, _, err := runUninstallWithFlags(t, nil, map[string]string{"claude": "true"})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !strings.Contains(stdout.String(), "removed claude from "+path) {
		t.Fatalf("missing removed line: %q", stdout.String())
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "repokeeper") {
		t.Fatalf("entry still present: %s", raw)
	}
}

func TestUninstallNoOpWhenAbsent(t *testing.T) {
	home := withInstallEnv(t)
	// Create .claude.json so Detect passes, but with no repokeeper entry.
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := withAssumeYes(t, true)
	defer restore()

	stdout, _, err := runUninstallWithFlags(t, nil, map[string]string{"claude": "true"})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected silent no-op, got: %q", stdout.String())
	}
}

func TestUninstallPromptDeclineKeepsEntry(t *testing.T) {
	home := withInstallEnv(t)
	path := seedClaudeEntry(t, home)
	restore := withAssumeYes(t, false)
	defer restore()

	stdin := strings.NewReader("n\n")
	stdout, stderr, err := runUninstallWithFlags(t, stdin, map[string]string{"claude": "true"})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if strings.Contains(stdout.String(), "removed") {
		t.Fatalf("should not have removed on decline: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "cancelled") {
		t.Fatalf("expected cancelled message on stderr: %q", stderr.String())
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "repokeeper") {
		t.Fatalf("entry was removed despite decline: %s", raw)
	}
}

func TestUninstallNonTTYEmptyStdinAborts(t *testing.T) {
	home := withInstallEnv(t)
	path := seedClaudeEntry(t, home)
	restore := withAssumeYes(t, false)
	defer restore()

	// Empty stdin mimics non-interactive invocation.
	stdout, stderr, err := runUninstallWithFlags(t, strings.NewReader(""), map[string]string{"claude": "true"})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if strings.Contains(stdout.String(), "removed") {
		t.Fatalf("should not remove on EOF: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "cancelled") {
		t.Fatalf("expected cancelled on EOF: %q", stderr.String())
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "repokeeper") {
		t.Fatalf("entry was removed on EOF: %s", raw)
	}
}

func TestUninstallCodexProjectScopeIsError(t *testing.T) {
	withInstallEnv(t)
	restore := withAssumeYes(t, true)
	defer restore()
	_, _, err := runUninstallWithFlags(t, nil, map[string]string{
		"codex": "true",
		"scope": "project",
	})
	if err == nil {
		t.Fatal("expected error for --codex --scope project")
	}
	if !strings.Contains(err.Error(), "codex") || !strings.Contains(err.Error(), "project") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUninstallNoRuntimeDetected(t *testing.T) {
	withInstallEnv(t)
	restore := withAssumeYes(t, true)
	defer restore()
	_, _, err := runUninstallWithFlags(t, nil, nil)
	if err == nil {
		t.Fatal("expected error when no runtime is detected")
	}
	if !strings.Contains(err.Error(), "no MCP-capable runtime detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
