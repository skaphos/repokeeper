// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func readMCPToolCases() []MCPToolCase {
	clean := func(workspace *MaterializedWorkspace) string {
		return workspace.Repositories["clean metadata repository"].CheckoutPath
	}
	return []MCPToolCase{
		{Name: "list_repositories", ReadOnly: true, Arguments: emptyMCPArguments, Assert: requireList("repositories")},
		{Name: "get_repository_context", ReadOnly: true, Arguments: func(workspace *MaterializedWorkspace) map[string]any { return map[string]any{"repo": clean(workspace)} }, Assert: requireFields("repo_id", "path", "bare", "head", "tracking", "submodules")},
		{Name: "get_workspace_config", ReadOnly: true, Arguments: emptyMCPArguments, Assert: requireFields("config_path", "registry_stale_days", "defaults", "repo_count")},
		{Name: "build_workspace_inventory", ReadOnly: true, Arguments: func(*MaterializedWorkspace) map[string]any { return map[string]any{"filter": "all", "concurrency": 1} }, Assert: requireFields("generated_at", "repos")},
		{Name: "select_repositories", ReadOnly: true, Arguments: func(*MaterializedWorkspace) map[string]any { return map[string]any{"name_match": "clean"} }, Assert: requireList("repositories")},
		{Name: "get_repo_metadata", ReadOnly: true, Arguments: func(workspace *MaterializedWorkspace) map[string]any { return map[string]any{"repo": clean(workspace)} }, Assert: requireFields("apiVersion", "kind", "name", "entrypoints", "paths", "related_repos")},
		{Name: "get_authoritative_paths", ReadOnly: true, Arguments: func(workspace *MaterializedWorkspace) map[string]any { return map[string]any{"repo": clean(workspace)} }, Assert: requireFields("authoritative", "low_value", "entrypoints")},
		{Name: "get_related_repositories", ReadOnly: true, Arguments: func(workspace *MaterializedWorkspace) map[string]any { return map[string]any{"repo": clean(workspace)} }, Assert: requireList("repositories")},
		{Name: "plan_sync", ReadOnly: true, Arguments: func(*MaterializedWorkspace) map[string]any {
			return map[string]any{"filter": "all", "update_local": false, "push_local": false}
		}, Assert: requireList("plan")},
	}
}

func emptyMCPArguments(*MaterializedWorkspace) map[string]any { return map[string]any{} }

func requireFields(fields ...string) func(*mcp.CallToolResult) error {
	return func(result *mcp.CallToolResult) error {
		if err := requireToolSuccess(result); err != nil {
			return err
		}
		return requireObjectFields(result, fields...)
	}
}

func requireList(field string) func(*mcp.CallToolResult) error {
	return func(result *mcp.CallToolResult) error {
		if err := requireToolSuccess(result); err != nil {
			return err
		}
		items, err := requireListField(result, field)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return fmt.Errorf("structuredContent field %q is empty", field)
		}
		return nil
	}
}
