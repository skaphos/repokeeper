// SPDX-License-Identifier: MIT
package tui

import tea "charm.land/bubbletea/v2"

func (m tuiModel) View() tea.View {
	content := m.renderCurrentView()
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m tuiModel) renderCurrentView() string {
	switch m.mode {
	case viewList:
		return renderListView(m)
	default:
		return renderListView(m)
	}
}
