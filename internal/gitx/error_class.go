// SPDX-License-Identifier: MIT
package gitx

import (
	"context"
	"errors"
	"strings"
)

// ClassifyError maps git/process errors into broad actionable categories.
//
// In production, git failures surface as *exec.ExitError (or a wrapped
// error from GitRunner.Run carrying git's stderr text), never as a
// sentinel error value, so classification is done entirely by matching
// text in err.Error(). See GitRunner.Run for why that text is forced to
// the C locale before it reaches here.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}

	msg := strings.ToLower(err.Error())
	// Heuristics are intentionally broad to keep categories actionable for users.
	switch {
	case containsAny(msg, "permission denied", "authentication failed", "access denied", "publickey", "could not read username", "credential"):
		return "auth"
	case containsAny(msg, "could not resolve host", "network is unreachable", "connection timed out", "failed to connect", "temporary failure in name resolution", "tls handshake timeout"):
		return "network"
	case containsAny(msg, "timeout", "timed out", "deadline exceeded"):
		return "timeout"
	case containsAny(msg, "not a git repository", "bad object", "corrupt", "object file"):
		return "corrupt"
	case containsAny(msg, "repository not found", "couldn't find remote ref", "remote ref does not exist", "no such remote"):
		return "missing_remote"
	default:
		return "unknown"
	}
}

func containsAny(msg string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
