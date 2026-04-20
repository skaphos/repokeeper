// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetInstallListFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_ = installListCmd.Flags().Set("scope", "user")
		_ = installListCmd.Flags().Set("json", "false")
	})
}

func runInstallListWithFlags(t *testing.T, flags map[string]string) (stdout, stderr *bytes.Buffer, err error) {
	t.Helper()
	resetInstallListFlags(t)
	for k, v := range flags {
		if err := installListCmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set flag %s=%s: %v", k, v, err)
		}
	}
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	installListCmd.SetOut(stdout)
	installListCmd.SetErr(stderr)
	t.Cleanup(func() {
		installListCmd.SetOut(os.Stdout)
		installListCmd.SetErr(os.Stderr)
	})
	err = installListCmd.RunE(installListCmd, nil)
	return
}

func TestInstallListAllNotRegistered(t *testing.T) {
	withInstallEnv(t)
	stdout, _, err := runInstallListWithFlags(t, nil)
	if err != nil {
		t.Fatalf("install list: %v", err)
	}
	out := stdout.String()
	for _, name := range []string{"claude", "codex", "opencode"} {
		if !strings.Contains(out, name) {
			t.Fatalf("missing %s in table: %q", name, out)
		}
	}
	if !strings.Contains(out, "not registered") {
		t.Fatalf("expected 'not registered' state: %q", out)
	}
}

func TestInstallListRegisteredStateMatchesExecutable(t *testing.T) {
	home := withInstallEnv(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	seed := fmt.Sprintf(`{"mcpServers":{"repokeeper":{"command":%q,"args":["mcp"]}}}`, exe)
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runInstallListWithFlags(t, nil)
	if err != nil {
		t.Fatalf("install list: %v", err)
	}
	lines := strings.Split(stdout.String(), "\n")
	var claudeLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "claude") {
			claudeLine = line
			break
		}
	}
	if claudeLine == "" {
		t.Fatalf("no claude row: %q", stdout.String())
	}
	if !strings.Contains(claudeLine, "registered") {
		t.Fatalf("claude row missing registered state: %q", claudeLine)
	}
	if strings.Contains(claudeLine, "stale") {
		t.Fatalf("claude should not be stale when command matches os.Executable: %q", claudeLine)
	}
}

func TestInstallListStaleWhenCommandDiffers(t *testing.T) {
	home := withInstallEnv(t)
	seed := `{"mcpServers":{"repokeeper":{"command":"/old/path/repokeeper","args":["mcp"]}}}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runInstallListWithFlags(t, nil)
	if err != nil {
		t.Fatalf("install list: %v", err)
	}
	if !strings.Contains(stdout.String(), "registered (stale)") {
		t.Fatalf("expected stale state: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "/old/path/repokeeper") {
		t.Fatalf("expected stale command in output: %q", stdout.String())
	}
}

func TestInstallListJSON(t *testing.T) {
	home := withInstallEnv(t)
	seed := `{"mcpServers":{"repokeeper":{"command":"/old/path/repokeeper","args":["mcp"]}}}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runInstallListWithFlags(t, map[string]string{"json": "true"})
	if err != nil {
		t.Fatalf("install list --json: %v", err)
	}
	var doc struct {
		Scope    string    `json:"scope"`
		Runtimes []listRow `json:"runtimes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, stdout.String())
	}
	if doc.Scope != "user" {
		t.Fatalf("scope: got %q", doc.Scope)
	}
	found := false
	for _, r := range doc.Runtimes {
		if r.Name == "claude" {
			found = true
			if r.State != "registered (stale)" {
				t.Fatalf("claude state: %q", r.State)
			}
			if r.Command != "/old/path/repokeeper" {
				t.Fatalf("claude command: %q", r.Command)
			}
		}
	}
	if !found {
		t.Fatal("claude row missing from JSON output")
	}
}

func TestInstallListProjectCodexUnsupported(t *testing.T) {
	withInstallEnv(t)
	// Use a project cwd so project-scope paths land in a tempdir.
	cwd := t.TempDir()
	prevCwd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevCwd) })

	stdout, _, err := runInstallListWithFlags(t, map[string]string{"scope": "project"})
	if err != nil {
		t.Fatalf("install list --scope project: %v", err)
	}
	lines := strings.Split(stdout.String(), "\n")
	var codexLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "codex") {
			codexLine = line
			break
		}
	}
	if codexLine == "" {
		t.Fatalf("no codex row: %q", stdout.String())
	}
	if !strings.Contains(codexLine, "unsupported") {
		t.Fatalf("codex should report unsupported in project scope: %q", codexLine)
	}
}

func TestInstallListInvalidScope(t *testing.T) {
	withInstallEnv(t)
	_, _, err := runInstallListWithFlags(t, map[string]string{"scope": "bogus"})
	if err == nil {
		t.Fatal("expected invalid-scope error")
	}
}
