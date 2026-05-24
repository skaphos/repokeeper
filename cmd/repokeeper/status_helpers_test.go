// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestRelatedReposString(t *testing.T) {
	t.Parallel()

	if got := relatedReposString(nil); got != "-" {
		t.Fatalf("expected dash for empty related repos, got %q", got)
	}

	got := relatedReposString([]model.RepoMetadataRelatedRepo{{RepoID: "z/repo", Relationship: "depends-on"}, {RepoID: "a/repo"}})
	if got != "a/repo,z/repo:depends-on" {
		t.Fatalf("expected sorted related repos string, got %q", got)
	}
}

// TestStatusJSONOutputIncludesAPIVersion locks the SKA-208 contract: the
// `get`/`status -o json` envelope carries a top-level apiVersion equal to the
// schema constant, in both the normal and --only diverged shapes, without
// dropping the existing generated_at/repos fields.
func TestStatusJSONOutputIncludesAPIVersion(t *testing.T) {
	t.Parallel()

	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{RepoID: "github.com/org/healthy", Path: "/repos/healthy", Tracking: model.Tracking{Status: model.TrackingEqual}},
			{
				RepoID:   "github.com/org/diverged",
				Path:     "/repos/diverged",
				Tracking: model.Tracking{Status: model.TrackingDiverged, Upstream: "origin/main"},
			},
		},
	}

	t.Run("normal output", func(t *testing.T) {
		t.Parallel()
		raw, err := json.Marshal(buildStatusJSONOutput(report, false))
		if err != nil {
			t.Fatalf("marshal status json: %v", err)
		}
		var doc struct {
			APIVersion  string            `json:"apiVersion"`
			GeneratedAt string            `json:"generated_at"`
			Repos       []json.RawMessage `json:"repos"`
			Diverged    []json.RawMessage `json:"diverged"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal status json: %v\n%s", err, raw)
		}
		if doc.APIVersion != statusJSONAPIVersion {
			t.Errorf("apiVersion = %q, want %q", doc.APIVersion, statusJSONAPIVersion)
		}
		if len(doc.Repos) != len(report.Repos) {
			t.Errorf("repos length = %d, want %d (existing field must be preserved)", len(doc.Repos), len(report.Repos))
		}
		if doc.Diverged != nil {
			t.Errorf("normal output must not include diverged array, got %v", doc.Diverged)
		}
	})

	t.Run("diverged output", func(t *testing.T) {
		t.Parallel()
		raw, err := json.Marshal(buildStatusJSONOutput(report, true))
		if err != nil {
			t.Fatalf("marshal diverged json: %v", err)
		}
		var doc struct {
			APIVersion string            `json:"apiVersion"`
			Repos      []json.RawMessage `json:"repos"`
			Diverged   []json.RawMessage `json:"diverged"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal diverged json: %v\n%s", err, raw)
		}
		if doc.APIVersion != statusJSONAPIVersion {
			t.Errorf("diverged apiVersion = %q, want %q", doc.APIVersion, statusJSONAPIVersion)
		}
		if len(doc.Diverged) != 1 {
			t.Errorf("diverged advice length = %d, want 1", len(doc.Diverged))
		}
	})
}
