// SPDX-License-Identifier: MIT
package mcpserver

import (
	"fmt"
	"path/filepath"

	"github.com/skaphos/repokeeper/internal/registry"
)

// resolveRepo resolves a repo identifier (repo_id or absolute path) to a
// registry entry. It searches by repo_id first, then falls back to path match.
func resolveRepo(reg *registry.Registry, repo string) (*registry.Entry, error) {
	if reg == nil {
		return nil, fmt.Errorf("registry not loaded")
	}

	// Try repo_id match first.
	if entry := reg.FindByRepoID(repo); entry != nil {
		return entry, nil
	}

	// Try absolute path match.
	if filepath.IsAbs(repo) {
		for i := range reg.Entries {
			if reg.Entries[i].Path == repo {
				return &reg.Entries[i], nil
			}
		}
	}

	return nil, fmt.Errorf("repository %q not found in registry", repo)
}
