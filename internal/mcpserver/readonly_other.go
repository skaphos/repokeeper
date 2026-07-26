// SPDX-License-Identifier: MIT

//go:build !unix

package mcpserver

// isReadOnlyErrno always reports false off unix.
//
// EROFS is a POSIX errno; there is no portable equivalent to test for here, and
// the read-only workspace contract belongs to the container image, which is
// Linux-only. Inventing a read-only diagnosis on a platform that cannot produce
// one would be worse than saying nothing.
//
// This does not disable the explanation entirely on these platforms: isReadOnly
// also matches git's stderr text, which is platform-independent. A Windows user
// running against a genuinely read-only volume still gets the translation if git
// reports it; they simply do not get the errno shortcut.
func isReadOnlyErrno(error) bool { return false }
