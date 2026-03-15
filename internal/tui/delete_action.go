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

var deleteOptions = []string{"Cancel", "Unregister only", "Delete from disk"}

func renderDeleteConfirmView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Delete Repository"))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  Repo:  %s\n", m.deleteRepoID))
	b.WriteString(fmt.Sprintf("  Path:  %s\n", m.deleteRepoPath))
	b.WriteByte('\n')
	b.WriteString("  Unregister only — removes from repokeeper, files stay on disk\n")
	b.WriteString("  Delete from disk — unregisters AND permanently deletes the directory\n")
	b.WriteByte('\n')
	b.WriteString("  " + renderModalButtons(deleteOptions, m.modalCursor))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("←/→ or h/l: select  enter: confirm  esc: cancel"))
	return b.String()
}
