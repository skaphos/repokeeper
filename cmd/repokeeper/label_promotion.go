// SPDX-License-Identifier: MIT
package repokeeper

func mergePromotedLabels(shared, local map[string]string) map[string]string {
	if len(shared) == 0 && len(local) == 0 {
		return nil
	}
	merged := cloneMetadataMap(shared)
	if merged == nil {
		merged = make(map[string]string, len(local))
	}
	for key, value := range local {
		if _, exists := merged[key]; exists {
			continue
		}
		merged[key] = value
	}
	return normalizeMetadataMap(merged)
}
