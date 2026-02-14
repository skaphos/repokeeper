package repokeeper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestImportTargetRelativePath(t *testing.T) {
	entry := registry.Entry{
		RepoID: "github.com/org/repo-a",
		Path:   "/source/root/team/repo-a",
	}

	if got := importTargetRelativePath(entry, []string{"/source/root"}); got != "team/repo-a" {
		t.Fatalf("expected root-relative target, got %q", got)
	}

	if got := importTargetRelativePath(entry, []string{"/other/root"}); got != "repo-a" {
		t.Fatalf("expected basename fallback, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-z", Path: ""}
	if got := importTargetRelativePath(entry, nil); got != "repo-z" {
		t.Fatalf("expected repo-id fallback, got %q", got)
	}
}

func TestCloneImportedReposReportsSpecificTargetConflicts(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/repo-a",
					Path:      "/source/root/team/repo-a",
					RemoteURL: "git@github.com:org/repo-a.git",
					Status:    registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Config: config.Config{Roots: []string{"/source/root"}}}
	target := filepath.Join(cwd, "team", "repo-a")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	cmd := &cobra.Command{}
	err := cloneImportedRepos(cmd, cfg, bundle, cwd, false)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "import target conflicts detected") {
		t.Fatalf("expected conflict summary error, got: %v", err)
	}
	if !strings.Contains(err.Error(), target) {
		t.Fatalf("expected target path in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "github.com/org/repo-a") {
		t.Fatalf("expected repo id in error, got: %v", err)
	}
}

func TestCloneImportedReposSkipsLocalEntriesWithoutRemoteURL(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:   "local:/source/root/team/repo-a",
					Path:     "/source/root/team/repo-a",
					LastSeen: time.Now(),
					Status:   registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Config: config.Config{Roots: []string{"/source/root"}}}

	cmd := &cobra.Command{}
	if err := cloneImportedRepos(cmd, cfg, bundle, cwd, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Registry.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(cfg.Registry.Entries))
	}
	entry := cfg.Registry.Entries[0]
	if entry.Status != registry.StatusMissing {
		t.Fatalf("expected local entry to be missing after skip, got %q", entry.Status)
	}
	if got, want := entry.Path, filepath.Join(cwd, "team", "repo-a"); got != want {
		t.Fatalf("expected rewritten path %q, got %q", want, got)
	}
}

func TestCloneImportedReposErrorsForNonLocalMissingRemoteURL(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID: "github.com/org/repo-a",
					Path:   "/source/root/team/repo-a",
					Status: registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Config: config.Config{Roots: []string{"/source/root"}}}

	cmd := &cobra.Command{}
	err := cloneImportedRepos(cmd, cfg, bundle, cwd, false)
	if err == nil {
		t.Fatal("expected error for non-local repo missing remote_url")
	}
	if !strings.Contains(err.Error(), "missing remote_url in bundle") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportCommandArgsValidation(t *testing.T) {
	if importCmd.Args == nil {
		t.Fatal("expected import command args validator")
	}
	if err := importCmd.Args(importCmd, []string{"a.yaml", "b.yaml"}); err == nil {
		t.Fatal("expected too-many-args validation error")
	}
	if err := importCmd.Args(importCmd, []string{}); err != nil {
		t.Fatalf("expected zero args to be valid (stdin), got: %v", err)
	}
	if err := importCmd.Args(importCmd, []string{"bundle.yaml"}); err != nil {
		t.Fatalf("expected one arg to be valid, got: %v", err)
	}
}
