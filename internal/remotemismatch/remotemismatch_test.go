// SPDX-License-Identifier: MIT
package remotemismatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/v2/internal/model"
	"github.com/skaphos/repokeeper/v2/internal/registry"
	"github.com/skaphos/repokeeper/v2/internal/vcs"
)

type adapterStub struct {
	setRemoteCalls []string
	setRemoteErr   error
}

func (a *adapterStub) Name() string                                            { return "stub" }
func (a *adapterStub) IsRepo(context.Context, string) (bool, error)            { return true, nil }
func (a *adapterStub) IsBare(context.Context, string) (bool, error)            { return false, nil }
func (a *adapterStub) Remotes(context.Context, string) ([]model.Remote, error) { return nil, nil }
func (a *adapterStub) Head(context.Context, string) (model.Head, error)        { return model.Head{}, nil }
func (a *adapterStub) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return nil, nil
}
func (a *adapterStub) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{}, nil
}
func (a *adapterStub) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (a *adapterStub) Fetch(context.Context, string) error                 { return nil }
func (a *adapterStub) PullRebase(context.Context, string) error            { return nil }
func (a *adapterStub) Push(context.Context, string) error                  { return nil }
func (a *adapterStub) SetUpstream(context.Context, string, string, string) error {
	return nil
}
func (a *adapterStub) SetRemoteURL(_ context.Context, dir, remote, remoteURL string) error {
	a.setRemoteCalls = append(a.setRemoteCalls, dir+":"+remote+":"+remoteURL)
	return a.setRemoteErr
}
func (a *adapterStub) StashPush(context.Context, string, string) (bool, error)   { return false, nil }
func (a *adapterStub) StashPop(context.Context, string) error                    { return nil }
func (a *adapterStub) ResetHard(context.Context, string) error                   { return nil }
func (a *adapterStub) CleanFD(context.Context, string) error                     { return nil }
func (a *adapterStub) Clone(context.Context, string, string, string, bool) error { return nil }
func (a *adapterStub) NormalizeURL(rawURL string) string                         { return rawURL }
func (a *adapterStub) PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) == 0 {
		return ""
	}
	return remoteNames[0]
}

func TestParseReconcileMode(t *testing.T) {
	mode, err := ParseReconcileMode("")
	if err != nil || mode != ReconcileNone {
		t.Fatalf("expected empty mode to map to none, got %q (%v)", mode, err)
	}
	mode, err = ParseReconcileMode("registry")
	if err != nil || mode != ReconcileRegistry {
		t.Fatalf("expected registry mode, got %q (%v)", mode, err)
	}
	mode, err = ParseReconcileMode("GIT")
	if err != nil || mode != ReconcileGit {
		t.Fatalf("expected git mode, got %q (%v)", mode, err)
	}
	if _, err := ParseReconcileMode("invalid"); err == nil {
		t.Fatal("expected invalid mode to error")
	}
}

func TestBuildPlansAndApplyRegistry(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: "/tmp/repo-a", RemoteURL: "git@github.com:other/repo-a.git"},
		},
	}
	repos := []model.RepoStatus{
		{
			RepoID:        "github.com/org/repo-a",
			Path:          "/tmp/repo-a",
			PrimaryRemote: "origin",
			Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/repo-a.git"}},
		},
	}

	plans := BuildPlans(repos, reg, &adapterStub{}, ReconcileRegistry)
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans))
	}
	if plans[0].Action == "" {
		t.Fatalf("expected planned action, got %+v", plans[0])
	}

	fixedNow := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := ApplyPlans(context.Background(), plans, reg, ReconcileRegistry, &adapterStub{}, func() time.Time { return fixedNow }); err != nil {
		t.Fatalf("apply registry plans: %v", err)
	}
	if got := reg.Entries[0].RemoteURL; got != "git@github.com:org/repo-a.git" {
		t.Fatalf("expected registry url update, got %q", got)
	}
}

func TestBuildPlansSkipsAndSelection(t *testing.T) {
	repos := []model.RepoStatus{
		{
			RepoID:        "github.com/org/repo-a",
			Path:          "/tmp/repo-a",
			PrimaryRemote: "origin",
			Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/repo-a.git"}},
		},
		{
			RepoID:        "github.com/org/repo-b",
			Path:          "/tmp/repo-b",
			PrimaryRemote: "origin",
			Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/repo-b.git"}},
		},
	}
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: "/tmp/repo-a", RemoteURL: "git@github.com:other/repo-a.git"},
			{RepoID: "github.com/org/repo-b", Path: "/tmp/other-path", RemoteURL: "git@github.com:other/repo-b.git"},
		},
	}
	adapter := &adapterStub{}

	if got := BuildPlans(repos, reg, adapter, ReconcileNone); got != nil {
		t.Fatalf("expected nil plans for none mode, got %+v", got)
	}
	if got := BuildPlans(repos, nil, adapter, ReconcileRegistry); got != nil {
		t.Fatalf("expected nil plans for nil registry, got %+v", got)
	}
	if got := BuildPlans(repos, reg, nil, ReconcileRegistry); got != nil {
		t.Fatalf("expected nil plans for nil adapter, got %+v", got)
	}

	plans := BuildPlans(repos, reg, adapter, ReconcileGit)
	if len(plans) != 2 {
		t.Fatalf("expected two plans, got %d", len(plans))
	}
	if plans[1].EntryIndex != 1 {
		t.Fatalf("expected repo-id fallback match to second entry, got %+v", plans[1])
	}
}

// normalizingAdapter normalizes URLs like the real git adapter, so repo IDs
// derive from remote URLs the way they do in a live status.
type normalizingAdapter struct {
	adapterStub
}

func (a *normalizingAdapter) NormalizeURL(rawURL string) string {
	return vcs.NewGitURLNormalizer().NormalizeURL(rawURL)
}

// A real remote URL change alters the live repo ID, so the checkout must be
// bound to its registry entry by path rather than by the new ID (#361).
func TestBuildPlansFollowsCheckoutAcrossRemoteURLChange(t *testing.T) {
	adapter := &normalizingAdapter{}
	oldURL := "https://host/old.git"
	newURL := "https://host/new.git"
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: adapter.NormalizeURL(oldURL), Path: "/tmp/repo", RemoteURL: oldURL},
		},
	}
	repos := []model.RepoStatus{
		{
			RepoID:        adapter.NormalizeURL(newURL),
			Path:          "/tmp/repo",
			PrimaryRemote: "origin",
			Remotes:       []model.Remote{{Name: "origin", URL: newURL}},
		},
	}

	plans := BuildPlans(repos, reg, adapter, ReconcileRegistry)
	if len(plans) != 1 {
		t.Fatalf("expected one plan for the changed remote, got %+v", plans)
	}
	if plans[0].EntryIndex != 0 || plans[0].RegistryURL != oldURL || plans[0].RepoRemoteURL != newURL {
		t.Fatalf("expected plan against the same-path entry, got %+v", plans[0])
	}

	if err := ApplyPlans(context.Background(), plans, reg, ReconcileRegistry, adapter, nil); err != nil {
		t.Fatalf("apply registry plans: %v", err)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("expected the entry to be updated in place, got %+v", reg.Entries)
	}
	entry := reg.Entries[0]
	if entry.RemoteURL != newURL {
		t.Fatalf("expected registry remote_url %q, got %q", newURL, entry.RemoteURL)
	}
	if entry.RepoID != adapter.NormalizeURL(newURL) {
		t.Fatalf("expected repo_id to follow the accepted remote, got %q", entry.RepoID)
	}
	if again := BuildPlans(repos, reg, adapter, ReconcileRegistry); len(again) != 0 {
		t.Fatalf("expected no mismatch after accepting the remote, got %+v", again)
	}
}

// Git mode restores the registry URL on the checkout, so the entry's identity
// already matches the remote it will have and must not change.
func TestBuildPlansGitModeAcrossRemoteURLChange(t *testing.T) {
	adapter := &normalizingAdapter{}
	oldURL := "https://host/old.git"
	newURL := "https://host/new.git"
	oldID := adapter.NormalizeURL(oldURL)
	reg := &registry.Registry{
		Entries: []registry.Entry{{RepoID: oldID, Path: "/tmp/repo", RemoteURL: oldURL}},
	}
	repos := []model.RepoStatus{{
		RepoID:        adapter.NormalizeURL(newURL),
		Path:          "/tmp/repo",
		PrimaryRemote: "origin",
		Remotes:       []model.Remote{{Name: "origin", URL: newURL}},
	}}

	plans := BuildPlans(repos, reg, adapter, ReconcileGit)
	if len(plans) != 1 {
		t.Fatalf("expected one plan for the changed remote, got %+v", plans)
	}
	if err := ApplyPlans(context.Background(), plans, reg, ReconcileGit, adapter, nil); err != nil {
		t.Fatalf("apply git plans: %v", err)
	}
	if want := "/tmp/repo:origin:" + oldURL; len(adapter.setRemoteCalls) != 1 || adapter.setRemoteCalls[0] != want {
		t.Fatalf("expected set-url %q, got %v", want, adapter.setRemoteCalls)
	}
	if reg.Entries[0].RepoID != oldID || reg.Entries[0].RemoteURL != oldURL {
		t.Fatalf("expected registry entry unchanged in git mode, got %+v", reg.Entries[0])
	}
}

func TestApplyPlansGit(t *testing.T) {
	plans := []Plan{
		{
			Path:          "/tmp/repo-a",
			PrimaryRemote: "origin",
			RegistryURL:   "git@github.com:org/repo-a.git",
		},
	}
	adapter := &adapterStub{}
	if err := ApplyPlans(context.Background(), plans, &registry.Registry{}, ReconcileGit, adapter, nil); err != nil {
		t.Fatalf("apply git plans: %v", err)
	}
	if len(adapter.setRemoteCalls) != 1 {
		t.Fatalf("expected one set-remote call, got %d", len(adapter.setRemoteCalls))
	}

	adapter = &adapterStub{setRemoteErr: errors.New("boom")}
	if err := ApplyPlans(context.Background(), plans, &registry.Registry{}, ReconcileGit, adapter, nil); err == nil {
		t.Fatal("expected git apply error")
	}
}

func TestApplyPlansEdgeModes(t *testing.T) {
	if err := ApplyPlans(context.Background(), nil, nil, ReconcileNone, nil, nil); err != nil {
		t.Fatalf("expected empty plans to no-op, got %v", err)
	}

	plans := []Plan{{Path: "/tmp/repo-a", PrimaryRemote: "origin", RegistryURL: "git@github.com:org/repo-a.git"}}
	if err := ApplyPlans(context.Background(), plans, &registry.Registry{}, ReconcileGit, nil, nil); err == nil {
		t.Fatal("expected nil adapter error for git mode")
	}

	reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "r1", Path: "/tmp/repo-a", RemoteURL: "before"}}}
	if err := ApplyPlans(context.Background(), []Plan{{EntryIndex: 9, RepoRemoteURL: "after"}}, reg, ReconcileRegistry, nil, nil); err != nil {
		t.Fatalf("expected out-of-range entry index to be ignored, got %v", err)
	}
	if got := reg.Entries[0].RemoteURL; got != "before" {
		t.Fatalf("expected unchanged entry for invalid index, got %q", got)
	}
}
