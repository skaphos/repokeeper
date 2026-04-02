// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	resourceURIConfig   = "repokeeper://config"
	resourceURIRegistry = "repokeeper://registry"
	// Single template that covers both repo entry and repo metadata URIs.
	// Dispatch is handled internally by checking for a /metadata suffix.
	resourceURIRepoTpl = "repokeeper://repo/{+repo_id}"
)

// --- resource definitions ---

func configResource() mcp.Resource {
	return mcp.NewResource(resourceURIConfig, "Workspace Configuration",
		mcp.WithResourceDescription("Current RepoKeeper workspace configuration"),
		mcp.WithMIMEType("application/json"),
	)
}

func registryResource() mcp.Resource {
	return mcp.NewResource(resourceURIRegistry, "Registry Snapshot",
		mcp.WithResourceDescription("Full registry snapshot of all tracked repositories"),
		mcp.WithMIMEType("application/json"),
	)
}

func repoTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(resourceURIRepoTpl, "Repository Entry or Metadata",
		mcp.WithTemplateDescription("Single registry entry by repo_id (append /metadata for repo-local metadata)"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// --- resource handlers ---

func (s *MCPServer) handleConfigResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	cfg := s.engine.Config()
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: resourceURIConfig, MIMEType: "application/json", Text: string(data)},
	}, nil
}

func (s *MCPServer) handleRegistryResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	reg := s.engine.Registry()
	if reg == nil {
		return nil, fmt.Errorf("registry not loaded")
	}
	data, err := json.Marshal(reg)
	if err != nil {
		return nil, fmt.Errorf("marshaling registry: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: resourceURIRegistry, MIMEType: "application/json", Text: string(data)},
	}, nil
}

// handleRepoResource dispatches to either the registry entry handler or the
// metadata handler based on whether the URI ends with /metadata.
func (s *MCPServer) handleRepoResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	repoID := extractRepoID(req.Params.URI)
	if repoID == "" {
		return nil, fmt.Errorf("could not determine repo_id from URI %q", req.Params.URI)
	}

	if strings.HasSuffix(repoID, "/metadata") {
		actualID := strings.TrimSuffix(repoID, "/metadata")
		return s.serveRepoMetadata(ctx, req.Params.URI, actualID)
	}

	return s.serveRepoEntry(req.Params.URI, repoID)
}

func (s *MCPServer) serveRepoEntry(uri, repoID string) ([]mcp.ResourceContents, error) {
	reg := s.engine.Registry()
	entry, err := resolveRepo(reg, repoID)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshaling entry: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(data)},
	}, nil
}

func (s *MCPServer) serveRepoMetadata(ctx context.Context, uri, repoID string) ([]mcp.ResourceContents, error) {
	reg := s.engine.Registry()
	entry, err := resolveRepo(reg, repoID)
	if err != nil {
		return nil, err
	}

	status, err := s.engine.InspectRepo(ctx, entry.Path)
	if err != nil {
		return nil, err
	}

	if status.RepoMetadata == nil {
		return nil, fmt.Errorf("no metadata found for repository %q", repoID)
	}

	data, err := json.Marshal(status.RepoMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(data)},
	}, nil
}

// extractRepoID parses a repo_id from a resource URI by stripping the
// repokeeper://repo/ prefix.
func extractRepoID(uri string) string {
	const prefix = "repokeeper://repo/"
	if !strings.HasPrefix(uri, prefix) {
		return ""
	}
	return strings.TrimPrefix(uri, prefix)
}
