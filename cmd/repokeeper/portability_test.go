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

func TestIsDirectoryEmpty(t *testing.T) {
	dir := t.TempDir()
	empty, err := isDirectoryEmpty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !empty {
		t.Fatal("expected new temp dir to be empty")
	}

	f := filepath.Join(dir, "x")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	empty, err = isDirectoryEmpty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty {
		t.Fatal("expected dir with files to be non-empty")
	}
}

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
