package remotemismatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type adapterStub struct {
	setRemoteCalls []string
	setRemoteErr   error
}

func (a *adapterStub) Name() string { return "stub" }
func (a *adapterStub) IsRepo(context.Context, string) (bool, error) { return true, nil }
func (a *adapterStub) IsBare(context.Context, string) (bool, error) { return false, nil }
func (a *adapterStub) Remotes(context.Context, string) ([]model.Remote, error) { return nil, nil }
func (a *adapterStub) Head(context.Context, string) (model.Head, error) { return model.Head{}, nil }
func (a *adapterStub) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return nil, nil
}
func (a *adapterStub) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{}, nil
}
func (a *adapterStub) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (a *adapterStub) Fetch(context.Context, string) error                  { return nil }
func (a *adapterStub) PullRebase(context.Context, string) error             { return nil }
func (a *adapterStub) Push(context.Context, string) error                   { return nil }
func (a *adapterStub) SetUpstream(context.Context, string, string, string) error {
	return nil
}
func (a *adapterStub) SetRemoteURL(_ context.Context, dir, remote, remoteURL string) error {
	a.setRemoteCalls = append(a.setRemoteCalls, dir+":"+remote+":"+remoteURL)
	return a.setRemoteErr
}
func (a *adapterStub) StashPush(context.Context, string, string) (bool, error) { return false, nil }
func (a *adapterStub) StashPop(context.Context, string) error                   { return nil }
func (a *adapterStub) Clone(context.Context, string, string, string, bool) error {
	return nil
}
func (a *adapterStub) NormalizeURL(rawURL string) string { return rawURL }
func (a *adapterStub) PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) == 0 {
		return ""
	}
	return remoteNames[0]
}

func TestParseReconcileMode(t *testing.T) {
	mode, err := ParseReconcileMode("registry")
	if err != nil || mode != ReconcileRegistry {
		t.Fatalf("expected registry mode, got %q (%v)", mode, err)
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
