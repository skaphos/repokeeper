// SPDX-License-Identifier: MIT
package pathutil

import (
	"os"
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

func TestWriteFileAtomic(t *testing.T) {
	t.Run("creates a new file with the requested mode", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "new.yaml")
		if err := WriteFileAtomic(path, []byte("hello\n"), 0o640); err != nil {
			t.Fatalf("WriteFileAtomic: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != "hello\n" {
			t.Fatalf("content = %q, want %q", got, "hello\n")
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if info.Mode().Perm() != 0o640 {
				t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
			}
		}
	})

	t.Run("overwrites and preserves the existing file mode", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "existing.yaml")
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		// perm arg differs from the on-disk mode; the on-disk mode must win.
		if err := WriteFileAtomic(path, []byte("new-content"), 0o644); err != nil {
			t.Fatalf("WriteFileAtomic: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != "new-content" {
			t.Fatalf("content = %q, want %q", got, "new-content")
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("mode = %v, want preserved 0600", info.Mode().Perm())
			}
		}
	})

	t.Run("does not leave temp files behind", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "clean.yaml")
		if err := WriteFileAtomic(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFileAtomic: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		if len(entries) != 1 || entries[0].Name() != "clean.yaml" {
			t.Fatalf("expected only the destination file, got %v", entries)
		}
	})

	t.Run("replaces a symlink instead of writing through it", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink semantics differ on Windows")
		}
		dir := t.TempDir()
		outside := filepath.Join(dir, "outside.txt")
		if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
			t.Fatalf("seed outside file: %v", err)
		}
		link := filepath.Join(dir, "link.yaml")
		if err := os.Symlink(outside, link); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		if err := WriteFileAtomic(link, []byte("payload"), 0o644); err != nil {
			t.Fatalf("WriteFileAtomic: %v", err)
		}
		// The symlink target must be untouched.
		if got, _ := os.ReadFile(outside); string(got) != "secret" {
			t.Fatalf("symlink target was modified: %q", got)
		}
		// link.yaml must now be a regular file holding the payload.
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("lstat link: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("expected link to be replaced by a regular file")
		}
		if got, _ := os.ReadFile(link); string(got) != "payload" {
			t.Fatalf("link content = %q, want payload", got)
		}
	})

	t.Run("errors when the parent directory is missing", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "missing-subdir", "file.yaml")
		if err := WriteFileAtomic(path, []byte("x"), 0o644); err == nil {
			t.Fatal("expected error writing into a non-existent directory")
		}
	})
}
