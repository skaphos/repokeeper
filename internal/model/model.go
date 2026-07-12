// SPDX-License-Identifier: MIT
// Package model defines the core data types used throughout RepoKeeper.
package model

import (
	"fmt"
	"strings"
	"time"
)

// Remote represents a single git remote.
type Remote struct {
	// Name is the configured remote name (for example, "origin").
	Name string `json:"name" yaml:"name"`
	// URL is the remote fetch/push URL.
	URL string `json:"url" yaml:"url"`
}

// Head represents the current HEAD state of a repo.
type Head struct {
	// Branch is the current branch name when HEAD is attached.
	Branch string `json:"branch" yaml:"branch"`
	// Detached reports whether HEAD is detached.
	Detached bool `json:"detached" yaml:"detached"`
}

// Worktree represents the working tree status. Nil for bare repos.
type Worktree struct {
	// Dirty indicates whether the worktree has any local modifications.
	Dirty bool `json:"dirty" yaml:"dirty"`
	// Staged is the count of staged file changes.
	Staged int `json:"staged" yaml:"staged"`
	// Unstaged is the count of unstaged file changes.
	Unstaged int `json:"unstaged" yaml:"unstaged"`
	// Untracked is the count of untracked files.
	Untracked int `json:"untracked" yaml:"untracked"`
}

// TrackingStatus enumerates the possible upstream tracking states.
type TrackingStatus string

const (
	TrackingAhead    TrackingStatus = "ahead"
	TrackingBehind   TrackingStatus = "behind"
	TrackingDiverged TrackingStatus = "diverged"
	TrackingEqual    TrackingStatus = "equal"
	TrackingGone     TrackingStatus = "gone"
	TrackingNone     TrackingStatus = "none"
)

// Tracking represents the upstream tracking relationship for the current branch.
type Tracking struct {
	// Upstream is the tracked upstream ref (for example, "origin/main").
	Upstream string `json:"upstream" yaml:"upstream"`
	// Status is the high-level relationship between local and upstream branches.
	Status TrackingStatus `json:"status" yaml:"status"`
	// Ahead is the number of commits local is ahead of upstream. Nil when unknown/not applicable.
	Ahead *int `json:"ahead" yaml:"ahead"` // nil when gone or none
	// Behind is the number of commits local is behind upstream. Nil when unknown/not applicable.
	Behind *int `json:"behind" yaml:"behind"` // nil when gone or none
}

// Submodules indicates whether the repo contains submodules.
type Submodules struct {
	// HasSubmodules indicates whether .gitmodules defines one or more submodules.
	HasSubmodules bool `json:"has_submodules" yaml:"has_submodules"`
}

// RemoteTrackingRefStatus describes remote-tracking refs that no longer exist
// on their configured remotes. InspectionError is non-empty when the remote
// could not be queried, so callers can distinguish an unavailable signal from
// a repository with no stale refs.
type RemoteTrackingRefStatus struct {
	StaleCount      int      `json:"stale_count" yaml:"stale_count"`
	Stale           []string `json:"stale,omitempty" yaml:"stale,omitempty"`
	InspectionError string   `json:"inspection_error,omitempty" yaml:"inspection_error,omitempty"`
}

// PruneCategory classifies a local branch by how safe it is to prune. Only
// PruneSafeToPrune is eligible for automated/batch prune; PruneProbablySafe
// carries positive evidence but is review-required and never auto-pruned.
type PruneCategory string

const (
	PruneKeep         PruneCategory = "keep"
	PruneSafeToPrune  PruneCategory = "safe_to_prune"
	PruneProbablySafe PruneCategory = "probably_safe"
	PruneNeedsReview  PruneCategory = "needs_review"
)

// ParsePruneCategory validates and parses a prune category value.
func ParsePruneCategory(raw string) (PruneCategory, error) {
	c := PruneCategory(strings.ToLower(strings.TrimSpace(raw)))
	switch c {
	case PruneKeep, PruneSafeToPrune, PruneProbablySafe, PruneNeedsReview:
		return c, nil
	default:
		return "", fmt.Errorf("unsupported prune category %q (expected one of: keep, safe_to_prune, probably_safe, needs_review)", raw)
	}
}

// PruneReason is a machine-readable code explaining a PruneCategory verdict.
type PruneReason string

const (
	ReasonCurrentBranch         PruneReason = "current_branch"
	ReasonCheckedOutElsewhere   PruneReason = "checked_out_elsewhere"
	ReasonBaseBranch            PruneReason = "base_branch"
	ReasonProtectedPattern      PruneReason = "protected_pattern"
	ReasonActiveUnmerged        PruneReason = "active_unmerged"
	ReasonSignalUnavailable     PruneReason = "signal_unavailable"
	ReasonUnmergedLocalWork     PruneReason = "unmerged_local_work"
	ReasonDivergedUnmerged      PruneReason = "diverged_unmerged"
	ReasonStaleUnmerged         PruneReason = "stale_unmerged"
	ReasonMergedIntoBase        PruneReason = "merged_into_base"
	ReasonPatchEquivalentToBase PruneReason = "patch_equivalent_to_base"
	ReasonBaseUnresolved        PruneReason = "base_unresolved"
)

// pruneReasonHints maps prune reason codes to operator-facing explanations.
var pruneReasonHints = map[PruneReason]string{
	ReasonCurrentBranch:         "branch is checked out in this worktree",
	ReasonCheckedOutElsewhere:   "branch is checked out in another worktree",
	ReasonBaseBranch:            "branch is the repository base branch",
	ReasonProtectedPattern:      "branch matches a protected pattern",
	ReasonActiveUnmerged:        "active branch not yet integrated into base",
	ReasonSignalUnavailable:     "integration status could not be determined",
	ReasonUnmergedLocalWork:     "branch has local commits not integrated into base",
	ReasonDivergedUnmerged:      "branch has diverged from its upstream and is not integrated",
	ReasonStaleUnmerged:         "branch is unintegrated and older than the stale threshold",
	ReasonMergedIntoBase:        "branch is reachable from base (fully merged)",
	ReasonPatchEquivalentToBase: "branch commits are patch-equivalent to base (likely squash/rebase merged)",
	ReasonBaseUnresolved:        "base branch could not be resolved for this repository",
}

// HintForReason returns operator-facing text for a prune reason, or empty string.
func HintForReason(r PruneReason) string {
	return pruneReasonHints[r]
}

// LocalBranch is a single local branch with its raw prune-safety signals and
// computed classification. Tri-state signals (*bool / *time.Time) are nil when
// the underlying check could not be run, so a failed check is never a silent
// false; the classifier maps unknown integration state to needs_review.
type LocalBranch struct {
	// Name is the short branch name (for example, "feature/x").
	Name string `json:"name" yaml:"name"`
	// IsCurrent reports whether the branch is checked out in this worktree.
	IsCurrent bool `json:"is_current" yaml:"is_current"`
	// CheckedOutElsewhere reports whether the branch is checked out in another linked worktree.
	CheckedOutElsewhere bool `json:"checked_out_elsewhere" yaml:"checked_out_elsewhere"`
	// Protected reports whether the branch matches a configured protected pattern.
	Protected bool `json:"protected" yaml:"protected"`
	// Upstream is the tracked upstream ref, if any (for example, "origin/feature/x").
	Upstream string `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	// UpstreamStatus is the high-level relationship to the upstream branch.
	UpstreamStatus TrackingStatus `json:"upstream_status" yaml:"upstream_status"`
	// Ahead is the commit count ahead of upstream. Nil when unknown/not applicable.
	Ahead *int `json:"ahead" yaml:"ahead"`
	// Behind is the commit count behind upstream. Nil when unknown/not applicable.
	Behind *int `json:"behind" yaml:"behind"`
	// MergedIntoBase reports reachability from the base branch. Nil when the check was unavailable.
	MergedIntoBase *bool `json:"merged_into_base" yaml:"merged_into_base"`
	// PatchEquivalentToBase reports patch-equivalence to base (squash/rebase merges). Nil when unavailable.
	PatchEquivalentToBase *bool `json:"patch_equivalent_to_base" yaml:"patch_equivalent_to_base"`
	// LastCommitAt is the branch tip committer date. Nil when unavailable.
	LastCommitAt *time.Time `json:"last_commit_at" yaml:"last_commit_at"`
	// Category is the computed prune-safety classification.
	Category PruneCategory `json:"category" yaml:"category"`
	// Reasons are the machine-readable codes explaining Category.
	Reasons []PruneReason `json:"reasons,omitempty" yaml:"reasons,omitempty"`
}

// LocalBranchStatus is the set of classified local branches for a repository.
// InspectionError is non-empty when branch enumeration failed, so callers can
// distinguish an unavailable signal from a repository with no local branches.
type LocalBranchStatus struct {
	Branches        []LocalBranch `json:"branches,omitempty" yaml:"branches,omitempty"`
	InspectionError string        `json:"inspection_error,omitempty" yaml:"inspection_error,omitempty"`
}

// SyncResult records the outcome of the last sync operation.
type SyncResult struct {
	// OK is true when the last sync completed successfully.
	OK bool `json:"ok" yaml:"ok"`
	// At is the timestamp of the last sync attempt.
	At time.Time `json:"at" yaml:"at"`
	// Error contains the sync error message when OK is false.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// RepoMetadataPaths groups path hints declared by a repository.
type RepoMetadataPaths struct {
	// Authoritative highlights the paths most worth consulting first.
	Authoritative []string `json:"authoritative,omitempty" yaml:"authoritative,omitempty"`
	// LowValue highlights paths that are usually less useful for navigation.
	LowValue []string `json:"low_value,omitempty" yaml:"low_value,omitempty"`
}

// RepoMetadataRelatedRepo describes a relationship to another repository.
type RepoMetadataRelatedRepo struct {
	// RepoID is the referenced repository identity.
	RepoID string `json:"repo_id" yaml:"repo_id"`
	// Relationship is an optional free-form relationship label.
	Relationship string `json:"relationship,omitempty" yaml:"relationship,omitempty"`
}

// RepoMetadata describes source-controlled repo-local metadata discovered at runtime.
type RepoMetadata struct {
	// APIVersion is an optional schema version marker for the metadata file.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	// Kind is an optional schema kind marker for the metadata file.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	// RepoID is an optional repo-local assertion of the repository identity.
	RepoID string `json:"repo_id,omitempty" yaml:"repo_id,omitempty"`
	// Name is an optional human-friendly repository name.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Labels are generic taxonomy labels declared by the repository.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	// Entrypoints are named starting points within the repository.
	Entrypoints map[string]string `json:"entrypoints,omitempty" yaml:"entrypoints,omitempty"`
	// Paths groups repository path relevance hints.
	Paths RepoMetadataPaths `json:"paths,omitempty" yaml:"paths,omitempty"`
	// Provides lists capabilities or artifacts this repository offers.
	Provides []string `json:"provides,omitempty" yaml:"provides,omitempty"`
	// RelatedRepos lists known related repositories and their relationships.
	RelatedRepos []RepoMetadataRelatedRepo `json:"related_repos,omitempty" yaml:"related_repos,omitempty"`
}

// RepoStatus is the full status report for a single repository.
type RepoStatus struct {
	// RepoID is the normalized identity for the repository (usually derived from remote URL).
	RepoID string `json:"repo_id" yaml:"repo_id"`
	// CheckoutID is an optional stable identifier for a specific local checkout of the repository.
	CheckoutID string `json:"checkout_id,omitempty" yaml:"checkout_id,omitempty"`
	// Path is the absolute local filesystem path to the repository.
	Path string `json:"path" yaml:"path"`
	// Type is the checkout type ("checkout" or "mirror").
	Type string `json:"type,omitempty" yaml:"type,omitempty"` // checkout | mirror
	// Labels are user-defined key/value metadata for classification and selection.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	// Annotations are user-defined key/value metadata for non-selector notes.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	// RepoMetadataFile is the repo-local metadata file path when one was discovered.
	RepoMetadataFile string `json:"repo_metadata_file,omitempty" yaml:"repo_metadata_file,omitempty"`
	// RepoMetadataError captures non-fatal repo-local metadata discovery or validation errors.
	RepoMetadataError string `json:"repo_metadata_error,omitempty" yaml:"repo_metadata_error,omitempty"`
	// RepoMetadataFingerprint is the last observed on-disk fingerprint for repo-local metadata state.
	RepoMetadataFingerprint string `json:"repo_metadata_fingerprint,omitempty" yaml:"repo_metadata_fingerprint,omitempty"`
	// RepoMetadata carries source-controlled repo-local metadata when available.
	RepoMetadata *RepoMetadata `json:"repo_metadata,omitempty" yaml:"repo_metadata,omitempty"`
	// Bare indicates whether the repository has no working tree.
	Bare bool `json:"bare" yaml:"bare"`
	// Remotes contains all configured remotes.
	Remotes []Remote `json:"remotes" yaml:"remotes"`
	// PrimaryRemote is the preferred remote name used for identity and sync behavior.
	PrimaryRemote string `json:"primary_remote" yaml:"primary_remote"`
	// Head describes the current HEAD branch/detached state.
	Head Head `json:"head" yaml:"head"`
	// Worktree is nil for bare repositories.
	Worktree *Worktree `json:"worktree" yaml:"worktree"` // nil for bare repos
	// Tracking describes upstream tracking status for the current branch.
	Tracking Tracking `json:"tracking" yaml:"tracking"`
	// Submodules indicates whether the repository contains submodules.
	Submodules Submodules `json:"submodules" yaml:"submodules"`
	// RemoteTrackingRefs describes refs that a fetch with prune would remove.
	RemoteTrackingRefs RemoteTrackingRefStatus `json:"remote_tracking_refs" yaml:"remote_tracking_refs"`
	// LocalBranches describes local branches classified by prune safety.
	LocalBranches LocalBranchStatus `json:"local_branches" yaml:"local_branches"`
	// LastSync is the latest sync outcome metadata when available.
	LastSync *SyncResult `json:"last_sync,omitempty" yaml:"last_sync,omitempty"`
	// Error holds repository-specific inspect or sync error text.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
	// ErrorClass is a coarse category for Error (for example, missing/auth/network).
	ErrorClass string `json:"error_class,omitempty" yaml:"error_class,omitempty"`
}

// StatusReport is the top-level output of the status command.
type StatusReport struct {
	// GeneratedAt is the timestamp when this report was produced.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	// Repos is the full set of repository status rows in the report.
	Repos []RepoStatus `json:"repos" yaml:"repos"`
}
