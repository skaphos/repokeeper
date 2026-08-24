// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import "path/filepath"

func mutationMCPToolCases() []MCPToolCase {
	return []MCPToolCase{
		{Name: "scan_workspace", ReadOnly: false, Arguments: func(workspace *MaterializedWorkspace) map[string]any {
			return map[string]any{"roots": []string{workspace.WorkspaceRoot}, "prune_stale": false}
		}, Assert: requireFields("discovered", "new", "missing", "pruned", "repos")},
		{Name: "set_labels", ReadOnly: false, Arguments: func(workspace *MaterializedWorkspace) map[string]any {
			return map[string]any{"repo": workspace.Repositories["clean metadata repository"].CheckoutPath, "set": map[string]any{"e2e": "true"}}
		}, Assert: requireFields("repo_id", "labels")},
		{Name: "add_repository", ReadOnly: false, Arguments: func(workspace *MaterializedWorkspace) map[string]any {
			return map[string]any{"url": workspace.Repositories["clean metadata repository"].RemoteURL, "path": addedRepositoryPath(workspace), "mirror": false}
		}, Assert: requireFields("repo_id", "path", "status")},
	}
}

func addedRepositoryPath(workspace *MaterializedWorkspace) string {
	return filepath.Join(workspace.WorkspaceRoot, "added repo")
}
