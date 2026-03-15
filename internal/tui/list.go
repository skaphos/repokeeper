// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"strings"
)

func renderListView(m tuiModel) string {
	if m.width == 0 {
		return ""
	}

	cols := defaultColumns()
	widths := distributeWidths(cols, m.width)

	var b strings.Builder

	count := len(m.repos)
	title := fmt.Sprintf("repokeeper — %d repo", count)
	if count != 1 {
		title += "s"
	}
	if m.loading {
		title += "  " + loadingStyle.Render("loading…")
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')

	b.WriteString(headerStyle.Render(renderHeader(cols, widths)))
	b.WriteByte('\n')

	b.WriteString(renderDivider(widths))
	b.WriteByte('\n')

	if m.err != nil {
		b.WriteString(errorTextStyle.Render(fmt.Sprintf("Error: %s", m.err)))
		b.WriteByte('\n')
	} else if len(m.repos) == 0 && !m.loading {
		b.WriteString(loadingStyle.Render("No repositories found. Run `repokeeper scan` first."))
		b.WriteByte('\n')
	} else {
		visible := visibleRows(m)
		start := m.offset
		end := start + visible
		if end > len(m.repos) {
			end = len(m.repos)
		}
		for i := start; i < end; i++ {
			row := renderStyledRow(cols, widths, m.repos[i])
			if i == m.cursor {
				row = cursorStyle.Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
		}
	}

	b.WriteString(statusBarStyle.Render("q: quit"))

	return b.String()
}

func visibleRows(m tuiModel) int {
	v := m.height - 4
	if v < 1 {
		v = 1
	}
	return v
}
