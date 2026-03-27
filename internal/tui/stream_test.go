// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestHandleRepoStatusUpdatesExistingRow(t *testing.T) {
	t.Parallel()

	partial := model.RepoStatus{RepoID: "a/b", Path: "/work/b"}
	m := tuiModel{repos: []model.RepoStatus{partial}}

	full := model.RepoStatus{RepoID: "a/b", Path: "/work/b", Head: model.Head{Branch: "main"}}
	nm, cmd := m.handleRepoStatus(repoStatusMsg{status: full})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	next := nm.(tuiModel)
	if len(next.repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(next.repos))
	}
	if next.repos[0].Head.Branch != "main" {
		t.Fatalf("expected branch=main, got %q", next.repos[0].Head.Branch)
	}
}

func TestHandleRepoStatusAddsNewRow(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	nm, _ := m.handleRepoStatus(repoStatusMsg{status: model.RepoStatus{RepoID: "new/repo"}})
	if len(nm.(tuiModel).repos) != 1 {
		t.Fatalf("expected 1 repo after add, got %d", len(nm.(tuiModel).repos))
	}
}

func TestHandleRepoStatusUpdatesFilteredResults(t *testing.T) {
	t.Parallel()

	partial := model.RepoStatus{RepoID: "acme/svc", Path: "/work/svc"}
	m := tuiModel{repos: []model.RepoStatus{partial}, filterText: "acme"}
	m.filteredRepos = filterRows(m.repos, m.filterText)

	full := model.RepoStatus{RepoID: "acme/svc", Path: "/work/svc", Head: model.Head{Branch: "main"}}
	nm, _ := m.handleRepoStatus(repoStatusMsg{status: full})
	next := nm.(tuiModel)
	if len(next.filteredRepos) != 1 {
		t.Fatalf("expected filtered repos to be updated, got %d", len(next.filteredRepos))
	}
}

func TestHandleRepoStatusDoesNotCollapseDuplicateRepoID(t *testing.T) {
	t.Parallel()

	m := tuiModel{repos: []model.RepoStatus{
		{RepoID: "acme/backend", Path: "/work/primary"},
		{RepoID: "acme/backend", Path: "/work/secondary"},
	}}

	nm, cmd := m.handleRepoStatus(repoStatusMsg{status: model.RepoStatus{
		RepoID: "acme/backend",
		Path:   "/work/secondary",
		Head:   model.Head{Branch: "feature/secondary"},
	}})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}

	next := nm.(tuiModel)
	if len(next.repos) != 2 {
		t.Fatalf("expected duplicate checkouts to stay distinct, got %d rows", len(next.repos))
	}
	if next.repos[0].Path != "/work/primary" {
		t.Fatalf("expected first checkout path to stay /work/primary, got %q", next.repos[0].Path)
	}
	if next.repos[1].Path != "/work/secondary" {
		t.Fatalf("expected second checkout path to stay /work/secondary, got %q", next.repos[1].Path)
	}
	if next.repos[0].Head.Branch != "" {
		t.Fatalf("expected first checkout to remain untouched, got branch %q", next.repos[0].Head.Branch)
	}
	if next.repos[1].Head.Branch != "feature/secondary" {
		t.Fatalf("expected second checkout branch to update, got %q", next.repos[1].Head.Branch)
	}
}

func TestLastRepoStatusClearsLoading(t *testing.T) {
	t.Parallel()

	m := tuiModel{
		repos:              []model.RepoStatus{{RepoID: "a", Path: "/a"}, {RepoID: "b", Path: "/b"}},
		loading:            true,
		pendingInspections: 2,
	}
	nm, _ := m.handleRepoStatus(repoStatusMsg{status: model.RepoStatus{RepoID: "a", Path: "/a"}})
	next := nm.(tuiModel)
	if !next.loading {
		t.Fatal("expected loading=true while 1 inspection pending")
	}
	nm2, _ := next.handleRepoStatus(repoStatusMsg{status: model.RepoStatus{RepoID: "b", Path: "/b"}})
	next2 := nm2.(tuiModel)
	if next2.loading {
		t.Fatal("expected loading=false after last inspection")
	}
}

func TestInitUsesStreamWhenRegistryHasEntries(t *testing.T) {
	t.Parallel()

	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "a", Path: "/tmp/a", Status: registry.StatusPresent},
			{RepoID: "b", Path: "/tmp/b", Status: registry.StatusPresent},
		},
	}
	eng := &mockEngine{reg: reg}
	m := newModel(context.Background(), eng, reg, "")

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init() when registry has entries")
	}
}

func TestInitFallsBackToStatusWhenRegistryEmpty(t *testing.T) {
	t.Parallel()

	reg := &registry.Registry{}
	eng := &mockEngine{reg: reg, statusResult: &model.StatusReport{}}
	m := newModel(context.Background(), eng, reg, "")

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init()")
	}
	result := cmd()
	if _, ok := result.(statusReportMsg); !ok {
		t.Fatalf("expected statusReportMsg fallback, got %T", result)
	}
}

func TestWindowResizeUpdatesViewport(t *testing.T) {
	t.Parallel()

	m := tuiModel{width: 80, height: 24}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	next := nm.(tuiModel)
	if next.width != 160 || next.height != 50 {
		t.Fatalf("expected 160x50, got %dx%d", next.width, next.height)
	}
}

func TestHandleRepoStatusWritesRepoMetadataSnapshotBackToRegistry(t *testing.T) {
	t.Parallel()

	reg := &registry.Registry{Entries: []registry.Entry{{
		RepoID: "acme/repo", Path: "/work/repo", Status: registry.StatusPresent,
	}}}
	m := tuiModel{engine: &mockEngine{reg: reg}}
	status := model.RepoStatus{
		RepoID:                  "acme/repo",
		Path:                    "/work/repo",
		RepoMetadataFile:        "/work/repo/.repokeeper-repo.yaml",
		RepoMetadataFingerprint: "file:/work/repo/.repokeeper-repo.yaml:1:1",
		RepoMetadata:            &model.RepoMetadata{Name: "Repo"},
	}

	nm, _ := m.handleRepoStatus(repoStatusMsg{status: status})
	_ = nm

	entry := reg.FindEntry("acme/repo", "/work/repo")
	if entry == nil {
		t.Fatal("expected registry entry")
	}
	if entry.RepoMetadataFile != status.RepoMetadataFile {
		t.Fatalf("expected metadata file %q, got %q", status.RepoMetadataFile, entry.RepoMetadataFile)
	}
	if entry.RepoMetadataFingerprint == "" {
		t.Fatal("expected metadata fingerprint to be cached")
	}
	if entry.RepoMetadata == nil || entry.RepoMetadata.Name != "Repo" {
		t.Fatalf("expected metadata payload cached, got %+v", entry.RepoMetadata)
	}
}

func TestEmptyRegistryRenderNoRepos(t *testing.T) {
	t.Parallel()

	m := tuiModel{width: 120, height: 24, loading: false}
	content := renderListView(m)
	if content == "" {
		t.Fatal("expected non-empty content")
	}
}
