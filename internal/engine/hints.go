// SPDX-License-Identifier: MIT
package engine

// errorHints maps error class strings to operator-facing remediation advice.
var errorHints = map[string]string{
	"auth":           "check SSH keys or credentials for this remote",
	"network":        "verify network connectivity and remote host availability",
	"timeout":        "try increasing --timeout or check network latency",
	"corrupt":        "consider running 'git fsck' in the repository",
	"missing_remote": "verify the remote URL in registry matches the actual remote",
}

// hintForErrorClass returns a remediation hint for the given error class, or empty string if none.
func hintForErrorClass(class string) string {
	return errorHints[class]
}

// logSyncFailureHint emits a Warnf hint via the engine logger when the result
// represents a failure with a known, actionable error class. Hints are only
// visible in verbose output (Logger.Warnf is suppressed at the default level).
func (e *Engine) logSyncFailureHint(result SyncResult) {
	if result.OK {
		return
	}
	if hint := hintForErrorClass(result.ErrorClass); hint != "" {
		e.logger.Warnf("hint for %s (%s): %s", result.Path, result.ErrorClass, hint)
	}
}
