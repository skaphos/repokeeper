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

// newResourceContents renders a resource payload inside the adapter contract
// envelope.
//
// MCP resources are adapter-facing in exactly the way the tools are: they are
// advertised with MIME type application/json and any MCP client can read them.
// Before 2.0.0 they emitted raw domain objects with no version marker, which
// left three JSON surfaces outside a contract that claims to cover every one.
func newResourceContents(uri, key string, payload any) ([]mcp.ResourceContents, error) {
	data, err := json.Marshal(newEnvelope(key, payload))
	if err != nil {
		return nil, fmt.Errorf("marshaling %s resource: %w", key, err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(data)},
	}, nil
}

func (s *MCPServer) handleConfigResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	cfg := s.engine.Config()
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	return newResourceContents(resourceURIConfig, "config", cfg)
}

func (s *MCPServer) handleRegistryResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	reg := s.engine.Registry()
	if reg == nil {
		return nil, fmt.Errorf("registry not loaded")
	}
	return newResourceContents(resourceURIRegistry, "registry", reg.Redacted())
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

	return newResourceContents(uri, "repository", entry.Redacted())
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

	return newResourceContents(uri, "metadata", status.RepoMetadata)
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
