// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestHandleDetailKey(t *testing.T) {
	t.Parallel()

	for _, key := range []tea.KeyPressMsg{
		{Code: 'q'},
		{Code: tea.KeyEscape},
		{Code: tea.KeyBackspace},
	} {
		m := tuiModel{mode: viewDetail}
		next, _ := m.handleDetailKey(key)
		if next.(tuiModel).mode != viewList {
			t.Fatalf("expected viewList for %q", key.String())
		}
	}

	m := tuiModel{mode: viewDetail}
	next, _ := m.handleDetailKey(tea.KeyPressMsg{Code: 'x'})
	if next.(tuiModel).mode != viewDetail {
		t.Fatal("expected unhandled key to keep detail mode")
	}
}

func TestHandleListKey(t *testing.T) {
	t.Parallel()

	repos := []model.RepoStatus{{RepoID: "acme/a", Path: "/tmp/a", PrimaryRemote: "origin", Head: model.Head{Branch: "main"}}}
	reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "acme/a", Path: "/tmp/a", Branch: "main", Status: registry.StatusPresent}}}
	eng := &mockEngine{reg: reg, cfg: &config.Config{Defaults: config.Defaults{MainBranch: "main"}}}

	qModel, qCmd := (tuiModel{}).handleListKey(tea.KeyPressMsg{Code: 'q'})
	if qCmd == nil {
		t.Fatal("expected quit cmd")
	}
	if _, ok := qCmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", qCmd())
	}
	if qModel.(tuiModel).mode != viewList {
		t.Fatal("expected list mode")
	}

	m := tuiModel{repos: repos, width: 100, height: 30, engine: eng}
	next, _ := m.handleListKey(tea.KeyPressMsg{Code: 'j'})
	if next.(tuiModel).cursor != 0 {
		t.Fatal("expected clamp at last row")
	}

	m = tuiModel{repos: append(repos, model.RepoStatus{RepoID: "acme/b"}), width: 100, height: 30, engine: eng, cursor: 1}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'k'})
	if next.(tuiModel).cursor != 0 {
		t.Fatal("expected move up")
	}

	m = tuiModel{repos: repos}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: '/'})
	if !next.(tuiModel).filterMode {
		t.Fatal("expected filter mode")
	}

	m = tuiModel{repos: repos, cursor: 0}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewDetail {
		t.Fatal("expected detail mode")
	}

	m = tuiModel{repos: repos, cursor: 0}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: ' '})
	if !next.(tuiModel).selected["acme/a"] {
		t.Fatal("expected selected repo")
	}

	m = tuiModel{repos: append(repos, model.RepoStatus{RepoID: "acme/b"})}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'a'})
	if len(next.(tuiModel).selected) != 2 {
		t.Fatal("expected all selected")
	}

	m = tuiModel{repos: repos, cursor: 0, engine: eng}
	next, cmd := m.handleListKey(tea.KeyPressMsg{Code: 's'})
	if next.(tuiModel).mode != viewSyncPlan || cmd == nil {
		t.Fatal("expected sync plan mode with cmd")
	}

	m = tuiModel{repos: repos, cursor: 0, engine: eng}
	_, cmd = m.handleListKey(tea.KeyPressMsg{Code: 'e'})
	if cmd == nil {
		t.Fatal("expected edit prepare cmd")
	}

	m = tuiModel{repos: repos, cursor: 0, engine: eng}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'r'})
	if next.(tuiModel).mode != viewRepairConfirm {
		t.Fatal("expected repair confirm mode")
	}

	m = tuiModel{repos: repos, cursor: 0}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	if next.(tuiModel).mode != viewResetConfirm {
		t.Fatal("expected reset confirm mode")
	}

	m = tuiModel{repos: repos, cursor: 0}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if next.(tuiModel).mode != viewDeleteConfirm {
		t.Fatal("expected delete confirm mode")
	}

	m = tuiModel{}
	next, _ = m.handleListKey(tea.KeyPressMsg{Code: 'n'})
	if next.(tuiModel).mode != viewAdd {
		t.Fatal("expected add mode")
	}
}

func TestHandleFilterKey(t *testing.T) {
	t.Parallel()

	repos := []model.RepoStatus{{RepoID: "acme/backend"}, {RepoID: "tools/cli"}}
	m := tuiModel{repos: repos, filterMode: true, filterText: "ac"}
	next, _ := m.handleFilterKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.filterMode || nm.filterText != "" {
		t.Fatal("expected esc to clear and exit filter mode")
	}

	m = tuiModel{repos: repos, filterMode: true}
	next, _ = m.handleFilterKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).filterMode {
		t.Fatal("expected enter to leave filter mode")
	}

	m = tuiModel{repos: repos, filterMode: true, filterText: "abc", filteredRepos: repos}
	next, _ = m.handleFilterKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	nm = next.(tuiModel)
	if nm.filterText != "ab" {
		t.Fatalf("expected backspace to trim text, got %q", nm.filterText)
	}

	m = tuiModel{repos: repos, filterMode: true}
	next, _ = m.handleFilterKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	nm = next.(tuiModel)
	if nm.filterText != "a" || len(nm.filteredRepos) != 1 {
		t.Fatal("expected typed char to update filtering")
	}
}

func TestHandleResetConfirmKey(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewResetConfirm, resetRepoID: "acme/a", modalCursor: 1, engine: &mockEngine{}}
	next, _ := m.handleResetConfirmKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.mode != viewList || nm.resetRepoID != "" || nm.modalCursor != 0 {
		t.Fatal("expected esc to cancel reset")
	}

	m = tuiModel{mode: viewResetConfirm, resetRepoID: "acme/a", modalCursor: 1, engine: &mockEngine{}}
	next, cmd := m.handleResetConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm = next.(tuiModel)
	if nm.mode != viewList || cmd == nil {
		t.Fatal("expected confirm reset command")
	}
	if _, ok := cmd().(resetDoneMsg); !ok {
		t.Fatalf("expected resetDoneMsg, got %T", cmd())
	}

	m = tuiModel{mode: viewResetConfirm, resetRepoID: "acme/a", modalCursor: 0, engine: &mockEngine{}}
	next, cmd = m.handleResetConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd != nil {
		t.Fatal("expected cancel branch on cursor=0")
	}

	m = tuiModel{mode: viewResetConfirm, modalCursor: 1}
	next, _ = m.handleResetConfirmKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if next.(tuiModel).modalCursor != 0 {
		t.Fatal("expected move left")
	}
	next, _ = next.(tuiModel).handleResetConfirmKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if next.(tuiModel).modalCursor != 1 {
		t.Fatal("expected move right")
	}
}

func TestHandleDeleteConfirmKey(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewDeleteConfirm, deleteRepoID: "acme/a", deleteRepoPath: "/tmp/a", modalCursor: 1, engine: &mockEngine{}}
	next, _ := m.handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.mode != viewList || nm.deleteRepoID != "" || nm.deleteRepoPath != "" {
		t.Fatal("expected esc to cancel delete")
	}

	m = tuiModel{mode: viewDeleteConfirm, deleteRepoID: "acme/a", modalCursor: 0, engine: &mockEngine{}}
	next, cmd := m.handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd != nil {
		t.Fatal("expected cursor=0 cancel")
	}

	m = tuiModel{mode: viewDeleteConfirm, deleteRepoID: "acme/a", modalCursor: 1, engine: &mockEngine{}}
	next, cmd = m.handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd == nil {
		t.Fatal("expected unregister command")
	}
	if _, ok := cmd().(deleteDoneMsg); !ok {
		t.Fatalf("expected deleteDoneMsg, got %T", cmd())
	}

	m = tuiModel{mode: viewDeleteConfirm, deleteRepoID: "acme/a", modalCursor: 2, engine: &mockEngine{}}
	next, cmd = m.handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd == nil {
		t.Fatal("expected delete-from-disk command")
	}

	m = tuiModel{mode: viewDeleteConfirm, modalCursor: 2}
	next, _ = m.handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	if next.(tuiModel).modalCursor != 1 {
		t.Fatal("expected move left")
	}
	next, _ = next.(tuiModel).handleDeleteConfirmKey(tea.KeyPressMsg{Code: tea.KeyRight})
	if next.(tuiModel).modalCursor != 2 {
		t.Fatal("expected move right")
	}
}

func TestHandleRepairConfirmKey(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewRepairConfirm, repairRepoID: "acme/a", repairTargetUpstream: "origin/main", modalCursor: 1, engine: &mockEngine{}}
	next, _ := m.handleRepairConfirmKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.mode != viewList || nm.repairRepoID != "" || nm.repairTargetUpstream != "" {
		t.Fatal("expected esc to cancel repair")
	}

	m = tuiModel{mode: viewRepairConfirm, repairRepoID: "acme/a", repairTargetUpstream: "", modalCursor: 1}
	next, _ = m.handleRepairConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList {
		t.Fatal("expected enter with empty target to cancel")
	}

	m = tuiModel{mode: viewRepairConfirm, repairRepoID: "acme/a", repairTargetUpstream: "origin/main", modalCursor: 0, engine: &mockEngine{}}
	next, cmd := m.handleRepairConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd != nil {
		t.Fatal("expected cursor=0 to cancel")
	}

	m = tuiModel{mode: viewRepairConfirm, repairRepoID: "acme/a", repairTargetUpstream: "origin/main", modalCursor: 1, engine: &mockEngine{}}
	next, cmd = m.handleRepairConfirmKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd == nil {
		t.Fatal("expected repair command")
	}
	if _, ok := cmd().(repairDoneMsg); !ok {
		t.Fatalf("expected repairDoneMsg, got %T", cmd())
	}
}

func TestHandleAddKey(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewAdd, addURL: "x", addPath: "y", addMirror: true, addField: addFieldPath}
	next, _ := m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.mode != viewList || nm.addURL != "" || nm.addPath != "" || nm.addMirror || nm.addField != addFieldURL {
		t.Fatal("expected esc reset add form")
	}

	m = tuiModel{mode: viewAdd, addField: addFieldURL}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).addField != addFieldURL {
		t.Fatal("expected empty URL to stay on URL field")
	}

	m = tuiModel{mode: viewAdd, addField: addFieldURL, addURL: "https://github.com/acme/repo.git"}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).addField != addFieldPath {
		t.Fatal("expected URL enter to move to path")
	}

	m = next.(tuiModel)
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).addField != addFieldMirror {
		t.Fatal("expected path enter to move to mirror")
	}

	m = next.(tuiModel)
	m.engine = &mockEngine{}
	next, cmd := m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(tuiModel).mode != viewList || cmd == nil {
		t.Fatal("expected mirror enter to execute add")
	}
	if _, ok := cmd().(addDoneMsg); !ok {
		t.Fatalf("expected addDoneMsg, got %T", cmd())
	}

	m = tuiModel{mode: viewAdd, addField: addFieldURL, addURL: "abc"}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if next.(tuiModel).addURL != "ab" {
		t.Fatal("expected URL backspace")
	}

	m = tuiModel{mode: viewAdd, addField: addFieldPath, addPath: "xyz"}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if next.(tuiModel).addPath != "xy" {
		t.Fatal("expected path backspace")
	}

	m = tuiModel{mode: viewAdd, addField: addFieldMirror, addMirror: false}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: ' '})
	if !next.(tuiModel).addMirror {
		t.Fatal("expected space to toggle mirror")
	}

	m = tuiModel{mode: viewAdd, addField: addFieldURL}
	next, _ = m.handleAddKey(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if next.(tuiModel).addURL != "h" {
		t.Fatal("expected typing append to URL")
	}
}

func TestEditRepairResetDeleteAddDoneHandlers(t *testing.T) {
	t.Parallel()

	t.Run("handleEditReady valid", func(t *testing.T) {
		t.Parallel()
		m := tuiModel{}
		_, cmd := m.handleEditReady(editReadyMsg{editorCmd: exec.Command("true")})
		if cmd == nil {
			t.Fatal("expected non-nil exec cmd")
		}
	})

	t.Run("handleEditDone variants", func(t *testing.T) {
		t.Parallel()
		eng := &mockEngine{statusResult: &model.StatusReport{}}
		m := tuiModel{mode: viewDetail, engine: eng}

		next, _ := m.handleEditDone(editDoneMsg{repoID: "acme/a", err: errors.New("boom")})
		nm := next.(tuiModel)
		if !nm.statusIsError || !strings.Contains(nm.statusMsg, "edit error") {
			t.Fatal("expected edit error status")
		}

		next, _ = m.handleEditDone(editDoneMsg{repoID: "acme/a", saved: false})
		nm = next.(tuiModel)
		if nm.statusMsg != "no changes" || nm.statusIsError {
			t.Fatal("expected no changes status")
		}

		next, cmd := m.handleEditDone(editDoneMsg{repoID: "acme/a", saved: true})
		nm = next.(tuiModel)
		if nm.statusMsg != "updated acme/a" || cmd == nil {
			t.Fatal("expected updated status and refresh cmd")
		}
		if _, ok := cmd().(statusReportMsg); !ok {
			t.Fatalf("expected statusReportMsg, got %T", cmd())
		}
	})

	t.Run("handleRepairDone variants", func(t *testing.T) {
		t.Parallel()
		eng := &mockEngine{statusResult: &model.StatusReport{}}
		m := tuiModel{engine: eng, repairRepoID: "acme/a", repairTargetUpstream: "origin/main"}

		next, _ := m.handleRepairDone(repairDoneMsg{err: errors.New("bad")})
		nm := next.(tuiModel)
		if !nm.statusIsError || !strings.Contains(nm.statusMsg, "repair error") {
			t.Fatal("expected repair error status")
		}

		next, cmd := m.handleRepairDone(repairDoneMsg{result: engine.RepairUpstreamResult{RepoID: "acme/a", Action: "repaired"}})
		nm = next.(tuiModel)
		if nm.statusMsg != "repaired: acme/a" || cmd == nil {
			t.Fatal("expected repaired status and refresh cmd")
		}

		next, cmd = m.handleRepairDone(repairDoneMsg{result: engine.RepairUpstreamResult{RepoID: "acme/a", Action: "unchanged"}})
		nm = next.(tuiModel)
		if nm.statusMsg != "unchanged: acme/a" || cmd != nil {
			t.Fatal("expected non-repaired status without cmd")
		}
	})

	t.Run("handleResetDone variants", func(t *testing.T) {
		t.Parallel()
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "acme/a", Path: "/tmp/a", Status: registry.StatusPresent}}}
		eng := &mockEngine{reg: reg, statusResult: &model.StatusReport{}}
		m := tuiModel{engine: eng}

		next, _ := m.handleResetDone(resetDoneMsg{repoID: "acme/a", err: errors.New("bad")})
		nm := next.(tuiModel)
		if !nm.statusIsError || !strings.Contains(nm.statusMsg, "reset error") {
			t.Fatal("expected reset error")
		}

		next, cmd := m.handleResetDone(resetDoneMsg{repoID: "acme/a"})
		nm = next.(tuiModel)
		if nm.statusMsg != "reset: acme/a" || cmd == nil || nm.pendingInspections != 1 {
			t.Fatal("expected success status and stream refresh cmd")
		}
	})

	t.Run("handleDeleteDone variants", func(t *testing.T) {
		t.Parallel()
		m := tuiModel{
			repos:         []model.RepoStatus{{RepoID: "acme/a"}, {RepoID: "acme/b"}},
			filteredRepos: []model.RepoStatus{{RepoID: "acme/a"}},
			filterText:    "acme/a",
			cursor:        1,
		}

		next, _ := m.handleDeleteDone(deleteDoneMsg{repoID: "acme/a", err: errors.New("bad")})
		nm := next.(tuiModel)
		if !nm.statusIsError || !strings.Contains(nm.statusMsg, "delete error") {
			t.Fatal("expected delete error")
		}

		next, _ = m.handleDeleteDone(deleteDoneMsg{repoID: "acme/a"})
		nm = next.(tuiModel)
		if nm.statusIsError || nm.statusMsg != "deleted: acme/a" || len(nm.repos) != 1 {
			t.Fatal("expected delete success and repo removal")
		}
	})

	t.Run("handleAddDone variants", func(t *testing.T) {
		t.Parallel()
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "acme/a", Path: "/tmp/a", Status: registry.StatusPresent}}}
		eng := &mockEngine{reg: reg, statusResult: &model.StatusReport{}}
		m := tuiModel{engine: eng, addURL: "u", addPath: "p", addMirror: true, addField: addFieldMirror}

		next, _ := m.handleAddDone(addDoneMsg{repoID: "acme/a", err: errors.New("bad")})
		nm := next.(tuiModel)
		if !nm.statusIsError || !strings.Contains(nm.statusMsg, "add error") || nm.addURL != "" {
			t.Fatal("expected add error and field reset")
		}

		next, cmd := m.handleAddDone(addDoneMsg{repoID: "acme/a"})
		nm = next.(tuiModel)
		if nm.statusMsg != "added: acme/a" || cmd == nil || nm.pendingInspections != 1 {
			t.Fatal("expected add success and refresh cmd")
		}
	})
}

func TestHandleSyncProgressAndStarts(t *testing.T) {
	t.Parallel()

	m := tuiModel{}
	next, _ := m.handleSyncProgress(syncProgressMsg{result: engine.SyncResult{RepoID: "acme/a"}, started: true})
	nm := next.(tuiModel)
	if !nm.syncProgress["acme/a"].Planned {
		t.Fatal("expected started progress to mark Planned")
	}

	repos := []model.RepoStatus{{RepoID: "acme/a", Path: "/tmp/a", PrimaryRemote: "origin", Head: model.Head{Branch: "main"}}}
	reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "acme/a", Path: "/tmp/a", Branch: "main", Status: registry.StatusPresent}}}
	eng := &mockEngine{reg: reg, cfg: &config.Config{Defaults: config.Defaults{MainBranch: "main"}}}

	next, _ = (tuiModel{repos: repos, cursor: 0}).startReset()
	if next.(tuiModel).mode != viewResetConfirm {
		t.Fatal("expected reset confirm mode")
	}

	next, _ = (tuiModel{repos: repos, cursor: 0}).startDelete()
	if next.(tuiModel).mode != viewDeleteConfirm {
		t.Fatal("expected delete confirm mode")
	}

	next, _ = (tuiModel{addURL: "x", addPath: "y", addMirror: true, addField: addFieldMirror}).startAdd()
	nm = next.(tuiModel)
	if nm.mode != viewAdd || nm.addURL != "" || nm.addPath != "" || nm.addMirror || nm.addField != addFieldURL {
		t.Fatal("expected add reset")
	}

	next, _ = (tuiModel{repos: repos, cursor: 0, engine: eng}).startRepair()
	nm = next.(tuiModel)
	if nm.mode != viewRepairConfirm || nm.repairTargetUpstream != "origin/main" {
		t.Fatal("expected repair target resolution")
	}
}

func TestHelpersAndPathResolution(t *testing.T) {
	t.Parallel()

	if got := repoType(model.RepoStatus{}); got != "checkout" {
		t.Fatalf("expected checkout, got %q", got)
	}
	if got := repoType(model.RepoStatus{Type: "mirror"}); got != "mirror" {
		t.Fatalf("expected mirror, got %q", got)
	}

	if got := deltaOrDash(model.RepoStatus{}); got != "-" {
		t.Fatalf("expected -, got %q", got)
	}
	ahead := 2
	if got := deltaOrDash(model.RepoStatus{Tracking: model.Tracking{Ahead: &ahead}}); got != "+2" {
		t.Fatalf("expected +2, got %q", got)
	}

	if got := dirtyDisplay(model.RepoStatus{Bare: true}); !strings.Contains(got, "bare") {
		t.Fatalf("expected bare display, got %q", got)
	}
	if got := dirtyDisplay(model.RepoStatus{Worktree: nil}); got != "-" {
		t.Fatalf("expected -, got %q", got)
	}
	if got := dirtyDisplay(model.RepoStatus{Worktree: &model.Worktree{Dirty: true, Staged: 1, Unstaged: 2, Untracked: 3}}); !strings.Contains(got, "staged:1") {
		t.Fatalf("expected dirty counts, got %q", got)
	}
	if got := dirtyDisplay(model.RepoStatus{Worktree: &model.Worktree{Dirty: false}}); got != "clean" {
		t.Fatalf("expected clean, got %q", got)
	}

	if got := errorDisplay(model.RepoStatus{}); got != "-" {
		t.Fatalf("expected -, got %q", got)
	}
	if got := errorDisplay(model.RepoStatus{ErrorClass: "network", Error: "timeout"}); got != "[network] timeout" {
		t.Fatalf("expected classed error, got %q", got)
	}
	if got := errorDisplay(model.RepoStatus{Error: "timeout"}); got != "timeout" {
		t.Fatalf("expected plain error, got %q", got)
	}

	if got := upstreamDisplay(model.RepoStatus{}); got != "-" {
		t.Fatalf("expected -, got %q", got)
	}
	if got := upstreamDisplay(model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/main"}}); got != "origin/main" {
		t.Fatalf("expected upstream, got %q", got)
	}

	if got := resolvedAddPath(tuiModel{addPath: "/tmp/x"}); got != "/tmp/x" {
		t.Fatalf("expected explicit path, got %q", got)
	}
	if got := resolvedAddPath(tuiModel{addURL: "https://github.com/acme/tool.git"}); filepath.Base(got) != "tool" {
		t.Fatalf("expected default path suffix /tool, got %q", got)
	}
	if got := resolvedAddPath(tuiModel{}); got != "" {
		t.Fatalf("expected empty resolved path, got %q", got)
	}

	if got := defaultClonePath("https://github.com/acme/repo.git"); got != "repo" {
		t.Fatalf("expected repo, got %q", got)
	}
	if got := repoNameFromURL("git@github.com:acme/repo.git"); got != "repo" {
		t.Fatalf("expected repo from ssh url, got %q", got)
	}
	if got := repoNameFromURL("ssh://git@example.com/acme/repo.git"); got != "repo" {
		t.Fatalf("expected repo from ssh scheme, got %q", got)
	}
	if got := repoNameFromURL("github.com:acme/repo"); got != "repo" {
		t.Fatalf("expected repo with colon handling, got %q", got)
	}

	m := tuiModel{addField: addFieldURL}
	m = m.addFieldAppend("abc")
	if m.addURL != "abc" {
		t.Fatal("expected URL append")
	}
	m.addField = addFieldPath
	m = m.addFieldAppend("/tmp")
	if m.addPath != "/tmp" {
		t.Fatal("expected path append")
	}

	repos := []model.RepoStatus{{RepoID: "a"}, {RepoID: "b"}}
	out := removeRepoByID(repos, "a")
	if len(out) != 1 || out[0].RepoID != "b" {
		t.Fatalf("unexpected removeRepoByID result: %#v", out)
	}
}

func TestResolveRepairTarget(t *testing.T) {
	t.Parallel()

	eng := &mockEngine{}
	_, _, err := resolveRepairTarget(tuiModel{engine: eng})
	if err == nil || !strings.Contains(err.Error(), "no repo selected") {
		t.Fatal("expected no repo selected error")
	}

	repos := []model.RepoStatus{{RepoID: "acme/a", PrimaryRemote: "origin", Head: model.Head{Branch: "main"}}}
	_, _, err = resolveRepairTarget(tuiModel{engine: eng, repos: repos})
	if err == nil || !strings.Contains(err.Error(), "registry not available") {
		t.Fatal("expected registry not available error")
	}

	eng.reg = &registry.Registry{Entries: []registry.Entry{{RepoID: "acme/a", Path: "/tmp/a", Status: registry.StatusPresent}}}
	_, _, err = resolveRepairTarget(tuiModel{engine: eng, repos: []model.RepoStatus{{RepoID: "acme/a", PrimaryRemote: "origin", Head: model.Head{Detached: true}}}})
	if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
		t.Fatal("expected detached HEAD error")
	}

	_, _, err = resolveRepairTarget(tuiModel{engine: eng, repos: []model.RepoStatus{{RepoID: "acme/a", Head: model.Head{Branch: "main"}}}})
	if err == nil || !strings.Contains(err.Error(), "no remote configured") {
		t.Fatal("expected no remote configured error")
	}

	eng.cfg = &config.Config{Defaults: config.Defaults{MainBranch: "main"}}
	repoID, target, err := resolveRepairTarget(tuiModel{engine: eng, repos: repos})
	if err != nil || repoID != "acme/a" || target != "origin/main" {
		t.Fatalf("unexpected success: repoID=%q target=%q err=%v", repoID, target, err)
	}
}

func TestModalHelpers(t *testing.T) {
	t.Parallel()

	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyLeft}, {Code: 'h'}} {
		left, right := isModalNav(key)
		if !left || right {
			t.Fatalf("expected left nav for %q", key.String())
		}
	}
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyRight}, {Code: 'l'}, {Code: tea.KeyTab}} {
		left, right := isModalNav(key)
		if left || !right {
			t.Fatalf("expected right nav for %q", key.String())
		}
	}
	for _, key := range []tea.KeyPressMsg{{Code: 'j'}, {Code: 'k'}} {
		left, right := isModalNav(key)
		if left || right {
			t.Fatalf("expected j/k to produce no modal nav, got left=%v right=%v for %q", left, right, key.String())
		}
	}

	m := tuiModel{modalCursor: 0}
	m = modalMoveLeft(m, 2)
	if m.modalCursor != 0 {
		t.Fatal("expected left clamp at 0")
	}
	m = modalMoveRight(m, 2)
	if m.modalCursor != 1 {
		t.Fatal("expected right move")
	}
	m = modalMoveRight(m, 2)
	if m.modalCursor != 1 {
		t.Fatal("expected right clamp at max")
	}

	a := renderModalButtons([]string{"Cancel", "Confirm"}, 0)
	b := renderModalButtons([]string{"Cancel", "Confirm"}, 1)
	if !strings.Contains(a, "Cancel") || !strings.Contains(b, "Confirm") || a == b {
		t.Fatal("expected modal button rendering differences by selection")
	}
}

func TestViewsAndRendering(t *testing.T) {
	t.Parallel()

	baseRepo := model.RepoStatus{
		RepoID:        "acme/backend",
		Path:          "/work/backend",
		Type:          "checkout",
		PrimaryRemote: "origin",
		Head:          model.Head{Branch: "main"},
		Tracking:      model.Tracking{Status: model.TrackingEqual, Upstream: "origin/main"},
		Worktree:      &model.Worktree{Dirty: true, Staged: 1, Unstaged: 2, Untracked: 3},
		Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:acme/backend.git"}},
		Labels:        map[string]string{"team": "platform"},
		Annotations:   map[string]string{"owner": "devx"},
		LastSync:      &model.SyncResult{OK: false, At: time.Now().Add(-2 * time.Hour), Error: "fetch failed"},
	}

	detail := renderDetailView(tuiModel{repos: []model.RepoStatus{baseRepo}, cursor: 0, width: 100})
	for _, want := range []string{"Repository: acme/backend", "Path:", "/work/backend", "Remotes", "Labels", "Annotations", "Last Sync", "Error:"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail view missing %q", want)
		}
	}

	addURL := renderAddView(tuiModel{mode: viewAdd, width: 100, addField: addFieldURL})
	if !strings.Contains(addURL, "Add Repository") || !strings.Contains(addURL, "URL:") {
		t.Fatal("expected add view URL state")
	}
	addPath := renderAddView(tuiModel{mode: viewAdd, width: 100, addField: addFieldPath, addURL: "https://github.com/acme/backend.git"})
	if !strings.Contains(addPath, "Checkout location") {
		t.Fatal("expected add view path state")
	}
	addMirror := renderAddView(tuiModel{mode: viewAdd, width: 100, addField: addFieldMirror, addMirror: true})
	if !strings.Contains(addMirror, "Mirror clone") || !strings.Contains(addMirror, "space to toggle") {
		t.Fatal("expected add view mirror state")
	}

	deleteView := renderDeleteConfirmView(tuiModel{width: 100, deleteRepoID: "acme/backend", deleteRepoPath: "/work/backend"})
	if !strings.Contains(deleteView, "acme/backend") || !strings.Contains(deleteView, "/work/backend") {
		t.Fatal("expected delete confirm details")
	}

	resetView := renderResetConfirmView(tuiModel{width: 100, resetRepoID: "acme/backend"})
	if !strings.Contains(resetView, "acme/backend") || !strings.Contains(resetView, "WARNING") {
		t.Fatal("expected reset warning")
	}

	repairView := renderRepairConfirmView(tuiModel{width: 100, repairRepoID: "acme/backend", repairTargetUpstream: "origin/main"})
	if !strings.Contains(repairView, "origin/main") {
		t.Fatal("expected repair target")
	}
	repairNoTarget := renderRepairConfirmView(tuiModel{width: 100, repairTargetUpstream: ""})
	if !strings.Contains(repairNoTarget, "Cannot determine target upstream") {
		t.Fatal("expected no-target repair message")
	}

	plan := renderSyncPlanView(tuiModel{width: 100, syncPlan: []engine.SyncResult{{RepoID: "acme/backend", Action: "git fetch", Planned: true}}})
	if !strings.Contains(plan, "Sync Plan") || !strings.Contains(plan, "git fetch") {
		t.Fatal("expected sync plan rows")
	}
	planEmpty := renderSyncPlanView(tuiModel{width: 100})
	if !strings.Contains(planEmpty, "No actions planned") {
		t.Fatal("expected empty sync plan message")
	}

	progress := renderSyncProgressView(tuiModel{
		width:        100,
		syncPlan:     []engine.SyncResult{{RepoID: "acme/backend"}, {RepoID: "acme/cli"}},
		syncProgress: map[string]engine.SyncResult{"acme/backend": {RepoID: "acme/backend", Planned: true}, "acme/cli": {RepoID: "acme/cli", OK: true, Outcome: engine.SyncOutcomeFetched}},
	})
	if !strings.Contains(progress, "running") || !strings.Contains(progress, "fetched") {
		t.Fatal("expected progress state rendering")
	}
	progressDoneErr := renderSyncProgressView(tuiModel{
		width:        100,
		syncDone:     true,
		syncErr:      errors.New("boom"),
		syncPlan:     []engine.SyncResult{{RepoID: "acme/backend"}},
		syncProgress: map[string]engine.SyncResult{"acme/backend": {RepoID: "acme/backend", OK: false, Error: "failed"}},
	})
	if !strings.Contains(progressDoneErr, "Sync complete") || !strings.Contains(progressDoneErr, "Error:") {
		t.Fatal("expected completed progress rendering with errors")
	}

	list := renderListView(tuiModel{width: 100, height: 20, repos: []model.RepoStatus{baseRepo}})
	if !strings.Contains(list, "acme/backend") {
		t.Fatal("expected list row rendering")
	}

	if renderListView(tuiModel{width: 0}) != "" {
		t.Fatal("expected empty list with width=0")
	}
	if !strings.Contains(renderListView(tuiModel{width: 100, height: 20}), "No repositories found") {
		t.Fatal("expected empty repo message")
	}
	if !strings.Contains(renderListView(tuiModel{width: 100, height: 20, loading: true}), "loading") {
		t.Fatal("expected loading list title")
	}
	if !strings.Contains(renderListView(tuiModel{width: 100, height: 20, err: errors.New("boom")}), "Error:") {
		t.Fatal("expected list error row")
	}
	if !strings.Contains(renderListView(tuiModel{width: 100, height: 20, repos: []model.RepoStatus{baseRepo}, filterText: "zzz", filteredRepos: nil}), "No matches") {
		t.Fatal("expected no matches row")
	}
	if !strings.Contains(renderListView(tuiModel{width: 100, height: 20, statusMsg: "ok"}), "ok") {
		t.Fatal("expected status message row")
	}
}

func TestViewDispatchAndOtherHelpers(t *testing.T) {
	t.Parallel()

	m := tuiModel{width: 100, height: 20}
	v := m.View()
	if !v.AltScreen {
		t.Fatal("expected alt screen")
	}

	if !strings.Contains((tuiModel{mode: viewList, width: 100, height: 20}).renderCurrentView(), "repokeeper") {
		t.Fatal("expected list dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewDetail, width: 100, repos: []model.RepoStatus{{RepoID: "a"}}}).renderCurrentView(), "Repository:") {
		t.Fatal("expected detail dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewRepairConfirm, width: 100}).renderCurrentView(), "Repair Upstream") {
		t.Fatal("expected repair dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewResetConfirm, width: 100}).renderCurrentView(), "Reset Repository") {
		t.Fatal("expected reset dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewDeleteConfirm, width: 100}).renderCurrentView(), "Delete Repository") {
		t.Fatal("expected delete dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewAdd, width: 100}).renderCurrentView(), "Add Repository") {
		t.Fatal("expected add dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewSyncPlan, width: 100}).renderCurrentView(), "Sync Plan") {
		t.Fatal("expected sync plan dispatch")
	}
	if !strings.Contains((tuiModel{mode: viewProgress, width: 100}).renderCurrentView(), "Syncing") {
		t.Fatal("expected progress dispatch")
	}

	if got := visibleRows(tuiModel{height: 2}); got != 1 {
		t.Fatalf("expected min visible rows 1, got %d", got)
	}
	if got := visibleRows(tuiModel{height: 30}); got != 26 {
		t.Fatalf("expected visible rows 26, got %d", got)
	}

	if got := relativeTime(time.Now().Add(-15 * time.Second)); got != "just now" {
		t.Fatalf("expected just now, got %q", got)
	}
	if got := relativeTime(time.Now().Add(-3 * time.Minute)); !strings.Contains(got, "m ago") {
		t.Fatalf("expected minute display, got %q", got)
	}
	if got := relativeTime(time.Now().Add(-2 * time.Hour)); !strings.Contains(got, "h ago") {
		t.Fatalf("expected hour display, got %q", got)
	}
	if got := relativeTime(time.Now().Add(-48 * time.Hour)); !strings.Contains(got, "d ago") {
		t.Fatalf("expected day display, got %q", got)
	}

	eng := &mockEngine{statusResult: &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "a"}}}}
	if cmd := streamStatusCmd(context.Background(), eng, []registry.Entry{{RepoID: "a", Path: "/tmp/a"}}); cmd == nil {
		t.Fatal("expected streamStatusCmd non-nil")
	}
	cmd := loadStatusCmd(context.Background(), eng)
	if cmd == nil {
		t.Fatal("expected loadStatusCmd non-nil")
	}
	if _, ok := cmd().(statusReportMsg); !ok {
		t.Fatalf("expected statusReportMsg, got %T", cmd())
	}

	refreshWithEntries := refreshStatusCmd(context.Background(), &mockEngine{reg: &registry.Registry{Entries: []registry.Entry{{RepoID: "a", Path: "/tmp/a"}}}})
	if refreshWithEntries == nil {
		t.Fatal("expected refresh cmd with registry entries")
	}
	refreshWithoutEntries := refreshStatusCmd(context.Background(), &mockEngine{statusResult: &model.StatusReport{}})
	if refreshWithoutEntries == nil {
		t.Fatal("expected refresh cmd without entries")
	}
	if _, ok := refreshWithoutEntries().(statusReportMsg); !ok {
		t.Fatalf("expected statusReportMsg, got %T", refreshWithoutEntries())
	}
}
