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
	s.registerResources()
	return s
}

// Inner returns the underlying mcp-go server for transport binding.
func (s *MCPServer) Inner() *server.MCPServer { return s.inner }

func (s *MCPServer) registerTools() {
	s.inner.AddTools(
		// Phase 1: core read tools
		server.ServerTool{Tool: listRepositoriesTool(), Handler: s.handleListRepositories},
		server.ServerTool{Tool: getRepositoryContextTool(), Handler: s.handleGetRepositoryContext},
		server.ServerTool{Tool: getWorkspaceConfigTool(), Handler: s.handleGetWorkspaceConfig},
		// Phase 2: full read surface
		server.ServerTool{Tool: buildWorkspaceInventoryTool(), Handler: s.handleBuildWorkspaceInventory},
		server.ServerTool{Tool: selectRepositoriesTool(), Handler: s.handleSelectRepositories},
		server.ServerTool{Tool: getRepoMetadataTool(), Handler: s.handleGetRepoMetadata},
		server.ServerTool{Tool: getAuthoritativePathsTool(), Handler: s.handleGetAuthoritativePaths},
		server.ServerTool{Tool: getRelatedRepositoriesTool(), Handler: s.handleGetRelatedRepositories},
		// Phase 3: mutation tools
		server.ServerTool{Tool: scanWorkspaceTool(), Handler: s.handleScanWorkspace},
		server.ServerTool{Tool: planSyncTool(), Handler: s.handlePlanSync},
		server.ServerTool{Tool: executeSyncTool(), Handler: s.handleExecuteSync},
		server.ServerTool{Tool: setLabelsTool(), Handler: s.handleSetLabels},
		server.ServerTool{Tool: addRepositoryTool(), Handler: s.handleAddRepository},
		server.ServerTool{Tool: removeRepositoryTool(), Handler: s.handleRemoveRepository},
	)
}

func (s *MCPServer) registerResources() {
	s.inner.AddResource(configResource(), s.handleConfigResource)
	s.inner.AddResource(registryResource(), s.handleRegistryResource)
	s.inner.AddResourceTemplate(repoTemplate(), s.handleRepoResource)
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

func buildWorkspaceInventoryTool() mcp.Tool {
	return mcp.NewTool("build_workspace_inventory",
		mcp.WithDescription("Full live health check across all repos. Runs VCS inspect on each registered repository. Slower than list_repositories but returns current git state."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("filter",
			mcp.Description("Health filter: all, errors, dirty, clean, gone, diverged, missing (default: all)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label filter (e.g. team=platform,role=service)"),
		),
		mcp.WithNumber("concurrency",
			mcp.Description("Max parallel inspections (default from config)"),
		),
	)
}

func selectRepositoriesTool() mcp.Tool {
	return mcp.NewTool("select_repositories",
		mcp.WithDescription("Query repos by combining label selectors, field selectors, and free-text name matching. Returns matched repo IDs and paths without full status detail."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("label_selector",
			mcp.Description("Label filter (e.g. team=platform,env=prod)"),
		),
		mcp.WithString("field_selector",
			mcp.Description("Field filter (e.g. tracking.status=behind, worktree.dirty=true)"),
		),
		mcp.WithString("name_match",
			mcp.Description("Substring match on repo_id"),
		),
	)
}

func getRepoMetadataTool() mcp.Tool {
	return mcp.NewTool("get_repo_metadata",
		mcp.WithDescription("Source-controlled repo-local metadata only. Returns null if no metadata file exists in the repository."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
	)
}

func getAuthoritativePathsTool() mcp.Tool {
	return mcp.NewTool("get_authoritative_paths",
		mcp.WithDescription("Returns the authoritative and low-value path hints for a repo. Quick way to know where to look first and what to avoid."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
	)
}

func getRelatedRepositoriesTool() mcp.Tool {
	return mcp.NewTool("get_related_repositories",
		mcp.WithDescription("Given a repo, returns its declared related repos with relationship types and cross-reference to registry for local paths."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
	)
}

// --- Phase 3: Mutation tool definitions ---

func scanWorkspaceTool() mcp.Tool {
	return mcp.NewTool("scan_workspace",
		mcp.WithDescription("Discover repos under configured or specified roots and update the registry."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(false),
		}),
		mcp.WithArray("roots",
			mcp.Description("Override scan roots (default: config roots)"),
		),
		mcp.WithBoolean("prune_stale",
			mcp.Description("Remove entries marked missing beyond staleness threshold"),
		),
	)
}

func planSyncTool() mcp.Tool {
	return mcp.NewTool("plan_sync",
		mcp.WithDescription("Preview what a sync would do without executing. Always dry-run — never mutates."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(true),
		}),
		mcp.WithString("filter",
			mcp.Description("Health filter: all, errors, dirty, clean, gone, diverged, missing"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label filter"),
		),
		mcp.WithBoolean("update_local",
			mcp.Description("Include local rebase in plan"),
		),
		mcp.WithBoolean("push_local",
			mcp.Description("Include push in plan"),
		),
	)
}

func executeSyncTool() mcp.Tool {
	return mcp.NewTool("execute_sync",
		mcp.WithDescription("Execute a sync operation. Requires explicit confirm=true as a safety gate."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
		}),
		mcp.WithString("filter",
			mcp.Description("Health filter: all, errors, dirty, clean, gone, diverged, missing"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label filter"),
		),
		mcp.WithBoolean("update_local",
			mcp.Description("Enable local rebase"),
		),
		mcp.WithBoolean("push_local",
			mcp.Description("Enable push"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force operations on diverged repos"),
		),
		mcp.WithBoolean("confirm",
			mcp.Description("Must be true to execute. Safety gate."),
			mcp.Required(),
		),
	)
}

func setLabelsTool() mcp.Tool {
	return mcp.NewTool("set_labels",
		mcp.WithDescription("Set or remove machine-local labels on a repository."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(false),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
		mcp.WithObject("set",
			mcp.Description("Labels to set (key-value map)"),
		),
		mcp.WithArray("remove",
			mcp.Description("Label keys to remove"),
		),
	)
}

func addRepositoryTool() mcp.Tool {
	return mcp.NewTool("add_repository",
		mcp.WithDescription("Clone a repository and register it in the workspace."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint: boolPtr(false),
		}),
		mcp.WithString("url",
			mcp.Description("Remote URL to clone"),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("Local target path"),
			mcp.Required(),
		),
		mcp.WithBoolean("mirror",
			mcp.Description("Clone as bare mirror"),
		),
	)
}

func removeRepositoryTool() mcp.Tool {
	return mcp.NewTool("remove_repository",
		mcp.WithDescription("Remove a repository from the registry. Default is tracking-only removal (does not delete files)."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
		}),
		mcp.WithString("repo",
			mcp.Description("Repository identifier (repo_id or absolute path)"),
			mcp.Required(),
		),
		mcp.WithBoolean("delete_files",
			mcp.Description("Also delete files on disk (default: false)"),
		),
	)
}

func boolPtr(v bool) *bool { return &v }
