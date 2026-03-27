// SPDX-License-Identifier: MIT
package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
)

func TestToggleSelectionAddAndRemove(t *testing.T) {
	t.Parallel()

	repos := []model.RepoStatus{{RepoID: "a"}, {RepoID: "b"}}
	m := tuiModel{repos: repos, cursor: 0, width: 120, height: 20}

	m2 := m.toggleSelection()
	if !m2.selected["a"] {
		t.Fatal("expected a to be selected")
	}

	m3 := m2.toggleSelection()
	if m3.selected["a"] {
		t.Fatal("expected a to be deselected")
	}
}

func TestToggleSelectAll(t *testing.T) {
	t.Parallel()

	repos := []model.RepoStatus{{RepoID: "a"}, {RepoID: "b"}, {RepoID: "c"}}
	m := tuiModel{repos: repos, width: 120, height: 20}

	m2 := m.toggleSelectAll()
	if len(m2.selected) != 3 {
		t.Fatalf("expected 3 selected, got %d", len(m2.selected))
	}

	m3 := m2.toggleSelectAll()
	if len(m3.selected) != 0 {
		t.Fatalf("expected 0 selected (deselect all), got %d", len(m3.selected))
	}
}

func TestStartSyncSwitchesToSyncPlanMode(t *testing.T) {
	t.Parallel()

	eng := &mockEngine{statusResult: &model.StatusReport{}}
	m := tuiModel{engine: eng, repos: []model.RepoStatus{{RepoID: "a"}}}
	nm, cmd := m.startSync()
	if nm.(tuiModel).mode != viewSyncPlan {
		t.Fatalf("expected viewSyncPlan, got %v", nm.(tuiModel).mode)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from startSync")
	}
}

func TestHandleSyncPlanSuccess(t *testing.T) {
	t.Parallel()

	plan := []engine.SyncResult{{RepoID: "a", Planned: true, Action: "git fetch"}}
	m := tuiModel{mode: viewSyncPlan, loading: true}
	nm, _ := m.handleSyncPlan(syncPlanMsg{plan: plan})
	next := nm.(tuiModel)
	if next.loading {
		t.Fatal("expected loading=false after plan received")
	}
	if len(next.syncPlan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(next.syncPlan))
	}
}

func TestHandleSyncPlanError(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewSyncPlan}
	nm, _ := m.handleSyncPlan(syncPlanMsg{err: errors.New("network error")})
	next := nm.(tuiModel)
	if next.mode != viewList {
		t.Fatalf("expected viewList on error, got %v", next.mode)
	}
	if next.syncErr == nil {
		t.Fatal("expected syncErr set")
	}
}

func TestHandleSyncPlanKeyConfirm(t *testing.T) {
	t.Parallel()

	plan := []engine.SyncResult{{RepoID: "a", Planned: true}}
	m := tuiModel{mode: viewSyncPlan, syncPlan: plan, modalCursor: 1}
	nm, cmd := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	next := nm.(tuiModel)
	if next.mode != viewProgress {
		t.Fatalf("expected viewProgress on enter with cursor=1, got %v", next.mode)
	}
	if cmd == nil {
		t.Fatal("expected cmd after confirmation")
	}
}

func TestHandleSyncPlanKeyCancel(t *testing.T) {
	t.Parallel()

	plan := []engine.SyncResult{{RepoID: "a", Planned: true}}
	m := tuiModel{mode: viewSyncPlan, syncPlan: plan, modalCursor: 0}
	nm, _ := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	next := nm.(tuiModel)
	if next.mode != viewList {
		t.Fatalf("expected viewList on esc, got %v", next.mode)
	}
	if next.syncPlan != nil {
		t.Fatal("expected syncPlan cleared on cancel")
	}
}

func TestHandleSyncPlanKeyCancelViaEnterOnCancel(t *testing.T) {
	t.Parallel()

	plan := []engine.SyncResult{{RepoID: "a", Planned: true}}
	m := tuiModel{mode: viewSyncPlan, syncPlan: plan, modalCursor: 0}
	nm, cmd := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	next := nm.(tuiModel)
	if next.mode != viewList {
		t.Fatalf("expected viewList when cursor=0 (Cancel), got %v", next.mode)
	}
	if cmd != nil {
		t.Fatal("expected nil cmd when cancelling via modal")
	}
}

func TestModalNavMovesRight(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewSyncPlan, modalCursor: 0}
	nm, _ := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if nm.(tuiModel).modalCursor != 1 {
		t.Fatalf("expected modalCursor=1, got %d", nm.(tuiModel).modalCursor)
	}
}

func TestModalNavMovesLeft(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewSyncPlan, modalCursor: 1}
	nm, _ := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if nm.(tuiModel).modalCursor != 0 {
		t.Fatalf("expected modalCursor=0, got %d", nm.(tuiModel).modalCursor)
	}
}

func TestModalNavClampsAtEdges(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewSyncPlan, modalCursor: 0}
	nm, _ := m.handleSyncPlanKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if nm.(tuiModel).modalCursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", nm.(tuiModel).modalCursor)
	}
}

func TestHandleSyncDone(t *testing.T) {
	t.Parallel()

	results := []engine.SyncResult{{RepoID: "a", OK: true, Outcome: engine.SyncOutcomeFetched}}
	m := tuiModel{mode: viewProgress, syncProgress: make(map[string]engine.SyncResult)}
	nm, _ := m.handleSyncDone(syncDoneMsg{results: results})
	next := nm.(tuiModel)
	if !next.syncDone {
		t.Fatal("expected syncDone=true")
	}
	if next.syncProgress["a"].Outcome != engine.SyncOutcomeFetched {
		t.Fatalf("expected fetched outcome in progress map")
	}
}

func TestSyncProgressKeysByCheckoutIdentity(t *testing.T) {
	t.Parallel()

	m := tuiModel{syncPlan: []engine.SyncResult{
		{RepoID: "acme/backend", Path: "/work/primary"},
		{RepoID: "acme/backend", Path: "/work/secondary"},
	}}

	next, _ := m.handleSyncProgress(syncProgressMsg{result: engine.SyncResult{RepoID: "acme/backend", Path: "/work/primary"}, started: true})
	next, _ = next.(tuiModel).handleSyncProgress(syncProgressMsg{result: engine.SyncResult{RepoID: "acme/backend", Path: "/work/secondary", OK: true, Outcome: engine.SyncOutcomeFetched}, started: false})
	nm := next.(tuiModel)

	if len(nm.syncProgress) != 2 {
		t.Fatalf("expected sync progress to track both checkouts separately, got %d entries", len(nm.syncProgress))
	}

	progress := renderSyncProgressView(nm)
	if !strings.Contains(progress, "running") {
		t.Fatalf("expected progress view to keep one checkout running, got %q", progress)
	}
	if !strings.Contains(progress, "fetched") {
		t.Fatalf("expected progress view to show one checkout fetched, got %q", progress)
	}
}

func TestHandleSyncProgressKey_WhenDoneReturnsToList(t *testing.T) {
	t.Parallel()

	eng := &mockEngine{statusResult: &model.StatusReport{}}
	m := tuiModel{mode: viewProgress, syncDone: true, engine: eng}
	nm, cmd := m.handleSyncProgressKey(tea.KeyPressMsg{Code: 'q'})
	next := nm.(tuiModel)
	if next.mode != viewList {
		t.Fatalf("expected viewList, got %v", next.mode)
	}
	if cmd == nil {
		t.Fatal("expected refresh cmd after returning to list")
	}
}
