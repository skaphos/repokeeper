// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type deleteDoneMsg struct {
	repoID string
	err    error
}

func deleteRepoCmd(eng EngineAPI, repoID, cfgPath string, deleteFiles bool) tea.Cmd {
	return func() tea.Msg {
		err := eng.DeleteRepo(context.Background(), repoID, cfgPath, deleteFiles)
		return deleteDoneMsg{repoID: repoID, err: err}
	}
}

func renderDeleteConfirmView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Delete Repository"))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  Repo: %s\n", m.deleteRepoID))
	b.WriteString(fmt.Sprintf("  Path: %s\n", m.deleteRepoPath))
	b.WriteByte('\n')
	b.WriteString("  What would you like to do?\n")
	b.WriteByte('\n')
	b.WriteString("  [u] Unregister only  — remove from repokeeper registry, keep files on disk\n")
	b.WriteString("  [d] Delete files too — unregister AND permanently delete from disk\n")
	b.WriteString("  [n] Cancel (default)\n")
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("u: unregister  d: delete from disk  n/esc: cancel (default)"))
	return b.String()
}
