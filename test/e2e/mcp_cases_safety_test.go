// SPDX-License-Identifier: MIT
//go:build integration

package e2e

func safetyMCPToolCases() []MCPToolCase {
	return []MCPToolCase{
		{Name: "execute_sync", ReadOnly: false, SafetyCase: true, Arguments: func(*MaterializedWorkspace) map[string]any {
			return map[string]any{"filter": "all", "update_local": false, "push_local": false, "confirm": true}
		}, Assert: requireList("results")},
		{Name: "remove_repository", ReadOnly: false, SafetyCase: true, Arguments: func(workspace *MaterializedWorkspace) map[string]any {
			return map[string]any{"repo": addedRepositoryPath(workspace), "delete_files": true, "confirm": true}
		}, Assert: requireFields("repo_id", "removed")},
	}
}
