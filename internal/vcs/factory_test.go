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
		{name: "multi", raw: "git,hg,bzr", want: []string{"git", "hg", "bzr"}},
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
	name       string
	repoPaths  map[string]bool
	fetchCalls int
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

func TestMultiAdapterRoutesByPath(t *testing.T) {
	gitAdapter := &multiStubAdapter{name: "git", repoPaths: map[string]bool{"/git-repo": true}}
	hgAdapter := &multiStubAdapter{name: "hg", repoPaths: map[string]bool{"/hg-repo": true}}
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
