// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/prune"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// inspectLocalBranches enumerates and classifies the repository's local branches
// for prune safety. It is read-only. Unsupported backends and bare repos yield an
// empty result; a resolvable base that git cannot query yields per-branch
// signal_unavailable via the classifier, while an unresolvable base yields
// base_unresolved. Nothing here deletes a branch.
func (e *Engine) inspectLocalBranches(ctx context.Context, path, primary, repoID string, head model.Head, tracking model.Tracking, bare bool) model.LocalBranchStatus {
	if bare {
		return model.LocalBranchStatus{}
	}
	inspector, ok := e.adapter.(vcs.LocalBranchInspector)
	if !ok {
		return model.LocalBranchStatus{}
	}

	// baseName is normally an unqualified local branch name, but the
	// branch_policy.base_branch override may be remote-qualified (e.g.
	// "origin/main"). Separate the local name — used for classification, so the
	// base branch is recognized by isBaseBranch — from the remote-tracking ref
	// used for git reachability/patch queries, so a stale local base does not
	// yield false "not merged" (ADR-0015) and we never double-prefix
	// ("origin/origin/main").
	baseName := e.resolveBaseBranchName(repoID, path, tracking)
	primaryRemote := strings.TrimSpace(primary)
	localBase, queryBase := baseName, baseName
	if primaryRemote != "" && baseName != "" {
		if strings.HasPrefix(baseName, primaryRemote+"/") {
			localBase = strings.TrimPrefix(baseName, primaryRemote+"/")
		} else {
			queryBase = primaryRemote + "/" + baseName
		}
	}

	signals, err := inspector.InspectLocalBranches(ctx, path, queryBase)
	if err != nil {
		if e.logger != nil {
			e.logger.Warnf("local branch inspection failed for %s: %v", path, err)
		}
		return model.LocalBranchStatus{InspectionError: err.Error()}
	}

	policy := e.branchPolicy(localBase)
	now := time.Now()
	branches := make([]model.LocalBranch, 0, len(signals))
	for _, s := range signals {
		lb := model.LocalBranch{
			Name:                  s.Name,
			Upstream:              s.Upstream,
			UpstreamStatus:        upstreamStatusFromSignal(s),
			MergedIntoBase:        s.MergedIntoBase,
			PatchEquivalentToBase: s.PatchEquivalentToBase,
		}
		lb.IsCurrent = !head.Detached && s.Name == head.Branch
		lb.CheckedOutElsewhere = strings.TrimSpace(s.WorktreePath) != "" && !lb.IsCurrent
		if s.HasLastCommit {
			last := s.LastCommit
			lb.LastCommitAt = &last
		}
		if protected, perr := prune.MatchesProtected(s.Name, policy.ProtectedPatterns); perr == nil {
			lb.Protected = protected
		}
		if localBase == "" {
			// No base could be resolved for this repo: surface for review rather
			// than ever proposing a prune.
			lb.Category = model.PruneNeedsReview
			lb.Reasons = []model.PruneReason{model.ReasonBaseUnresolved}
		} else {
			lb.Category, lb.Reasons = prune.Classify(lb, policy, now)
		}
		branches = append(branches, lb)
	}
	return model.LocalBranchStatus{Branches: branches}
}

// resolveBaseBranchName resolves the merge-into-base reference for a repository,
// mirroring repairResolveTargetBranch: an explicit config override wins, then the
// registry's recorded branch, then the upstream-derived branch, then the
// workspace default. Returns "" when nothing resolves.
func (e *Engine) resolveBaseBranchName(repoID, path string, tracking model.Tracking) string {
	if e.cfg != nil {
		if b := strings.TrimSpace(e.cfg.BranchPolicy.BaseBranch); b != "" {
			return b
		}
	}
	if e.registry != nil {
		if entry := e.registry.FindEntry(repoID, path); entry != nil {
			if b := strings.TrimSpace(entry.Branch); b != "" {
				return b
			}
		}
	}
	if up := strings.TrimSpace(tracking.Upstream); up != "" {
		if parts := strings.SplitN(up, "/", 2); len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}
	if e.cfg != nil {
		if b := strings.TrimSpace(e.cfg.Defaults.MainBranch); b != "" {
			return b
		}
	}
	return ""
}

// branchPolicy maps machine config into the config-free classifier policy,
// injecting the already-resolved per-repo base branch.
func (e *Engine) branchPolicy(baseName string) prune.Policy {
	policy := prune.Policy{BaseBranch: baseName}
	if e.cfg != nil {
		policy.ProtectedPatterns = e.cfg.BranchPolicy.ProtectedPatterns
		policy.StaleDays = e.cfg.BranchPolicy.StaleDays
		policy.RequireMerged = e.cfg.BranchPolicy.RequireMerged
	}
	return policy
}

// upstreamStatusFromSignal maps the raw for-each-ref tracking hints to the model
// enum the classifier keys on.
func upstreamStatusFromSignal(s vcs.LocalBranchSignal) model.TrackingStatus {
	if strings.TrimSpace(s.Upstream) == "" {
		return model.TrackingNone
	}
	if strings.Contains(s.Track, "gone") {
		return model.TrackingGone
	}
	switch s.TrackShort {
	case ">":
		return model.TrackingAhead
	case "<":
		return model.TrackingBehind
	case "<>":
		return model.TrackingDiverged
	default:
		return model.TrackingEqual
	}
}
