// SPDX-License-Identifier: MIT
// Package prune classifies local branches by prune safety. It is a pure,
// dependency-light package (it imports only model) so the classification logic
// can be exhaustively table-tested in isolation and reused by CLI, TUI, MCP, and
// future prune planning without pulling in engine wiring.
package prune

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
)

// Policy is the branch-retention policy the classifier consumes. It is
// config-free: the engine maps config.BranchPolicy into it, resolving BaseBranch
// per repository first (see ADR-0015).
type Policy struct {
	// ProtectedPatterns are glob patterns (path.Match) whose matching branches
	// are never prune candidates. The engine uses these to populate
	// model.LocalBranch.Protected before calling Classify.
	ProtectedPatterns []string
	// BaseBranch is the already-resolved base branch name for this repository.
	BaseBranch string
	// StaleDays, when > 0, escalates an unintegrated branch older than this many
	// days to needs_review. 0 disables staleness escalation.
	StaleDays int
	// RequireMerged, when true (the default), trusts only reachability as merge
	// proof: a patch-equivalent-only branch is surfaced for review rather than as
	// a probably_safe prune candidate.
	RequireMerged bool
}

// MatchesProtected reports whether branch matches any protected glob pattern. It
// returns an error if a pattern is malformed, so config validation can fail
// closed rather than silently treating a bad pattern as no-match.
func MatchesProtected(branch string, patterns []string) (bool, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false, nil
	}
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		ok, err := path.Match(p, branch)
		if err != nil {
			return false, fmt.Errorf("invalid protected branch pattern %q: %w", pattern, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// Classify returns the prune-safety category and reason codes for a branch. It
// is pure and first-match-wins. Raw signals (IsCurrent, CheckedOutElsewhere,
// Protected, upstream state, merged/patch-equivalence, recency) must already be
// populated on b; the engine does that from git state before calling Classify.
//
// The durable invariants (ADR-0014): protected/current/base/worktree-held
// branches are always keep; a positive integration signal (reachability or, when
// permitted, patch-equivalence) is required for any prune verdict; an unknown
// integration signal is always needs_review; only safe_to_prune is
// auto-prune-eligible; the conservative fallback is keep.
func Classify(b model.LocalBranch, p Policy, now time.Time) (model.PruneCategory, []model.PruneReason) {
	// 1. keep: branches that must never be prune candidates.
	switch {
	case b.IsCurrent:
		return model.PruneKeep, []model.PruneReason{model.ReasonCurrentBranch}
	case b.CheckedOutElsewhere:
		return model.PruneKeep, []model.PruneReason{model.ReasonCheckedOutElsewhere}
	case isBaseBranch(b.Name, p.BaseBranch):
		return model.PruneKeep, []model.PruneReason{model.ReasonBaseBranch}
	case b.Protected:
		return model.PruneKeep, []model.PruneReason{model.ReasonProtectedPattern}
	}

	reachable := b.MergedIntoBase != nil && *b.MergedIntoBase
	patchEquivalent := b.PatchEquivalentToBase != nil && *b.PatchEquivalentToBase

	// 2. safe_to_prune: reachable from base (fully merged). The only
	// auto-prune-eligible verdict.
	if reachable {
		return model.PruneSafeToPrune, []model.PruneReason{model.ReasonMergedIntoBase}
	}

	// 3. patch-equivalent (squash/rebase merge) but not reachable. Review-required
	// evidence; under RequireMerged it is surfaced as needs_review rather than a
	// prune candidate, so the conservative default never emits probably_safe.
	if patchEquivalent {
		if p.RequireMerged {
			return model.PruneNeedsReview, []model.PruneReason{model.ReasonPatchEquivalentToBase}
		}
		return model.PruneProbablySafe, []model.PruneReason{model.ReasonPatchEquivalentToBase}
	}

	// 4. Integration state could not be fully established (a check was
	// unavailable): be conservative, never infer "not merged" from a nil signal.
	if b.MergedIntoBase == nil || b.PatchEquivalentToBase == nil {
		return model.PruneNeedsReview, []model.PruneReason{model.ReasonSignalUnavailable}
	}

	// 5. Definitively not integrated. Classify by upstream state and staleness.
	category := model.PruneKeep
	var reasons []model.PruneReason
	switch b.UpstreamStatus {
	case model.TrackingNone, model.TrackingGone:
		category = model.PruneNeedsReview
		reasons = append(reasons, model.ReasonUnmergedLocalWork)
	case model.TrackingDiverged:
		category = model.PruneNeedsReview
		reasons = append(reasons, model.ReasonDivergedUnmerged)
	}
	if isStale(b.LastCommitAt, p.StaleDays, now) {
		category = model.PruneNeedsReview
		reasons = append(reasons, model.ReasonStaleUnmerged)
	}
	if category == model.PruneKeep {
		reasons = []model.PruneReason{model.ReasonActiveUnmerged}
	}
	return category, reasons
}

func isBaseBranch(name, base string) bool {
	base = strings.TrimSpace(base)
	return base != "" && strings.TrimSpace(name) == base
}

func isStale(last *time.Time, staleDays int, now time.Time) bool {
	if staleDays <= 0 || last == nil {
		return false
	}
	return last.Before(now.AddDate(0, 0, -staleDays))
}
