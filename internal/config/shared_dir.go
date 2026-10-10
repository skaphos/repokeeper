// SPDX-License-Identifier: MIT

//go:build !windows

package config

import "os"

// isSharedDir reports whether dir is world-writable or sticky, which marks it
// as a shared location (like /tmp) that config discovery must not walk into.
// See ADR-0019; shared_dir_windows.go implements the same rule with ACLs.
func isSharedDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode.Perm()&0o002 != 0 || mode&os.ModeSticky != 0
}
