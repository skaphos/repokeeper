// SPDX-License-Identifier: MIT
package gitx

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrAuthFailure marks authentication/authorization failures.
	ErrAuthFailure = errors.New("git auth error")
	// ErrNetworkFailure marks network/transport failures.
	ErrNetworkFailure = errors.New("git network error")
	// ErrCorruptRepo marks corrupt or invalid-repository failures.
	ErrCorruptRepo = errors.New("git corrupt repository")
	// ErrMissingRemoteRef marks missing upstream/ref/remote failures.
	ErrMissingRemoteRef = errors.New("git missing remote")
)

// ClassifyError maps git/process errors into broad actionable categories.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	if errors.Is(err, ErrAuthFailure) {
		return "auth"
	}
	if errors.Is(err, ErrNetworkFailure) {
		return "network"
	}
	if errors.Is(err, ErrCorruptRepo) {
		return "corrupt"
	}
	if errors.Is(err, ErrMissingRemoteRef) {
		return "missing_remote"
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
