// SPDX-License-Identifier: MIT
package vcs

import (
	"github.com/skaphos/repokeeper/internal/gitx"
)

// ErrorClassifier classifies VCS operation errors into stable categories.
type ErrorClassifier interface {
	ClassifyError(err error) string
}

// URLNormalizer normalizes VCS remote URLs to canonical form.
type URLNormalizer interface {
	NormalizeURL(rawURL string) string
}

// gitErrorClassifier implements ErrorClassifier by delegating to gitx.ClassifyError.
type gitErrorClassifier struct{}

// ClassifyError delegates to gitx.ClassifyError.
func (gitErrorClassifier) ClassifyError(err error) string {
	return gitx.ClassifyError(err)
}

// NewGitErrorClassifier returns a new ErrorClassifier backed by gitx.ClassifyError.
func NewGitErrorClassifier() ErrorClassifier {
	return gitErrorClassifier{}
}

// gitURLNormalizer implements URLNormalizer by delegating to gitx.NormalizeURL.
type gitURLNormalizer struct{}

// NormalizeURL delegates to gitx.NormalizeURL.
func (gitURLNormalizer) NormalizeURL(rawURL string) string {
	return gitx.NormalizeURL(rawURL)
}

// NewGitURLNormalizer returns a new URLNormalizer backed by gitx.NormalizeURL.
func NewGitURLNormalizer() URLNormalizer {
	return gitURLNormalizer{}
}
