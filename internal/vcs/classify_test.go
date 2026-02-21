// SPDX-License-Identifier: MIT
package vcs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func TestGitErrorClassifier(t *testing.T) {
	classifier := vcs.NewGitErrorClassifier()

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "auth failure",
			err:      gitx.ErrAuthFailure,
			expected: "auth",
		},
		{
			name:     "network failure",
			err:      gitx.ErrNetworkFailure,
			expected: "network",
		},
		{
			name:     "corrupt repo",
			err:      gitx.ErrCorruptRepo,
			expected: "corrupt",
		},
		{
			name:     "missing remote ref",
			err:      gitx.ErrMissingRemoteRef,
			expected: "missing_remote",
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: "timeout",
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: "unknown",
		},
		{
			name:     "permission denied heuristic",
			err:      errors.New("permission denied"),
			expected: "auth",
		},
		{
			name:     "network unreachable heuristic",
			err:      errors.New("network is unreachable"),
			expected: "network",
		},
		{
			name:     "not a git repository heuristic",
			err:      errors.New("not a git repository"),
			expected: "corrupt",
		},
		{
			name:     "repository not found heuristic",
			err:      errors.New("repository not found"),
			expected: "missing_remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err)
			if got != tt.expected {
				t.Errorf("ClassifyError(%v) = %q, want %q", tt.err, got, tt.expected)
			}
		})
	}
}

func TestGitErrorClassifierMatchesGitx(t *testing.T) {
	classifier := vcs.NewGitErrorClassifier()

	testErrors := []error{
		nil,
		gitx.ErrAuthFailure,
		gitx.ErrNetworkFailure,
		gitx.ErrCorruptRepo,
		gitx.ErrMissingRemoteRef,
		context.DeadlineExceeded,
		context.Canceled,
		errors.New("permission denied"),
		errors.New("network is unreachable"),
		errors.New("not a git repository"),
		errors.New("repository not found"),
		errors.New("unknown error"),
	}

	for _, err := range testErrors {
		classifierResult := classifier.ClassifyError(err)
		gitxResult := gitx.ClassifyError(err)
		if classifierResult != gitxResult {
			t.Errorf("ClassifyError(%v) = %q, but gitx.ClassifyError(%v) = %q", err, classifierResult, err, gitxResult)
		}
	}
}

func TestGitURLNormalizer(t *testing.T) {
	normalizer := vcs.NewGitURLNormalizer()

	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "empty URL",
			rawURL:   "",
			expected: "",
		},
		{
			name:     "SSH shorthand with .git",
			rawURL:   "git@github.com:Org/Repo.git",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "SSH shorthand without .git",
			rawURL:   "git@github.com:Org/Repo",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "HTTPS URL with .git",
			rawURL:   "https://github.com/Org/Repo.git",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "HTTPS URL without .git",
			rawURL:   "https://github.com/Org/Repo",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "git protocol URL",
			rawURL:   "git://github.com/Org/Repo.git",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "uppercase host lowercased",
			rawURL:   "https://GitHub.com/Org/Repo.git",
			expected: "github.com/Org/Repo",
		},
		{
			name:     "trailing slashes stripped after .git",
			rawURL:   "https://github.com/Org/Repo.git/",
			expected: "github.com/Org/Repo.git",
		},
		{
			name:     "SSH with port",
			rawURL:   "ssh://git@github.com:22/Org/Repo.git",
			expected: "github.com/Org/Repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.NormalizeURL(tt.rawURL)
			if got != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.rawURL, got, tt.expected)
			}
		})
	}
}

func TestGitURLNormalizerMatchesGitx(t *testing.T) {
	normalizer := vcs.NewGitURLNormalizer()

	testURLs := []string{
		"",
		"git@github.com:Org/Repo.git",
		"git@github.com:Org/Repo",
		"https://github.com/Org/Repo.git",
		"https://github.com/Org/Repo",
		"git://github.com/Org/Repo.git",
		"https://GitHub.com/Org/Repo.git",
		"https://github.com/Org/Repo.git/",
		"ssh://git@github.com:22/Org/Repo.git",
	}

	for _, url := range testURLs {
		normalizerResult := normalizer.NormalizeURL(url)
		gitxResult := gitx.NormalizeURL(url)
		if normalizerResult != gitxResult {
			t.Errorf("NormalizeURL(%q) = %q, but gitx.NormalizeURL(%q) = %q", url, normalizerResult, url, gitxResult)
		}
	}
}
