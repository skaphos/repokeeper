// SPDX-License-Identifier: MIT

package registry_test

import (
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/registry"
)

const (
	secretURL   = "https://alice:ghp_supersecrettoken@github.com/org/repo.git"
	secretToken = "ghp_supersecrettoken"
)

func TestEntryRedactedStripsCredentials(t *testing.T) {
	t.Parallel()

	entry := registry.Entry{RepoID: "github.com/org/repo", RemoteURL: secretURL}

	if got := entry.Redacted().RemoteURL; strings.Contains(got, secretToken) {
		t.Errorf("redacted RemoteURL still contains the credential: %q", got)
	}
}

// TestEntryRedactedDoesNotMutateReceiver guards the same hazard as the model
// counterpart: these entries are the in-memory registry written back to disk.
func TestEntryRedactedDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()

	entry := registry.Entry{RemoteURL: secretURL}
	_ = entry.Redacted()

	if entry.RemoteURL != secretURL {
		t.Errorf("Redacted mutated its receiver: %q", entry.RemoteURL)
	}
}

func TestRegistryRedactedRedactsEveryEntry(t *testing.T) {
	t.Parallel()

	reg := &registry.Registry{Entries: []registry.Entry{
		{RemoteURL: secretURL},
		{RemoteURL: secretURL},
	}}

	for i, entry := range reg.Redacted().Entries {
		if strings.Contains(entry.RemoteURL, secretToken) {
			t.Errorf("entries[%d] still contains the credential: %q", i, entry.RemoteURL)
		}
	}
	// The original registry is the one that gets saved; it must be untouched.
	for i, entry := range reg.Entries {
		if entry.RemoteURL != secretURL {
			t.Errorf("Redacted mutated source entries[%d]: %q", i, entry.RemoteURL)
		}
	}
}

func TestRegistryRedactedHandlesNilReceiver(t *testing.T) {
	t.Parallel()

	var reg *registry.Registry
	if got := reg.Redacted(); got != nil {
		t.Errorf("nil registry redacted to %v, want nil", got)
	}
}
