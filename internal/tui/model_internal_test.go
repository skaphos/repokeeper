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

func TestHandleKeyQuitCtrlC(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	next, cmd := m.handleKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
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
