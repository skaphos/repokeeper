// SPDX-License-Identifier: MIT

package mcpserver

import (
	"fmt"
	"strings"
)

// A tool that fails because the workspace was mounted read-only must say so.
//
// The container image documents a read-only bind mount as its default, which is
// Principle VII made concrete: inspection always works, mutation is refused. But
// a refusal is only useful if it explains itself. Left untranslated, a mutating
// tool under a read-only mount surfaces a bare "read-only file system" path
// error, or a git error about being unable to write into .git -- both are
// failures, and neither tells the caller what to do, which Principle VI treats
// as a defect.
//
// The mutating tools are deliberately still advertised. Hiding them under a
// read-only mount would be a silently reduced surface; a tool that exists and
// explains why it cannot run is honest, one that vanishes is not.
//
// Two failure shapes have to be recognised, because the mutating tools split
// into two groups that fail differently:
//
//   - The registry writers (add_repository, remove_repository, set_labels,
//     scan_workspace) write files directly, so the failure arrives as a
//     structural EROFS and is matched with errors.Is.
//   - execute_sync runs git as a subprocess. As gitx.ClassifyError documents,
//     "git failures surface as *exec.ExitError (or a wrapped error from
//     GitRunner.Run carrying git's stderr text), never as a sentinel error
//     value" -- so errors.Is cannot see EROFS through it, and the text is the
//     only signal available.
//
// Matching git's text is safe here for the same reason gitx already relies on
// it: GitRunner.Run forces LC_ALL=C on every invocation specifically so stderr
// is stable, untranslated English. That is a deliberate guarantee of this
// codebase, not an assumption being made about the user's environment.

// readOnlyAdvice is appended to a translated failure. It names the cause and the
// remedy, in the terms the user configured: the ":ro" suffix on the bind mount.
//
// Credentials are stated as conditional rather than required. Most mutating
// tools here only rewrite the registry or a working tree -- add_repository,
// remove_repository and set_labels need nothing but a writable mount. Telling
// every caller to supply git credentials would send them looking for a second
// problem they do not have, which is its own kind of unhelpful error.
const readOnlyAdvice = "the workspace is mounted read-only, so this tool cannot write to it; " +
	"remount it read-write (drop the \":ro\" suffix from the bind mount) to enable it, " +
	"and additionally supply git credentials if the tool reaches a remote, as sync execution does " +
	"-- read-only inspection tools are unaffected"

// explainReadOnly translates a write failure caused by a read-only filesystem
// into a refusal that names the cause and the remedy. Any other error is
// returned unchanged: this must not reinterpret failures it does not understand.
func explainReadOnly(err error) error {
	if err == nil || !isReadOnly(err) {
		return err
	}
	return fmt.Errorf("%s: %w", readOnlyAdvice, err)
}

// isReadOnly reports whether err was caused by a read-only filesystem, by
// either route: a structural errno from a direct write, or git's own stderr
// text from a subprocess that could not write.
func isReadOnly(err error) bool {
	return isReadOnlyErrno(err) || mentionsReadOnlyFilesystem(err)
}

// readOnlyGitMarkers are the phrases git emits, in the C locale, when it cannot
// write because the filesystem is read-only. The first is strerror(EROFS)
// passed through verbatim and covers most cases; the others catch the lock and
// object-write paths, which are what a fetch into a read-only .git hits first.
var readOnlyGitMarkers = []string{
	"read-only file system",
	"could not lock config file",
	"unable to create",
}

// mentionsReadOnlyFilesystem matches git's stderr text. It requires the
// read-only phrase itself for the broader markers, so an "unable to create"
// caused by something else -- a missing parent directory, a permissions
// problem -- is not misdiagnosed as a read-only mount.
func mentionsReadOnlyFilesystem(err error) bool {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "read-only file system") {
		return true
	}
	// The remaining markers are only meaningful alongside the errno text; on
	// their own they describe failures with entirely different remedies.
	for _, marker := range readOnlyGitMarkers[1:] {
		if strings.Contains(msg, marker) && strings.Contains(msg, "read-only") {
			return true
		}
	}
	return false
}
