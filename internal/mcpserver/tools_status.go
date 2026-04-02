// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/selector"
)

// --- build_workspace_inventory ---

type inventoryResponse struct {
	GeneratedAt string               `json:"generated_at"`
	Repos       []inventoryRepoEntry `json:"repos"`
}

type inventoryRepoEntry struct {
	RepoID      string            `json:"repo_id"`
	Path        string            `json:"path"`
	Type        string            `json:"type,omitempty"`
	Bare        bool              `json:"bare"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Head     model.Head      `json:"head"`
	Worktree *model.Worktree `json:"worktree,omitempty"`
	Tracking model.Tracking  `json:"tracking"`

	RepoMetadata *model.RepoMetadata `json:"repo_metadata,omitempty"`

	Error      string `json:"error,omitempty"`
	ErrorClass string `json:"error_class,omitempty"`
}

func (s *MCPServer) handleBuildWorkspaceInventory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filterRaw := strings.ToLower(strings.TrimSpace(req.GetString("filter", "all")))
	labelSelectorRaw := req.GetString("label_selector", "")
	concurrency := req.GetInt("concurrency", 0)

	fk := engine.FilterKind(filterRaw)

	report, err := s.engine.Status(ctx, engine.StatusOptions{
		Filter:      fk,
		Concurrency: concurrency,
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var labelReqs []selector.LabelRequirement
	if labelSelectorRaw != "" {
		var parseErr error
		labelReqs, parseErr = selector.ParseLabelSelector(labelSelectorRaw)
		if parseErr != nil {
			return mcp.NewToolResultError(parseErr.Error()), nil
		}
	}

	reg := s.engine.Registry()
	repos := make([]inventoryRepoEntry, 0, len(report.Repos))
	for _, rs := range report.Repos {
		labels := enrichLabels(reg, rs.RepoID, rs.Labels)
		if !selector.LabelsMatchSelector(labels, labelReqs) {
			continue
		}

		repos = append(repos, inventoryRepoEntry{
			RepoID:       rs.RepoID,
			Path:         rs.Path,
			Type:         rs.Type,
			Bare:         rs.Bare,
			Labels:       labels,
			Annotations:  enrichAnnotations(reg, rs.RepoID),
			Head:         rs.Head,
			Worktree:     rs.Worktree,
			Tracking:     rs.Tracking,
			RepoMetadata: rs.RepoMetadata,
			Error:        rs.Error,
			ErrorClass:   rs.ErrorClass,
		})
	}

	resp := inventoryResponse{
		GeneratedAt: report.GeneratedAt.Format(time.RFC3339),
		Repos:       repos,
	}
	return mcp.NewToolResultJSON(resp)
}

// --- select_repositories ---

type selectRepoEntry struct {
	RepoID      string            `json:"repo_id"`
	Path        string            `json:"path"`
	Labels      map[string]string `json:"labels,omitempty"`
	MatchReason string            `json:"match_reason"`
}

func (s *MCPServer) handleSelectRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	labelSelectorRaw := req.GetString("label_selector", "")
	fieldSelectorRaw := req.GetString("field_selector", "")
	nameMatch := strings.TrimSpace(req.GetString("name_match", ""))

	fk := engine.FilterAll
	if fieldSelectorRaw != "" {
		var err error
		fk, err = selector.ParseFieldSelectorFilter(fieldSelectorRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	var labelReqs []selector.LabelRequirement
	if labelSelectorRaw != "" {
		var err error
		labelReqs, err = selector.ParseLabelSelector(labelSelectorRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	report, err := s.engine.Status(ctx, engine.StatusOptions{Filter: fk})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reg := s.engine.Registry()
	entries := make([]selectRepoEntry, 0, len(report.Repos))
	for _, rs := range report.Repos {
		labels := enrichLabels(reg, rs.RepoID, rs.Labels)
		if !selector.LabelsMatchSelector(labels, labelReqs) {
			continue
		}
		if nameMatch != "" && !strings.Contains(rs.RepoID, nameMatch) {
			continue
		}

		entries = append(entries, selectRepoEntry{
			RepoID:      rs.RepoID,
			Path:        rs.Path,
			Labels:      labels,
			MatchReason: buildMatchReason(fieldSelectorRaw, labelSelectorRaw, nameMatch),
		})
	}

	return mcp.NewToolResultJSON(entries)
}

// --- shared helpers ---

// enrichLabels merges status labels with registry labels (registry takes precedence).
func enrichLabels(reg *registry.Registry, repoID string, statusLabels map[string]string) map[string]string {
	var regLabels map[string]string
	if reg != nil {
		if entry := reg.FindByRepoID(repoID); entry != nil {
			regLabels = entry.Labels
		}
	}
	return mergeLabels(regLabels, statusLabels)
}

// enrichAnnotations returns annotations from the registry entry if available.
func enrichAnnotations(reg *registry.Registry, repoID string) map[string]string {
	if reg == nil {
		return nil
	}
	if entry := reg.FindByRepoID(repoID); entry != nil {
		return entry.Annotations
	}
	return nil
}

func buildMatchReason(fieldSelector, labelSelector, nameMatch string) string {
	var parts []string
	if fieldSelector != "" {
		parts = append(parts, "field:"+fieldSelector)
	}
	if labelSelector != "" {
		parts = append(parts, "label:"+labelSelector)
	}
	if nameMatch != "" {
		parts = append(parts, "name:"+nameMatch)
	}
	if len(parts) == 0 {
		return "all"
	}
	return strings.Join(parts, ",")
}
