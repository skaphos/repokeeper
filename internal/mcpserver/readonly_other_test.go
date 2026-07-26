// SPDX-License-Identifier: MIT

//go:build !unix

package mcpserver

// readOnlyError returns nil off unix: there is no read-only bind mount to
// detect, so the tests that depend on one skip rather than assert a behavior
// this platform does not have.
func readOnlyError() error { return nil }
