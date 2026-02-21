// SPDX-License-Identifier: MIT
package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCleanNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "foo/bar",
			expected: filepath.Clean("foo/bar"),
		},
		{
			name:     "path with dot",
			input:    "foo/./bar",
			expected: filepath.Clean("foo/./bar"),
		},
		{
			name:     "path with double dot",
			input:    "foo/../bar",
			expected: filepath.Clean("foo/../bar"),
		},
		{
			name:     "trailing slash",
			input:    "foo/bar/",
			expected: filepath.Clean("foo/bar/"),
		},
		{
			name:     "absolute path",
			input:    "/foo/bar",
			expected: filepath.Clean("/foo/bar"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: filepath.Clean(""),
		},
		{
			name:     "single dot",
			input:    ".",
			expected: filepath.Clean("."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanNormalize(tt.input)
			if got != tt.expected {
				t.Errorf("CleanNormalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCanonicalNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "foo/bar",
			expected: canonicalExpected("foo/bar"),
		},
		{
			name:     "path with dot",
			input:    "foo/./bar",
			expected: canonicalExpected("foo/./bar"),
		},
		{
			name:     "path with double dot",
			input:    "foo/../bar",
			expected: canonicalExpected("foo/../bar"),
		},
		{
			name:     "trailing slash",
			input:    "foo/bar/",
			expected: canonicalExpected("foo/bar/"),
		},
		{
			name:     "absolute path",
			input:    "/foo/bar",
			expected: canonicalExpected("/foo/bar"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: canonicalExpected(""),
		},
		{
			name:     "single dot",
			input:    ".",
			expected: canonicalExpected("."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalNormalize(tt.input)
			if got != tt.expected {
				t.Errorf("CanonicalNormalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIgnoredPathSet(t *testing.T) {
	tests := []struct {
		name      string
		paths     []string
		normalize func(string) string
		expected  map[string]struct{}
	}{
		{
			name:      "empty list",
			paths:     []string{},
			normalize: CleanNormalize,
			expected:  map[string]struct{}{},
		},
		{
			name:      "single path",
			paths:     []string{"foo/bar"},
			normalize: CleanNormalize,
			expected: map[string]struct{}{
				filepath.Clean("foo/bar"): {},
			},
		},
		{
			name:      "multiple paths",
			paths:     []string{"foo/bar", "baz/qux"},
			normalize: CleanNormalize,
			expected: map[string]struct{}{
				filepath.Clean("foo/bar"): {},
				filepath.Clean("baz/qux"): {},
			},
		},
		{
			name:      "skip empty paths",
			paths:     []string{"foo/bar", "", "  ", "baz/qux"},
			normalize: CleanNormalize,
			expected: map[string]struct{}{
				filepath.Clean("foo/bar"): {},
				filepath.Clean("baz/qux"): {},
			},
		},
		{
			name:      "with CanonicalNormalize",
			paths:     []string{"foo/bar", "baz/qux"},
			normalize: CanonicalNormalize,
			expected: map[string]struct{}{
				canonicalExpected("foo/bar"): {},
				canonicalExpected("baz/qux"): {},
			},
		},
		{
			name:      "duplicate paths deduplicated",
			paths:     []string{"foo/bar", "foo/bar"},
			normalize: CleanNormalize,
			expected: map[string]struct{}{
				filepath.Clean("foo/bar"): {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IgnoredPathSet(tt.paths, tt.normalize)
			if len(got) != len(tt.expected) {
				t.Errorf("IgnoredPathSet length = %d, want %d", len(got), len(tt.expected))
			}
			for key := range tt.expected {
				if _, ok := got[key]; !ok {
					t.Errorf("IgnoredPathSet missing key %q", key)
				}
			}
			for key := range got {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("IgnoredPathSet unexpected key %q", key)
				}
			}
		})
	}
}

func TestCanonicalNormalizeWindowsBehavior(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase path",
			input:    "FOO\\BAR",
			expected: strings.ToLower(filepath.Clean("FOO\\BAR")),
		},
		{
			name:     "mixed case path",
			input:    "Foo\\Bar",
			expected: strings.ToLower(filepath.Clean("Foo\\Bar")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalNormalize(tt.input)
			if got != tt.expected {
				t.Errorf("CanonicalNormalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func canonicalExpected(p string) string {
	key := filepath.Clean(p)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}
