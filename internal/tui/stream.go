// SPDX-License-Identifier: MIT
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/repometa"
)

type repoStatusMsg struct {
	status model.RepoStatus
}

func streamStatusCmd(ctx context.Context, eng EngineAPI, entries []registry.Entry) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(entries))
	for _, entry := range entries {
		entry := entry
		cmds = append(cmds, inspectEntryCmd(ctx, eng, entry))
	}
	return tea.Batch(cmds...)
}

func inspectEntryCmd(ctx context.Context, eng EngineAPI, entry registry.Entry) tea.Cmd {
	return func() tea.Msg {
		status, err := eng.InspectRepo(ctx, entry.Path)
		if err != nil {
			partial := model.RepoStatus{
				RepoID:     entry.RepoID,
				CheckoutID: entry.CheckoutID,
				Path:       entry.Path,
				Type:       entry.Type,
				Error:      err.Error(),
				ErrorClass: "inspect",
			}
			registry.SeedRepoMetadataStatus(entry, &partial)
			repometa.Apply(&partial)
			return repoStatusMsg{status: partial}
		}
		if status.RepoID == "" {
			status.RepoID = entry.RepoID
		}
		if status.CheckoutID == "" {
			status.CheckoutID = entry.CheckoutID
		}
		if entry.Type != "" {
			status.Type = entry.Type
		}
		status.Labels = cloneRegistryMetadataMap(entry.Labels)
		status.Annotations = cloneRegistryMetadataMap(entry.Annotations)
		return repoStatusMsg{status: *status}
	}
}

func cloneRegistryMetadataMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}
