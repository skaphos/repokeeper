// SPDX-License-Identifier: MIT
package mcpserver

import (
	"fmt"
	"path/filepath"

	"github.com/skaphos/repokeeper/internal/registry"
)

// resolveRepo resolves a repo identifier (repo_id or absolute path) to a
// registry entry.
//
// An absolute path selects a single checkout unambiguously, so it is matched
// first. A repo_id may resolve to multiple local checkouts; rather than
// silently acting on an arbitrary one (which would let set_labels/get_* mutate
// or report on the wrong checkout), an ambiguous repo_id is rejected with a
// clear error, mirroring engine/actions.go's DeleteRepo behavior. Callers with
// more than one checkout must disambiguate with the checkout's absolute path.
func resolveRepo(reg *registry.Registry, repo string) (*registry.Entry, error) {
	if reg == nil {
		return nil, fmt.Errorf("registry not loaded")
	}

	// An absolute path identifies exactly one checkout. Normalize both sides
	// with filepath.Clean so equivalent-but-non-canonical forms (e.g. a
	// trailing slash, or "." / ".." segments) still match the stored path.
	if filepath.IsAbs(repo) {
		want := filepath.Clean(repo)
		for i := range reg.Entries {
			if filepath.Clean(reg.Entries[i].Path) == want {
				return &reg.Entries[i], nil
			}
		}
		return nil, fmt.Errorf("repository %q not found in registry", repo)
	}

	// Otherwise treat the identifier as a repo_id, which may map to several
	// checkouts. Collect all matches so ambiguity can be detected.
	var matches []int
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == repo {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("repository %q not found in registry", repo)
	case 1:
		return &reg.Entries[matches[0]], nil
	default:
		return nil, fmt.Errorf("repo %q is ambiguous: found %d local checkouts; pass the checkout's absolute path instead", repo, len(matches))
	}
}
