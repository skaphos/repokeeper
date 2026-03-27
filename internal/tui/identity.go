// SPDX-License-Identifier: MIT
package tui

import (
	"strings"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
)

const checkoutIdentitySeparator = "\x00"

func syncResultIdentityKey(result engine.SyncResult) string {
	if path := strings.TrimSpace(result.Path); path != "" {
		return result.RepoID + checkoutIdentitySeparator + path
	}
	return result.RepoID
}

func sameRepoCheckout(current, incoming model.RepoStatus) bool {
	currentCheckoutID := strings.TrimSpace(current.CheckoutID)
	incomingCheckoutID := strings.TrimSpace(incoming.CheckoutID)
	if currentCheckoutID != "" && incomingCheckoutID != "" {
		return current.RepoID == incoming.RepoID && currentCheckoutID == incomingCheckoutID
	}

	currentPath := strings.TrimSpace(current.Path)
	incomingPath := strings.TrimSpace(incoming.Path)
	if currentPath != "" && incomingPath != "" {
		return currentPath == incomingPath
	}

	return false
}

func findRepoStatusIndex(rows []model.RepoStatus, incoming model.RepoStatus) int {
	repoIDMatches := 0
	repoIDIndex := -1

	for i, current := range rows {
		if sameRepoCheckout(current, incoming) {
			return i
		}
		if current.RepoID != "" && current.RepoID == incoming.RepoID {
			repoIDMatches++
			repoIDIndex = i
		}
	}

	if repoIDMatches == 1 {
		return repoIDIndex
	}

	return -1
}

func duplicateSyncRepoCounts(plan []engine.SyncResult) map[string]int {
	counts := make(map[string]int, len(plan))
	for _, item := range plan {
		counts[item.RepoID]++
	}
	return counts
}

func syncResultDisplayName(result engine.SyncResult, repoCounts map[string]int) string {
	if repoCounts[result.RepoID] > 1 {
		if path := strings.TrimSpace(result.Path); path != "" {
			return result.RepoID + " @ " + path
		}
	}
	return result.RepoID
}
