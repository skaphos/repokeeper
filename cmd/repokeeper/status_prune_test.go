// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/skaphos/repokeeper/internal/model"
)

func pruneRepoFixture() model.RepoStatus {
	return model.RepoStatus{
		Path:     "/repos/testrepo",
		Head:     model.Head{Branch: "main"},
		Tracking: model.Tracking{Status: model.TrackingNone},
		LocalBranches: model.LocalBranchStatus{Branches: []model.LocalBranch{
			{Name: "main", Category: model.PruneKeep, Reasons: []model.PruneReason{model.ReasonCurrentBranch}},
			{Name: "feature/done", Category: model.PruneSafeToPrune, Reasons: []model.PruneReason{model.ReasonMergedIntoBase}},
			{Name: "feature/wip", Category: model.PruneNeedsReview, Reasons: []model.PruneReason{model.ReasonUnmergedLocalWork}},
		}},
	}
}

func TestWriteStatusDetailsPruneClassification(t *testing.T) {
	t.Parallel()
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	if err := writeStatusDetails(cmd, pruneRepoFixture(), "/repos", nil); err != nil {
		t.Fatalf("writeStatusDetails returned error: %v", err)
	}
	got := out.String()

	if !strings.Contains(got, "LOCAL_BRANCH_PRUNE: safe_to_prune=1 probably_safe=0 needs_review=1 keep=1\n") {
		t.Errorf("missing/incorrect summary line in:\n%s", got)
	}
	if !strings.Contains(got, "PRUNE_SAFE_TO_PRUNE: feature/done\n") {
		t.Errorf("missing safe_to_prune list in:\n%s", got)
	}
	if !strings.Contains(got, "PRUNE_NEEDS_REVIEW: feature/wip\n") {
		t.Errorf("missing needs_review list in:\n%s", got)
	}
	// keep branches are not listed by name; probably_safe is empty so it is omitted.
	if strings.Contains(got, "PRUNE_PROBABLY_SAFE:") {
		t.Errorf("empty probably_safe category should be omitted:\n%s", got)
	}
	if strings.Contains(got, "PRUNE_KEEP:") {
		t.Errorf("keep category should never be listed by name:\n%s", got)
	}
}

func TestWriteStatusDetailsPruneInspectionError(t *testing.T) {
	t.Parallel()
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	repo := model.RepoStatus{
		Path:          "/repos/testrepo",
		Head:          model.Head{Branch: "main"},
		Tracking:      model.Tracking{Status: model.TrackingNone},
		LocalBranches: model.LocalBranchStatus{InspectionError: "boom"},
	}
	if err := writeStatusDetails(cmd, repo, "/repos", nil); err != nil {
		t.Fatalf("writeStatusDetails returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "LOCAL_BRANCH_PRUNE: ?\n") {
		t.Errorf("expected '?' summary on inspection error:\n%s", got)
	}
	if !strings.Contains(got, "LOCAL_BRANCH_INSPECTION_ERROR: boom\n") {
		t.Errorf("expected inspection error line:\n%s", got)
	}
}

func TestStatusJSONIncludesLocalBranches(t *testing.T) {
	t.Parallel()
	report := &model.StatusReport{
		GeneratedAt: time.Unix(0, 0).UTC(),
		Repos:       []model.RepoStatus{pruneRepoFixture()},
	}
	raw, err := json.Marshal(buildStatusJSONOutput(report, false))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, `"local_branches"`) {
		t.Errorf("status JSON missing local_branches key:\n%s", s)
	}
	if !strings.Contains(s, `"category":"safe_to_prune"`) {
		t.Errorf("status JSON missing classified branch:\n%s", s)
	}
}
