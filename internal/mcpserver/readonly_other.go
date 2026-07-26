// SPDX-License-Identifier: MIT

//go:build !unix

package mcpserver

// isReadOnlyFS always reports false off unix.
//
// The read-only workspace contract is a property of the container image, which
// is Linux-only: there is no read-only bind mount to detect here. Returning
// false means explainReadOnly passes every error through untouched, which is
// the correct behavior -- inventing a read-only diagnosis on a platform that
// cannot produce one would be worse than saying nothing.
func isReadOnlyFS(error) bool { return false }
