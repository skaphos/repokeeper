package vcs

import (
	"context"
	"errors"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

var errUnsupportedForBzr = errors.New("operation unsupported for bzr adapter")

// BzrAdapter implements Adapter for Bazaar repositories.
type BzrAdapter struct{}

func NewBzrAdapter() *BzrAdapter { return &BzrAdapter{} }

func (b *BzrAdapter) Name() string { return "bzr" }

func (b *BzrAdapter) IsRepo(ctx context.Context, dir string) (bool, error) {
	if _, err := runCommand(ctx, dir, "bzr", "root"); err != nil {
		return false, nil
	}
	return true, nil
}

func (b *BzrAdapter) IsBare(context.Context, string) (bool, error) { return false, nil }

func (b *BzrAdapter) Remotes(context.Context, string) ([]model.Remote, error) { return nil, nil }

func (b *BzrAdapter) Head(ctx context.Context, dir string) (model.Head, error) {
	branch, err := runCommand(ctx, dir, "bzr", "nick")
	if err != nil {
		return model.Head{}, err
	}
	return model.Head{Branch: strings.TrimSpace(branch), Detached: false}, nil
}

func (b *BzrAdapter) WorktreeStatus(ctx context.Context, dir string) (*model.Worktree, error) {
	out, err := runCommand(ctx, dir, "bzr", "status", "--short")
	if err != nil {
		return nil, err
	}
	return &model.Worktree{Dirty: strings.TrimSpace(out) != ""}, nil
}

func (b *BzrAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingNone}, nil
}

func (b *BzrAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }

func (b *BzrAdapter) Fetch(context.Context, string) error { return errUnsupportedForBzr }

func (b *BzrAdapter) PullRebase(context.Context, string) error { return errUnsupportedForBzr }

func (b *BzrAdapter) Push(ctx context.Context, dir string) error {
	_, err := runCommand(ctx, dir, "bzr", "push")
	return err
}

func (b *BzrAdapter) SetUpstream(context.Context, string, string, string) error {
	return errUnsupportedForBzr
}

func (b *BzrAdapter) SetRemoteURL(context.Context, string, string, string) error {
	return errUnsupportedForBzr
}

func (b *BzrAdapter) StashPush(context.Context, string, string) (bool, error) {
	return false, errUnsupportedForBzr
}

func (b *BzrAdapter) StashPop(context.Context, string) error { return errUnsupportedForBzr }

func (b *BzrAdapter) Clone(ctx context.Context, remoteURL, targetPath, _ string, mirror bool) error {
	if mirror {
		return errUnsupportedForBzr
	}
	_, err := runCommand(ctx, "", "bzr", "branch", remoteURL, targetPath)
	return err
}

func (b *BzrAdapter) NormalizeURL(rawURL string) string {
	return strings.TrimSpace(strings.ToLower(rawURL))
}

func (b *BzrAdapter) PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) == 0 {
		return ""
	}
	return remoteNames[0]
}
