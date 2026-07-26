// SPDX-License-Identifier: MIT

//go:build unix

package mcpserver

import "syscall"

// readOnlyError returns the error a write to a read-only filesystem produces on
// this platform, or nil where the condition cannot arise.
func readOnlyError() error { return syscall.EROFS }
