// SPDX-License-Identifier: MIT
package repometa

import (
	"os"
	"path/filepath"
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

func writeRepoMetadataFile(t *testing.T, path string, metadata model.RepoMetadata) {
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}
