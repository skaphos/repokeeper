// SPDX-License-Identifier: MIT
// Package registry handles persistence and staleness detection for the
// per-machine repo registry.
package registry

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// EntryStatus represents whether a registry entry's path is still valid.
type EntryStatus string

const (
	StatusPresent EntryStatus = "present"
	StatusMissing EntryStatus = "missing"
	StatusMoved   EntryStatus = "moved"
)

// Entry is a single repo entry in the registry.
type Entry struct {
	RepoID      string            `yaml:"repo_id"`
	Path        string            `yaml:"path"`
	RemoteURL   string            `yaml:"remote_url"`
	Type        string            `yaml:"type,omitempty"` // checkout | mirror
	Branch      string            `yaml:"branch,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	LastSeen    time.Time         `yaml:"last_seen,omitempty"`
	Status      EntryStatus       `yaml:"status"`
}

// Registry is the per-machine mapping of repo identities to local paths.
type Registry struct {
	UpdatedAt time.Time `yaml:"updated_at,omitempty"`
	Entries   []Entry   `yaml:"repos"`
}

// Load reads a registry file from the given path.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// Save writes the registry to the given path.
func Save(reg *Registry, path string) error {
	if reg == nil {
		return errors.New("registry is nil")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(reg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Upsert adds or updates an entry in the registry by repo_id.
// If the repo_id already exists, it updates path, last_seen, and status.
// If new, it appends the entry.
func (r *Registry) Upsert(entry Entry) {
	for i := range r.Entries {
		if r.Entries[i].RepoID == entry.RepoID {
			if r.Entries[i].Path != entry.Path {
				entry.Status = StatusMoved
			} else if entry.Status == "" {
				entry.Status = StatusPresent
			}
			if entry.Type == "" {
				entry.Type = r.Entries[i].Type
			}
			if entry.Branch == "" {
				entry.Branch = r.Entries[i].Branch
			}
			if entry.Labels == nil && len(r.Entries[i].Labels) > 0 {
				entry.Labels = cloneStringMap(r.Entries[i].Labels)
			}
			if entry.Annotations == nil && len(r.Entries[i].Annotations) > 0 {
				entry.Annotations = cloneStringMap(r.Entries[i].Annotations)
			}
			r.Entries[i].Path = entry.Path
			r.Entries[i].RemoteURL = entry.RemoteURL
			r.Entries[i].Type = entry.Type
			r.Entries[i].Branch = entry.Branch
			r.Entries[i].Labels = entry.Labels
			r.Entries[i].Annotations = entry.Annotations
			r.Entries[i].LastSeen = entry.LastSeen
			r.Entries[i].Status = entry.Status
			return
		}
	}
	if entry.Status == "" {
		entry.Status = StatusPresent
	}
	r.Entries = append(r.Entries, entry)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ValidatePaths checks all entries against the filesystem and marks
// entries as missing or moved as appropriate.
func (r *Registry) ValidatePaths() error {
	for i := range r.Entries {
		_, err := os.Stat(r.Entries[i].Path)
		if err != nil {
			if os.IsNotExist(err) {
				r.Entries[i].Status = StatusMissing
				continue
			}
			return err
		}
		r.Entries[i].Status = StatusPresent
	}
	return nil
}

// PruneStale removes entries marked as missing that are older than
// the given threshold.
func (r *Registry) PruneStale(olderThan time.Duration) int {
	if olderThan <= 0 {
		return 0
	}
	now := time.Now()
	var kept []Entry
	pruned := 0
	for _, entry := range r.Entries {
		if entry.Status == StatusMissing && entry.LastSeen.Before(now.Add(-olderThan)) {
			pruned++
			continue
		}
		kept = append(kept, entry)
	}
	r.Entries = kept
	return pruned
}

// FindByRepoID returns the entry matching the given repo_id, or nil.
func (r *Registry) FindByRepoID(repoID string) *Entry {
	for i := range r.Entries {
		if r.Entries[i].RepoID == repoID {
			return &r.Entries[i]
		}
	}
	return nil
}
