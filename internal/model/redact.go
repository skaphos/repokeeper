// SPDX-License-Identifier: MIT

package model

import "github.com/skaphos/repokeeper/v2/internal/urlutil"

// Redacted returns a copy of the status with embedded credentials stripped from
// every remote URL.
//
// Adapter-facing output is a contract surface consumed by external IDE plugins,
// which routinely log or display payloads. A remote configured as
// https://user:token@host/repo.git would otherwise travel verbatim into a
// plugin's log. See specs/004-adapter-contract FR-021.
//
// This returns a copy rather than redacting in place: the same RepoStatus
// values feed the registry write path, and redacting a URL that is later
// persisted would corrupt the user's configuration. Callers must apply this at
// the output boundary, never upstream of a save.
//
// SSH remotes are returned unchanged -- they authenticate with keys rather than
// embedded secrets, so there is nothing in the URL to strip.
func (r RepoStatus) Redacted() RepoStatus {
	if len(r.Remotes) == 0 {
		return r
	}

	redacted := r
	redacted.Remotes = make([]Remote, len(r.Remotes))
	copy(redacted.Remotes, r.Remotes)
	for i := range redacted.Remotes {
		redacted.Remotes[i].URL = urlutil.RedactCredentials(redacted.Remotes[i].URL)
	}
	return redacted
}

// RedactedStatuses applies Redacted to a collection, returning a new slice. A
// nil input yields a nil output so callers keep control of the empty-collection
// representation the contract requires.
func RedactedStatuses(statuses []RepoStatus) []RepoStatus {
	if statuses == nil {
		return nil
	}
	out := make([]RepoStatus, len(statuses))
	for i, status := range statuses {
		out[i] = status.Redacted()
	}
	return out
}
