// Package model defines the core data types used throughout RepoKeeper.
package model

import "time"

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

// SyncResult records the outcome of the last sync operation.
type SyncResult struct {
	// OK is true when the last sync completed successfully.
	OK bool `json:"ok" yaml:"ok"`
	// At is the timestamp of the last sync attempt.
	At time.Time `json:"at" yaml:"at"`
	// Error contains the sync error message when OK is false.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// RepoStatus is the full status report for a single repository.
type RepoStatus struct {
	// RepoID is the normalized identity for the repository (usually derived from remote URL).
	RepoID string `json:"repo_id" yaml:"repo_id"`
	// Path is the absolute local filesystem path to the repository.
	Path string `json:"path" yaml:"path"`
	// Type is the checkout type ("checkout" or "mirror").
	Type string `json:"type,omitempty" yaml:"type,omitempty"` // checkout | mirror
	// Labels are user-defined key/value metadata for classification and selection.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	// Annotations are user-defined key/value metadata for non-selector notes.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
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
