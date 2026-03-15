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
	widths := distributeWidths(cols, m.width-1)
	list := m.visibleList()

	var b strings.Builder

	count := len(m.repos)
	title := fmt.Sprintf("repokeeper — %d repo", count)
	if count != 1 {
		title += "s"
	}
	if m.loading {
		title += "  " + loadingStyle.Render("loading…")
	}
	if m.filterText != "" {
		title += fmt.Sprintf("  [%d match", len(list))
		if len(list) != 1 {
			title += "es"
		}
		title += "]"
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
	} else if len(list) == 0 && !m.loading {
		if m.filterText != "" {
			b.WriteString(loadingStyle.Render(fmt.Sprintf("No matches for %q", m.filterText)))
		} else {
			b.WriteString(loadingStyle.Render("No repositories found. Run `repokeeper scan` first."))
		}
		b.WriteByte('\n')
	} else {
		visible := visibleRows(m)
		start := m.offset
		end := start + visible
		if end > len(list) {
			end = len(list)
		}
		for i := start; i < end; i++ {
			row := renderStyledRow(cols, widths, list[i])
			sel := " "
			if m.selected[list[i].RepoID] {
				sel = "●"
			}
			if i == m.cursor {
				row = cursorStyle.Render(sel + row)
			} else {
				row = sel + row
			}
			b.WriteString(row)
			b.WriteByte('\n')
		}
	}

	if m.filterMode {
		b.WriteString(statusBarStyle.Render(fmt.Sprintf("Filter: %s█", m.filterText)))
	} else if m.statusMsg != "" {
		style := statusBarStyle
		if m.statusIsError {
			style = errorTextStyle
		}
		b.WriteString(style.Render(m.statusMsg))
	} else {
		selCount := len(m.selected)
		helpKeys := "↑↓/jk: nav  space: select  a: all  s: sync  e: edit  r: repair  /: filter  f5: refresh  q: quit"
		if m.filterText != "" {
			helpKeys = "↑↓/jk: nav  space: select  a: all  s: sync  e: edit  r: repair  /: filter  esc: clear  f5: refresh  q: quit"
		}
		if selCount > 0 {
			helpKeys = fmt.Sprintf("%d selected  |  ", selCount) + helpKeys
		}
		b.WriteString(statusBarStyle.Render(helpKeys))
	}

	return b.String()
}

func visibleRows(m tuiModel) int {
	v := m.height - 4
	if v < 1 {
		v = 1
	}
	return v
}
