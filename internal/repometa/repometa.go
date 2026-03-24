// SPDX-License-Identifier: MIT
package repometa

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
	"go.yaml.in/yaml/v3"
)

const (
	PreferredFilename = ".repokeeper-repo.yaml"
	LegacyFilename    = "repokeeper.yaml"
	APIVersion        = "repokeeper/v1"
	Kind              = "RepoMetadata"
)

var ErrNotFound = errors.New("repo metadata file not found")

func Load(repoRoot string) (string, *model.RepoMetadata, error) {
	path, err := discoverPath(repoRoot)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return path, nil, err
	}
	var metadata model.RepoMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return path, nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	metadata = normalize(metadata)
	if err := Validate(&metadata); err != nil {
		return path, nil, err
	}
	return path, &metadata, nil
}

func Save(repoRoot string, metadata *model.RepoMetadata, force bool) (string, error) {
	if metadata == nil {
		return "", fmt.Errorf("repo metadata is required")
	}
	normalized := normalize(*metadata)
	if normalized.APIVersion == "" {
		normalized.APIVersion = APIVersion
	}
	if normalized.Kind == "" {
		normalized.Kind = Kind
	}
	if err := Validate(&normalized); err != nil {
		return "", err
	}
	target, cleanupPaths, err := savePath(repoRoot, force)
	if err != nil {
		return "", err
	}
	if !force {
		if _, err := os.Stat(target); err == nil {
			return "", fmt.Errorf("repo metadata already exists at %q (use --force to overwrite)", target)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	data, err := yaml.Marshal(&normalized)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	for _, cleanupPath := range cleanupPaths {
		if cleanupPath == target {
			continue
		}
		if err := os.Remove(cleanupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return target, nil
}

func Apply(status *model.RepoStatus) {
	if status == nil || strings.TrimSpace(status.Path) == "" {
		return
	}
	path, metadata, err := Load(status.Path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return
		}
		if path != "" {
			status.RepoMetadataFile = path
		}
		status.RepoMetadataError = err.Error()
		status.RepoMetadata = nil
		return
	}
	status.RepoMetadataFile = path
	status.RepoMetadataError = ""
	status.RepoMetadata = metadata
	if metadata != nil && strings.TrimSpace(metadata.RepoID) != "" && strings.TrimSpace(status.RepoID) != "" && metadata.RepoID != status.RepoID {
		status.RepoMetadataError = fmt.Sprintf("repo metadata repo_id %q does not match discovered repo_id %q", metadata.RepoID, status.RepoID)
	}
}

func discoverPath(repoRoot string) (string, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		return "", ErrNotFound
	}
	preferred := filepath.Join(root, PreferredFilename)
	legacy := filepath.Join(root, LegacyFilename)
	preferredExists, err := fileExists(preferred)
	if err != nil {
		return "", err
	}
	legacyExists, err := fileExists(legacy)
	if err != nil {
		return "", err
	}
	switch {
	case preferredExists && legacyExists:
		return "", fmt.Errorf("multiple repo metadata files found: %q and %q", preferred, legacy)
	case preferredExists:
		return preferred, nil
	case legacyExists:
		return legacy, nil
	default:
		return "", ErrNotFound
	}
}

func savePath(repoRoot string, force bool) (string, []string, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		return "", nil, fmt.Errorf("repo root is required")
	}
	preferred := filepath.Join(root, PreferredFilename)
	legacy := filepath.Join(root, LegacyFilename)
	preferredExists, err := fileExists(preferred)
	if err != nil {
		return "", nil, err
	}
	legacyExists, err := fileExists(legacy)
	if err != nil {
		return "", nil, err
	}
	if preferredExists && legacyExists {
		if !force {
			return "", nil, fmt.Errorf("multiple repo metadata files found: %q and %q", preferred, legacy)
		}
		return preferred, []string{legacy}, nil
	}
	if preferredExists {
		return preferred, nil, nil
	}
	if legacyExists {
		return legacy, nil, nil
	}
	return preferred, nil, nil
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func Validate(metadata *model.RepoMetadata) error {
	if metadata == nil {
		return fmt.Errorf("repo metadata is required")
	}
	if metadata.APIVersion != "" && metadata.APIVersion != APIVersion {
		return fmt.Errorf("unsupported repo metadata apiVersion %q", metadata.APIVersion)
	}
	if metadata.Kind != "" && metadata.Kind != Kind {
		return fmt.Errorf("unsupported repo metadata kind %q", metadata.Kind)
	}
	if strings.TrimSpace(metadata.RepoID) == "" && strings.TrimSpace(metadata.Name) == "" && len(metadata.Labels) == 0 && len(metadata.Entrypoints) == 0 && len(metadata.Paths.Authoritative) == 0 && len(metadata.Paths.LowValue) == 0 && len(metadata.Provides) == 0 && len(metadata.RelatedRepos) == 0 {
		return fmt.Errorf("repo metadata must declare at least one non-empty field")
	}
	for key := range metadata.Labels {
		if err := validateMetadataKey(key); err != nil {
			return err
		}
	}
	for key, value := range metadata.Entrypoints {
		if err := validateMetadataKey(key); err != nil {
			return fmt.Errorf("invalid entrypoint key: %w", err)
		}
		if err := validateRelativePath(value, "entrypoint"); err != nil {
			return err
		}
	}
	for _, path := range metadata.Paths.Authoritative {
		if err := validateRelativePath(path, "authoritative path"); err != nil {
			return err
		}
	}
	for _, path := range metadata.Paths.LowValue {
		if err := validateRelativePath(path, "low-value path"); err != nil {
			return err
		}
	}
	for _, value := range metadata.Provides {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("provides entries cannot be empty")
		}
	}
	for _, related := range metadata.RelatedRepos {
		if strings.TrimSpace(related.RepoID) == "" {
			return fmt.Errorf("related_repos entries require repo_id")
		}
	}
	return nil
}

func normalize(metadata model.RepoMetadata) model.RepoMetadata {
	metadata.RepoID = strings.TrimSpace(metadata.RepoID)
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.APIVersion = strings.TrimSpace(metadata.APIVersion)
	metadata.Kind = strings.TrimSpace(metadata.Kind)
	metadata.Labels = normalizeMap(metadata.Labels)
	metadata.Entrypoints = normalizeRelativePathMap(metadata.Entrypoints)
	metadata.Paths.Authoritative = normalizeRelativePathSlice(metadata.Paths.Authoritative)
	metadata.Paths.LowValue = normalizeRelativePathSlice(metadata.Paths.LowValue)
	metadata.Provides = normalizeSlice(metadata.Provides)
	if len(metadata.RelatedRepos) > 0 {
		related := make([]model.RepoMetadataRelatedRepo, 0, len(metadata.RelatedRepos))
		for _, item := range metadata.RelatedRepos {
			repoID := strings.TrimSpace(item.RepoID)
			relationship := strings.TrimSpace(item.Relationship)
			if repoID == "" && relationship == "" {
				continue
			}
			related = append(related, model.RepoMetadataRelatedRepo{RepoID: repoID, Relationship: relationship})
		}
		sort.SliceStable(related, func(i, j int) bool {
			if related[i].RepoID == related[j].RepoID {
				return related[i].Relationship < related[j].Relationship
			}
			return related[i].RepoID < related[j].RepoID
		})
		metadata.RelatedRepos = related
	} else {
		metadata.RelatedRepos = nil
	}
	return metadata
}

func normalizeMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		out[trimmedKey] = trimmedValue
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, value := range in {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func normalizeRelativePathMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		out[trimmedKey] = filepath.ToSlash(trimmedValue)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeRelativePathSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, value := range in {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, filepath.ToSlash(trimmed))
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func validateMetadataKey(key string) error {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if strings.ContainsAny(trimmed, " \t\r\n=") {
		return fmt.Errorf("key %q cannot contain whitespace or '='", key)
	}
	return nil
}

func validateRelativePath(path string, label string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("%s %q must be relative", label, path)
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q must stay within the repository root", label, path)
	}
	return nil
}
