// SPDX-License-Identifier: MIT
package tui

import tea "charm.land/bubbletea/v2"

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
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m tuiModel) handleStatusReport(msg statusReportMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	if msg.report != nil {
		m.repos = msg.report.Repos
	}
	m.cursor = 0
	m.offset = 0
	return m, nil
}
