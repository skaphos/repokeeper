// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/selector"
)

// listRepoEntry is the JSON shape returned by list_repositories for each entry.
type listRepoEntry struct {
	RepoID      string            `json:"repo_id"`
	CheckoutID  string            `json:"checkout_id,omitempty"`
	Path        string            `json:"path"`
	RemoteURL   string            `json:"remote_url,omitempty"`
	Type        string            `json:"type,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Status      string            `json:"status"`
	LastSeen    string            `json:"last_seen"`
}

func (s *MCPServer) handleListRepositories(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reg := s.engine.Registry()
	if reg == nil {
		return mcp.NewToolResultError("registry not loaded (run scan first)"), nil
	}

	labelSelectorRaw := req.GetString("label_selector", "")
	statusFilter := strings.ToLower(strings.TrimSpace(req.GetString("status", "")))

	var labelReqs []selector.LabelRequirement
	if labelSelectorRaw != "" {
		var err error
		labelReqs, err = selector.ParseLabelSelector(labelSelectorRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	entries := make([]listRepoEntry, 0, len(reg.Entries))
	for _, e := range reg.Entries {
		if !matchesStatusFilter(e.Status, statusFilter) {
			continue
		}
		if !selector.LabelsMatchSelector(e.Labels, labelReqs) {
			continue
		}
		entries = append(entries, listRepoEntry{
			RepoID:      e.RepoID,
			CheckoutID:  e.CheckoutID,
			Path:        e.Path,
			RemoteURL:   e.RemoteURL,
			Type:        e.Type,
			Labels:      e.Labels,
			Annotations: e.Annotations,
			Status:      string(e.Status),
			LastSeen:    e.LastSeen.Format("2006-01-02T15:04:05Z"),
		})
	}

	return mcp.NewToolResultJSON(entries)
}

func matchesStatusFilter(entryStatus registry.EntryStatus, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.EqualFold(string(entryStatus), filter)
}
