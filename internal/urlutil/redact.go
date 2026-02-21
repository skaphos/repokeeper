// SPDX-License-Identifier: MIT
package urlutil

import (
	"net/url"
	"strings"
)

// RedactCredentials strips embedded credentials from URLs for safe display in verbose/debug output.
//
// It handles both standard URLs (https://, ssh://) and SSH shorthand (git@host:path).
// Returns the original string unchanged if no credentials are present or if the URL is malformed.
//
// SSH URLs (ssh://, git@) are returned unchanged because SSH uses key-based authentication,
// not embedded credentials.
//
// Examples:
//
//	https://user:token@github.com/org/repo.git → https://***@github.com/org/repo.git
//	https://user@github.com/org/repo.git → https://***@github.com/org/repo.git
//	git@github.com:org/repo.git → git@github.com:org/repo.git (unchanged)
//	ssh://git@github.com/org/repo.git → ssh://git@github.com/org/repo.git (unchanged)
//	https://github.com/org/repo.git → https://github.com/org/repo.git (unchanged)
func RedactCredentials(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Try to parse as a standard URL first
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Scheme != "" {
		// SSH URLs use key-based auth, not embedded credentials; return unchanged
		if parsed.Scheme == "ssh" || parsed.Scheme == "git" {
			return rawURL
		}

		// Check if there are credentials in the URL
		if parsed.User != nil {
			// Reconstruct the URL with redacted credentials
			userinfo := "***"
			if parsed.User.Username() != "" {
				// Has username (with or without password)
				userinfo = "***"
			}
			// Build the redacted URL
			redacted := parsed.Scheme + "://" + userinfo + "@" + parsed.Host + parsed.Path
			if parsed.RawQuery != "" {
				redacted += "?" + parsed.RawQuery
			}
			if parsed.Fragment != "" {
				redacted += "#" + parsed.Fragment
			}
			return redacted
		}
		// No credentials, return as-is
		return rawURL
	}

	// Handle SSH shorthand: git@host:path (no protocol)
	// SSH shorthand doesn't have embedded credentials in the URL itself
	// (credentials are handled via SSH keys), so return unchanged
	if strings.Contains(rawURL, "@") && !strings.Contains(rawURL, "://") {
		// This is likely SSH shorthand; return unchanged
		return rawURL
	}

	// Malformed or unrecognized format; return unchanged
	return rawURL
}
