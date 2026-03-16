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

	visible := visibleRows(m)
	renderedRows := 0

	if m.err != nil {
		b.WriteString(errorTextStyle.Render(fmt.Sprintf("Error: %s", m.err)))
		b.WriteByte('\n')
		renderedRows = 1
	} else if len(list) == 0 && !m.loading {
		if m.filterText != "" {
			b.WriteString(loadingStyle.Render(fmt.Sprintf("No matches for %q", m.filterText)))
		} else {
			b.WriteString(loadingStyle.Render("No repositories found. Run `repokeeper scan` first."))
		}
		b.WriteByte('\n')
		renderedRows = 1
	} else {
		start := m.offset
		end := start + visible
		if end > len(list) {
			end = len(list)
		}
		for i := start; i < end; i++ {
			row := renderStyledRow(cols, widths, list[i])
			if i == m.cursor {
				row = cursorStyle.Render(row)
			} else if m.selected[list[i].RepoID] {
				row = selectedStyle.Render(trackingRowStyle(list[i]).Render(row))
			} else {
				row = trackingRowStyle(list[i]).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
			renderedRows++
		}
	}

	// Pad with blank lines so the footer is pinned to the bottom of the terminal.
	for range visible - renderedRows {
		b.WriteByte('\n')
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
		helpKeys := "↑↓/jk: nav  space: select  a: all  s: sync  e: edit  r: repair  ctrl+x: reset  ctrl+d: delete  n: add  /: filter  f5: refresh  q: quit"
		if m.filterText != "" {
			helpKeys = "↑↓/jk: nav  space: select  a: all  s: sync  e: edit  r: repair  ctrl+x: reset  ctrl+d: delete  n: add  /: filter  esc: clear  f5: refresh  q: quit"
		}
		selCount := len(m.selected)
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
