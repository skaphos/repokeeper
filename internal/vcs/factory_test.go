package vcs

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestParseAdapterSelection(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   []string
		hasErr bool
	}{
		{name: "default", raw: "", want: []string{"git"}},
		{name: "single", raw: "hg", want: []string{"hg"}},
		{name: "multi", raw: "git,hg", want: []string{"git", "hg"}},
		{name: "dedupe", raw: "git,git,hg", want: []string{"git", "hg"}},
		{name: "invalid", raw: "svn", hasErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAdapterSelection(tc.raw)
			if tc.hasErr {
				if err == nil {
					t.Fatalf("expected parse error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("selection len = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("selection[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

type multiStubAdapter struct {
	name             string
	repoPaths        map[string]bool
	fetchCalls       int
	localUpdateOK    bool
	localUpdateWhy   string
	fetchActionLabel string
}

func (m *multiStubAdapter) Name() string { return m.name }
func (m *multiStubAdapter) IsRepo(_ context.Context, dir string) (bool, error) {
	return m.repoPaths[dir], nil
}
func (m *multiStubAdapter) IsBare(context.Context, string) (bool, error) { return false, nil }
func (m *multiStubAdapter) Remotes(context.Context, string) ([]model.Remote, error) {
	return nil, nil
}
func (m *multiStubAdapter) Head(context.Context, string) (model.Head, error) {
	return model.Head{}, nil
}
func (m *multiStubAdapter) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return &model.Worktree{}, nil
}
func (m *multiStubAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingNone}, nil
}
func (m *multiStubAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (m *multiStubAdapter) Fetch(context.Context, string) error {
	m.fetchCalls++
	return nil
}
func (m *multiStubAdapter) PullRebase(context.Context, string) error {
	return errors.New("unsupported")
}
func (m *multiStubAdapter) Push(context.Context, string) error { return errors.New("unsupported") }
func (m *multiStubAdapter) SetUpstream(context.Context, string, string, string) error {
	return errors.New("unsupported")
}
func (m *multiStubAdapter) SetRemoteURL(context.Context, string, string, string) error {
	return errors.New("unsupported")
}
func (m *multiStubAdapter) StashPush(context.Context, string, string) (bool, error) {
	return false, errors.New("unsupported")
}
func (m *multiStubAdapter) StashPop(context.Context, string) error { return errors.New("unsupported") }
func (m *multiStubAdapter) Clone(context.Context, string, string, string, bool) error {
	return errors.New("unsupported")
}
func (m *multiStubAdapter) NormalizeURL(rawURL string) string { return rawURL }
func (m *multiStubAdapter) PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) > 0 {
		return remoteNames[0]
	}
	return ""
}
func (m *multiStubAdapter) SupportsLocalUpdate(context.Context, string) (bool, string, error) {
	return m.localUpdateOK, m.localUpdateWhy, nil
}
func (m *multiStubAdapter) FetchAction(context.Context, string) (string, error) {
	if m.fetchActionLabel == "" {
		return "git fetch --all --prune --prune-tags --no-recurse-submodules", nil
	}
	return m.fetchActionLabel, nil
}

func TestMultiAdapterRoutesByPath(t *testing.T) {
	gitAdapter := &multiStubAdapter{name: "git", repoPaths: map[string]bool{"/git-repo": true}, localUpdateOK: true}
	hgAdapter := &multiStubAdapter{name: "hg", repoPaths: map[string]bool{"/hg-repo": true}, localUpdateOK: false}
	multi := &MultiAdapter{
		adapters: []Adapter{gitAdapter, hgAdapter},
		byPath:   map[string]Adapter{},
	}

	ok, err := multi.IsRepo(context.Background(), "/git-repo")
	if err != nil || !ok {
		t.Fatalf("expected git repo detection, got ok=%v err=%v", ok, err)
	}
	if err := multi.Fetch(context.Background(), "/git-repo"); err != nil {
		t.Fatalf("expected git fetch route, got %v", err)
	}

	ok, err = multi.IsRepo(context.Background(), "/hg-repo")
	if err != nil || !ok {
		t.Fatalf("expected hg repo detection, got ok=%v err=%v", ok, err)
	}
	if err := multi.Fetch(context.Background(), "/hg-repo"); err != nil {
		t.Fatalf("expected hg fetch route, got %v", err)
	}

	if gitAdapter.fetchCalls != 1 || hgAdapter.fetchCalls != 1 {
		t.Fatalf("unexpected fetch call routing git=%d hg=%d", gitAdapter.fetchCalls, hgAdapter.fetchCalls)
	}
}

func TestMultiAdapterRoutesCapabilityMethodsByPath(t *testing.T) {
	gitAdapter := &multiStubAdapter{
		name:             "git",
		repoPaths:        map[string]bool{"/git-repo": true},
		localUpdateOK:    true,
		fetchActionLabel: "git fetch --all --prune --prune-tags --no-recurse-submodules",
	}
	hgAdapter := &multiStubAdapter{
		name:             "hg",
		repoPaths:        map[string]bool{"/hg-repo": true},
		localUpdateOK:    false,
		localUpdateWhy:   "local update unsupported for vcs hg",
		fetchActionLabel: "hg pull",
	}
	multi := &MultiAdapter{
		adapters: []Adapter{gitAdapter, hgAdapter},
		byPath:   map[string]Adapter{},
	}

	ok, reason, err := multi.SupportsLocalUpdate(context.Background(), "/hg-repo")
	if err != nil {
		t.Fatalf("supports local update returned error: %v", err)
	}
	if ok {
		t.Fatal("expected hg local update unsupported")
	}
	if reason != "local update unsupported for vcs hg" {
		t.Fatalf("unexpected local update reason: %q", reason)
	}

	action, err := multi.FetchAction(context.Background(), "/hg-repo")
	if err != nil {
		t.Fatalf("fetch action returned error: %v", err)
	}
	if action != "hg pull" {
		t.Fatalf("unexpected fetch action: %q", action)
	}
}

func TestNewAdapterForSelection(t *testing.T) {
	adapter, err := NewAdapterForSelection("git")
	if err != nil {
		t.Fatalf("git selection error: %v", err)
	}
	if adapter.Name() != "git" {
		t.Fatalf("unexpected adapter: %s", adapter.Name())
	}

	adapter, err = NewAdapterForSelection("hg")
	if err != nil {
		t.Fatalf("hg selection error: %v", err)
	}
	if adapter.Name() != "hg" {
		t.Fatalf("unexpected adapter: %s", adapter.Name())
	}

	adapter, err = NewAdapterForSelection("git,hg")
	if err != nil {
		t.Fatalf("multi selection error: %v", err)
	}
	if adapter.Name() != "multi" {
		t.Fatalf("unexpected adapter: %s", adapter.Name())
	}

	if _, err := NewAdapterForSelection("svn"); err == nil {
		t.Fatal("expected empty selection error")
	}
}

func TestMultiAdapterDelegatesAllMethods(t *testing.T) {
	adapter := &multiStubAdapter{
		name:             "git",
		repoPaths:        map[string]bool{"/repo": true},
		localUpdateOK:    true,
		fetchActionLabel: "git fetch --all --prune --prune-tags --no-recurse-submodules",
	}
	multi := &MultiAdapter{
		adapters: []Adapter{adapter},
		byPath:   map[string]Adapter{},
	}
	ctx := context.Background()

	if multi.Name() != "multi" {
		t.Fatalf("unexpected multi adapter name: %s", multi.Name())
	}
	if ok, err := multi.IsRepo(ctx, "/repo"); err != nil || !ok {
		t.Fatalf("IsRepo unexpected result: ok=%v err=%v", ok, err)
	}
	if _, err := multi.IsBare(ctx, "/repo"); err != nil {
		t.Fatalf("IsBare unexpected error: %v", err)
	}
	if _, err := multi.Remotes(ctx, "/repo"); err != nil {
		t.Fatalf("Remotes unexpected error: %v", err)
	}
	if _, err := multi.Head(ctx, "/repo"); err != nil {
		t.Fatalf("Head unexpected error: %v", err)
	}
	if _, err := multi.WorktreeStatus(ctx, "/repo"); err != nil {
		t.Fatalf("WorktreeStatus unexpected error: %v", err)
	}
	if _, err := multi.TrackingStatus(ctx, "/repo"); err != nil {
		t.Fatalf("TrackingStatus unexpected error: %v", err)
	}
	if _, err := multi.HasSubmodules(ctx, "/repo"); err != nil {
		t.Fatalf("HasSubmodules unexpected error: %v", err)
	}
	if err := multi.Fetch(ctx, "/repo"); err != nil {
		t.Fatalf("Fetch unexpected error: %v", err)
	}
	if err := multi.PullRebase(ctx, "/repo"); err == nil {
		t.Fatal("expected PullRebase unsupported error")
	}
	if err := multi.Push(ctx, "/repo"); err == nil {
		t.Fatal("expected Push unsupported error")
	}
	if err := multi.SetUpstream(ctx, "/repo", "origin/main", "main"); err == nil {
		t.Fatal("expected SetUpstream unsupported error")
	}
	if err := multi.SetRemoteURL(ctx, "/repo", "origin", "git@github.com:org/repo.git"); err == nil {
		t.Fatal("expected SetRemoteURL unsupported error")
	}
	if _, err := multi.StashPush(ctx, "/repo", "msg"); err == nil {
		t.Fatal("expected StashPush unsupported error")
	}
	if err := multi.StashPop(ctx, "/repo"); err == nil {
		t.Fatal("expected StashPop unsupported error")
	}
	if err := multi.Clone(ctx, "git@github.com:org/repo.git", "/tmp/repo", "main", false); err == nil {
		t.Fatal("expected Clone unsupported error")
	}
	if got := multi.NormalizeURL("git@github.com:org/repo.git"); got == "" {
		t.Fatal("expected normalized url")
	}
	if got := multi.PrimaryRemote([]string{"origin"}); got != "origin" {
		t.Fatalf("unexpected primary remote: %q", got)
	}
	if ok, _, err := multi.SupportsLocalUpdate(ctx, "/repo"); err != nil || !ok {
		t.Fatalf("SupportsLocalUpdate unexpected result: ok=%v err=%v", ok, err)
	}
	if action, err := multi.FetchAction(ctx, "/repo"); err != nil || action == "" {
		t.Fatalf("FetchAction unexpected result: action=%q err=%v", action, err)
	}
}
