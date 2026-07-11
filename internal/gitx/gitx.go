// SPDX-License-Identifier: MIT
// Package gitx provides helpers for executing git commands and parsing
// their output. It shells out to the installed git binary.
package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/obs"
)

// Runner executes git commands in a given repo directory.
// This interface allows mocking in tests.
type Runner interface {
	// Run executes a git command in the given directory and returns
	// trimmed stdout only. Any stderr text is folded into the returned
	// error (see GitRunner.Run) and never appears in the returned string,
	// so callers can safely parse the result as machine-readable output.
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
	// Force the C locale so git's stderr text is stable, untranslated
	// English: ClassifyError string-matches stderr, and on a non-English
	// locale every classification would otherwise degrade to "unknown".
	// A later duplicate key wins in cmd.Env, so this reliably overrides
	// whatever locale the parent process is running under.
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")

	// Capture stdout and stderr separately (mirrors internal/vcs's
	// runCommand) instead of CombinedOutput(), which merges the two.
	// An exit-0 git warning on stderr (e.g. an unknown gitconfig key)
	// would otherwise corrupt callers that parse stdout as
	// machine-readable output (IsRepo/IsBare/ParsePorcelainStatus).
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		if errText := strings.TrimSpace(stderr.String()); errText != "" {
			return out, fmt.Errorf("%s: %w", errText, err)
		}
		return out, err
	}
	return out, nil
}

// IsRepo checks whether the given path is inside a git working tree.
func IsRepo(ctx context.Context, r Runner, dir string, logger obs.Logger) (bool, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		// Treat probe failure as "not a repo" to keep discovery/status resilient.
		if logger != nil {
			logger.Warnf("IsRepo probe failed for %s: %v", dir, err)
		}
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

// IsBare checks whether the given path is a bare git repository.
func IsBare(ctx context.Context, r Runner, dir string, logger obs.Logger) (bool, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "--is-bare-repository")
	if err != nil {
		// Mirror IsRepo behavior: command failure should not hard-fail callers.
		if logger != nil {
			logger.Warnf("IsBare probe failed for %s: %v", dir, err)
		}
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

// Remotes returns all configured remotes for the repo.
func Remotes(ctx context.Context, r Runner, dir string, logger obs.Logger) ([]model.Remote, error) {
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
			if logger != nil {
				logger.Warnf("remote get-url %s failed for %s: %v", name, dir, err)
			}
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
			// "[gone]" is the most reliable indicator that upstream ref disappeared.
			return model.Tracking{
				Upstream: e.Upstream,
				Status:   model.TrackingGone,
			}, nil
		}

		// Get ahead/behind counts
		revOut, revErr := r.Run(ctx, dir, "rev-list", "--left-right", "--count", head+"..."+e.Upstream)
		if revErr != nil {
			// Fall back to lightweight tracking hints if precise counts are unavailable.
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
	out, err := r.Run(ctx, dir, "-c", "fetch.recurseSubmodules=false", "fetch", "--all", "--prune", "--prune-tags", "--no-recurse-submodules")
	return wrapRunError("git fetch", out, err)
}

// PullRebase runs a safe pull --rebase with submodule recursion disabled.
func PullRebase(ctx context.Context, r Runner, dir string) error {
	out, err := r.Run(ctx, dir, "-c", "fetch.recurseSubmodules=false", "pull", "--rebase", "--no-recurse-submodules")
	return wrapRunError("git pull --rebase", out, err)
}

// Push publishes local commits on the current branch to its upstream.
func Push(ctx context.Context, r Runner, dir string) error {
	out, err := r.Run(ctx, dir, "push")
	return wrapRunError("git push", out, err)
}

// SetUpstream configures the current local branch to track the given upstream.
func SetUpstream(ctx context.Context, r Runner, dir, upstream, branch string) error {
	upstream = strings.TrimSpace(upstream)
	branch = strings.TrimSpace(branch)
	if err := rejectFlagLike("upstream", upstream); err != nil {
		return err
	}
	if err := rejectFlagLike("branch", branch); err != nil {
		return err
	}
	out, err := r.Run(ctx, dir, "branch", "--set-upstream-to", upstream, branch)
	return wrapRunError("git branch --set-upstream-to", out, err)
}

// SetRemoteURL updates the URL for a named remote.
func SetRemoteURL(ctx context.Context, r Runner, dir, remote, remoteURL string) error {
	remote = strings.TrimSpace(remote)
	remoteURL = strings.TrimSpace(remoteURL)
	if err := rejectFlagLike("remote", remote); err != nil {
		return err
	}
	if err := rejectFlagLike("remote URL", remoteURL); err != nil {
		return err
	}
	out, err := r.Run(ctx, dir, "remote", "set-url", remote, remoteURL)
	return wrapRunError("git remote set-url", out, err)
}

// StashPush stashes current worktree changes (including untracked files).
// Returns true when a stash entry was created.
func StashPush(ctx context.Context, r Runner, dir, message string) (bool, error) {
	args := []string{"stash", "push", "-u"}
	if strings.TrimSpace(message) != "" {
		args = append(args, "-m", message)
	}
	out, err := r.Run(ctx, dir, args...)
	if err != nil {
		return false, wrapRunError("git stash push", out, err)
	}
	return !strings.Contains(strings.ToLower(out), "no local changes to save"), nil
}

// StashPop reapplies the most recent stash entry.
func StashPop(ctx context.Context, r Runner, dir string) error {
	out, err := r.Run(ctx, dir, "stash", "pop")
	return wrapRunError("git stash pop", out, err)
}

// Clone runs a clone operation. Branch is ignored for mirror clones.
func ResetHard(ctx context.Context, r Runner, dir string) error {
	out, err := r.Run(ctx, dir, "reset", "--hard", "HEAD")
	return wrapRunError("git reset --hard HEAD", out, err)
}

func CleanFD(ctx context.Context, r Runner, dir string) error {
	out, err := r.Run(ctx, dir, "clean", "-f", "-d")
	return wrapRunError("git clean -f -d", out, err)
}

func Clone(ctx context.Context, r Runner, remoteURL, targetPath, branch string, mirror bool) error {
	// remoteURL and targetPath are passed to git verbatim: leading/trailing
	// whitespace is legal in local paths, and the flag-injection guard only
	// needs to reject values whose first byte is '-' (a leading space cannot be
	// parsed as an option), so trimming would only corrupt valid inputs.
	branch = strings.TrimSpace(branch)
	if err := rejectFlagLike("remote URL", remoteURL); err != nil {
		return err
	}
	if targetPath != "" {
		if err := rejectFlagLike("target path", targetPath); err != nil {
			return err
		}
	}

	args := []string{"clone"}
	if mirror {
		args = append(args, "--mirror")
	} else if branch != "" {
		if err := rejectFlagLike("branch", branch); err != nil {
			return err
		}
		args = append(args, "--branch", branch, "--single-branch")
	}
	args = append(args, remoteURL, targetPath)
	out, err := r.Run(ctx, "", args...)
	return wrapRunError("git clone", out, err)
}

// rejectFlagLike rejects a positional argument (URL, path, ref, or remote
// name) that begins with "-". git parses such values as an option rather
// than a literal positional argument, so an attacker-controlled remote URL,
// branch, or remote name beginning with "-" could otherwise smuggle
// arbitrary flags into the git invocation.
func rejectFlagLike(field, value string) error {
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("gitx: %s must not start with '-': %q", field, value)
	}
	return nil
}

func wrapRunError(op, output string, err error) error {
	if err == nil {
		return nil
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return fmt.Errorf("%s: %w", op, err)
	}
	return fmt.Errorf("%s: %s: %w", op, trimmed, err)
}
