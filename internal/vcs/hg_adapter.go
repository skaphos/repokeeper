package vcs

import (
	"context"
	"errors"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

var errUnsupportedForHg = errors.New("operation unsupported for hg adapter")

// HgAdapter implements Adapter for Mercurial repositories.
type HgAdapter struct{}

func NewHgAdapter() *HgAdapter { return &HgAdapter{} }

func (h *HgAdapter) Name() string { return "hg" }

func (h *HgAdapter) IsRepo(ctx context.Context, dir string) (bool, error) {
	if _, err := runCommand(ctx, dir, "hg", "root"); err != nil {
		return false, nil
	}
	return true, nil
}

func (h *HgAdapter) IsBare(context.Context, string) (bool, error) { return false, nil }

func (h *HgAdapter) Remotes(ctx context.Context, dir string) ([]model.Remote, error) {
	url, err := runCommand(ctx, dir, "hg", "paths", "default")
	if err != nil || strings.TrimSpace(url) == "" {
		return nil, nil
	}
	return []model.Remote{{Name: "default", URL: strings.TrimSpace(url)}}, nil
}

func (h *HgAdapter) Head(ctx context.Context, dir string) (model.Head, error) {
	branch, err := runCommand(ctx, dir, "hg", "branch")
	if err != nil {
		return model.Head{}, err
	}
	return model.Head{Branch: strings.TrimSpace(branch), Detached: false}, nil
}

func (h *HgAdapter) WorktreeStatus(ctx context.Context, dir string) (*model.Worktree, error) {
	out, err := runCommand(ctx, dir, "hg", "status")
	if err != nil {
		return nil, err
	}
	dirty := strings.TrimSpace(out) != ""
	return &model.Worktree{Dirty: dirty}, nil
}

func (h *HgAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingNone}, nil
}

func (h *HgAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }

func (h *HgAdapter) Fetch(ctx context.Context, dir string) error {
	_, err := runCommand(ctx, dir, "hg", "pull")
	return err
}

func (h *HgAdapter) PullRebase(context.Context, string) error { return errUnsupportedForHg }

func (h *HgAdapter) Push(ctx context.Context, dir string) error {
	_, err := runCommand(ctx, dir, "hg", "push")
	return err
}

func (h *HgAdapter) SetUpstream(context.Context, string, string, string) error {
	return errUnsupportedForHg
}

func (h *HgAdapter) SetRemoteURL(context.Context, string, string, string) error {
	return errUnsupportedForHg
}

func (h *HgAdapter) StashPush(context.Context, string, string) (bool, error) {
	return false, errUnsupportedForHg
}

func (h *HgAdapter) StashPop(context.Context, string) error { return errUnsupportedForHg }

func (h *HgAdapter) Clone(ctx context.Context, remoteURL, targetPath, branch string, mirror bool) error {
	args := []string{"clone"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--updaterev", strings.TrimSpace(branch))
	}
	if mirror {
		return errUnsupportedForHg
	}
	args = append(args, remoteURL, targetPath)
	_, err := runCommand(ctx, "", "hg", args...)
	return err
}

func (h *HgAdapter) NormalizeURL(rawURL string) string {
	trimmed := strings.TrimSpace(strings.ToLower(rawURL))
	return strings.TrimSuffix(trimmed, ".hg")
}

func (h *HgAdapter) PrimaryRemote(remoteNames []string) string {
	for _, name := range remoteNames {
		if name == "default" {
			return name
		}
	}
	if len(remoteNames) == 0 {
		return ""
	}
	return remoteNames[0]
}

// SupportsLocalUpdate documents current sync safety boundaries for hg.
func (h *HgAdapter) SupportsLocalUpdate(context.Context, string) (bool, string, error) {
	return false, "local update unsupported for vcs hg", nil
}

// FetchAction returns the human-readable safe fetch action for hg.
func (h *HgAdapter) FetchAction(context.Context, string) (string, error) {
	return "hg pull", nil
}
