// SPDX-License-Identifier: MIT
package gitx

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// localBranchFormat enumerates local branches with the signals the prune
// classifier needs. It is NUL-delimited ("%00") because "|" is a legal ref
// character; unlike the pipe-delimited TrackingStatus format, this is a brand
// new git invocation with no existing mock-runner fixtures pinned to it, so the
// NUL delimiter is free of the coupling that reverted the earlier fix.
const localBranchFormat = "%(refname:short)%00%(upstream:short)%00%(upstream:track)%00%(upstream:trackshort)%00%(committerdate:iso-strict)%00%(worktreepath)"

// LocalBranchInfo is the raw per-branch enumeration for one local branch. It is
// git-layer data; the engine maps it (plus merged/patch-equivalence signals)
// into model.LocalBranch and classifies it.
type LocalBranchInfo struct {
	Name          string
	Upstream      string
	Track         string // e.g. "[ahead 2]", "[gone]", ""
	TrackShort    string // e.g. ">", "<", "<>", "="
	LastCommit    time.Time
	HasLastCommit bool
	WorktreePath  string // non-empty when checked out in a (possibly other) worktree
}

// LocalBranches enumerates every local branch with its upstream, tracking,
// recency, and worktree-checkout signals.
func LocalBranches(ctx context.Context, r Runner, dir string) ([]LocalBranchInfo, error) {
	out, err := r.Run(ctx, dir, "for-each-ref", "--format="+localBranchFormat, "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref refs/heads: %w", err)
	}
	return ParseLocalBranches(out), nil
}

// MergedBranches returns the set of local branches reachable from base (i.e.
// fully merged via a merge commit or fast-forward). Squash/rebase merges are not
// reachable and are intentionally absent; patch-equivalence covers those. base
// must resolve to an existing ref or git errors.
func MergedBranches(ctx context.Context, r Runner, dir, base string) (map[string]bool, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return nil, fmt.Errorf("merged-branch check requires a base ref")
	}
	out, err := r.Run(ctx, dir, "for-each-ref", "--merged="+base, "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref --merged=%q: %w", base, err)
	}
	merged := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			merged[name] = true
		}
	}
	return merged, nil
}

// PatchEquivalentToBase reports whether every commit unique to branch is
// patch-equivalent to a commit already in base (the squash/rebase-merge case),
// via git cherry. base must resolve to an existing ref or git errors.
func PatchEquivalentToBase(ctx context.Context, r Runner, dir, base, branch string) (bool, error) {
	base = strings.TrimSpace(base)
	branch = strings.TrimSpace(branch)
	if base == "" || branch == "" {
		return false, fmt.Errorf("patch-equivalence check requires base and branch refs")
	}
	out, err := r.Run(ctx, dir, "cherry", base, branch)
	if err != nil {
		return false, fmt.Errorf("git cherry %q %q: %w", base, branch, err)
	}
	return ParseCherryEquivalent(out), nil
}

// ParseLocalBranches parses the NUL-delimited localBranchFormat output.
func ParseLocalBranches(output string) []LocalBranchInfo {
	if output == "" {
		return nil
	}
	var infos []LocalBranchInfo
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		info := LocalBranchInfo{}
		if len(parts) > 0 {
			info.Name = parts[0]
		}
		if len(parts) > 1 {
			info.Upstream = parts[1]
		}
		if len(parts) > 2 {
			info.Track = parts[2]
		}
		if len(parts) > 3 {
			info.TrackShort = parts[3]
		}
		if len(parts) > 4 {
			if ts := strings.TrimSpace(parts[4]); ts != "" {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					info.LastCommit = t
					info.HasLastCommit = true
				}
			}
		}
		if len(parts) > 5 {
			info.WorktreePath = strings.TrimSpace(parts[5])
		}
		infos = append(infos, info)
	}
	return infos
}

// ParseCherryEquivalent reports whether git cherry output shows every unique
// commit already integrated (patch-equivalent). A "+" prefixed line marks a
// commit with no equivalent in base; its presence means not patch-equivalent.
// Empty output (no unique commits) is treated as patch-equivalent.
func ParseCherryEquivalent(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "+") {
			return false
		}
	}
	return true
}
