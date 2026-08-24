// SPDX-License-Identifier: MIT
//go:build integration

package compatibility

import (
	"strings"
	"testing"
)

const validDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestValidatePinnedInputRejectsEmbeddedCredentials(t *testing.T) {
	for _, rawURL := range []string{
		"https://user:password@example.com/git.tar.xz",
		"https://token@example.com/git.tar.xz",
	} {
		err := validatePinnedInput("cells[0].provisioner", rawURL, validDigest)
		if err == nil {
			t.Fatalf("validatePinnedInput(%q) accepted a URL embedding credentials", rawURL)
		}
		if !strings.Contains(err.Error(), "must not embed credentials") {
			t.Fatalf("validatePinnedInput(%q) error = %v, want a credential rejection", rawURL, err)
		}
	}
}

func TestValidatePinnedInputAcceptsCredentialFreeHTTPS(t *testing.T) {
	if err := validatePinnedInput("cells[0].provisioner", "https://example.com/git.tar.xz", validDigest); err != nil {
		t.Fatalf("validatePinnedInput rejected a credential-free HTTPS URL: %v", err)
	}
}
