// SPDX-License-Identifier: MIT
package prune_test

import (
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/prune"
)

func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }

func TestClassify(t *testing.T) {
	now := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -120)
	recent := now.AddDate(0, 0, -1)

	base := prune.Policy{BaseBranch: "main", RequireMerged: true, StaleDays: 90}

	tests := []struct {
		name        string
		branch      model.LocalBranch
		policy      prune.Policy
		wantCat     model.PruneCategory
		wantReasons []model.PruneReason
	}{
		// --- keep guards (precedence, first-match-wins) ---
		{
			name:        "current branch is kept even when merged",
			branch:      model.LocalBranch{Name: "feature", IsCurrent: true, MergedIntoBase: boolPtr(true)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonCurrentBranch},
		},
		{
			name:        "checked out in another worktree is kept",
			branch:      model.LocalBranch{Name: "feature", CheckedOutElsewhere: true, MergedIntoBase: boolPtr(true)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonCheckedOutElsewhere},
		},
		{
			name:        "base branch is kept",
			branch:      model.LocalBranch{Name: "main", MergedIntoBase: boolPtr(true)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonBaseBranch},
		},
		{
			name:        "protected branch is kept even when merged",
			branch:      model.LocalBranch{Name: "release/1.0", Protected: true, MergedIntoBase: boolPtr(true)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonProtectedPattern},
		},
		// --- safe_to_prune (reachability) ---
		{
			name:        "reachable-merged is safe_to_prune",
			branch:      model.LocalBranch{Name: "feature", MergedIntoBase: boolPtr(true), PatchEquivalentToBase: boolPtr(false)},
			policy:      base,
			wantCat:     model.PruneSafeToPrune,
			wantReasons: []model.PruneReason{model.ReasonMergedIntoBase},
		},
		// --- patch-equivalence gated by RequireMerged ---
		{
			name:        "patch-equivalent under RequireMerged is needs_review",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingGone, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(true)},
			policy:      prune.Policy{BaseBranch: "main", RequireMerged: true},
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonPatchEquivalentToBase},
		},
		{
			name:        "patch-equivalent with RequireMerged=false is probably_safe",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingGone, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(true)},
			policy:      prune.Policy{BaseBranch: "main", RequireMerged: false},
			wantCat:     model.PruneProbablySafe,
			wantReasons: []model.PruneReason{model.ReasonPatchEquivalentToBase},
		},
		// --- the closed data-loss hole: upstream gone, no integration evidence ---
		{
			name:        "upstream gone without integration evidence is needs_review, not probably_safe",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingGone, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(recent)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonUnmergedLocalWork},
		},
		// --- signal unavailable (tri-state nil) ---
		{
			name:        "both integration signals unknown is needs_review",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingAhead},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonSignalUnavailable},
		},
		{
			name:        "merged known-false but patch unknown is needs_review (can't rule out squash)",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingGone, MergedIntoBase: boolPtr(false)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonSignalUnavailable},
		},
		// --- definitively not integrated ---
		{
			name:        "no upstream, unmerged local work is needs_review",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingNone, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(recent)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonUnmergedLocalWork},
		},
		{
			name:        "diverged and unmerged is needs_review",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingDiverged, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(recent)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonDivergedUnmerged},
		},
		{
			name:        "active branch pushed to live upstream, not stale, is kept",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingAhead, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(recent)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonActiveUnmerged},
		},
		{
			name:        "equal-to-upstream, not stale, is kept",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingEqual, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(recent)},
			policy:      base,
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonActiveUnmerged},
		},
		// --- staleness escalation ---
		{
			name:        "stale active branch escalates to needs_review",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingAhead, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(old)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonStaleUnmerged},
		},
		{
			name:        "stale unmerged with gone upstream carries both reasons",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingGone, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(old)},
			policy:      base,
			wantCat:     model.PruneNeedsReview,
			wantReasons: []model.PruneReason{model.ReasonUnmergedLocalWork, model.ReasonStaleUnmerged},
		},
		{
			name:        "staleness disabled (StaleDays=0) keeps old active branch",
			branch:      model.LocalBranch{Name: "feature", UpstreamStatus: model.TrackingAhead, MergedIntoBase: boolPtr(false), PatchEquivalentToBase: boolPtr(false), LastCommitAt: timePtr(old)},
			policy:      prune.Policy{BaseBranch: "main", RequireMerged: true, StaleDays: 0},
			wantCat:     model.PruneKeep,
			wantReasons: []model.PruneReason{model.ReasonActiveUnmerged},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCat, gotReasons := prune.Classify(tc.branch, tc.policy, now)
			if gotCat != tc.wantCat {
				t.Errorf("category = %q, want %q", gotCat, tc.wantCat)
			}
			if !reasonsEqual(gotReasons, tc.wantReasons) {
				t.Errorf("reasons = %v, want %v", gotReasons, tc.wantReasons)
			}
		})
	}
}

func TestMatchesProtected(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		patterns []string
		want     bool
		wantErr  bool
	}{
		{name: "exact match", branch: "main", patterns: []string{"main", "master"}, want: true},
		{name: "glob match", branch: "release/1.0", patterns: []string{"release/*"}, want: true},
		{name: "no match", branch: "feature/x", patterns: []string{"main", "release/*"}, want: false},
		{name: "empty branch never matches", branch: "", patterns: []string{"*"}, want: false},
		{name: "empty patterns skipped", branch: "main", patterns: []string{"", "  "}, want: false},
		{name: "malformed glob errors", branch: "release/x", patterns: []string{"release/["}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := prune.MatchesProtected(tc.branch, tc.patterns)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("MatchesProtected(%q, %v) = %v, want %v", tc.branch, tc.patterns, got, tc.want)
			}
		})
	}
}

func reasonsEqual(a, b []model.PruneReason) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
