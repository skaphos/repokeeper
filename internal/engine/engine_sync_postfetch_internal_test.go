// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"slices"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/model"
	"github.com/skaphos/repokeeper/v2/internal/registry"
)

// fetchTransitionAdapter reports one tracking state until Fetch runs and
// another afterwards, simulating a fetch that discovers upstream changes.
type fetchTransitionAdapter struct {
	*planAdapter
	branch  string
	before  model.TrackingStatus
	after   model.TrackingStatus
	fetched bool
}

func (a *fetchTransitionAdapter) Head(context.Context, string) (model.Head, error) {
	return model.Head{Branch: a.branch}, nil
}

func (a *fetchTransitionAdapter) Fetch(ctx context.Context, dir string) error {
	a.fetched = true
	return a.planAdapter.Fetch(ctx, dir)
}

func (a *fetchTransitionAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	status := a.before
	if a.fetched {
		status = a.after
	}
	return model.Tracking{Status: status, Upstream: "origin/main"}, nil
}

// A planned --update-local sync decides from pre-fetch state; executing it must
// re-decide after the fetch, as the direct apply path does (#358).
func TestPlannedLocalUpdateRedecidesAfterFetch(t *testing.T) {
	for _, tc := range []struct {
		name          string
		branch        string
		before, after model.TrackingStatus
		opts          SyncOptions
		wantPlanSkip  string
		wantCalls     []string
		wantOutcome   OutcomeKind
		wantSkip      string
	}{
		{
			name:         "up to date before fetch, behind after: rebases",
			before:       model.TrackingEqual,
			after:        model.TrackingBehind,
			opts:         SyncOptions{UpdateLocal: true},
			wantPlanSkip: SyncReasonAlreadyUpToDate,
			wantCalls:    []string{"fetch:/repo", "pull:/repo"},
			wantOutcome:  SyncOutcomeRebased,
		},
		{
			name:        "behind before fetch, up to date after: skips the rebase",
			before:      model.TrackingBehind,
			after:       model.TrackingEqual,
			opts:        SyncOptions{UpdateLocal: true},
			wantCalls:   []string{"fetch:/repo"},
			wantOutcome: SyncOutcomeSkippedLocalUpdate,
			wantSkip:    SyncReasonAlreadyUpToDate,
		},
		{
			name:        "ahead before fetch, diverged after: does not push",
			before:      model.TrackingAhead,
			after:       model.TrackingDiverged,
			opts:        SyncOptions{UpdateLocal: true, PushLocal: true},
			wantCalls:   []string{"fetch:/repo"},
			wantOutcome: SyncOutcomeSkippedLocalUpdate,
			wantSkip:    "branch has diverged (use --force to rebase anyway)",
		},
		{
			name:         "protected branch stays skipped when the fetch finds new commits",
			branch:       "main",
			before:       model.TrackingEqual,
			after:        model.TrackingBehind,
			opts:         SyncOptions{UpdateLocal: true, ProtectedBranches: []string{"main"}},
			wantPlanSkip: `branch "main" is protected`,
			wantCalls:    []string{"fetch:/repo"},
			wantOutcome:  SyncOutcomeSkippedLocalUpdate,
			wantSkip:     `branch "main" is protected`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &fetchTransitionAdapter{planAdapter: &planAdapter{}, branch: tc.branch, before: tc.before, after: tc.after}
			eng := newPlanExecEngine(adapter)
			entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

			plan, executed := eng.planAndExecute(t, entry, tc.opts)

			if plan.SkipReason != tc.wantPlanSkip {
				t.Fatalf("plan skip reason = %q, want %q (plan: %+v)", plan.SkipReason, tc.wantPlanSkip, plan)
			}
			if !slices.Equal(adapter.calls, tc.wantCalls) {
				t.Fatalf("adapter calls = %v, want %v", adapter.calls, tc.wantCalls)
			}
			if !executed.OK || executed.Outcome != tc.wantOutcome {
				t.Fatalf("executed outcome = %q ok=%v, want %q (result: %+v)", executed.Outcome, executed.OK, tc.wantOutcome, executed)
			}
			if executed.SkipReason != tc.wantSkip {
				t.Fatalf("executed skip reason = %q, want %q", executed.SkipReason, tc.wantSkip)
			}
			wantError := ""
			if tc.wantSkip != "" {
				wantError = SyncErrorSkippedLocalUpdatePrefix + tc.wantSkip
			}
			if executed.Error != wantError {
				t.Fatalf("executed error = %q, want %q", executed.Error, wantError)
			}
		})
	}
}
