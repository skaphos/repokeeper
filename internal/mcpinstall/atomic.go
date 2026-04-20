// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to path via a temp-file-plus-rename sequence
// so that readers never observe a truncated or partially-written file.
//
// Mode handling: if path already exists as a regular file (or as a
// symlink to one), its existing mode is preserved. Otherwise, the
// supplied mode is used. This keeps us from tightening or loosening
// permissions on user-managed config files we don't own.
//
// Symlink handling: if path is a symlink, it is resolved and the
// write lands on the target, leaving the symlink in place. The temp
// file is created alongside the final destination (after symlink
// resolution) so os.Rename stays on the same filesystem.
//
// The parent directory of the resolved target must already exist;
// WriteAtomic does not create intermediate directories. Callers that
// need mkdir-p should do it before calling.
func WriteAtomic(path string, data []byte, mode fs.FileMode) error {
	resolved, existingMode, existed, err := resolveTarget(path)
	if err != nil {
		return err
	}
	if existed {
		mode = existingMode
	}
	dir := filepath.Dir(resolved)
	tmp, err := os.CreateTemp(dir, ".mcpinstall.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, resolved); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// resolveTarget returns the concrete path writes should land on (after
// symlink resolution), the existing regular-file mode (if any), and
// whether the target file existed at call time.
func resolveTarget(path string) (string, fs.FileMode, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, 0, false, nil
		}
		return "", 0, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", 0, false, fmt.Errorf("resolve symlink %q: %w", path, err)
		}
		targetInfo, err := os.Stat(target)
		if err != nil {
			return "", 0, false, fmt.Errorf("stat symlink target %q: %w", target, err)
		}
		return target, targetInfo.Mode().Perm(), true, nil
	}
	return path, info.Mode().Perm(), true, nil
}
