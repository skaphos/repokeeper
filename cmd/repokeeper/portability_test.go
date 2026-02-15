package repokeeper

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
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
	bundle := exportBundle{Config: config.Config{}}
	target := filepath.Join(cwd, "repo-a")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
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
	bundle := exportBundle{Config: config.Config{}}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
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
	if got, want := entry.Path, filepath.Join(cwd, "repo-a"); got != want {
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
	bundle := exportBundle{Config: config.Config{}}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
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

func TestPopulateExportBranches(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/repos/r1", Status: registry.StatusPresent},
			{RepoID: "r2", Path: "/repos/r2", Status: registry.StatusPresent, Branch: "keep-me"},
			{RepoID: "r3", Path: "/repos/r3", Status: registry.StatusMissing, Branch: "stale"},
			{RepoID: "r4", Path: "/repos/r4", Type: "mirror", Status: registry.StatusPresent, Branch: "mirror-branch"},
		},
	}

	populateExportBranches(context.Background(), reg, func(_ context.Context, path string) (model.Head, error) {
		switch path {
		case "/repos/r1":
			return model.Head{Branch: "feature/a"}, nil
		case "/repos/r2":
			return model.Head{}, errors.New("head failed")
		case "/repos/r4":
			return model.Head{Branch: "should-not-apply"}, nil
		default:
			return model.Head{}, nil
		}
	})

	if got, want := reg.Entries[0].Branch, "feature/a"; got != want {
		t.Fatalf("expected branch %q, got %q", want, got)
	}
	if got, want := reg.Entries[1].Branch, "keep-me"; got != want {
		t.Fatalf("expected existing branch %q to remain, got %q", want, got)
	}
	if got, want := reg.Entries[2].Branch, "stale"; got != want {
		t.Fatalf("expected missing entry branch %q to remain, got %q", want, got)
	}
	if got, want := reg.Entries[3].Branch, "mirror-branch"; got != want {
		t.Fatalf("expected mirror branch %q to remain, got %q", want, got)
	}
}
