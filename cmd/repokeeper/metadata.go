// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"strings"
)

func parseMetadataAssignments(inputs []string, flagName string) (map[string]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(inputs))
	for _, raw := range inputs {
		expr := strings.TrimSpace(raw)
		if expr == "" {
			continue
		}
		tokens := strings.SplitN(expr, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid %s value %q: expected key=value", flagName, raw)
		}
		key := strings.TrimSpace(tokens[0])
		value := strings.TrimSpace(tokens[1])
		if err := validateMetadataKey(key, flagName); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

func parseMetadataKeys(inputs []string, flagName string) ([]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(inputs))
	for _, raw := range inputs {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if err := validateMetadataKey(key, flagName); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, nil
}

func validateMetadataKey(key, flagName string) error {
	if key == "" {
		return fmt.Errorf("invalid %s key: cannot be empty", flagName)
	}
	if strings.ContainsAny(key, " \t\r\n=") {
		return fmt.Errorf("invalid %s key %q: keys cannot contain whitespace or '='", flagName, key)
	}
	return nil
}

func cloneMetadataMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func normalizeMetadataMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return in
}
