// SPDX-License-Identifier: MIT
package urlutil

import (
	"testing"
)

func TestRedactCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS with user and password",
			input:    "https://user:token@github.com/org/repo.git",
			expected: "https://***@github.com/org/repo.git",
		},
		{
			name:     "HTTPS with user only",
			input:    "https://user@github.com/org/repo.git",
			expected: "https://***@github.com/org/repo.git",
		},
		{
			name:     "HTTPS without credentials",
			input:    "https://github.com/org/repo.git",
			expected: "https://github.com/org/repo.git",
		},
		{
			name:     "SSH shorthand (git@host:path)",
			input:    "git@github.com:org/repo.git",
			expected: "git@github.com:org/repo.git",
		},
		{
			name:     "SSH protocol URL",
			input:    "ssh://git@github.com/org/repo.git",
			expected: "ssh://git@github.com/org/repo.git",
		},
		{
			name:     "file:// URL",
			input:    "file:///local/path/to/repo",
			expected: "file:///local/path/to/repo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "malformed URL",
			input:    "not a valid url at all",
			expected: "not a valid url at all",
		},
		{
			name:     "HTTP with credentials",
			input:    "http://admin:secret@example.com/repo.git",
			expected: "http://***@example.com/repo.git",
		},
		{
			name:     "HTTPS with port and credentials",
			input:    "https://user:pass@github.com:443/org/repo.git",
			expected: "https://***@github.com:443/org/repo.git",
		},
		{
			name:     "git protocol (key-based, unchanged)",
			input:    "git://user:pass@github.com/org/repo.git",
			expected: "git://user:pass@github.com/org/repo.git",
		},
		{
			name:     "URL with query string and credentials",
			input:    "https://user:pass@github.com/org/repo.git?param=value",
			expected: "https://***@github.com/org/repo.git?param=value",
		},
		{
			name:     "URL with fragment and credentials",
			input:    "https://user:pass@github.com/org/repo.git#section",
			expected: "https://***@github.com/org/repo.git#section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			if result != tt.expected {
				t.Errorf("RedactCredentials(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
