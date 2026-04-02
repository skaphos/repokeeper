// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/internal/config"
)

// workspaceConfigResponse is the JSON shape for get_workspace_config.
type workspaceConfigResponse struct {
	ConfigPath   string         `json:"config_path"`
	Exclude      []string       `json:"exclude,omitempty"`
	IgnoredPaths []string       `json:"ignored_paths,omitempty"`
	RegistryPath string         `json:"registry_path,omitempty"`
	Defaults     configDefaults `json:"defaults"`
	RepoCount    int            `json:"repo_count"`
}

type configDefaults struct {
	RemoteName     string `json:"remote_name"`
	MainBranch     string `json:"main_branch"`
	Concurrency    int    `json:"concurrency"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func (s *MCPServer) handleGetWorkspaceConfig(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := s.engine.Config()
	if cfg == nil {
		return mcp.NewToolResultError("config not loaded"), nil
	}

	repoCount := 0
	if reg := s.engine.Registry(); reg != nil {
		repoCount = len(reg.Entries)
	}

	resp := workspaceConfigResponse{
		ConfigPath:   s.cfgPath,
		Exclude:      cfg.Exclude,
		IgnoredPaths: cfg.IgnoredPaths,
		RegistryPath: cfg.RegistryPath,
		Defaults: configDefaults{
			RemoteName:     cfgDefault(cfg.Defaults.RemoteName, config.DefaultConfig().Defaults.RemoteName),
			MainBranch:     cfgDefault(cfg.Defaults.MainBranch, config.DefaultConfig().Defaults.MainBranch),
			Concurrency:    intDefault(cfg.Defaults.Concurrency, config.DefaultConfig().Defaults.Concurrency),
			TimeoutSeconds: intDefault(cfg.Defaults.TimeoutSeconds, config.DefaultConfig().Defaults.TimeoutSeconds),
		},
		RepoCount: repoCount,
	}

	return mcp.NewToolResultJSON(resp)
}

func cfgDefault(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func intDefault(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}
