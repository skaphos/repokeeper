package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type planAdapter struct {
	fetchErrByDir map[string]error
	pushErrByDir  map[string]error
	pullErrByDir  map[string]error
	cloneErrByDir map[string]error
	stashErrByDir map[string]error
	popErrByDir   map[string]error
	stashCreated  bool
	calls         []string
}

func (p *planAdapter) Name() string                                            { return "plan" }
func (p *planAdapter) IsRepo(context.Context, string) (bool, error)            { return true, nil }
func (p *planAdapter) IsBare(context.Context, string) (bool, error)            { return false, nil }
func (p *planAdapter) Remotes(context.Context, string) ([]model.Remote, error) { return nil, nil }
func (p *planAdapter) Head(context.Context, string) (model.Head, error)        { return model.Head{}, nil }
func (p *planAdapter) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return &model.Worktree{}, nil
}
func (p *planAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingNone}, nil
}
func (p *planAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (p *planAdapter) Fetch(_ context.Context, dir string) error {
	p.calls = append(p.calls, "fetch:"+dir)
	return p.fetchErrByDir[dir]
}
func (p *planAdapter) PullRebase(_ context.Context, dir string) error {
	p.calls = append(p.calls, "pull:"+dir)
	return p.pullErrByDir[dir]
}
func (p *planAdapter) Push(_ context.Context, dir string) error {
	p.calls = append(p.calls, "push:"+dir)
	return p.pushErrByDir[dir]
}
func (p *planAdapter) SetUpstream(_ context.Context, dir, upstream, branch string) error {
	p.calls = append(p.calls, "set-upstream:"+dir+":"+upstream+":"+branch)
	return nil
}
func (p *planAdapter) SetRemoteURL(_ context.Context, dir, remote, remoteURL string) error {
	p.calls = append(p.calls, "set-remote-url:"+dir+":"+remote+":"+remoteURL)
	return nil
}
func (p *planAdapter) StashPush(_ context.Context, dir, _ string) (bool, error) {
	p.calls = append(p.calls, "stash-push:"+dir)
	return p.stashCreated, p.stashErrByDir[dir]
}
func (p *planAdapter) StashPop(_ context.Context, dir string) error {
	p.calls = append(p.calls, "stash-pop:"+dir)
	return p.popErrByDir[dir]
}
func (p *planAdapter) Clone(_ context.Context, _ string, targetPath, _ string, _ bool) error {
	p.calls = append(p.calls, "clone:"+targetPath)
	return p.cloneErrByDir[targetPath]
}
func (p *planAdapter) NormalizeURL(rawURL string) string { return rawURL }
func (p *planAdapter) PrimaryRemote(_ []string) string   { return "origin" }

func TestPullRebaseSkipReasonTable(t *testing.T) {
	tests := []struct {
		name   string
		status *model.RepoStatus
		dirty  bool
		force  bool
		prot   []string
		allow  bool
		want   string
	}{
		{name: "nil", status: nil, want: "unknown status"},
		{name: "bare", status: &model.RepoStatus{Bare: true}, want: "bare repository"},
		{name: "detached", status: &model.RepoStatus{Head: model.Head{Detached: true}}, want: "detached HEAD"},
		{
			name:   "protected",
			status: &model.RepoStatus{Head: model.Head{Branch: "main"}},
			prot:   []string{"main"},
			want:   "branch \"main\" is protected",
		},
		{
			name:   "unknown dirty",
			status: &model.RepoStatus{Head: model.Head{Branch: "main"}},
			allow:  true,
			prot:   []string{"release/*"},
			want:   "dirty state unknown",
		},
		{
			name:   "dirty",
			status: &model.RepoStatus{Head: model.Head{Branch: "main"}, Worktree: &model.Worktree{Dirty: true}},
			prot:   []string{"release/*"},
			want:   "dirty working tree",
		},
		{
			name: "gone",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingGone},
			},
			prot: []string{"release/*"},
			want: "upstream no longer exists",
		},
		{
			name: "not tracking",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingNone},
			},
			prot: []string{"release/*"},
			want: "branch is not tracking an upstream",
		},
		{
			name: "not main",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "feature"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingBehind, Upstream: "origin/develop"},
			},
			prot: []string{"release/*"},
			want: "upstream \"origin/develop\" is not main",
		},
		{
			name: "ahead",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingAhead, Upstream: "origin/main"},
			},
			prot: []string{"release/*"},
			want: "branch has local commits to push",
		},
		{
			name: "diverged no force",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingDiverged, Upstream: "origin/main"},
			},
			prot: []string{"release/*"},
			want: "branch has diverged (use --force to rebase anyway)",
		},
		{
			name: "equal",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingEqual, Upstream: "origin/main"},
			},
			prot: []string{"release/*"},
			want: "already up to date",
		},
		{
			name: "rebase allowed",
			status: &model.RepoStatus{
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingBehind, Upstream: "origin/main"},
			},
			prot: []string{"release/*"},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pullRebaseSkipReason(tc.status, PullRebasePolicyOptions{
				MainBranch:           "main",
				RebaseDirty:          tc.dirty,
				Force:                tc.force,
				ProtectedBranches:    tc.prot,
				AllowProtectedRebase: tc.allow,
			})
			if got != tc.want {
				t.Fatalf("pullRebaseSkipReason() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFilterStatusDefaultKind(t *testing.T) {
	if !filterStatus(FilterKind("something-new"), model.RepoStatus{}, nil) {
		t.Fatal("expected unknown filter kind to default to true")
	}
}

func TestFindRegistryEntryForStatusAndReplace(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "repo", Path: "/first"},
			{RepoID: "repo", Path: "/second"},
		},
	}

	byExact := findRegistryEntryForStatus(reg, model.RepoStatus{RepoID: "repo", Path: "/second"})
	if byExact == nil || byExact.Path != "/second" {
		t.Fatalf("expected exact match, got %#v", byExact)
	}

	byRepo := findRegistryEntryForStatus(reg, model.RepoStatus{RepoID: "repo", Path: "/missing"})
	if byRepo == nil || byRepo.Path != "/first" {
		t.Fatalf("expected repo-id fallback, got %#v", byRepo)
	}

	updated := replaceRegistryEntry(reg.Entries, registry.Entry{RepoID: "repo", Path: "/second", RemoteURL: "git@github.com:org/repo.git"})
	if updated[1].RemoteURL == "" {
		t.Fatalf("expected replaced entry to be updated: %#v", updated[1])
	}

	notFound := replaceRegistryEntry(reg.Entries, registry.Entry{RepoID: "new", Path: "/new"})
	if len(notFound) != len(reg.Entries) {
		t.Fatalf("expected no append for missing entry, got len=%d", len(notFound))
	}
}

func TestHasRemoteMismatchCases(t *testing.T) {
	if hasRemoteMismatch(model.RepoStatus{RepoID: "github.com/org/repo"}, registry.Entry{}) {
		t.Fatal("expected false for blank registry remote")
	}
	if hasRemoteMismatch(model.RepoStatus{}, registry.Entry{RemoteURL: "git@github.com:org/repo.git"}) {
		t.Fatal("expected false for blank status repo id")
	}
	if hasRemoteMismatch(
		model.RepoStatus{RepoID: "github.com/org/repo"},
		registry.Entry{RemoteURL: "git@github.com:org/repo.git"},
	) {
		t.Fatal("expected normalized same remotes to match")
	}
	if !hasRemoteMismatch(
		model.RepoStatus{RepoID: "github.com/other/repo"},
		registry.Entry{RemoteURL: "git@github.com:org/repo.git"},
	) {
		t.Fatal("expected mismatch for differing remotes")
	}
}

func TestMatchesProtectedBranch(t *testing.T) {
	if matchesProtectedBranch("", []string{"main"}) {
		t.Fatal("expected blank branch to never match")
	}
	if matchesProtectedBranch("main", []string{""}) {
		t.Fatal("expected blank pattern to be ignored")
	}
	if !matchesProtectedBranch("release/v1", []string{"release/*"}) {
		t.Fatal("expected glob match for protected branch")
	}
	if matchesProtectedBranch("main", []string{"["}) {
		t.Fatal("expected invalid glob pattern to be ignored")
	}
}

func TestFilterStatusKindsAndSorts(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "repo-clean", Path: "/repos/clean", RemoteURL: "git@github.com:org/clean.git"},
			{RepoID: "github.com/other/dirty", Path: "/repos/dirty", RemoteURL: "git@github.com:org/dirty.git"},
			{RepoID: "missing", Path: "/repos/missing", Status: registry.StatusMissing},
		},
	}
	clean := model.RepoStatus{
		RepoID:   "github.com/org/clean",
		Path:     "/repos/clean",
		Worktree: &model.Worktree{Dirty: false},
		Tracking: model.Tracking{Status: model.TrackingEqual},
	}
	dirty := model.RepoStatus{
		RepoID:   "github.com/other/dirty",
		Path:     "/repos/dirty",
		Worktree: &model.Worktree{Dirty: true},
		Tracking: model.Tracking{Status: model.TrackingDiverged},
		Error:    "boom",
	}
	missing := model.RepoStatus{RepoID: "missing", Path: "/repos/missing", ErrorClass: "missing"}
	gone := model.RepoStatus{RepoID: "gone", Path: "/repos/gone", Tracking: model.Tracking{Status: model.TrackingGone}}

	if !filterStatus(FilterAll, clean, reg) {
		t.Fatal("expected all filter to include repo")
	}
	if !filterStatus(FilterErrors, dirty, reg) {
		t.Fatal("expected errors filter to include repo with error")
	}
	if !filterStatus(FilterDirty, dirty, reg) {
		t.Fatal("expected dirty filter to include dirty repo")
	}
	if !filterStatus(FilterClean, clean, reg) {
		t.Fatal("expected clean filter to include clean repo")
	}
	if !filterStatus(FilterMissing, missing, reg) {
		t.Fatal("expected missing filter to include missing repo")
	}
	if !filterStatus(FilterGone, gone, reg) {
		t.Fatal("expected gone filter to include gone repo")
	}
	if !filterStatus(FilterDiverged, dirty, reg) {
		t.Fatal("expected diverged filter to include diverged repo")
	}
	if !filterStatus(FilterRemoteMismatch, dirty, reg) {
		t.Fatal("expected remote mismatch filter to include mismatched repo")
	}

	repos := []model.RepoStatus{{RepoID: "b"}, {RepoID: "a"}}
	sortRepoStatuses(repos)
	if repos[0].RepoID != "a" {
		t.Fatalf("expected repos sorted by repo id, got %#v", repos)
	}

	results := []SyncResult{{RepoID: "b"}, {RepoID: "a"}}
	sortSyncResults(results)
	if results[0].RepoID != "a" {
		t.Fatalf("expected sync results sorted by repo id, got %#v", results)
	}
}

func TestExecuteSyncPlanAppliesPlannedActions(t *testing.T) {
	adapter := &planAdapter{
		stashCreated: true,
	}
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "clone", Path: "/repos/clone", RemoteURL: "git@github.com:org/clone.git", Branch: "main", Status: registry.StatusMissing},
		},
	}
	eng := &Engine{registry: reg, adapter: adapter}
	plan := []SyncResult{
		{RepoID: "fetch", Path: "/repos/fetch", OK: true, Error: "dry-run", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "rebase", Path: "/repos/rebase", OK: true, Error: "dry-run", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git stash push -u -m \"repokeeper: pre-rebase stash\" && git pull --rebase --no-recurse-submodules && git stash pop"},
		{RepoID: "push", Path: "/repos/push", OK: true, Error: "dry-run", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git push"},
		{RepoID: "clone", Path: "/repos/clone", OK: true, Error: "dry-run", Action: "git clone --branch main --single-branch git@github.com:org/clone.git /repos/clone"},
		{RepoID: "skip", Path: "/repos/skip", OK: true, Error: "skipped-no-upstream"},
	}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: true})
	if err != nil {
		t.Fatalf("execute sync plan: %v", err)
	}
	if len(results) != len(plan) {
		t.Fatalf("expected %d results, got %d", len(plan), len(results))
	}
	outcome := map[string]SyncOutcome{}
	for _, res := range results {
		outcome[res.RepoID] = res.Outcome
	}
	if outcome["fetch"] != SyncOutcomeFetched {
		t.Fatalf("expected fetched outcome, got %q", outcome["fetch"])
	}
	if outcome["rebase"] != SyncOutcomeStashedRebased {
		t.Fatalf("expected stashed_rebased outcome, got %q", outcome["rebase"])
	}
	if outcome["push"] != SyncOutcomePushed {
		t.Fatalf("expected pushed outcome, got %q", outcome["push"])
	}
	if outcome["clone"] != SyncOutcomeCheckoutMissing {
		t.Fatalf("expected checkout_missing outcome, got %q", outcome["clone"])
	}
	if reg.Entries[0].Status != registry.StatusPresent {
		t.Fatalf("expected clone entry to be marked present, got %q", reg.Entries[0].Status)
	}
}

func TestExecuteSyncPlanStopsOnFailureWhenConfigured(t *testing.T) {
	adapter := &planAdapter{
		fetchErrByDir: map[string]error{
			"/repos/a": errors.New("network down"),
		},
	}
	eng := &Engine{
		registry: &registry.Registry{},
		adapter:  adapter,
	}
	plan := []SyncResult{
		{RepoID: "a", Path: "/repos/a", OK: true, Error: "dry-run", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "b", Path: "/repos/b", OK: true, Error: "dry-run", Action: "git push"},
	}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected execution to stop after first failure, got %d results", len(results))
	}
	if results[0].OK || results[0].Outcome != SyncOutcomeFailedFetch {
		t.Fatalf("expected failed_fetch result, got %#v", results[0])
	}
}

func TestPullRebaseSkipReasonUsesConfiguredMainBranch(t *testing.T) {
	status := &model.RepoStatus{
		Head:     model.Head{Branch: "develop"},
		Worktree: &model.Worktree{Dirty: false},
		Tracking: model.Tracking{Status: model.TrackingBehind, Upstream: "origin/develop"},
	}
	if got := pullRebaseSkipReason(status, PullRebasePolicyOptions{
		MainBranch:           "develop",
		RebaseDirty:          false,
		Force:                false,
		ProtectedBranches:    nil,
		AllowProtectedRebase: true,
	}); got != "" {
		t.Fatalf("expected configured main branch to allow rebase, got %q", got)
	}
	if got := pullRebaseSkipReason(status, PullRebasePolicyOptions{
		MainBranch:           "main",
		RebaseDirty:          false,
		Force:                false,
		ProtectedBranches:    nil,
		AllowProtectedRebase: true,
	}); got != "upstream \"origin/develop\" is not main" {
		t.Fatalf("expected branch mismatch reason, got %q", got)
	}
}

func TestNewInitializesDefaultAdapter(t *testing.T) {
	eng := New(&config.Config{}, &registry.Registry{}, nil)
	if eng.Adapter() == nil {
		t.Fatal("expected engine.New to set default adapter when nil")
	}
}
