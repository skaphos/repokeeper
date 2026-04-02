// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// --- get_repo_metadata ---

func (s *MCPServer) handleGetRepoMetadata(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if status.RepoMetadata == nil {
		return mcp.NewToolResultText("null"), nil
	}

	return mcp.NewToolResultJSON(status.RepoMetadata)
}

// --- get_authoritative_paths ---

type authoritativePathsResponse struct {
	Authoritative []string          `json:"authoritative"`
	LowValue      []string          `json:"low_value"`
	Entrypoints   map[string]string `json:"entrypoints"`
}

func (s *MCPServer) handleGetAuthoritativePaths(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if status.RepoMetadata == nil {
		return mcp.NewToolResultError("no repo metadata found for " + repoArg), nil
	}

	resp := authoritativePathsResponse{
		Authoritative: status.RepoMetadata.Paths.Authoritative,
		LowValue:      status.RepoMetadata.Paths.LowValue,
		Entrypoints:   status.RepoMetadata.Entrypoints,
	}
	return mcp.NewToolResultJSON(resp)
}

// --- get_related_repositories ---

type relatedRepoEntry struct {
	RepoID       string `json:"repo_id"`
	Relationship string `json:"relationship,omitempty"`
	Path         string `json:"path,omitempty"`
	Status       string `json:"status,omitempty"`
}

func (s *MCPServer) handleGetRelatedRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if status.RepoMetadata == nil || len(status.RepoMetadata.RelatedRepos) == 0 {
		return mcp.NewToolResultJSON([]relatedRepoEntry{})
	}

	entries := make([]relatedRepoEntry, 0, len(status.RepoMetadata.RelatedRepos))
	for _, rel := range status.RepoMetadata.RelatedRepos {
		re := relatedRepoEntry{
			RepoID:       rel.RepoID,
			Relationship: rel.Relationship,
		}
		if reg != nil {
			if related := reg.FindByRepoID(rel.RepoID); related != nil {
				re.Path = related.Path
				re.Status = string(related.Status)
			}
		}
		entries = append(entries, re)
	}

	return mcp.NewToolResultJSON(entries)
}
