// SPDX-License-Identifier: MIT
package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
)

// IgnoredPathSet constructs a set of normalized paths from a list of path strings.
// The normalize function is applied to each path to produce the canonical key.
// Empty paths (after trimming whitespace) are skipped.
func IgnoredPathSet(paths []string, normalize func(string) string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out[normalize(p)] = struct{}{}
	}
	return out
}

// CleanNormalize normalizes a path using filepath.Clean.
// This matches the behavior of engine.go's ignoredPathSet().
func CleanNormalize(p string) string {
	return filepath.Clean(p)
}

// CanonicalNormalize normalizes a path using filepath.Clean and applies
// case-folding on Windows. This matches the behavior of portability.go's
// canonicalPathKey() and ignoredPathSet().
func CanonicalNormalize(p string) string {
	key := filepath.Clean(p)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}
