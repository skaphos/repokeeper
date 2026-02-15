package repokeeper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestEditRunEDetachedHead(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo-detached")
	mustRunGit(t, filepath.Dir(repo), "init", repo)
	mustRunGit(t, repo, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, repo, "checkout", "--detach", "HEAD")

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	regPath := filepath.Join(tmp, "registry.yaml")
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-detached", Path: repo, Status: registry.StatusPresent},
		},
	}
	cfg := config.DefaultConfig()
	cfg.Registry = reg
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", regPath)
	_ = editCmd.Flags().Set("set-upstream", "origin/main")

	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-detached"})
	if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
		t.Fatalf("expected detached head error, got %v", err)
	}
}

func TestEditRunERunnerFailure(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo-no-upstream")
	mustRunGit(t, filepath.Dir(repo), "init", repo)
	mustRunGit(t, repo, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, repo, "remote", "add", "origin", "git@github.com:org/repo-no-upstream.git")

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	regPath := filepath.Join(tmp, "registry.yaml")
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-no-upstream", Path: repo, Status: registry.StatusPresent},
		},
	}
	cfg := config.DefaultConfig()
	cfg.Registry = reg
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", regPath)
	_ = editCmd.Flags().Set("set-upstream", "origin/main")

	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-no-upstream"})
	if err == nil || !strings.Contains(err.Error(), "git branch --set-upstream-to") {
		t.Fatalf("expected set-upstream runner error, got %v", err)
	}
}

func TestEditRunERegistryOverrideLoadError(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", filepath.Join(t.TempDir(), "missing-registry.yaml"))
	_ = editCmd.Flags().Set("set-upstream", "origin/main")

	err := editCmd.RunE(editCmd, []string{"github.com/org/repo"})
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected registry load os-not-exist error, got %v", err)
	}
}

func TestEditRejectsInvalidSetUpstreamFormat(t *testing.T) {
	cfgPath, regPath := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", regPath)
	_ = editCmd.Flags().Set("set-upstream", "origin")
	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-missing"})
	if err == nil || !strings.Contains(err.Error(), "expected remote/branch") {
		t.Fatalf("expected invalid set-upstream format error, got %v", err)
	}
}
