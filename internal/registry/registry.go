// SPDX-License-Identifier: MIT
// Package registry handles persistence and staleness detection for the
// per-machine repo registry.
package registry

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
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
	RepoID                  string              `yaml:"repo_id"`
	CheckoutID              string              `yaml:"checkout_id,omitempty"`
	Path                    string              `yaml:"path"`
	RemoteURL               string              `yaml:"remote_url"`
	Type                    string              `yaml:"type,omitempty"` // checkout | mirror
	Branch                  string              `yaml:"branch,omitempty"`
	Labels                  map[string]string   `yaml:"labels,omitempty"`
	Annotations             map[string]string   `yaml:"annotations,omitempty"`
	RepoMetadataFile        string              `yaml:"repo_metadata_file,omitempty"`
	RepoMetadataError       string              `yaml:"repo_metadata_error,omitempty"`
	RepoMetadataFingerprint string              `yaml:"repo_metadata_fingerprint,omitempty"`
	RepoMetadata            *model.RepoMetadata `yaml:"repo_metadata,omitempty"`
	LastSeen                time.Time           `yaml:"last_seen,omitempty"`
	Status                  EntryStatus         `yaml:"status"`
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

// Upsert adds or updates an entry in the registry by repo_id + checkout_id.
// If the repo_id + checkout_id already exists, it updates path, last_seen, and status.
// If new, it appends the entry.
func (r *Registry) Upsert(entry Entry) {
	entry.CheckoutID = checkoutIDFromEntry(entry)

	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == entry.RepoID && r.Entries[i].CheckoutID == entry.CheckoutID {
			merged := mergeRegistryEntry(r.Entries[i], entry)
			if !sameRegistryPath(r.Entries[i].Path, entry.Path) {
				merged.Status = StatusMoved
			}
			r.Entries[i] = merged
			r.collapseDuplicatePaths(i)
			return
		}
	}

	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if sameRegistryPath(r.Entries[i].Path, entry.Path) {
			r.Entries[i] = mergeRegistryEntry(r.Entries[i], entry)
			r.collapseDuplicatePaths(i)
			return
		}
	}
	if entry.Status == "" {
		entry.Status = StatusPresent
	}
	r.Entries = append(r.Entries, entry)
}

func (r *Registry) backfillCheckoutID(index int) {
	if r == nil || index < 0 || index >= len(r.Entries) {
		return
	}
	r.Entries[index].CheckoutID = checkoutIDFromEntry(r.Entries[index])
}

func checkoutIDFromEntry(entry Entry) string {
	if entry.CheckoutID != "" {
		return entry.CheckoutID
	}
	return defaultCheckoutIDFromPath(entry.Path)
}

func defaultCheckoutIDFromPath(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
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

func mergeRegistryEntry(existing, incoming Entry) Entry {
	merged := incoming
	merged.CheckoutID = checkoutIDFromEntry(merged)
	if merged.Status == "" {
		merged.Status = StatusPresent
	}
	if merged.Type == "" {
		merged.Type = existing.Type
	}
	if merged.Branch == "" {
		merged.Branch = existing.Branch
	}
	if merged.Labels == nil && len(existing.Labels) > 0 {
		merged.Labels = cloneStringMap(existing.Labels)
	}
	if merged.Annotations == nil && len(existing.Annotations) > 0 {
		merged.Annotations = cloneStringMap(existing.Annotations)
	}
	if merged.RepoMetadataFile == "" {
		merged.RepoMetadataFile = existing.RepoMetadataFile
	}
	if merged.RepoMetadataError == "" {
		merged.RepoMetadataError = existing.RepoMetadataError
	}
	if merged.RepoMetadataFingerprint == "" {
		merged.RepoMetadataFingerprint = existing.RepoMetadataFingerprint
	}
	if merged.RepoMetadata == nil && existing.RepoMetadata != nil {
		merged.RepoMetadata = cloneRepoMetadata(existing.RepoMetadata)
	}
	return merged
}

func sameRegistryPath(a, b string) bool {
	left, ok := canonicalRegistryPath(a)
	if !ok {
		return false
	}
	right, ok := canonicalRegistryPath(b)
	if !ok {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func canonicalRegistryPath(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false
	}
	return filepath.Clean(trimmed), true
}

func (r *Registry) collapseDuplicatePaths(keepIndex int) {
	if r == nil || keepIndex < 0 || keepIndex >= len(r.Entries) {
		return
	}
	keep := r.Entries[keepIndex]
	collapsed := make([]Entry, 0, len(r.Entries))
	for i := range r.Entries {
		if i == keepIndex {
			continue
		}
		if sameRegistryPath(r.Entries[i].Path, keep.Path) {
			keep = mergeRegistryEntry(r.Entries[i], keep)
			continue
		}
		collapsed = append(collapsed, r.Entries[i])
	}
	collapsed = append(collapsed, keep)
	r.Entries = collapsed
}

func SeedRepoMetadataStatus(entry Entry, status *model.RepoStatus) {
	if status == nil {
		return
	}
	status.RepoMetadataFile = entry.RepoMetadataFile
	status.RepoMetadataError = entry.RepoMetadataError
	status.RepoMetadataFingerprint = entry.RepoMetadataFingerprint
	status.RepoMetadata = cloneRepoMetadata(entry.RepoMetadata)
}

func StoreRepoMetadataStatus(entry *Entry, status model.RepoStatus) {
	if entry == nil {
		return
	}
	entry.RepoMetadataFile = status.RepoMetadataFile
	entry.RepoMetadataError = status.RepoMetadataError
	entry.RepoMetadataFingerprint = status.RepoMetadataFingerprint
	entry.RepoMetadata = cloneRepoMetadata(status.RepoMetadata)
}

func cloneRepoMetadata(in *model.RepoMetadata) *model.RepoMetadata {
	if in == nil {
		return nil
	}
	out := *in
	out.Labels = cloneStringMap(in.Labels)
	out.Entrypoints = cloneStringMap(in.Entrypoints)
	out.Paths.Authoritative = cloneStringSlice(in.Paths.Authoritative)
	out.Paths.LowValue = cloneStringSlice(in.Paths.LowValue)
	out.Provides = cloneStringSlice(in.Provides)
	if len(in.RelatedRepos) > 0 {
		out.RelatedRepos = make([]model.RepoMetadataRelatedRepo, len(in.RelatedRepos))
		copy(out.RelatedRepos, in.RelatedRepos)
	} else {
		out.RelatedRepos = nil
	}
	return &out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
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
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == repoID {
			return &r.Entries[i]
		}
	}
	return nil
}

// FindEntriesByRepoID returns all entries matching the given repo_id.
func (r *Registry) FindEntriesByRepoID(repoID string) []Entry {
	if r == nil {
		return nil
	}
	entries := make([]Entry, 0)
	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == repoID {
			entries = append(entries, r.Entries[i])
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

// FindByRepoIDAndCheckoutID returns the entry matching repoID and checkoutID,
// or nil if not found.
func (r *Registry) FindByRepoIDAndCheckoutID(repoID, checkoutID string) *Entry {
	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == repoID && r.Entries[i].CheckoutID == checkoutID {
			return &r.Entries[i]
		}
	}
	return nil
}

// FindEntry returns the entry matching repoID and path (exact match first,
// then repoID-only fallback), or nil if not found.
func (r *Registry) FindEntry(repoID, path string) *Entry {
	idx := r.FindEntryIndex(repoID, path)
	if idx < 0 {
		return nil
	}
	return &r.Entries[idx]
}

// FindEntryIndex returns the index of the entry matching repoID and path
// (exact match first, then repoID-only fallback), or -1 if not found.
func (r *Registry) FindEntryIndex(repoID, path string) int {
	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == repoID && r.Entries[i].Path == path {
			return i
		}
	}
	for i := range r.Entries {
		r.backfillCheckoutID(i)
		if r.Entries[i].RepoID == repoID {
			return i
		}
	}
	return -1
}
