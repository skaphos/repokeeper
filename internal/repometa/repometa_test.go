// SPDX-License-Identifier: MIT
package repometa

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"go.yaml.in/yaml/v3"
)

func TestLoadPrefersHiddenFile(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	hidden := filepath.Join(repo, PreferredFilename)
	legacy := filepath.Join(repo, LegacyFilename)
	writeRepoMetadataFile(t, hidden, model.RepoMetadata{Name: "Hidden"})
	writeRepoMetadataFile(t, legacy, model.RepoMetadata{Name: "Legacy"})

	_, _, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), "multiple repo metadata files") {
		t.Fatalf("expected multiple-files error, got %v", err)
	}

	if err := os.Remove(legacy); err != nil {
		t.Fatalf("remove legacy metadata: %v", err)
	}
	path, metadata, err := Load(repo)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	if path != hidden {
		t.Fatalf("expected hidden metadata path %q, got %q", hidden, path)
	}
	if metadata == nil || metadata.Name != "Hidden" {
		t.Fatalf("expected hidden metadata, got %#v", metadata)
	}
}

func TestLoadRejectsTraversingPaths(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path := filepath.Join(repo, PreferredFilename)
	writeRepoMetadataFile(t, path, model.RepoMetadata{
		Entrypoints: map[string]string{"readme": "../README.md"},
	})

	_, _, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), "must stay within the repository root") {
		t.Fatalf("expected traversal validation error, got %v", err)
	}
}

func TestSaveWritesCanonicalFile(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path, err := Save(repo, &model.RepoMetadata{
		Name:   "Repo",
		Labels: map[string]string{"role": "docs"},
	}, false)
	if err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if path != filepath.Join(repo, PreferredFilename) {
		t.Fatalf("expected preferred path, got %q", path)
	}
	loadedPath, metadata, err := Load(repo)
	if err != nil {
		t.Fatalf("reload metadata: %v", err)
	}
	if loadedPath != path {
		t.Fatalf("expected load path %q, got %q", path, loadedPath)
	}
	if metadata.APIVersion != APIVersion || metadata.Kind != Kind {
		t.Fatalf("expected canonical schema markers, got %#v", metadata)
	}
	if metadata.Labels["role"] != "docs" {
		t.Fatalf("expected saved labels, got %#v", metadata.Labels)
	}
}

func TestSaveForceResolvesDualFileConflict(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	hidden := filepath.Join(repo, PreferredFilename)
	legacy := filepath.Join(repo, LegacyFilename)
	writeRepoMetadataFile(t, hidden, model.RepoMetadata{Name: "Hidden"})
	writeRepoMetadataFile(t, legacy, model.RepoMetadata{Name: "Legacy"})

	path, err := Save(repo, &model.RepoMetadata{
		Name:   "Resolved",
		Labels: map[string]string{"role": "docs"},
	}, true)
	if err != nil {
		t.Fatalf("force save metadata: %v", err)
	}
	if path != hidden {
		t.Fatalf("expected preferred file to win, got %q", path)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("expected legacy file removed, got err=%v", err)
	}
	_, metadata, err := Load(repo)
	if err != nil {
		t.Fatalf("load resolved metadata: %v", err)
	}
	if metadata.Name != "Resolved" {
		t.Fatalf("expected resolved metadata, got %#v", metadata)
	}
}

func TestValidateRejectsInvalidMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		metadata *model.RepoMetadata
		wantErr  string
	}{
		{name: "nil metadata", metadata: nil, wantErr: "repo metadata is required"},
		{name: "unsupported api version", metadata: &model.RepoMetadata{APIVersion: "v2", Name: "Repo"}, wantErr: "unsupported repo metadata apiVersion"},
		{name: "unsupported kind", metadata: &model.RepoMetadata{Kind: "Other", Name: "Repo"}, wantErr: "unsupported repo metadata kind"},
		{name: "empty metadata", metadata: &model.RepoMetadata{}, wantErr: "must declare at least one non-empty field"},
		{name: "invalid label key", metadata: &model.RepoMetadata{Name: "Repo", Labels: map[string]string{"bad key": "docs"}}, wantErr: "cannot contain whitespace or '='"},
		{name: "invalid entrypoint key", metadata: &model.RepoMetadata{Name: "Repo", Entrypoints: map[string]string{"bad key": "README.md"}}, wantErr: "invalid entrypoint key"},
		{name: "absolute entrypoint path", metadata: &model.RepoMetadata{Name: "Repo", Entrypoints: map[string]string{"readme": testAbsolutePath()}}, wantErr: "must be relative"},
		{name: "traversing authoritative path", metadata: &model.RepoMetadata{Name: "Repo", Paths: model.RepoMetadataPaths{Authoritative: []string{"../docs"}}}, wantErr: "must stay within the repository root"},
		{name: "empty provides entry", metadata: &model.RepoMetadata{Name: "Repo", Provides: []string{"docs", "  "}}, wantErr: "provides entries cannot be empty"},
		{name: "missing related repo id", metadata: &model.RepoMetadata{Name: "Repo", RelatedRepos: []model.RepoMetadataRelatedRepo{{Relationship: "depends-on"}}}, wantErr: "related_repos entries require repo_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(tt.metadata)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func testAbsolutePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(`C:\`, "tmp", "README.md")
	}
	return filepath.Join(string(filepath.Separator), "tmp", "README.md")
}

func TestNormalizeTrimsFiltersAndSorts(t *testing.T) {
	t.Parallel()
	metadata := normalize(model.RepoMetadata{
		APIVersion: "  ",
		Kind:       "  ",
		RepoID:     "  github.com/example/repo  ",
		Name:       "  Example Repo  ",
		Labels: map[string]string{
			" role ": " docs ",
			" ":      "skip",
			"team":   " ",
		},
		Entrypoints: map[string]string{
			" readme ": " README.md ",
			"":         "skip",
		},
		Paths: model.RepoMetadataPaths{
			Authoritative: []string{" docs ", "", "cmd"},
			LowValue:      []string{" tmp ", "  ", "vendor"},
		},
		Provides: []string{" cli ", "", "api"},
		RelatedRepos: []model.RepoMetadataRelatedRepo{
			{RepoID: " github.com/example/z ", Relationship: " sibling "},
			{RepoID: "", Relationship: ""},
			{RepoID: " github.com/example/a ", Relationship: " depends-on "},
		},
	})

	if metadata.APIVersion != "" || metadata.Kind != "" {
		t.Fatalf("expected empty schema markers after trim, got %#v", metadata)
	}
	if metadata.RepoID != "github.com/example/repo" || metadata.Name != "Example Repo" {
		t.Fatalf("expected trimmed repo identity fields, got %#v", metadata)
	}
	if !reflect.DeepEqual(metadata.Labels, map[string]string{"role": "docs"}) {
		t.Fatalf("unexpected normalized labels: %#v", metadata.Labels)
	}
	if !reflect.DeepEqual(metadata.Entrypoints, map[string]string{"readme": "README.md"}) {
		t.Fatalf("unexpected normalized entrypoints: %#v", metadata.Entrypoints)
	}
	if !reflect.DeepEqual(metadata.Paths.Authoritative, []string{"cmd", "docs"}) {
		t.Fatalf("unexpected authoritative paths: %#v", metadata.Paths.Authoritative)
	}
	if !reflect.DeepEqual(metadata.Paths.LowValue, []string{"tmp", "vendor"}) {
		t.Fatalf("unexpected low-value paths: %#v", metadata.Paths.LowValue)
	}
	if !reflect.DeepEqual(metadata.Provides, []string{"api", "cli"}) {
		t.Fatalf("unexpected provides: %#v", metadata.Provides)
	}
	if want := []model.RepoMetadataRelatedRepo{{RepoID: "github.com/example/a", Relationship: "depends-on"}, {RepoID: "github.com/example/z", Relationship: "sibling"}}; !reflect.DeepEqual(metadata.RelatedRepos, want) {
		t.Fatalf("unexpected related repos: %#v", metadata.RelatedRepos)
	}

	empty := normalize(model.RepoMetadata{
		Labels:      map[string]string{"": "skip"},
		Entrypoints: map[string]string{"": "skip"},
		Paths: model.RepoMetadataPaths{
			Authoritative: []string{"  "},
			LowValue:      []string{"  "},
		},
		Provides:     []string{"  "},
		RelatedRepos: []model.RepoMetadataRelatedRepo{{RepoID: "", Relationship: ""}},
	})
	if empty.Labels != nil || empty.Entrypoints != nil || empty.Paths.Authoritative != nil || empty.Paths.LowValue != nil || empty.Provides != nil || len(empty.RelatedRepos) != 0 {
		t.Fatalf("expected empty collections to normalize to nil, got %#v", empty)
	}
}

func TestApplyPopulatesStatusMetadata(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	missing := &model.RepoStatus{}
	Apply(nil)
	Apply(missing)
	if missing.RepoMetadata != nil || missing.RepoMetadataFile != "" || missing.RepoMetadataError != "" {
		t.Fatalf("expected blank status to remain untouched, got %#v", missing)
	}

	notFound := &model.RepoStatus{Path: repo, RepoID: "github.com/example/repo"}
	Apply(notFound)
	if notFound.RepoMetadata != nil || notFound.RepoMetadataFile != "" || notFound.RepoMetadataError != "" {
		t.Fatalf("expected missing metadata file to be ignored, got %#v", notFound)
	}

	invalidPath := filepath.Join(repo, PreferredFilename)
	if err := os.WriteFile(invalidPath, []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nentrypoints:\n  readme: ../README.md\n"), 0o644); err != nil {
		t.Fatalf("write invalid metadata: %v", err)
	}
	invalid := &model.RepoStatus{Path: repo, RepoID: "github.com/example/repo"}
	Apply(invalid)
	if invalid.RepoMetadata != nil {
		t.Fatalf("expected invalid metadata to be discarded, got %#v", invalid.RepoMetadata)
	}
	if invalid.RepoMetadataFile != invalidPath {
		t.Fatalf("expected metadata file path %q, got %q", invalidPath, invalid.RepoMetadataFile)
	}
	if !strings.Contains(invalid.RepoMetadataError, "must stay within the repository root") {
		t.Fatalf("expected validation error, got %q", invalid.RepoMetadataError)
	}

	validPath := filepath.Join(repo, PreferredFilename)
	writeRepoMetadataFile(t, validPath, model.RepoMetadata{Name: "Repo", RepoID: "github.com/example/repo", Provides: []string{"guides"}})
	matched := &model.RepoStatus{Path: repo, RepoID: "github.com/example/repo"}
	Apply(matched)
	if matched.RepoMetadataFile != validPath || matched.RepoMetadata == nil || matched.RepoMetadata.Name != "Repo" {
		t.Fatalf("expected metadata to load successfully, got %#v", matched)
	}
	if matched.RepoMetadataError != "" {
		t.Fatalf("expected no metadata error, got %q", matched.RepoMetadataError)
	}

	mismatched := &model.RepoStatus{Path: repo, RepoID: "github.com/example/other"}
	Apply(mismatched)
	if mismatched.RepoMetadata == nil || mismatched.RepoMetadata.RepoID != "github.com/example/repo" {
		t.Fatalf("expected metadata to remain attached on repo id mismatch, got %#v", mismatched)
	}
	if !strings.Contains(mismatched.RepoMetadataError, "does not match discovered repo_id") {
		t.Fatalf("expected mismatch warning, got %q", mismatched.RepoMetadataError)
	}
}

func TestApplyUsesSnapshotWhenFingerprintMatches(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path := filepath.Join(repo, PreferredFilename)
	writeRepoMetadataFile(t, path, model.RepoMetadata{
		RepoID:   "github.com/example/repo",
		Name:     "Cached Repo",
		Provides: []string{"guides"},
	})

	cached := &model.RepoMetadata{
		APIVersion: APIVersion,
		Kind:       Kind,
		RepoID:     "github.com/example/repo",
		Name:       "Cached Repo",
		Provides:   []string{"guides"},
	}
	status := &model.RepoStatus{
		Path:                    repo,
		RepoID:                  "github.com/example/repo",
		RepoMetadataFile:        path,
		RepoMetadataFingerprint: mustMetadataFingerprint(t, repo),
		RepoMetadata:            cached,
	}

	rewriteFilePreservingFingerprint(t, path, []byte("not: [valid yaml"))

	Apply(status)

	if status.RepoMetadataFile != path {
		t.Fatalf("expected cached metadata file %q to be reused, got %q", path, status.RepoMetadataFile)
	}
	if !reflect.DeepEqual(status.RepoMetadata, cached) {
		t.Fatalf("expected cached snapshot to be reused, got %#v", status.RepoMetadata)
	}
	if status.RepoMetadataError != "" {
		t.Fatalf("expected no metadata error when cached fingerprint matches, got %q", status.RepoMetadataError)
	}
}

func TestApplyRefreshesWhenFingerprintChanges(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path := filepath.Join(repo, PreferredFilename)
	writeRepoMetadataFile(t, path, model.RepoMetadata{
		RepoID:   "github.com/example/repo",
		Name:     "Old Snapshot",
		Provides: []string{"guides"},
	})

	status := &model.RepoStatus{
		Path:                    repo,
		RepoID:                  "github.com/example/repo",
		RepoMetadataFile:        path,
		RepoMetadataFingerprint: mustMetadataFingerprint(t, repo),
		RepoMetadata: &model.RepoMetadata{
			APIVersion: APIVersion,
			Kind:       Kind,
			RepoID:     "github.com/example/repo",
			Name:       "Old Snapshot",
			Provides:   []string{"guides"},
		},
		RepoMetadataError: "cached metadata error should be replaced on refresh",
	}

	writeRepoMetadataFile(t, path, model.RepoMetadata{
		RepoID:      "github.com/example/repo",
		Name:        "Fresh Snapshot",
		Labels:      map[string]string{"role": "docs"},
		Entrypoints: map[string]string{"readme": "README.md"},
		Provides:    []string{"guides", "api"},
	})

	Apply(status)

	if status.RepoMetadataFile != path {
		t.Fatalf("expected refreshed metadata file path %q, got %q", path, status.RepoMetadataFile)
	}
	if status.RepoMetadata == nil {
		t.Fatalf("expected refreshed metadata snapshot, got nil")
	}
	if status.RepoMetadata.Name != "Fresh Snapshot" {
		t.Fatalf("expected refreshed metadata name, got %#v", status.RepoMetadata)
	}
	if status.RepoMetadata.Labels["role"] != "docs" {
		t.Fatalf("expected refreshed labels, got %#v", status.RepoMetadata.Labels)
	}
	if status.RepoMetadata.Entrypoints["readme"] != "README.md" {
		t.Fatalf("expected refreshed entrypoints, got %#v", status.RepoMetadata.Entrypoints)
	}
	if status.RepoMetadataError != "" {
		t.Fatalf("expected changed fingerprint to clear cached error, got %q", status.RepoMetadataError)
	}
}

func TestApplyHandlesMetadataDeletion(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path := filepath.Join(repo, PreferredFilename)
	writeRepoMetadataFile(t, path, model.RepoMetadata{
		RepoID:   "github.com/example/repo",
		Name:     "Cached Repo",
		Provides: []string{"guides"},
	})

	status := &model.RepoStatus{
		Path:                    repo,
		RepoID:                  "github.com/example/repo",
		RepoMetadataFile:        path,
		RepoMetadataFingerprint: mustMetadataFingerprint(t, repo),
		RepoMetadata: &model.RepoMetadata{
			APIVersion: APIVersion,
			Kind:       Kind,
			RepoID:     "github.com/example/repo",
			Name:       "Cached Repo",
			Provides:   []string{"guides"},
		},
		RepoMetadataError: "stale cached error",
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove metadata file: %v", err)
	}

	Apply(status)

	if status.RepoMetadataFile != "" {
		t.Fatalf("expected metadata file to clear after deletion, got %q", status.RepoMetadataFile)
	}
	if status.RepoMetadata != nil {
		t.Fatalf("expected metadata snapshot to clear after deletion, got %#v", status.RepoMetadata)
	}
	if status.RepoMetadataError != "" {
		t.Fatalf("expected deletion to surface missing-file semantics without stale error, got %q", status.RepoMetadataError)
	}
}

func TestApplyCachesDualFileConflict(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	hidden := filepath.Join(repo, PreferredFilename)
	legacy := filepath.Join(repo, LegacyFilename)
	writeRepoMetadataFile(t, hidden, model.RepoMetadata{
		RepoID: "github.com/example/repo",
		Name:   "Cached Repo",
	})

	status := &model.RepoStatus{
		Path:                    repo,
		RepoID:                  "github.com/example/repo",
		RepoMetadataFile:        hidden,
		RepoMetadataFingerprint: mustMetadataFingerprint(t, repo),
		RepoMetadata: &model.RepoMetadata{
			APIVersion: APIVersion,
			Kind:       Kind,
			RepoID:     "github.com/example/repo",
			Name:       "Cached Repo",
		},
	}

	writeRepoMetadataFile(t, legacy, model.RepoMetadata{
		RepoID: "github.com/example/repo",
		Name:   "Legacy Repo",
	})

	Apply(status)

	if status.RepoMetadataFile != "" {
		t.Fatalf("expected dual-file conflict to clear stale selected metadata file, got %q", status.RepoMetadataFile)
	}
	if status.RepoMetadata != nil {
		t.Fatalf("expected dual-file conflict to discard cached metadata snapshot, got %#v", status.RepoMetadata)
	}
	if !strings.Contains(status.RepoMetadataError, "multiple repo metadata files") {
		t.Fatalf("expected dual-file conflict error, got %q", status.RepoMetadataError)
	}
}

func TestApplyCachesInvalidMetadataError(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	path := filepath.Join(repo, PreferredFilename)
	if err := os.WriteFile(path, []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nentrypoints:\n  readme: ../README.md\n# cached-invalid-padding\n"), 0o644); err != nil {
		t.Fatalf("write invalid metadata: %v", err)
	}

	status := &model.RepoStatus{
		Path:                    repo,
		RepoID:                  "github.com/example/repo",
		RepoMetadataFile:        path,
		RepoMetadataFingerprint: mustMetadataFingerprint(t, repo),
		RepoMetadataError:       "entrypoint \"../README.md\" must stay within the repository root",
	}

	rewriteFilePreservingFingerprint(t, path, []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Now Valid\n"))

	Apply(status)

	if status.RepoMetadataFile != path {
		t.Fatalf("expected cached invalid metadata path %q to be preserved, got %q", path, status.RepoMetadataFile)
	}
	if status.RepoMetadata != nil {
		t.Fatalf("expected cached invalid snapshot to remain empty, got %#v", status.RepoMetadata)
	}
	if !strings.Contains(status.RepoMetadataError, "must stay within the repository root") {
		t.Fatalf("expected cached invalid metadata error to be reused, got %q", status.RepoMetadataError)
	}
}

func mustMarshalRepoMetadata(t *testing.T, metadata model.RepoMetadata) []byte {
	t.Helper()
	if metadata.APIVersion == "" {
		metadata.APIVersion = APIVersion
	}
	if metadata.Kind == "" {
		metadata.Kind = Kind
	}
	data, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	return data
}

func rewriteFilePreservingFingerprint(t *testing.T, path string, replacement []byte) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat metadata file: %v", err)
	}
	if len(replacement) > int(info.Size()) {
		t.Fatalf("replacement content length %d exceeds preserved fingerprint size %d", len(replacement), info.Size())
	}
	padded := append([]byte(nil), replacement...)
	if len(padded) < int(info.Size()) {
		padded = append(padded, []byte(strings.Repeat(" ", int(info.Size())-len(padded)))...)
	}
	if err := os.WriteFile(path, padded, 0o644); err != nil {
		t.Fatalf("rewrite metadata file: %v", err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatalf("restore metadata file times: %v", err)
	}
}

func writeRepoMetadataFile(t *testing.T, path string, metadata model.RepoMetadata) {
	t.Helper()
	data := mustMarshalRepoMetadata(t, metadata)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

func mustMetadataFingerprint(t *testing.T, repo string) string {
	t.Helper()
	state := discoverMetadataState(repo)
	if state.err != nil {
		t.Fatalf("discover metadata state: %v", state.err)
	}
	if state.fingerprint == "" {
		t.Fatalf("expected metadata fingerprint for %q", repo)
	}
	return state.fingerprint
}
