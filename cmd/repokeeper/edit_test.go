// SPDX-License-Identifier: MIT
package repokeeper

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

func writeEditorFixtureScript(t *testing.T, dir, repoPath string, invalidYAML bool) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		scriptPath := filepath.Join(dir, "editor.cmd")
		var script string
		if invalidYAML {
			script = "@echo off\r\n> \"%~1\" echo not: [valid\r\n"
		} else {
			script = "@echo off\r\n" +
				"> \"%~1\" (\r\n" +
				"echo repo_id: github.com/org/repo-a\r\n" +
				"echo path: " + repoPath + "\r\n" +
				"echo remote_url: git@github.com:org/repo-a.git\r\n" +
				"echo labels:\r\n" +
				"echo   team: platform\r\n" +
				"echo annotations:\r\n" +
				"echo   owner: sre\r\n" +
				"echo last_seen: 2026-01-01T00:00:00Z\r\n" +
				"echo status: present\r\n" +
				")\r\n"
		}
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write editor script: %v", err)
		}
		return scriptPath
	}

	scriptPath := filepath.Join(dir, "editor.sh")
	var script string
	if invalidYAML {
		script = "#!/bin/sh\necho 'not: [valid' > \"$1\"\n"
	} else {
		script = "#!/bin/sh\ncat > \"$1\" <<'EOF'\nrepo_id: github.com/org/repo-a\npath: " + repoPath + "\nremote_url: git@github.com:org/repo-a.git\nlabels:\n  team: platform\nannotations:\n  owner: sre\nlast_seen: 2026-01-01T00:00:00Z\nstatus: present\nEOF\n"
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	return scriptPath
}

func TestTrackingBranchFromUpstream(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
		want     string
	}{
		{name: "simple branch", upstream: "origin/main", want: "main"},
		{
			// Regression test: branch names may themselves contain "/". Only
			// the remote (first path segment) must be stripped, not the last
			// segment — "upstream/release/v1" is remote "upstream" tracking
			// branch "release/v1", not branch "v1".
			name:     "branch name containing slashes",
			upstream: "upstream/release/v1",
			want:     "release/v1",
		},
		{name: "deeply nested branch name", upstream: "origin/feature/team/x", want: "feature/team/x"},
		{name: "no remote prefix", upstream: "main", want: "main"},
		{name: "empty upstream", upstream: "", want: ""},
		{name: "whitespace-only upstream", upstream: "   ", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := trackingBranchFromUpstream(tc.upstream); got != tc.want {
				t.Fatalf("trackingBranchFromUpstream(%q) = %q, want %q", tc.upstream, got, tc.want)
			}
		})
	}
}

func TestValidateEditedRegistryEntryUniqueness(t *testing.T) {
	base := func() registry.Entry {
		return registry.Entry{
			RepoID: "github.com/org/repo",
			Path:   "/repos/repo-a",
			Status: registry.StatusPresent,
		}
	}

	tests := []struct {
		name    string
		edited  registry.Entry
		reg     *registry.Registry
		index   int
		wantErr bool
	}{
		{
			name:   "no other entries",
			edited: base(),
			reg:    &registry.Registry{Entries: []registry.Entry{base()}},
			index:  0,
		},
		{
			name:   "same repo_id different checkout_id is allowed (multi-checkout)",
			edited: registry.Entry{RepoID: "github.com/org/repo", CheckoutID: "b", Path: "/repos/repo-b", Status: registry.StatusPresent},
			reg: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", CheckoutID: "a", Path: "/repos/repo-a", Status: registry.StatusPresent},
				{RepoID: "github.com/org/repo", CheckoutID: "b", Path: "/repos/repo-b", Status: registry.StatusPresent},
			}},
			index: 1,
		},
		{
			name:   "same repo_id different path, no checkout_id, is allowed (multi-checkout)",
			edited: registry.Entry{RepoID: "github.com/org/repo", Path: "/repos/repo-b", Status: registry.StatusPresent},
			reg: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", Path: "/repos/repo-a", Status: registry.StatusPresent},
				{RepoID: "github.com/org/repo", Path: "/repos/repo-b", Status: registry.StatusPresent},
			}},
			index: 1,
		},
		{
			name:   "same repo_id and same checkout_id is rejected",
			edited: registry.Entry{RepoID: "github.com/org/repo", CheckoutID: "a", Path: "/repos/repo-b", Status: registry.StatusPresent},
			reg: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", CheckoutID: "a", Path: "/repos/repo-a", Status: registry.StatusPresent},
				{RepoID: "github.com/org/repo", CheckoutID: "a", Path: "/repos/repo-b", Status: registry.StatusPresent},
			}},
			index:   1,
			wantErr: true,
		},
		{
			name:   "same repo_id and same path (no checkout_id) is rejected",
			edited: registry.Entry{RepoID: "github.com/org/repo", Path: "/repos/repo-a", Status: registry.StatusPresent},
			reg: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", Path: "/repos/repo-a", Status: registry.StatusPresent},
				{RepoID: "github.com/org/repo", Path: "/repos/repo-a", Status: registry.StatusPresent},
			}},
			index:   1,
			wantErr: true,
		},
		{
			name:   "different repo_id never conflicts",
			edited: registry.Entry{RepoID: "github.com/org/other", Path: "/repos/repo-a", Status: registry.StatusPresent},
			reg: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", Path: "/repos/repo-a", Status: registry.StatusPresent},
				{RepoID: "github.com/org/other", Path: "/repos/repo-a", Status: registry.StatusPresent},
			}},
			index: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEditedRegistryEntry(tc.edited, tc.reg, tc.index)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEditRunEUpdatesSingleRepoFromEditor(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/repo-a",
				Path:     repoPath,
				Status:   registry.StatusPresent,
				LastSeen: time.Now(),
			},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	editorScript := writeEditorFixtureScript(t, tmp, repoPath, false)

	prevEditor := os.Getenv("EDITOR")
	if err := os.Setenv("EDITOR", editorScript); err != nil {
		t.Fatalf("set editor env: %v", err)
	}
	defer func() { _ = os.Setenv("EDITOR", prevEditor) }()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", "")
	if err := editCmd.RunE(editCmd, []string{"github.com/org/repo-a"}); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	updatedCfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	entry := updatedCfg.Registry.FindByRepoID("github.com/org/repo-a")
	if entry == nil {
		t.Fatal("expected updated entry")
	}
	if got := entry.Labels["team"]; got != "platform" {
		t.Fatalf("expected label team=platform, got %q", got)
	}
	if got := entry.Annotations["owner"]; got != "sre" {
		t.Fatalf("expected annotation owner=sre, got %q", got)
	}
}

func TestEditRunERejectsInvalidEditedYAML(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	tmp := t.TempDir()
	editorScript := writeEditorFixtureScript(t, tmp, "", true)
	prevEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", editorScript)
	defer func() { _ = os.Setenv("EDITOR", prevEditor) }()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", "")
	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-missing"})
	if err == nil || !strings.Contains(err.Error(), "invalid edited yaml") {
		t.Fatalf("expected invalid yaml error, got %v", err)
	}
}

func TestEditRunEFailsWithoutEditor(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	prevEditor := os.Getenv("EDITOR")
	prevVisual := os.Getenv("VISUAL")
	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")
	defer func() {
		_ = os.Setenv("EDITOR", prevEditor)
		_ = os.Setenv("VISUAL", prevVisual)
	}()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", "")
	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-missing"})
	if err == nil || !strings.Contains(err.Error(), "set VISUAL or EDITOR") {
		t.Fatalf("expected editor-required error, got %v", err)
	}
}

func TestResolveEditorCommandParsesQuotedExecutable(t *testing.T) {
	prevEditor := os.Getenv("EDITOR")
	prevVisual := os.Getenv("VISUAL")
	defer func() {
		_ = os.Setenv("EDITOR", prevEditor)
		_ = os.Setenv("VISUAL", prevVisual)
	}()
	_ = os.Unsetenv("VISUAL")

	if runtime.GOOS == "windows" {
		_ = os.Setenv("EDITOR", `"C:\Program Files\Editor\editor.exe" --wait`)
		parts, err := resolveEditorCommand()
		if err != nil {
			t.Fatalf("resolve editor: %v", err)
		}
		if len(parts) != 2 || parts[0] != `C:\Program Files\Editor\editor.exe` || parts[1] != "--wait" {
			t.Fatalf("unexpected parsed editor parts: %#v", parts)
		}
		return
	}

	_ = os.Setenv("EDITOR", `"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" --wait`)
	parts, err := resolveEditorCommand()
	if err != nil {
		t.Fatalf("resolve editor: %v", err)
	}
	if len(parts) != 2 || parts[0] != "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" || parts[1] != "--wait" {
		t.Fatalf("unexpected parsed editor parts: %#v", parts)
	}
}
