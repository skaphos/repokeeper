// SPDX-License-Identifier: MIT
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type repoStatusMsg struct {
	status model.RepoStatus
}

func streamStatusCmd(eng EngineAPI, entries []registry.Entry) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(entries))
	for _, entry := range entries {
		entry := entry
		cmds = append(cmds, inspectEntryCmd(eng, entry))
	}
	return tea.Batch(cmds...)
}

func inspectEntryCmd(eng EngineAPI, entry registry.Entry) tea.Cmd {
	return func() tea.Msg {
		status, err := eng.InspectRepo(context.Background(), entry.Path)
		if err != nil {
			return repoStatusMsg{status: model.RepoStatus{
				RepoID:     entry.RepoID,
				Path:       entry.Path,
				Type:       entry.Type,
				Error:      err.Error(),
				ErrorClass: "inspect",
			}}
		}
		if status.RepoID == "" {
			status.RepoID = entry.RepoID
		}
		if entry.Type != "" {
			status.Type = entry.Type
		}
		status.Labels = entry.Labels
		status.Annotations = entry.Annotations
		return repoStatusMsg{status: *status}
	}
}
