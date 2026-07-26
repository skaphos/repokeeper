// SPDX-License-Identifier: MIT

//go:build unix

package mcpserver

import (
	"errors"
	"syscall"
)

// isReadOnlyFS reports whether err was caused by writing to a read-only
// filesystem.
//
// EROFS is matched structurally rather than by string. internal/gitx already
// forces LC_ALL=C on every git invocation precisely because it string-matches
// stderr, and that comment is the standing argument against adding another
// locale-sensitive check: errors.Is is stable across locales, git versions and
// wrapping.
func isReadOnlyFS(err error) bool {
	return errors.Is(err, syscall.EROFS)
}
