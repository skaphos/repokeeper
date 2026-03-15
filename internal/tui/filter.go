// SPDX-License-Identifier: MIT
package tui

import (
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

// filterRows returns the subset of repos that match the query string.
// Matching is case-insensitive and checks: RepoID, Path, Head.Branch,
// Tracking.Status, ErrorClass, and all label values.
func filterRows(repos []model.RepoStatus, query string) []model.RepoStatus {
	if query == "" {
		return repos
	}
	q := strings.ToLower(query)
	out := make([]model.RepoStatus, 0, len(repos))
	for _, r := range repos {
		if matchesFilter(r, q) {
			out = append(out, r)
		}
	}
	return out
}

// matchesFilter returns true when repo matches the lower-cased query.
func matchesFilter(r model.RepoStatus, q string) bool {
	if strings.Contains(strings.ToLower(r.RepoID), q) {
		return true
	}
	if strings.Contains(strings.ToLower(r.Path), q) {
		return true
	}
	if strings.Contains(strings.ToLower(r.Head.Branch), q) {
		return true
	}
	if strings.Contains(strings.ToLower(string(r.Tracking.Status)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(colValueStatus(r)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(r.ErrorClass), q) {
		return true
	}
	for _, v := range r.Labels {
		if strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	return false
}
