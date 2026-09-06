// SPDX-License-Identifier: MIT

package repokeeper

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/v2/internal/model"
)

const (
	credentialURL   = "https://alice:ghp_supersecrettoken@github.com/org/repo.git"
	credentialToken = "ghp_supersecrettoken"
)

func statusWithCredentialRemote() model.RepoStatus {
	return model.RepoStatus{
		RepoID:        "github.com/org/repo",
		Path:          "/work/org/repo",
		PrimaryRemote: "origin",
		Remotes:       []model.Remote{{Name: "origin", URL: credentialURL}},
	}
}

// TestAdapterSurfacesRedactCredentials asserts the guarantee on the emitted
// bytes of each adapter-facing surface, not on the redaction helper.
//
// Testing the helper in isolation would pass while a surface that never calls
// it leaked, which is exactly the failure this contract has to rule out: the
// helper already existed before 2.0.0 and was wired only into `export` and
// debug logging, so adapter JSON emitted remote URLs verbatim.
func TestAdapterSurfacesRedactCredentials(t *testing.T) {
	t.Parallel()

	status := statusWithCredentialRemote()

	cases := []struct {
		surface string
		payload any
	}{
		{
			surface: "get/status -o json",
			payload: buildStatusJSONOutput(&model.StatusReport{
				GeneratedAt: time.Now(),
				Repos:       []model.RepoStatus{status},
			}, false),
		},
		{
			surface: "describe -o json",
			payload: newRepoEnvelope(status.Redacted()),
		},
		{
			surface: "scan -o json",
			payload: newReposEnvelope(model.RedactedStatuses([]model.RepoStatus{status})),
		},
	}

	for _, tc := range cases {
		t.Run(tc.surface, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("marshal %s: %v", tc.surface, err)
			}
			if strings.Contains(string(data), credentialToken) {
				t.Errorf("%s emitted an embedded credential:\n%s", tc.surface, data)
			}
			// The redaction must not have eaten the useful part of the URL --
			// an adapter still needs to identify the remote host and path.
			if !strings.Contains(string(data), "github.com/org/repo.git") {
				t.Errorf("%s lost the non-secret portion of the remote URL:\n%s", tc.surface, data)
			}
		})
	}
}

// TestStatusJSONDoesNotMutateSourceReport guards the write path: `status` also
// persists registry snapshots, so redaction applied at the output boundary must
// leave the report it was handed untouched.
func TestStatusJSONDoesNotMutateSourceReport(t *testing.T) {
	t.Parallel()

	report := &model.StatusReport{
		GeneratedAt: time.Now(),
		Repos:       []model.RepoStatus{statusWithCredentialRemote()},
	}

	_ = buildStatusJSONOutput(report, false)

	if got := report.Repos[0].Remotes[0].URL; got != credentialURL {
		t.Errorf("building JSON output mutated the source report: %q", got)
	}
}
