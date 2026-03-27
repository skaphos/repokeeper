// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
)

type syncPlanMsg struct {
	plan []engine.SyncResult
	err  error
}

type syncProgressMsg struct {
	result  engine.SyncResult
	started bool
}

type syncDoneMsg struct {
	results []engine.SyncResult
	err     error
}

func buildSyncPlanCmd(ctx context.Context, eng EngineAPI, repoIDs map[string]bool) tea.Cmd {
	return func() tea.Msg {
		filter := engine.FilterAll
		plan, err := eng.Sync(ctx, engine.SyncOptions{
			Filter:          filter,
			DryRun:          true,
			ContinueOnError: true,
		})
		if err != nil {
			return syncPlanMsg{err: err}
		}
		if len(repoIDs) > 0 {
			filtered := plan[:0]
			for _, r := range plan {
				if repoIDs[r.RepoID] {
					filtered = append(filtered, r)
				}
			}
			plan = filtered
		}
		return syncPlanMsg{plan: plan}
	}
}

func renderSyncPlanView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sync Plan"))
	b.WriteByte('\n')
	b.WriteString(renderDivider([]int{m.width - 1}))
	b.WriteByte('\n')

	if len(m.syncPlan) == 0 {
		b.WriteString(loadingStyle.Render("No actions planned."))
		b.WriteByte('\n')
	} else {
		for _, item := range m.syncPlan {
			icon := "  "
			if item.Planned {
				icon = "→ "
			}
			line := fmt.Sprintf("%s%-30s  %s", icon, truncPad(item.RepoID, 30), item.Action)
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	b.WriteByte('\n')
	b.WriteString("  " + renderModalButtons([]string{"Cancel", "Confirm sync"}, m.modalCursor))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("←/→ or h/l: select  enter: confirm  esc: cancel"))
	return b.String()
}

func renderSyncProgressView(m tuiModel) string {
	var b strings.Builder
	repoCounts := duplicateSyncRepoCounts(m.syncPlan)

	done := 0
	total := len(m.syncPlan)
	for _, r := range m.syncProgress {
		if !r.Planned {
			done++
		}
	}

	title := fmt.Sprintf("Syncing — %d/%d", done, total)
	if m.syncDone {
		title = fmt.Sprintf("Sync complete — %d repos", total)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	b.WriteString(renderDivider([]int{m.width - 1}))
	b.WriteByte('\n')

	for _, item := range m.syncPlan {
		prog, ok := m.syncProgress[syncResultIdentityKey(item)]
		var statusText string
		if !ok {
			statusText = loadingStyle.Render("waiting…")
		} else if prog.Planned {
			statusText = loadingStyle.Render("running…")
		} else if prog.OK {
			statusText = statusEqualStyle.Render("✓ " + string(prog.Outcome))
		} else {
			statusText = errorTextStyle.Render("✗ " + prog.Error)
		}
		line := fmt.Sprintf("  %-30s  %s", truncPad(syncResultDisplayName(item, repoCounts), 30), statusText)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	if m.syncErr != nil {
		b.WriteString(errorTextStyle.Render(fmt.Sprintf("Error: %s", m.syncErr)))
		b.WriteByte('\n')
	}

	if m.syncDone {
		b.WriteString(statusBarStyle.Render("any key: return to list"))
	} else {
		b.WriteString(statusBarStyle.Render("syncing…"))
	}
	return b.String()
}
