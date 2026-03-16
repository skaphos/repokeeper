// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skaphos/repokeeper/internal/model"
)

type columnDef struct {
	title    string
	minWidth int
	flex     int
	value    func(model.RepoStatus) string
	styled   func(model.RepoStatus) string
}

func defaultColumns() []columnDef {
	return []columnDef{
		{title: "REPO", minWidth: 10, flex: 4, value: colValueRepo, styled: colStyledRepo},
		{title: "BRANCH", minWidth: 8, flex: 2, value: colValueBranch, styled: colStyledBranch},
		{title: "STATUS", minWidth: 8, flex: 1, value: colValueStatus, styled: colStyledStatus},
		{title: "±", minWidth: 5, flex: 1, value: colValueDelta, styled: colStyledPlain},
		{title: "DIRTY", minWidth: 5, flex: 1, value: colValueDirty, styled: colStyledDirty},
		{title: "ERROR", minWidth: 5, flex: 1, value: colValueError, styled: colStyledError},
		{title: "SYNCED", minWidth: 6, flex: 1, value: colValueSynced, styled: colStyledSynced},
	}
}

func colValueRepo(r model.RepoStatus) string {
	return r.RepoID
}

func colValueBranch(r model.RepoStatus) string {
	if r.Type == "mirror" {
		return "-"
	}
	if r.Head.Detached {
		return "detached"
	}
	return r.Head.Branch
}

func colValueStatus(r model.RepoStatus) string {
	switch r.Tracking.Status {
	case model.TrackingEqual:
		return "up to date"
	case model.TrackingAhead:
		return "ahead"
	case model.TrackingBehind:
		return "behind"
	case model.TrackingDiverged:
		return "diverged"
	case model.TrackingGone:
		return "gone"
	case model.TrackingNone:
		return "-"
	default:
		return string(r.Tracking.Status)
	}
}

func colValueDelta(r model.RepoStatus) string {
	t := r.Tracking
	ahead := t.Ahead != nil && *t.Ahead > 0
	behind := t.Behind != nil && *t.Behind > 0
	switch {
	case ahead && behind:
		return fmt.Sprintf("+%d/-%d", *t.Ahead, *t.Behind)
	case ahead:
		return fmt.Sprintf("+%d", *t.Ahead)
	case behind:
		return fmt.Sprintf("-%d", *t.Behind)
	default:
		return ""
	}
}

func colValueDirty(r model.RepoStatus) string {
	if r.Bare {
		return "-"
	}
	if r.Worktree == nil {
		return "-"
	}
	if r.Worktree.Dirty {
		return "✗"
	}
	return ""
}

func colValueError(r model.RepoStatus) string {
	return r.ErrorClass
}

func colValueSynced(r model.RepoStatus) string {
	if r.LastSync == nil {
		return ""
	}
	return relativeTime(r.LastSync.At)
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func colStyledRepo(r model.RepoStatus) string {
	return colValueRepo(r)
}

func colStyledBranch(r model.RepoStatus) string {
	return colValueBranch(r)
}

func colStyledStatus(r model.RepoStatus) string {
	v := colValueStatus(r)
	switch r.Tracking.Status {
	case model.TrackingEqual:
		return statusEqualStyle.Render(v)
	case model.TrackingAhead:
		return statusAheadStyle.Render(v)
	case model.TrackingBehind:
		return statusBehindStyle.Render(v)
	case model.TrackingDiverged:
		return statusDivergedStyle.Render(v)
	case model.TrackingGone:
		return statusGoneStyle.Render(v)
	default:
		return statusNoneStyle.Render(v)
	}
}

func colStyledPlain(r model.RepoStatus) string {
	return colValueDelta(r)
}

func colStyledDirty(r model.RepoStatus) string {
	v := colValueDirty(r)
	if v == "✗" {
		return dirtyStyle.Render(v)
	}
	return v
}

func colStyledError(r model.RepoStatus) string {
	v := colValueError(r)
	if v != "" {
		return errorClassStyle.Render(v)
	}
	return v
}

func colStyledSynced(r model.RepoStatus) string {
	if r.LastSync == nil {
		return ""
	}
	v := colValueSynced(r)
	if r.LastSync.OK {
		return syncedOKStyle.Render(v)
	}
	return syncedFailStyle.Render(v)
}

func distributeWidths(cols []columnDef, totalWidth int) []int {
	n := len(cols)
	if n == 0 {
		return nil
	}
	overhead := (n - 1) * 3
	available := totalWidth - overhead
	if available < 0 {
		available = 0
	}

	widths := make([]int, n)
	used := 0
	for i, col := range cols {
		widths[i] = col.minWidth
		used += col.minWidth
	}

	remaining := available - used
	if remaining > 0 {
		totalFlex := 0
		for _, col := range cols {
			totalFlex += col.flex
		}
		if totalFlex > 0 {
			for i, col := range cols {
				extra := (remaining * col.flex) / totalFlex
				widths[i] += extra
			}
		}
	}

	return widths
}

func renderHeader(cols []columnDef, widths []int) string {
	cells := make([]string, len(cols))
	for i, col := range cols {
		cells[i] = truncPad(col.title, widths[i])
	}
	return strings.Join(cells, " │ ")
}

func renderDivider(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		if w < 0 {
			w = 0
		}
		parts[i] = strings.Repeat("─", w)
	}
	return strings.Join(parts, "─┼─")
}

func renderRow(cols []columnDef, widths []int, repo model.RepoStatus) string {
	cells := make([]string, len(cols))
	for i, col := range cols {
		cells[i] = truncPad(col.value(repo), widths[i])
	}
	return strings.Join(cells, " │ ")
}

func renderStyledRow(cols []columnDef, widths []int, repo model.RepoStatus) string {
	cells := make([]string, len(cols))
	for i, col := range cols {
		plain := truncPad(col.value(repo), widths[i])
		styled := col.styled(repo)
		if utf8.RuneCountInString(stripAnsi(styled)) == utf8.RuneCountInString(plain) {
			styledRunes := utf8.RuneCountInString(stripAnsi(styled))
			padNeeded := widths[i] - styledRunes
			if padNeeded > 0 {
				cells[i] = styled + strings.Repeat(" ", padNeeded)
			} else {
				cells[i] = styled
			}
		} else {
			cells[i] = plain
		}
	}
	return strings.Join(cells, " │ ")
}

func truncPad(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		if width <= 1 {
			return strings.Repeat(" ", width)
		}
		return string(runes[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
