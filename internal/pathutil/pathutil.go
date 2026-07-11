// SPDX-License-Identifier: MIT
package pathutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// WriteFileAtomic writes data to path durably and atomically. It creates a
// temporary file in the same directory, writes and fsyncs it, then renames it
// over the destination so a concurrent reader or a crash mid-write never
// observes a truncated or partially written file (unlike os.WriteFile, which
// truncates in place).
//
// Mode handling: if path already exists as a regular file, its current mode is
// preserved; otherwise perm is applied. Because the write lands on a fresh temp
// file that is renamed into place, an existing symlink at path is replaced by a
// regular file rather than being followed — this deliberately refuses to write
// through a symlink to an arbitrary location.
//
// The parent directory must already exist; callers that need mkdir-p should do
// it beforehand.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is required")
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode().IsRegular() {
			perm = info.Mode().Perm()
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".repokeeper-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	// Best-effort fsync of the directory so the rename itself is durable.
	syncDir(dir)
	return nil
}

// syncDir flushes the directory entry so a rename survives a crash. Not all
// platforms or filesystems support directory fsync, so failures are ignored.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
