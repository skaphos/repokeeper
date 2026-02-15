package engine

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

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
			got := pullRebaseSkipReason(tc.status, tc.dirty, tc.force, tc.prot, tc.allow)
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
