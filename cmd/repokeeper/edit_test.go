// SPDX-License-Identifier: MIT
package repokeeper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestTrackingBranchFromUpstream(t *testing.T) {
	if got := trackingBranchFromUpstream("origin/main"); got != "main" {
		t.Fatalf("expected main, got %q", got)
	}
	if got := trackingBranchFromUpstream("upstream/release/v1"); got != "v1" {
		t.Fatalf("expected v1, got %q", got)
	}
	if got := trackingBranchFromUpstream(""); got != "" {
		t.Fatalf("expected empty for empty upstream, got %q", got)
	}
}

func TestValidateUpstreamRef(t *testing.T) {
	cases := []struct {
		name        string
		upstream    string
		expectError bool
	}{
		{name: "valid simple", upstream: "origin/main"},
		{name: "valid nested branch", upstream: "upstream/release/v1"},
		{name: "missing slash", upstream: "origin", expectError: true},
		{name: "missing remote", upstream: "/main", expectError: true},
		{name: "missing branch", upstream: "origin/", expectError: true},
		{name: "empty", upstream: "", expectError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUpstreamRef(tc.upstream)
			if tc.expectError && err == nil {
				t.Fatalf("expected error for %q", tc.upstream)
			}
			if !tc.expectError && err != nil {
				t.Fatalf("expected no error for %q, got %v", tc.upstream, err)
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

	editorScript := filepath.Join(tmp, "editor.sh")
	script := "#!/bin/sh\ncat > \"$1\" <<'EOF'\nrepo_id: github.com/org/repo-a\npath: " + repoPath + "\nremote_url: git@github.com:org/repo-a.git\nlabels:\n  team: platform\nannotations:\n  owner: sre\nlast_seen: 2026-01-01T00:00:00Z\nstatus: present\nEOF\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}

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
	editorScript := filepath.Join(tmp, "editor.sh")
	script := "#!/bin/sh\necho 'not: [valid' > \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
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
