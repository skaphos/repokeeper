package vcs

import (
	"context"

	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
)

// Adapter defines the VCS operations RepoKeeper relies on.
// Git is the default adapter; other VCS are stretch goals.
type Adapter interface {
	Name() string
	IsRepo(ctx context.Context, dir string) (bool, error)
	IsBare(ctx context.Context, dir string) (bool, error)
	Remotes(ctx context.Context, dir string) ([]model.Remote, error)
	Head(ctx context.Context, dir string) (model.Head, error)
	WorktreeStatus(ctx context.Context, dir string) (*model.Worktree, error)
	TrackingStatus(ctx context.Context, dir string) (model.Tracking, error)
	HasSubmodules(ctx context.Context, dir string) (bool, error)
	Fetch(ctx context.Context, dir string) error
	PullRebase(ctx context.Context, dir string) error
	Push(ctx context.Context, dir string) error
	SetUpstream(ctx context.Context, dir, upstream, branch string) error
	StashPush(ctx context.Context, dir, message string) (bool, error)
	StashPop(ctx context.Context, dir string) error
	Clone(ctx context.Context, remoteURL, targetPath, branch string, mirror bool) error
	NormalizeURL(rawURL string) string
	PrimaryRemote(remoteNames []string) string
}

// GitAdapter implements Adapter using the git CLI via gitx.
type GitAdapter struct {
	Runner gitx.Runner
}

func NewGitAdapter(runner gitx.Runner) *GitAdapter {
	if runner == nil {
		runner = &gitx.GitRunner{}
	}
	return &GitAdapter{Runner: runner}
}

func (g *GitAdapter) Name() string { return "git" }

func (g *GitAdapter) IsRepo(ctx context.Context, dir string) (bool, error) {
	return gitx.IsRepo(ctx, g.Runner, dir)
}

func (g *GitAdapter) IsBare(ctx context.Context, dir string) (bool, error) {
	return gitx.IsBare(ctx, g.Runner, dir)
}

func (g *GitAdapter) Remotes(ctx context.Context, dir string) ([]model.Remote, error) {
	return gitx.Remotes(ctx, g.Runner, dir)
}

func (g *GitAdapter) Head(ctx context.Context, dir string) (model.Head, error) {
	return gitx.Head(ctx, g.Runner, dir)
}

func (g *GitAdapter) WorktreeStatus(ctx context.Context, dir string) (*model.Worktree, error) {
	return gitx.WorktreeStatus(ctx, g.Runner, dir)
}

func (g *GitAdapter) TrackingStatus(ctx context.Context, dir string) (model.Tracking, error) {
	return gitx.TrackingStatus(ctx, g.Runner, dir)
}

func (g *GitAdapter) HasSubmodules(ctx context.Context, dir string) (bool, error) {
	return gitx.HasSubmodules(ctx, g.Runner, dir)
}

func (g *GitAdapter) Fetch(ctx context.Context, dir string) error {
	return gitx.Fetch(ctx, g.Runner, dir)
}

func (g *GitAdapter) PullRebase(ctx context.Context, dir string) error {
	return gitx.PullRebase(ctx, g.Runner, dir)
}

func (g *GitAdapter) Push(ctx context.Context, dir string) error {
	return gitx.Push(ctx, g.Runner, dir)
}

func (g *GitAdapter) SetUpstream(ctx context.Context, dir, upstream, branch string) error {
	return gitx.SetUpstream(ctx, g.Runner, dir, upstream, branch)
}

func (g *GitAdapter) StashPush(ctx context.Context, dir, message string) (bool, error) {
	return gitx.StashPush(ctx, g.Runner, dir, message)
}

func (g *GitAdapter) StashPop(ctx context.Context, dir string) error {
	return gitx.StashPop(ctx, g.Runner, dir)
}

func (g *GitAdapter) Clone(ctx context.Context, remoteURL, targetPath, branch string, mirror bool) error {
	return gitx.Clone(ctx, g.Runner, remoteURL, targetPath, branch, mirror)
}

func (g *GitAdapter) NormalizeURL(rawURL string) string {
	return gitx.NormalizeURL(rawURL)
}

func (g *GitAdapter) PrimaryRemote(remoteNames []string) string {
	return gitx.PrimaryRemote(remoteNames)
}
