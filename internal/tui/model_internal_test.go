// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type mockEngine struct {
	statusResult *model.StatusReport
	statusErr    error
	reg          *registry.Registry
	cfg          *config.Config
}

func (e *mockEngine) Status(ctx context.Context, opts engine.StatusOptions) (*model.StatusReport, error) {
	return e.statusResult, e.statusErr
}

func (e *mockEngine) Sync(ctx context.Context, opts engine.SyncOptions) ([]engine.SyncResult, error) {
	return nil, nil
}

func (e *mockEngine) ExecuteSyncPlanWithCallbacks(
	ctx context.Context,
	plan []engine.SyncResult,
	opts engine.SyncOptions,
	onStart engine.SyncStartCallback,
	onComplete engine.SyncResultCallback,
) ([]engine.SyncResult, error) {
	return nil, nil
}

func (e *mockEngine) InspectRepo(ctx context.Context, path string) (*model.RepoStatus, error) {
	return nil, nil
}

func (e *mockEngine) RepairUpstream(ctx context.Context, repoID, cfgPath string) (engine.RepairUpstreamResult, error) {
	return engine.RepairUpstreamResult{}, nil
}
func (e *mockEngine) ResetRepo(ctx context.Context, repoID, cfgPath string) error { return nil }
func (e *mockEngine) DeleteRepo(ctx context.Context, repoID, cfgPath string, deleteFiles bool) error {
	return nil
}
func (e *mockEngine) CloneAndRegister(ctx context.Context, remoteURL, targetPath, cfgPath string, mirror bool) error {
	return nil
}

func (e *mockEngine) Registry() *registry.Registry {
	return e.reg
}

func (e *mockEngine) Config() *config.Config {
	return e.cfg
}

func TestNewModelWithRegistry(t *testing.T) {
	t.Parallel()

	reg := &registry.Registry{Entries: []registry.Entry{
		{RepoID: "repo/a", Path: "/tmp/a", Type: "checkout", Branch: "main", Status: registry.StatusPresent},
		{RepoID: "repo/b", Path: "/tmp/b", Type: "mirror", Status: registry.StatusMissing},
	}}

	m := newModel(&mockEngine{}, reg, "")
	if len(m.repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(m.repos))
	}
	if !m.loading {
		t.Fatal("expected loading=true")
	}
	if m.mode != viewList {
		t.Fatalf("expected mode viewList, got %v", m.mode)
	}
	if m.pendingInspections != 2 {
		t.Fatalf("expected pendingInspections=2, got %d", m.pendingInspections)
	}
}

func TestNewModelWithNilRegistry(t *testing.T) {
	t.Parallel()

	m := newModel(&mockEngine{}, nil, "")
	if len(m.repos) != 0 {
		t.Fatalf("expected no repos, got %d", len(m.repos))
	}
}

func TestHandleStatusReportSuccess(t *testing.T) {
	t.Parallel()

	m := tuiModel{repos: []model.RepoStatus{{RepoID: "old"}}, loading: true, cursor: 3, offset: 4}
	report := &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "new"}}}

	next, cmd := m.handleStatusReport(statusReportMsg{report: report})
	if cmd != nil {
		t.Fatal("expected nil command")
	}
	nm := next.(tuiModel)
	if nm.loading {
		t.Fatal("expected loading=false")
	}
	if len(nm.repos) != 1 || nm.repos[0].RepoID != "new" {
		t.Fatalf("expected repos to be updated, got %#v", nm.repos)
	}
	if nm.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", nm.cursor)
	}
}

func TestHandleStatusReportError(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	m := tuiModel{repos: []model.RepoStatus{{RepoID: "keep"}}, loading: true}

	next, _ := m.handleStatusReport(statusReportMsg{err: errBoom})
	nm := next.(tuiModel)
	if nm.loading {
		t.Fatal("expected loading=false")
	}
	if !errors.Is(nm.err, errBoom) {
		t.Fatalf("expected error %v, got %v", errBoom, nm.err)
	}
	if len(nm.repos) != 1 || nm.repos[0].RepoID != "keep" {
		t.Fatalf("expected repos unchanged, got %#v", nm.repos)
	}
}

func TestHandleKeyQuitQ(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	next, cmd := m.handleKey(tea.KeyPressMsg{Code: 'q'})
	nm := next.(tuiModel)
	if nm.cursor != m.cursor || nm.offset != m.offset {
		t.Fatal("expected unchanged model state")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
}

func TestHandleKeyQuitCtrlCGlobal(t *testing.T) {
	t.Parallel()

	for _, mode := range []viewMode{viewList, viewDetail, viewSyncPlan, viewProgress} {
		m := tuiModel{mode: mode}
		_, cmd := m.handleKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Fatalf("expected quit command in mode %v", mode)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("expected tea.QuitMsg in mode %v, got %T", mode, cmd())
		}
	}

	m := tuiModel{filterMode: true}
	_, cmd := m.handleKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected quit command in filter mode")
	}
}

func TestHandleKeyUnhandled(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	next, cmd := m.handleKey(tea.KeyPressMsg{Code: 'j'})
	nm := next.(tuiModel)
	if nm.cursor != m.cursor || nm.offset != m.offset {
		t.Fatal("expected unchanged model state")
	}
	if cmd != nil {
		t.Fatal("expected nil command for unhandled key")
	}
}

func TestUpdateWindowSize(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	next, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatal("expected nil command")
	}
	nm := next.(tuiModel)
	if nm.width != 120 || nm.height != 40 {
		t.Fatalf("expected width=120,height=40 got width=%d,height=%d", nm.width, nm.height)
	}
}

func TestMoveCursorDown(t *testing.T) {
	t.Parallel()

	repos := make([]model.RepoStatus, 5)
	m := tuiModel{repos: repos, width: 120, height: 20}
	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
	if cmd != nil {
		t.Fatal("expected nil cmd")
	}
	if nm.(tuiModel).cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", nm.(tuiModel).cursor)
	}
}

func TestMoveCursorUp(t *testing.T) {
	t.Parallel()

	repos := make([]model.RepoStatus, 5)
	m := tuiModel{repos: repos, cursor: 2, width: 120, height: 20}
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	if nm.(tuiModel).cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", nm.(tuiModel).cursor)
	}
}

func TestMoveCursorClampAtZero(t *testing.T) {
	t.Parallel()

	repos := make([]model.RepoStatus, 3)
	m := tuiModel{repos: repos, cursor: 0, width: 120, height: 20}
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	if nm.(tuiModel).cursor != 0 {
		t.Fatalf("expected cursor=0 (clamped), got %d", nm.(tuiModel).cursor)
	}
}

func TestMoveCursorClampAtEnd(t *testing.T) {
	t.Parallel()

	repos := make([]model.RepoStatus, 3)
	m := tuiModel{repos: repos, cursor: 2, width: 120, height: 20}
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	if nm.(tuiModel).cursor != 2 {
		t.Fatalf("expected cursor=2 (clamped), got %d", nm.(tuiModel).cursor)
	}
}

func TestFilterModeActivation(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	nm, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	if !nm.(tuiModel).filterMode {
		t.Fatal("expected filterMode=true after /")
	}
}

func TestFilterModeTyping(t *testing.T) {
	t.Parallel()

	repos := []model.RepoStatus{{RepoID: "acme/backend"}, {RepoID: "tools/cli"}}
	m := tuiModel{repos: repos, filterMode: true}
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	next := nm.(tuiModel)
	if next.filterText != "a" {
		t.Fatalf("expected filterText='a', got %q", next.filterText)
	}
	if len(next.filteredRepos) != 1 || next.filteredRepos[0].RepoID != "acme/backend" {
		t.Fatalf("expected filteredRepos=[acme/backend], got %v", next.filteredRepos)
	}
}

func TestFilterModeEscClears(t *testing.T) {
	t.Parallel()

	m := tuiModel{filterMode: true, filterText: "abc", filteredRepos: []model.RepoStatus{{RepoID: "x"}}}
	nm, _ := m.handleFilterKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	next := nm.(tuiModel)
	if next.filterMode {
		t.Fatal("expected filterMode=false after esc")
	}
	if next.filterText != "" {
		t.Fatalf("expected filterText='', got %q", next.filterText)
	}
}

func TestEscInListClearsFilter(t *testing.T) {
	t.Parallel()

	m := tuiModel{filterText: "abc", filteredRepos: []model.RepoStatus{{RepoID: "x"}}}
	nm, _ := m.handleListKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	next := nm.(tuiModel)
	if next.filterText != "" {
		t.Fatalf("expected filterText='', got %q", next.filterText)
	}
}

func TestF5TriggersRefresh(t *testing.T) {
	t.Parallel()

	eng := &mockEngine{statusResult: &model.StatusReport{}}
	m := tuiModel{engine: eng}
	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	if !nm.(tuiModel).loading {
		t.Fatal("expected loading=true after f5")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from f5")
	}
	result := cmd()
	if _, ok := result.(statusReportMsg); !ok {
		t.Fatalf("expected statusReportMsg, got %T", result)
	}
}
