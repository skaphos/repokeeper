// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/obs"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// dirtyBehindAdapter reports a dirty worktree that is behind its upstream, so
// --update-local classifies it as a "dirty working tree" skip (without
// --rebase-dirty) or a stash+rebase candidate (with --rebase-dirty).
type dirtyBehindAdapter struct {
	*planAdapter
}

func (a *dirtyBehindAdapter) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return &model.Worktree{Dirty: true}, nil
}

func (a *dirtyBehindAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingBehind, Upstream: "origin/main"}, nil
}

func newPlanExecEngine(adapter vcs.Adapter) *Engine {
	return &Engine{
		cfg:        &config.Config{},
		registry:   &registry.Registry{},
		adapter:    adapter,
		classifier: vcs.NewGitErrorClassifier(),
		normalizer: vcs.NewGitURLNormalizer(),
		logger:     obs.NopLogger(),
	}
}

// planAndExecute mirrors the production sync flow: plan under dry-run, then apply
// the plan through the execute path.
func (e *Engine) planAndExecute(t *testing.T, entry registry.Entry, opts SyncOptions) (SyncResult, SyncResult) {
	t.Helper()
	planOpts := opts
	planOpts.DryRun = true
	plan := e.runSyncDryRun(context.Background(), entry, planOpts)
	execOpts := opts
	execOpts.DryRun = false
	execOpts.ContinueOnError = true
	results, err := e.ExecuteSyncPlanWithCallbacks(context.Background(), []SyncResult{plan}, execOpts, nil, nil)
	if err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 executed result, got %d", len(results))
	}
	return plan, results[0]
}

// Finding 1: --update-local must not drop the fetch for skip results. A dirty
// repo skips the local update, but the fetch still runs so remote-tracking refs
// do not go stale relative to a plain sync.
func TestUpdateLocalStillFetchesDirtySkip(t *testing.T) {
	adapter := &dirtyBehindAdapter{planAdapter: &planAdapter{}}
	eng := newPlanExecEngine(adapter)
	entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

	plan, executed := eng.planAndExecute(t, entry, SyncOptions{UpdateLocal: true})

	if !plan.Planned {
		t.Fatalf("expected skip plan to be Planned so the fetch runs: %+v", plan)
	}
	if len(plan.steps) != 1 || plan.steps[0] != syncStepFetch {
		t.Fatalf("expected fetch-only steps in skip plan, got %v", plan.steps)
	}
	if plan.SkipReason != SyncReasonDirtyWorkingTree {
		t.Fatalf("expected dirty skip reason preserved, got %q", plan.SkipReason)
	}
	if len(adapter.calls) != 1 || adapter.calls[0] != "fetch:/repo" {
		t.Fatalf("expected exactly one fetch during skipped --update-local, got %v", adapter.calls)
	}
	if executed.Outcome != SyncOutcomeSkippedLocalUpdate || !executed.OK {
		t.Fatalf("expected skipped-local-update outcome preserved, got %+v", executed)
	}
	if executed.SkipReason != SyncReasonDirtyWorkingTree {
		t.Fatalf("expected executed skip reason preserved, got %q", executed.SkipReason)
	}
}

// Finding 2: an hg fetch step must actually pull. The adapter reports local
// update unsupported (like hg) and a non-git fetch action; the execute path must
// still route the fetch through adapter.Fetch rather than substring-matching the
// git action text.
func TestUpdateLocalFetchesUnsupportedBackend(t *testing.T) {
	adapter := &unsupportedLocalUpdateAdapter{planAdapter: &planAdapter{}}
	eng := newPlanExecEngine(adapter)
	entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "https://hg.example/repo", Status: registry.StatusPresent}

	plan, executed := eng.planAndExecute(t, entry, SyncOptions{UpdateLocal: true})

	if plan.Action != "hg pull" {
		t.Fatalf("expected hg fetch action in plan, got %q", plan.Action)
	}
	if !plan.Planned || len(plan.steps) != 1 || plan.steps[0] != syncStepFetch {
		t.Fatalf("expected planned fetch step for unsupported backend, got %+v", plan)
	}
	if len(adapter.calls) != 1 || adapter.calls[0] != "fetch:/repo" {
		t.Fatalf("expected fetch to run for unsupported backend, got %v", adapter.calls)
	}
	if executed.Outcome != SyncOutcomeSkippedLocalUpdate || !executed.OK {
		t.Fatalf("expected skipped-local-update outcome, got %+v", executed)
	}
}

// Finding 3: --rebase-dirty must stash in the real execute path. On a dirty repo
// the plan must emit stash push/pop around the rebase, and executing it must run
// the stash operations (git pull --rebase has no autostash here).
func TestUpdateLocalRebaseDirtyStashesInExecutePath(t *testing.T) {
	adapter := &dirtyBehindAdapter{planAdapter: &planAdapter{stashCreated: true}}
	eng := newPlanExecEngine(adapter)
	entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

	plan, executed := eng.planAndExecute(t, entry, SyncOptions{UpdateLocal: true, RebaseDirty: true})

	wantSteps := []syncStep{syncStepFetch, syncStepStashPush, syncStepPullRebase, syncStepStashPop}
	if len(plan.steps) != len(wantSteps) {
		t.Fatalf("expected stash steps planned, got %v", plan.steps)
	}
	for i, s := range wantSteps {
		if plan.steps[i] != s {
			t.Fatalf("unexpected plan step %d: got %q want %q (all: %v)", i, plan.steps[i], s, plan.steps)
		}
	}
	wantCalls := []string{"fetch:/repo", "stash-push:/repo", "pull:/repo", "stash-pop:/repo"}
	if len(adapter.calls) != len(wantCalls) {
		t.Fatalf("expected stash+rebase call sequence, got %v", adapter.calls)
	}
	for i, c := range wantCalls {
		if adapter.calls[i] != c {
			t.Fatalf("unexpected call %d: got %q want %q (all: %v)", i, adapter.calls[i], c, adapter.calls)
		}
	}
	if executed.Outcome != SyncOutcomeStashedRebased || !executed.OK {
		t.Fatalf("expected stashed_rebased outcome, got %+v", executed)
	}
}

// Finding 5: ApplyRemoteMismatchPlans must use the engine's injected adapter, not
// a hardcoded git adapter, so custom/test adapters and --vcs git,hg are honored.
func TestApplyRemoteMismatchPlansUsesInjectedAdapter(t *testing.T) {
	adapter := &planAdapter{}
	eng := newPlanExecEngine(adapter)

	plans := []RemoteMismatchPlan{{
		RepoID:        "repo",
		Path:          "/repo",
		PrimaryRemote: "origin",
		RegistryURL:   "git@github.com:org/repo.git",
		EntryIndex:    0,
	}}
	if err := eng.ApplyRemoteMismatchPlans(context.Background(), plans, RemoteMismatchReconcileGit); err != nil {
		t.Fatalf("apply remote mismatch plans: %v", err)
	}
	want := "set-remote-url:/repo:origin:git@github.com:org/repo.git"
	if len(adapter.calls) != 1 || adapter.calls[0] != want {
		t.Fatalf("expected injected adapter to receive set-remote-url, got %v", adapter.calls)
	}
}

// Finding 7: ParseFilterKind validates filter values, defaulting empty to
// FilterAll and rejecting unknown values.
func TestParseFilterKind(t *testing.T) {
	cases := []struct {
		raw     string
		want    FilterKind
		wantErr bool
	}{
		{raw: "", want: FilterAll},
		{raw: "  ", want: FilterAll},
		{raw: "dirty", want: FilterDirty},
		{raw: "REMOTE-MISMATCH", want: FilterRemoteMismatch},
		{raw: "garbage", wantErr: true},
		{raw: "dir", wantErr: true},
	}
	for _, tc := range cases {
		got, err := ParseFilterKind(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseFilterKind(%q): expected error, got %q", tc.raw, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseFilterKind(%q): unexpected error %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseFilterKind(%q): got %q want %q", tc.raw, got, tc.want)
		}
	}
}

// Finding 7 (defense-in-depth): an unknown filter must fail closed in the sync
// path so it matches nothing rather than every repository.
func TestPrepareSyncEntryUnknownFilterFailsClosed(t *testing.T) {
	eng := newPlanExecEngine(&planAdapter{})
	present := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

	queue, immediate := eng.prepareSyncEntry(context.Background(), present, SyncOptions{Filter: FilterKind("bogus")}, 0)
	if queue || immediate != nil {
		t.Fatalf("expected unknown filter to fail closed (no queue, no result), got queue=%v immediate=%v", queue, immediate)
	}

	// A known non-inspect filter (all) still matches.
	queue, immediate = eng.prepareSyncEntry(context.Background(), present, SyncOptions{Filter: FilterAll}, 0)
	if !queue || immediate != nil {
		t.Fatalf("expected FilterAll to queue the entry, got queue=%v immediate=%v", queue, immediate)
	}
}
