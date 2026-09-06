// SPDX-License-Identifier: MIT

package registry

import "github.com/skaphos/repokeeper/v2/internal/urlutil"

// Redacted returns a copy of the entry with embedded credentials stripped from
// its remote URL. See model.RepoStatus.Redacted for the rationale; the same
// contract guarantee (FR-021) applies wherever a registry entry reaches an
// adapter-facing surface, which today means the MCP `registry` and per-repo
// resources.
//
// As with the model counterpart this copies rather than mutating: these entries
// are the in-memory registry that gets written back to disk, and redacting a
// persisted remote_url would break the user's ability to fetch.
func (e Entry) Redacted() Entry {
	redacted := e
	redacted.RemoteURL = urlutil.RedactCredentials(e.RemoteURL)
	return redacted
}

// Redacted returns a copy of the registry with every entry redacted. A nil
// receiver yields nil so callers can pass through a not-loaded registry
// unchanged.
func (r *Registry) Redacted() *Registry {
	if r == nil {
		return nil
	}

	redacted := &Registry{UpdatedAt: r.UpdatedAt}
	if r.Entries == nil {
		return redacted
	}
	redacted.Entries = make([]Entry, len(r.Entries))
	for i, entry := range r.Entries {
		redacted.Entries[i] = entry.Redacted()
	}
	return redacted
}
