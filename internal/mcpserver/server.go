// SPDX-License-Identifier: MIT
package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/skaphos/repokeeper/internal/obs"
)

// MCPServer wraps a mark3labs MCP server with access to the RepoKeeper engine.
type MCPServer struct {
	engine  EngineAPI
	cfgPath string
	logger  obs.Logger
	inner   *server.MCPServer
}

// New creates and configures an MCPServer with all tools and resources registered.
func New(eng EngineAPI, cfgPath, version string, logger obs.Logger) *MCPServer {
	if logger == nil {
		logger = obs.NopLogger()
	}

	s := &MCPServer{
		engine:  eng,
		cfgPath: cfgPath,
		logger:  logger,
		inner: server.NewMCPServer(
			"repokeeper",
			version,
			server.WithToolCapabilities(true),
			server.WithResourceCapabilities(true, false),
			server.WithRecovery(),
		),
	}

	s.registerTools()
	return s
}

// Inner returns the underlying mcp-go server for transport binding.
func (s *MCPServer) Inner() *server.MCPServer { return s.inner }

func (s *MCPServer) registerTools() {
	s.inner.AddTools(
		server.ServerTool{Tool: listRepositoriesTool(), Handler: s.handleListRepositories},
		server.ServerTool{Tool: getRepositoryContextTool(), Handler: s.handleGetRepositoryContext},
		server.ServerTool{Tool: getWorkspaceConfigTool(), Handler: s.handleGetWorkspaceConfig},
	)
}

// --- Tool definitions ---

func listRepositoriesTool() mcp.Tool {
	return mcp.NewTool("list_repositories",
		mcp.WithDescription("List all tracked repositories with summary info. Fast — reads registry only, no live git inspection."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("label_selector",
			mcp.Description("Label filter (e.g. team=platform,role=service)"),
		),
		mcp.WithString("status",
			mcp.Description("Registry status filter: present, missing, moved"),
		),
	)
}

func getRepositoryContextTool() mcp.Tool {
	return mcp.NewTool("get_repository_context",
		mcp.WithDescription("Deep context for a single repository — git state, labels, annotations, metadata, entrypoints, related repos. The 'tell me everything about this repo' tool."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
	)
}

func getWorkspaceConfigTool() mcp.Tool {
	return mcp.NewTool("get_workspace_config",
		mcp.WithDescription("Returns current RepoKeeper workspace configuration including roots, exclude patterns, defaults, and registry path."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
	)
}

func boolPtr(v bool) *bool { return &v }
