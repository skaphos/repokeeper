// SPDX-License-Identifier: MIT

//go:build unix

package mcpserver

import (
	"errors"
	"syscall"
)

// isReadOnlyErrno reports whether err was caused by writing to a read-only
// filesystem.
//
// This covers the direct-write path only -- the registry writers, which fail
// with a real errno. It cannot see a git subprocess failure: git reports
// through *exec.ExitError and stderr text, so EROFS never appears in the error
// chain. That case is handled by the text match in readonly.go, which is safe
// because GitRunner.Run pins the locale.
func isReadOnlyErrno(err error) bool {
	return errors.Is(err, syscall.EROFS)
}
