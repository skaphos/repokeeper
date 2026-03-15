// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case statusReportMsg:
		return m.handleStatusReport(msg)

	case syncPlanMsg:
		return m.handleSyncPlan(msg)

	case syncProgressMsg:
		return m.handleSyncProgress(msg)

	case syncDoneMsg:
		return m.handleSyncDone(msg)

	case repoStatusMsg:
		return m.handleRepoStatus(msg)

	case editReadyMsg:
		return m.handleEditReady(msg)

	case editDoneMsg:
		return m.handleEditDone(msg)

	case repairDoneMsg:
		return m.handleRepairDone(msg)

	case resetDoneMsg:
		return m.handleResetDone(msg)

	case deleteDoneMsg:
		return m.handleDeleteDone(msg)

	case addDoneMsg:
		return m.handleAddDone(msg)
	}
	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.mode {
	case viewSyncPlan:
		return m.handleSyncPlanKey(msg)
	case viewProgress:
		return m.handleSyncProgressKey(msg)
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewRepairConfirm:
		return m.handleRepairConfirmKey(msg)
	case viewResetConfirm:
		return m.handleResetConfirmKey(msg)
	case viewDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	case viewAdd:
		return m.handleAddKey(msg)
	default:
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleListKey(msg)
	}
}

func (m tuiModel) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "backspace":
		m.mode = viewList
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "j", "down":
		return m.moveCursor(1), nil

	case "k", "up":
		return m.moveCursor(-1), nil

	case "/":
		m.filterMode = true
		return m, nil

	case "f5":
		m.loading = true
		reg := m.engine.Registry()
		if reg != nil {
			m.pendingInspections = len(reg.Entries)
		}
		return m, refreshStatusCmd(m.engine)

	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.filteredRepos = nil
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	case "space":
		return m.toggleSelection(), nil

	case "a":
		return m.toggleSelectAll(), nil

	case "s":
		return m.startSync()

	case "enter":
		list := m.visibleList()
		if len(list) > 0 && m.cursor < len(list) {
			m.mode = viewDetail
		}
		return m, nil

	case "e":
		return m, prepareEditCmd(m)

	case "r":
		return m.startRepair()

	case "ctrl+x":
		return m.startReset()

	case "ctrl+d":
		return m.startDelete()

	case "n":
		return m.startAdd()
	}
	return m, nil
}

func (m tuiModel) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterText = ""
		m.filteredRepos = nil
		m.cursor = 0
		m.offset = 0
		return m, nil

	case "enter":
		m.filterMode = false
		return m, nil

	case "backspace":
		if len(m.filterText) > 0 {
			runes := []rune(m.filterText)
			m.filterText = string(runes[:len(runes)-1])
			m.filteredRepos = filterRows(m.repos, m.filterText)
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	default:
		if t := msg.Text; t != "" {
			m.filterText += t
			m.filteredRepos = filterRows(m.repos, m.filterText)
			m.cursor = 0
			m.offset = 0
		}
	}
	return m, nil
}

func (m tuiModel) handleSyncPlanKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if len(m.syncPlan) == 0 {
			m.mode = viewList
			return m, nil
		}
		m.mode = viewProgress
		m.syncProgress = make(map[string]engine.SyncResult)
		m.syncDone = false
		m.syncErr = nil
		return m, executeSyncCmd(m)
	case "n", "esc":
		m.mode = viewList
		m.syncPlan = nil
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleSyncProgressKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.syncDone {
		m.mode = viewList
		m.syncPlan = nil
		m.syncResults = nil
		m.syncProgress = nil
		m.syncDone = false
		m.syncErr = nil
		m.loading = true
		return m, refreshStatusCmd(m.engine)
	}
	return m, nil
}

func (m tuiModel) moveCursor(delta int) tuiModel {
	list := m.visibleList()
	n := len(list)
	if n == 0 {
		return m
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	visible := visibleRows(m)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	return m
}

func (m tuiModel) toggleSelection() tuiModel {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m
	}
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	id := list[m.cursor].RepoID
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		m.selected[id] = true
	}
	return m
}

func (m tuiModel) toggleSelectAll() tuiModel {
	list := m.visibleList()
	if len(list) == 0 {
		return m
	}
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	allSelected := len(m.selected) == len(list)
	if allSelected {
		m.selected = make(map[string]bool)
	} else {
		for _, r := range list {
			m.selected[r.RepoID] = true
		}
	}
	return m
}

func (m tuiModel) startSync() (tea.Model, tea.Cmd) {
	m.mode = viewSyncPlan
	m.syncPlan = nil
	m.loading = true
	return m, buildSyncPlanCmd(m.engine, m.selected)
}

func executeSyncCmd(m tuiModel) tea.Cmd {
	plan := m.syncPlan
	eng := m.engine
	prog := m.program
	return func() tea.Msg {
		onStart := func(r engine.SyncResult) {
			if prog != nil {
				prog.Send(syncProgressMsg{result: r, started: true})
			}
		}
		onComplete := func(r engine.SyncResult) {
			if prog != nil {
				prog.Send(syncProgressMsg{result: r, started: false})
			}
		}
		results, err := eng.ExecuteSyncPlanWithCallbacks(
			context.Background(),
			plan,
			engine.SyncOptions{ContinueOnError: true},
			onStart,
			onComplete,
		)
		return syncDoneMsg{results: results, err: err}
	}
}

func (m tuiModel) handleStatusReport(msg statusReportMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	if msg.report != nil {
		m.repos = msg.report.Repos
		if m.filterText != "" {
			m.filteredRepos = filterRows(m.repos, m.filterText)
		}
	}
	m.cursor = 0
	m.offset = 0
	return m, nil
}

func (m tuiModel) handleSyncPlan(msg syncPlanMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.syncErr = msg.err
		m.mode = viewList
		return m, nil
	}
	m.syncPlan = msg.plan
	return m, nil
}

func (m tuiModel) handleSyncProgress(msg syncProgressMsg) (tea.Model, tea.Cmd) {
	if m.syncProgress == nil {
		m.syncProgress = make(map[string]engine.SyncResult)
	}
	r := msg.result
	if msg.started {
		r.Planned = true
	}
	m.syncProgress[r.RepoID] = r
	return m, nil
}

func (m tuiModel) handleSyncDone(msg syncDoneMsg) (tea.Model, tea.Cmd) {
	m.syncResults = msg.results
	m.syncErr = msg.err
	m.syncDone = true
	if m.syncProgress == nil {
		m.syncProgress = make(map[string]engine.SyncResult)
	}
	for _, r := range msg.results {
		m.syncProgress[r.RepoID] = r
	}
	return m, nil
}

func (m tuiModel) handleRepoStatus(msg repoStatusMsg) (tea.Model, tea.Cmd) {
	updated := false
	for i, r := range m.repos {
		if r.RepoID == msg.status.RepoID || r.Path == msg.status.Path {
			m.repos[i] = msg.status
			updated = true
			break
		}
	}
	if !updated {
		m.repos = append(m.repos, msg.status)
	}
	if m.filterText != "" {
		m.filteredRepos = filterRows(m.repos, m.filterText)
	}
	m.pendingInspections--
	if m.pendingInspections <= 0 {
		m.loading = false
	}
	return m, nil
}

func (m tuiModel) startReset() (tea.Model, tea.Cmd) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m, nil
	}
	m.mode = viewResetConfirm
	m.resetRepoID = list[m.cursor].RepoID
	return m, nil
}

func (m tuiModel) handleResetConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		repoID := m.resetRepoID
		m.mode = viewList
		m.resetRepoID = ""
		return m, resetRepoCmd(m.engine, repoID, m.cfgPath)
	default:
		m.mode = viewList
		m.resetRepoID = ""
		return m, nil
	}
}

func (m tuiModel) handleResetDone(msg resetDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "reset error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "reset: " + msg.repoID
	m.statusIsError = false
	m.loading = true
	reg := m.engine.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		m.pendingInspections = len(reg.Entries)
		return m, streamStatusCmd(m.engine, reg.Entries)
	}
	return m, loadStatusCmd(m.engine)
}

func (m tuiModel) startDelete() (tea.Model, tea.Cmd) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m, nil
	}
	m.mode = viewDeleteConfirm
	m.deleteRepoID = list[m.cursor].RepoID
	m.deleteRepoPath = list[m.cursor].Path
	return m, nil
}

func (m tuiModel) handleDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "u":
		repoID := m.deleteRepoID
		m.mode = viewList
		m.deleteRepoID = ""
		m.deleteRepoPath = ""
		m.cursor = 0
		return m, deleteRepoCmd(m.engine, repoID, m.cfgPath, false)
	case "d":
		repoID := m.deleteRepoID
		m.mode = viewList
		m.deleteRepoID = ""
		m.deleteRepoPath = ""
		m.cursor = 0
		return m, deleteRepoCmd(m.engine, repoID, m.cfgPath, true)
	default:
		m.mode = viewList
		m.deleteRepoID = ""
		m.deleteRepoPath = ""
		return m, nil
	}
}

func (m tuiModel) handleDeleteDone(msg deleteDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "delete error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "deleted: " + msg.repoID
	m.statusIsError = false
	m.loading = true
	reg := m.engine.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		m.pendingInspections = len(reg.Entries)
		return m, streamStatusCmd(m.engine, reg.Entries)
	}
	return m, loadStatusCmd(m.engine)
}

func (m tuiModel) startAdd() (tea.Model, tea.Cmd) {
	m.mode = viewAdd
	m.addURL = ""
	m.addPath = ""
	m.addMirror = false
	m.addField = addFieldURL
	return m, nil
}

func (m tuiModel) handleAddKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewList
		m.addURL = ""
		m.addPath = ""
		m.addMirror = false
		m.addField = addFieldURL
		return m, nil

	case "enter":
		switch m.addField {
		case addFieldURL:
			if strings.TrimSpace(m.addURL) == "" {
				return m, nil
			}
			m.addField = addFieldPath
		case addFieldPath:
			m.addField = addFieldMirror
		case addFieldMirror:
			url := strings.TrimSpace(m.addURL)
			path := resolvedAddPath(m)
			if url == "" || path == "" {
				m.statusMsg = "URL and path are required"
				m.statusIsError = true
				m.mode = viewList
				return m, nil
			}
			m.mode = viewList
			return m, cloneAndRegisterCmd(m.engine, url, path, m.cfgPath, m.addMirror)
		}
		return m, nil

	case "backspace":
		switch m.addField {
		case addFieldURL:
			if len(m.addURL) > 0 {
				r := []rune(m.addURL)
				m.addURL = string(r[:len(r)-1])
			}
		case addFieldPath:
			if len(m.addPath) > 0 {
				r := []rune(m.addPath)
				m.addPath = string(r[:len(r)-1])
			}
		}
		return m, nil

	case "space":
		if m.addField == addFieldMirror {
			m.addMirror = !m.addMirror
		} else if t := msg.Text; t != "" {
			m = m.addFieldAppend(t)
		}
		return m, nil

	default:
		if t := msg.Text; t != "" {
			m = m.addFieldAppend(t)
		}
	}
	return m, nil
}

func (m tuiModel) addFieldAppend(text string) tuiModel {
	switch m.addField {
	case addFieldURL:
		m.addURL += text
	case addFieldPath:
		m.addPath += text
	}
	return m
}

func (m tuiModel) handleAddDone(msg addDoneMsg) (tea.Model, tea.Cmd) {
	m.addURL = ""
	m.addPath = ""
	m.addMirror = false
	m.addField = addFieldURL
	if msg.err != nil {
		m.statusMsg = "add error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "added: " + msg.repoID
	m.statusIsError = false
	m.loading = true
	reg := m.engine.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		m.pendingInspections = len(reg.Entries)
		return m, streamStatusCmd(m.engine, reg.Entries)
	}
	return m, loadStatusCmd(m.engine)
}

func (m tuiModel) startRepair() (tea.Model, tea.Cmd) {
	repoID, target, err := resolveRepairTarget(m)
	if err != nil {
		m.statusMsg = err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewRepairConfirm
	m.repairRepoID = repoID
	m.repairTargetUpstream = target
	return m, nil
}

func (m tuiModel) handleRepairConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.mode = viewList
		return m, repairUpstreamCmd(m.engine, m.repairRepoID, m.cfgPath)
	case "n", "esc":
		m.mode = viewList
		m.repairRepoID = ""
		m.repairTargetUpstream = ""
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleEditReady(msg editReadyMsg) (tea.Model, tea.Cmd) {
	_, cmd := handleEditReady(msg)
	return m, cmd
}

func (m tuiModel) handleEditDone(msg editDoneMsg) (tea.Model, tea.Cmd) {
	m.mode = viewList
	if msg.err != nil {
		m.statusMsg = "edit error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	if !msg.saved {
		m.statusMsg = "no changes"
		m.statusIsError = false
		return m, nil
	}
	m.statusMsg = "updated " + msg.repoID
	m.statusIsError = false
	m.loading = true
	reg := m.engine.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		m.pendingInspections = len(reg.Entries)
		return m, streamStatusCmd(m.engine, reg.Entries)
	}
	return m, loadStatusCmd(m.engine)
}

func (m tuiModel) handleRepairDone(msg repairDoneMsg) (tea.Model, tea.Cmd) {
	m.repairRepoID = ""
	m.repairTargetUpstream = ""
	if msg.err != nil {
		m.statusMsg = "repair error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusIsError = false
	switch msg.result.Action {
	case "repaired":
		m.statusMsg = "repaired: " + msg.result.RepoID
		m.loading = true
		reg := m.engine.Registry()
		if reg != nil && len(reg.Entries) > 0 {
			m.pendingInspections = len(reg.Entries)
			return m, streamStatusCmd(m.engine, reg.Entries)
		}
		return m, loadStatusCmd(m.engine)
	default:
		m.statusMsg = msg.result.Action + ": " + msg.result.RepoID
	}
	return m, nil
}

func refreshStatusCmd(eng EngineAPI) tea.Cmd {
	reg := eng.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		return streamStatusCmd(eng, reg.Entries)
	}
	return func() tea.Msg {
		report, err := eng.Status(context.Background(), engine.StatusOptions{
			Filter: engine.FilterAll,
		})
		return statusReportMsg{report: report, err: err}
	}
}
