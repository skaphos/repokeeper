// SPDX-License-Identifier: MIT

package model_test

import (
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/model"
)

const (
	secretURL    = "https://alice:ghp_supersecrettoken@github.com/org/repo.git"
	secretToken  = "ghp_supersecrettoken"
	sshRemoteURL = "git@github.com:org/repo.git"
)

func TestRepoStatusRedactedStripsCredentials(t *testing.T) {
	t.Parallel()

	status := model.RepoStatus{
		RepoID:  "github.com/org/repo",
		Remotes: []model.Remote{{Name: "origin", URL: secretURL}},
	}

	got := status.Redacted()
	if strings.Contains(got.Remotes[0].URL, secretToken) {
		t.Errorf("redacted URL still contains the credential: %q", got.Remotes[0].URL)
	}
	if !strings.Contains(got.Remotes[0].URL, "github.com/org/repo.git") {
		t.Errorf("redaction destroyed the usable part of the URL: %q", got.Remotes[0].URL)
	}
}

// TestRepoStatusRedactedDoesNotMutateReceiver is the guarantee that matters most
// here. The same RepoStatus values feed the registry write path, so redacting in
// place would persist a masked remote and break the user's ability to fetch.
func TestRepoStatusRedactedDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()

	status := model.RepoStatus{
		Remotes: []model.Remote{{Name: "origin", URL: secretURL}},
	}

	_ = status.Redacted()

	if status.Remotes[0].URL != secretURL {
		t.Errorf("Redacted mutated its receiver: %q", status.Remotes[0].URL)
	}
}

// TestRepoStatusRedactedLeavesSSHUnchanged: SSH remotes authenticate with keys,
// so there is no embedded secret to strip and mangling them would break tooling
// that round-trips the value.
func TestRepoStatusRedactedLeavesSSHUnchanged(t *testing.T) {
	t.Parallel()

	status := model.RepoStatus{
		Remotes: []model.Remote{{Name: "origin", URL: sshRemoteURL}},
	}

	if got := status.Redacted().Remotes[0].URL; got != sshRemoteURL {
		t.Errorf("SSH remote = %q, want unchanged %q", got, sshRemoteURL)
	}
}

func TestRedactedStatusesPreservesNil(t *testing.T) {
	t.Parallel()

	// Nil must survive as nil so the caller keeps control of the
	// empty-collection representation the contract requires ([] not null).
	if got := model.RedactedStatuses(nil); got != nil {
		t.Errorf("RedactedStatuses(nil) = %v, want nil", got)
	}
}

func TestRedactedStatusesRedactsEveryEntry(t *testing.T) {
	t.Parallel()

	statuses := []model.RepoStatus{
		{Remotes: []model.Remote{{Name: "origin", URL: secretURL}}},
		{Remotes: []model.Remote{{Name: "origin", URL: secretURL}}},
	}

	for i, got := range model.RedactedStatuses(statuses) {
		if strings.Contains(got.Remotes[0].URL, secretToken) {
			t.Errorf("statuses[%d] still contains the credential: %q", i, got.Remotes[0].URL)
		}
	}
}
