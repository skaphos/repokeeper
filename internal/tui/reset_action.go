// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type resetDoneMsg struct {
	repoID string
	err    error
}

func resetRepoCmd(eng EngineAPI, repoID, cfgPath string) tea.Cmd {
	return func() tea.Msg {
		err := eng.ResetRepo(context.Background(), repoID, cfgPath)
		return resetDoneMsg{repoID: repoID, err: err}
	}
}

func renderResetConfirmView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Reset Repository"))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  Repo: %s\n", m.resetRepoID))
	b.WriteByte('\n')
	b.WriteString(errorTextStyle.Render("  WARNING: This will run git reset --hard HEAD and git clean -f -d."))
	b.WriteByte('\n')
	b.WriteString(errorTextStyle.Render("  All uncommitted changes and untracked files will be permanently lost."))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString("  Proceed?")
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("y: confirm reset  n/esc: cancel (default)"))
	return b.String()
}
