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
	}
	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.filterMode {
		return m.handleFilterKey(msg)
	}
	return m.handleListKey(msg)
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

func refreshStatusCmd(eng EngineAPI) tea.Cmd {
	return func() tea.Msg {
		report, err := eng.Status(context.Background(), engine.StatusOptions{
			Filter: engine.FilterAll,
		})
		return statusReportMsg{report: report, err: err}
	}
}
