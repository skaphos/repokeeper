// SPDX-License-Identifier: MIT
package tui

import (
	"context"

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
	}
	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewSyncPlan:
		return m.handleSyncPlanKey(msg)
	case viewProgress:
		return m.handleSyncProgressKey(msg)
	default:
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleListKey(msg)
	}
}

func (m tuiModel) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
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
	return func() tea.Msg {
		results, err := eng.ExecuteSyncPlanWithCallbacks(
			context.Background(),
			plan,
			engine.SyncOptions{ContinueOnError: true},
			nil,
			nil,
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

func refreshStatusCmd(eng EngineAPI) tea.Cmd {
	return func() tea.Msg {
		report, err := eng.Status(context.Background(), engine.StatusOptions{
			Filter: engine.FilterAll,
		})
		return statusReportMsg{report: report, err: err}
	}
}
