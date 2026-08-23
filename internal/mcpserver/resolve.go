// SPDX-License-Identifier: MIT
package mcpserver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/skaphos/repokeeper/internal/registry"
)

// resolveRepo resolves a repo identifier (absolute path, checkout_id, or
// repo_id) to a registry entry.
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

	// An exact path identifies one checkout. Normalize both sides so
	// equivalent-but-non-canonical forms (e.g. a trailing slash, or "." /
	// ".." segments) still match the stored path.
	if filepath.IsAbs(repo) {
		want := filepath.Clean(repo)
		var pathMatches []int
		for i := range reg.Entries {
			if filepath.Clean(reg.Entries[i].Path) == want {
				pathMatches = append(pathMatches, i)
			}
		}
		if len(pathMatches) == 1 {
			return &reg.Entries[pathMatches[0]], nil
		}
		if len(pathMatches) > 1 {
			return nil, fmt.Errorf("repository path %q is ambiguous: found %d registry entries", repo, len(pathMatches))
		}
	}

	// checkout_id is the next-most-specific selector. Reject collisions rather
	// than picking an arbitrary repository when two entries share an ID.
	var checkoutMatches []int
	for i := range reg.Entries {
		if reg.Entries[i].CheckoutID == repo {
			checkoutMatches = append(checkoutMatches, i)
		}
	}
	if len(checkoutMatches) == 1 {
		return &reg.Entries[checkoutMatches[0]], nil
	}
	if len(checkoutMatches) > 1 {
		return nil, fmt.Errorf("checkout_id %q is ambiguous: found %d repositories; pass the checkout's absolute path instead", repo, len(checkoutMatches))
	}

	// Finally treat the identifier as a repo_id, which may map to several
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
		selectors := make([]string, 0, len(matches))
		for _, match := range matches {
			entry := reg.Entries[match]
			selectors = append(selectors, fmt.Sprintf("checkout_id=%q or path=%q", entry.CheckoutID, entry.Path))
		}
		return nil, fmt.Errorf("repo %q is ambiguous: found %d local checkouts; use %s", repo, len(matches), strings.Join(selectors, "; "))
	}
}
