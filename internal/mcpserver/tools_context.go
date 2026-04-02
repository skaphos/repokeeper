// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/internal/model"
)

// repoContextResponse is the JSON shape for get_repository_context.
type repoContextResponse struct {
	RepoID      string            `json:"repo_id"`
	CheckoutID  string            `json:"checkout_id,omitempty"`
	Path        string            `json:"path"`
	Type        string            `json:"type,omitempty"`
	Bare        bool              `json:"bare"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Remotes       []model.Remote   `json:"remotes,omitempty"`
	PrimaryRemote string           `json:"primary_remote,omitempty"`
	Head          model.Head       `json:"head"`
	Worktree      *model.Worktree  `json:"worktree,omitempty"`
	Tracking      model.Tracking   `json:"tracking"`
	Submodules    model.Submodules `json:"submodules"`

	RepoMetadata *model.RepoMetadata `json:"repo_metadata,omitempty"`

	Error      string `json:"error,omitempty"`
	ErrorClass string `json:"error_class,omitempty"`
}

func (s *MCPServer) handleGetRepositoryContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoArg, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: repo"), nil
	}

	reg := s.engine.Registry()
	entry, err := resolveRepo(reg, repoArg)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := s.engine.InspectRepo(ctx, entry.Path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp := repoContextResponse{
		RepoID:        status.RepoID,
		CheckoutID:    status.CheckoutID,
		Path:          status.Path,
		Type:          status.Type,
		Bare:          status.Bare,
		Labels:        mergeLabels(entry.Labels, status.Labels),
		Annotations:   entry.Annotations,
		Remotes:       status.Remotes,
		PrimaryRemote: status.PrimaryRemote,
		Head:          status.Head,
		Worktree:      status.Worktree,
		Tracking:      status.Tracking,
		Submodules:    status.Submodules,
		RepoMetadata:  status.RepoMetadata,
		Error:         status.Error,
		ErrorClass:    status.ErrorClass,
	}

	return mcp.NewToolResultJSON(resp)
}

// mergeLabels returns a combined map of registry labels and status labels,
// with registry labels taking precedence.
func mergeLabels(registryLabels, statusLabels map[string]string) map[string]string {
	if len(registryLabels) == 0 && len(statusLabels) == 0 {
		return nil
	}
	merged := make(map[string]string, len(registryLabels)+len(statusLabels))
	for k, v := range statusLabels {
		merged[k] = v
	}
	for k, v := range registryLabels {
		merged[k] = v
	}
	return merged
}
