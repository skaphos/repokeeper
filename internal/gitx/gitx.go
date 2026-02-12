// Package gitx provides helpers for executing git commands and parsing
// their output. It shells out to the installed git binary.
package gitx

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mfacenet/repokeeper/internal/model"
)

// Runner executes git commands in a given repo directory.
// This interface allows mocking in tests.
type Runner interface {
	// Run executes a git command in the given directory and returns
	// combined stdout/stderr output.
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

// GitRunner is the default Runner implementation that shells out to git.
type GitRunner struct {
	// GitBin is the path to the git binary. Defaults to "git".
	GitBin string
}

// Run executes a git command.
func (g *GitRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	bin := g.GitBin
	if bin == "" {
		bin = "git"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// IsRepo checks whether the given path is inside a git working tree.
func IsRepo(ctx context.Context, r Runner, dir string) (bool, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

// IsBare checks whether the given path is a bare git repository.
func IsBare(ctx context.Context, r Runner, dir string) (bool, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "--is-bare-repository")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

// Remotes returns all configured remotes for the repo.
func Remotes(ctx context.Context, r Runner, dir string) ([]model.Remote, error) {
	out, err := r.Run(ctx, dir, "remote")
	if err != nil {
		return nil, fmt.Errorf("git remote: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	names := strings.Split(strings.TrimSpace(out), "\n")
	var remotes []model.Remote
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		url, err := r.Run(ctx, dir, "remote", "get-url", name)
		if err != nil {
			continue
		}
		remotes = append(remotes, model.Remote{
			Name: name,
			URL:  strings.TrimSpace(url),
		})
	}
	return remotes, nil
}

// Head returns the current branch and detached state.
func Head(ctx context.Context, r Runner, dir string) (model.Head, error) {
	out, err := r.Run(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		// Detached HEAD — try to get the commit hash
		hash, hashErr := r.Run(ctx, dir, "rev-parse", "--short", "HEAD")
		if hashErr != nil {
			return model.Head{Detached: true}, nil
		}
		return model.Head{
			Branch:   strings.TrimSpace(hash),
			Detached: true,
		}, nil
	}
	return model.Head{
		Branch:   strings.TrimSpace(out),
		Detached: false,
	}, nil
}

// WorktreeStatus returns the working tree dirty/staged/unstaged/untracked counts.
func WorktreeStatus(ctx context.Context, r Runner, dir string) (*model.Worktree, error) {
	out, err := r.Run(ctx, dir, "status", "--porcelain=v1")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	return ParsePorcelainStatus(out), nil
}

// TrackingStatus returns upstream tracking info for the current branch.
func TrackingStatus(ctx context.Context, r Runner, dir string) (model.Tracking, error) {
	out, err := r.Run(ctx, dir, "for-each-ref", "--format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort)", "refs/heads")
	if err != nil {
		return model.Tracking{Status: model.TrackingNone}, nil
	}

	// Get current branch
	head, err := r.Run(ctx, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return model.Tracking{Status: model.TrackingNone}, nil
	}
	head = strings.TrimSpace(head)

	entries := ParseForEachRef(out)
	for _, e := range entries {
		if e.Branch != head {
			continue
		}
		if e.Upstream == "" {
			return model.Tracking{Status: model.TrackingNone}, nil
		}
		if strings.Contains(e.Track, "[gone]") {
			return model.Tracking{
				Upstream: e.Upstream,
				Status:   model.TrackingGone,
			}, nil
		}

		// Get ahead/behind counts
		revOut, revErr := r.Run(ctx, dir, "rev-list", "--left-right", "--count", head+"..."+e.Upstream)
		if revErr != nil {
			// Fall back to for-each-ref track info
			return trackingFromShort(e), nil
		}
		ahead, behind := ParseRevListCount(revOut)
		aheadPtr := &ahead
		behindPtr := &behind

		var status model.TrackingStatus
		switch {
		case ahead > 0 && behind > 0:
			status = model.TrackingDiverged
		case ahead > 0:
			status = model.TrackingAhead
		case behind > 0:
			status = model.TrackingBehind
		default:
			status = model.TrackingEqual
		}

		return model.Tracking{
			Upstream: e.Upstream,
			Status:   status,
			Ahead:    aheadPtr,
			Behind:   behindPtr,
		}, nil
	}

	return model.Tracking{Status: model.TrackingNone}, nil
}

func trackingFromShort(e ForEachRefEntry) model.Tracking {
	var status model.TrackingStatus
	switch e.TrackShort {
	case ">":
		status = model.TrackingAhead
	case "<":
		status = model.TrackingBehind
	case "<>":
		status = model.TrackingDiverged
	case "=":
		status = model.TrackingEqual
	default:
		status = model.TrackingNone
	}
	return model.Tracking{
		Upstream: e.Upstream,
		Status:   status,
	}
}

// HasSubmodules checks for the presence of submodules without recursing.
func HasSubmodules(ctx context.Context, r Runner, dir string) (bool, error) {
	_, err := r.Run(ctx, dir, "config", "--file", ".gitmodules", "--get-regexp", "submodule")
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Fetch runs a safe fetch with submodule recursion disabled.
func Fetch(ctx context.Context, r Runner, dir string) error {
	_, err := r.Run(ctx, dir, "-c", "fetch.recurseSubmodules=false", "fetch", "--all", "--prune", "--prune-tags", "--no-recurse-submodules")
	return err
}
