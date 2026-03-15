// SPDX-License-Identifier: MIT
package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func renderModalButtons(options []string, selected int) string {
	var b strings.Builder
	for i, opt := range options {
		if i > 0 {
			b.WriteString("   ")
		}
		label := "  " + opt + "  "
		if i == selected {
			b.WriteString(cursorStyle.Render(label))
		} else {
			b.WriteString(label)
		}
	}
	return b.String()
}

func modalMoveLeft(m tuiModel, n int) tuiModel {
	if m.modalCursor > 0 {
		m.modalCursor--
	}
	return m
}

func modalMoveRight(m tuiModel, n int) tuiModel {
	if m.modalCursor < n-1 {
		m.modalCursor++
	}
	return m
}

func isModalNav(msg tea.KeyPressMsg) (left, right bool) {
	switch msg.String() {
	case "left", "h", "k":
		return true, false
	case "right", "l", "j", "tab":
		return false, true
	}
	return false, false
}
