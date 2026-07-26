// SPDX-License-Identifier: MIT

package mcpserver

import "fmt"

// A tool that fails because the workspace was mounted read-only must say so.
//
// The container image documents a read-only bind mount as its default, which is
// Principle VII made concrete: inspection always works, mutation is refused. But
// a refusal is only useful if it explains itself. Left untranslated, a mutating
// tool under a read-only mount surfaces a bare "read-only file system" path
// error, or a git error about being unable to write a lock file -- both are
// failures, and neither tells the caller what to do, which Principle VI treats
// as a defect.
//
// The mutating tools are deliberately still advertised. Hiding them under a
// read-only mount would be a silently reduced surface; a tool that exists and
// explains why it cannot run is honest, one that vanishes is not.

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
	if err == nil || !isReadOnlyFS(err) {
		return err
	}
	return fmt.Errorf("%s: %w", readOnlyAdvice, err)
}
