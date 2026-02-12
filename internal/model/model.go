// Package model defines the core data types used throughout RepoKeeper.
package model

import "time"

// Remote represents a single git remote.
type Remote struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url" yaml:"url"`
}

// Head represents the current HEAD state of a repo.
type Head struct {
	Branch   string `json:"branch" yaml:"branch"`
	Detached bool   `json:"detached" yaml:"detached"`
}

// Worktree represents the working tree status. Nil for bare repos.
type Worktree struct {
	Dirty     bool `json:"dirty" yaml:"dirty"`
	Staged    int  `json:"staged" yaml:"staged"`
	Unstaged  int  `json:"unstaged" yaml:"unstaged"`
	Untracked int  `json:"untracked" yaml:"untracked"`
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
	Upstream string         `json:"upstream" yaml:"upstream"`
	Status   TrackingStatus `json:"status" yaml:"status"`
	Ahead    *int           `json:"ahead" yaml:"ahead"`   // nil when gone or none
	Behind   *int           `json:"behind" yaml:"behind"` // nil when gone or none
}

// Submodules indicates whether the repo contains submodules.
type Submodules struct {
	HasSubmodules bool `json:"has_submodules" yaml:"has_submodules"`
}

// SyncResult records the outcome of the last sync operation.
type SyncResult struct {
	OK    bool      `json:"ok" yaml:"ok"`
	At    time.Time `json:"at" yaml:"at"`
	Error string    `json:"error,omitempty" yaml:"error,omitempty"`
}

// RepoStatus is the full status report for a single repository.
type RepoStatus struct {
	RepoID        string      `json:"repo_id" yaml:"repo_id"`
	Path          string      `json:"path" yaml:"path"`
	Bare          bool        `json:"bare" yaml:"bare"`
	Remotes       []Remote    `json:"remotes" yaml:"remotes"`
	PrimaryRemote string      `json:"primary_remote" yaml:"primary_remote"`
	Head          Head        `json:"head" yaml:"head"`
	Worktree      *Worktree   `json:"worktree" yaml:"worktree"` // nil for bare repos
	Tracking      Tracking    `json:"tracking" yaml:"tracking"`
	Submodules    Submodules  `json:"submodules" yaml:"submodules"`
	LastSync      *SyncResult `json:"last_sync,omitempty" yaml:"last_sync,omitempty"`
	Error         string      `json:"error,omitempty" yaml:"error,omitempty"`
	ErrorClass    string      `json:"error_class,omitempty" yaml:"error_class,omitempty"`
}

// StatusReport is the top-level output of the status command.
type StatusReport struct {
	MachineID   string       `json:"machine_id" yaml:"machine_id"`
	GeneratedAt time.Time    `json:"generated_at" yaml:"generated_at"`
	Repos       []RepoStatus `json:"repos" yaml:"repos"`
}
