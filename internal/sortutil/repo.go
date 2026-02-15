package sortutil

import (
	"sort"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

// LessRepoIDPath provides deterministic ordering by repository identity first,
// then by path for multi-checkout scenarios.
func LessRepoIDPath(repoIDI, pathI, repoIDJ, pathJ string) bool {
	if repoIDI == repoIDJ {
		return pathI < pathJ
	}
	return repoIDI < repoIDJ
}

// SortRepoStatuses orders status rows by RepoID, then Path.
func SortRepoStatuses(statuses []model.RepoStatus) {
	sort.SliceStable(statuses, func(i, j int) bool {
		return LessRepoIDPath(statuses[i].RepoID, statuses[i].Path, statuses[j].RepoID, statuses[j].Path)
	})
}

// SortRegistryEntries orders registry entries by RepoID, then Path.
func SortRegistryEntries(entries []registry.Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return LessRepoIDPath(entries[i].RepoID, entries[i].Path, entries[j].RepoID, entries[j].Path)
	})
}
