// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/strutil"
)

type labelRequirement struct {
	key      string
	value    string
	hasValue bool
}

func parseLabelSelector(raw string) ([]labelRequirement, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strutil.SplitCSV(trimmed)
	if len(parts) == 0 {
		return nil, nil
	}
	reqs := make([]labelRequirement, 0, len(parts))
	for _, part := range parts {
		expr := strings.TrimSpace(part)
		if expr == "" {
			continue
		}
		tokens := strings.SplitN(expr, "=", 2)
		key := strings.TrimSpace(tokens[0])
		if err := validateMetadataKey(key, "--selector"); err != nil {
			return nil, err
		}
		req := labelRequirement{key: key}
		if len(tokens) == 2 {
			req.hasValue = true
			req.value = strings.TrimSpace(tokens[1])
		}
		reqs = append(reqs, req)
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("empty --selector expression")
	}
	return reqs, nil
}

func labelsMatchSelector(labels map[string]string, reqs []labelRequirement) bool {
	if len(reqs) == 0 {
		return true
	}
	for _, req := range reqs {
		got, ok := labels[req.key]
		if !ok {
			return false
		}
		if req.hasValue && got != req.value {
			return false
		}
	}
	return true
}
