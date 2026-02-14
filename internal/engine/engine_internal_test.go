package engine

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func TestScanUpdatesRegistry(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v %s", err, string(out))
	}

	cfg := &config.Config{Roots: []string{root}, Exclude: []string{}}
	reg := &registry.Registry{}
	eng := New(cfg, reg, vcs.NewGitAdapter(nil))
	statuses, err := eng.Scan(context.Background(), ScanOptions{})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("unexpected registry state: %+v", reg)
	}
}

func TestFilterAndSortHelpers(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{{RepoID: "r1", Status: registry.StatusMissing}},
	}
	if !filterStatus(FilterMissing, model.RepoStatus{RepoID: "r1"}, reg) {
		t.Fatal("expected missing filter match")
	}
	if !filterStatus(FilterErrors, model.RepoStatus{Error: "boom"}, reg) {
		t.Fatal("expected errors filter match")
	}

	repos := []model.RepoStatus{{RepoID: "b", Path: "/2"}, {RepoID: "a", Path: "/1"}}
	sortRepoStatuses(repos)
	if repos[0].RepoID != "a" {
		t.Fatalf("expected sorted repos, got %#v", repos)
	}

	results := []SyncResult{{RepoID: "b"}, {RepoID: "a"}}
	sortSyncResults(results)
	if results[0].RepoID != "a" {
		t.Fatalf("expected sorted sync results, got %#v", results)
	}
}
