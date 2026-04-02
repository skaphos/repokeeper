// SPDX-License-Identifier: MIT
// Package selector provides label and field selector parsing and matching
// for filtering registry entries. Shared by the CLI and MCP server.
package selector

import (
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/strutil"
)

// LabelRequirement represents a single label match expression (key or key=value).
type LabelRequirement struct {
	Key      string
	Value    string
	HasValue bool
}

// ParseLabelSelector parses a comma-separated label selector string into requirements.
// An empty string returns nil requirements and no error.
func ParseLabelSelector(raw string) ([]LabelRequirement, error) {
	return ParseLabelSelectorForFlag(raw, "--selector")
}

// ParseLabelSelectorForFlag parses a label selector with a custom flag name for error messages.
func ParseLabelSelectorForFlag(raw, flagName string) ([]LabelRequirement, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strutil.SplitCSV(trimmed)
	if len(parts) == 0 {
		return nil, nil
	}
	reqs := make([]LabelRequirement, 0, len(parts))
	for _, part := range parts {
		expr := strings.TrimSpace(part)
		if expr == "" {
			continue
		}
		tokens := strings.SplitN(expr, "=", 2)
		key := strings.TrimSpace(tokens[0])
		if err := validateKey(key, flagName); err != nil {
			return nil, err
		}
		req := LabelRequirement{Key: key}
		if len(tokens) == 2 {
			req.HasValue = true
			req.Value = strings.TrimSpace(tokens[1])
		}
		reqs = append(reqs, req)
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("empty %s expression", flagName)
	}
	return reqs, nil
}

// LabelsMatchSelector returns true if the given labels satisfy all requirements.
// An empty requirement list matches everything.
func LabelsMatchSelector(labels map[string]string, reqs []LabelRequirement) bool {
	if len(reqs) == 0 {
		return true
	}
	for _, req := range reqs {
		got, ok := labels[req.Key]
		if !ok {
			return false
		}
		if req.HasValue && got != req.Value {
			return false
		}
	}
	return true
}

// validateKey checks that a selector key is non-empty and contains no
// whitespace or '=' characters.
func validateKey(key, flagName string) error {
	if key == "" {
		return fmt.Errorf("invalid %s key: cannot be empty", flagName)
	}
	if strings.ContainsAny(key, " \t\r\n=") {
		return fmt.Errorf("invalid %s key %q: keys cannot contain whitespace or '='", flagName, key)
	}
	return nil
}
