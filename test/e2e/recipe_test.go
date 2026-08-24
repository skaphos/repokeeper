// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"go.yaml.in/yaml/v3"
)

type WorkspaceRecipe struct {
	SchemaVersion  int                  `json:"schema_version"`
	Name           string               `json:"name"`
	Repositories   []RepositoryRecipe   `json:"repositories"`
	MissingEntries []MissingEntryRecipe `json:"missing_entries"`
	Config         ConfigRecipe         `json:"config"`
}

type RepositoryRecipe struct {
	Name          string            `json:"name"`
	RemotePath    string            `json:"remote_path"`
	CheckoutPath  string            `json:"checkout_path"`
	Files         map[string]string `json:"files"`
	Branches      []BranchRecipe    `json:"branches"`
	CurrentBranch string            `json:"current_branch"`
	Upstream      UpstreamRecipe    `json:"upstream"`
	RemoteCommits []CommitRecipe    `json:"remote_commits"`
	DirtyFiles    map[string]string `json:"dirty_files"`
	Metadata      *MetadataRecipe   `json:"metadata"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
}

type BranchRecipe struct {
	Name    string         `json:"name"`
	Base    string         `json:"base"`
	Commits []CommitRecipe `json:"commits"`
}

type CommitRecipe struct {
	Message string            `json:"message"`
	Files   map[string]string `json:"files"`
}

type UpstreamRecipe struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

type MetadataRecipe struct {
	Name          string               `json:"name"`
	Entrypoints   map[string]string    `json:"entrypoints"`
	Authoritative []string             `json:"authoritative"`
	LowValue      []string             `json:"low_value"`
	Related       []RelationshipRecipe `json:"related_repositories"`
}

type RelationshipRecipe struct {
	Repository   string `json:"repository"`
	Relationship string `json:"relationship"`
}

type MissingEntryRecipe struct {
	Name       string `json:"name"`
	RepoID     string `json:"repo_id"`
	CheckoutID string `json:"checkout_id"`
	Path       string `json:"path"`
	RemoteURL  string `json:"remote_url"`
	Status     string `json:"status"`
}

type ConfigRecipe struct {
	ExternalRegistry bool     `json:"external_registry"`
	Exclude          []string `json:"exclude"`
}

type RegistrySeedMode int

const (
	SeedAllEntries RegistrySeedMode = iota
	SeedMissingOnly
)

type MaterializedWorkspace struct {
	Root           string
	WorkspaceRoot  string
	RemotesRoot    string
	SourcesRoot    string
	HomeRoot       string
	TempRoot       string
	CacheRoot      string
	ConfigPath     string
	RegistryPath   string
	Env            []string
	Repositories   map[string]MaterializedRepository
	MissingEntries map[string]MaterializedMissingEntry
}

type MaterializedRepository struct {
	Name         string
	CheckoutPath string
	RemotePath   string
	RemoteURL    string
	RepoID       string
	BaselineHEAD string
	DirtyHashes  map[string]string
}

type MaterializedMissingEntry struct {
	Name   string
	Path   string
	RepoID string
}

func loadRecipe(path string) (WorkspaceRecipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceRecipe{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var recipe WorkspaceRecipe
	if err := decoder.Decode(&recipe); err != nil {
		return WorkspaceRecipe{}, fmt.Errorf("decode recipe %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return WorkspaceRecipe{}, fmt.Errorf("decode recipe %s: trailing JSON: %w", path, err)
	}
	if err := validateRecipe(recipe); err != nil {
		return WorkspaceRecipe{}, err
	}
	return recipe, nil
}

func validateRecipe(recipe WorkspaceRecipe) error {
	if recipe.SchemaVersion != 1 {
		return fmt.Errorf("schema_version: got %d, want 1", recipe.SchemaVersion)
	}
	if strings.TrimSpace(recipe.Name) == "" {
		return fmt.Errorf("name: must not be empty")
	}
	names := map[string]string{}
	paths := map[string]string{}
	remotePaths := map[string]string{}
	fixtureNames := map[string]string{}
	repositoryNames := map[string]bool{}
	for i, repository := range recipe.Repositories {
		field := fmt.Sprintf("repositories[%d]", i)
		if err := uniqueName(names, repository.Name, field+".name"); err != nil {
			return err
		}
		repositoryNames[repository.Name] = true
		// safeFixtureName collapses every non-alphanumeric rune, so distinct
		// names can share one fixture identifier. That identifier becomes the
		// remote URL and therefore the repo ID, so a collision would make
		// materialization and registry state ambiguous.
		fixture := safeFixtureName(repository.Name)
		if fixture == "" {
			return fmt.Errorf("%s.name: %q leaves no characters for a fixture identifier", field, repository.Name)
		}
		if previous, exists := fixtureNames[fixture]; exists {
			return fmt.Errorf("%s.name: %q shares fixture identifier %q with %s", field, repository.Name, fixture, previous)
		}
		fixtureNames[fixture] = field + ".name"
		for _, candidate := range []struct{ name, value string }{{field + ".remote_path", repository.RemotePath}, {field + ".checkout_path", repository.CheckoutPath}} {
			if err := validatePortablePath(candidate.name, candidate.value); err != nil {
				return err
			}
		}
		if previous := overlappingPath(paths, repository.CheckoutPath); previous != "" {
			return fmt.Errorf("%s.checkout_path: collides with %s", field, previous)
		}
		paths[filepath.ToSlash(repository.CheckoutPath)] = field + ".checkout_path"
		if previous := overlappingPath(remotePaths, repository.RemotePath); previous != "" {
			return fmt.Errorf("%s.remote_path: collides with %s", field, previous)
		}
		remotePaths[filepath.ToSlash(repository.RemotePath)] = field + ".remote_path"
		if len(repository.Files) == 0 {
			return fmt.Errorf("%s.files: at least one committed file is required", field)
		}
		for path := range repository.Files {
			if err := validatePortablePath(field+".files["+path+"]", path); err != nil {
				return err
			}
		}
		for path := range repository.DirtyFiles {
			if err := validatePortablePath(field+".dirty_files["+path+"]", path); err != nil {
				return err
			}
		}
		branchNames := map[string]bool{"main": true}
		for branchIndex, branch := range repository.Branches {
			branchField := fmt.Sprintf("%s.branches[%d]", field, branchIndex)
			if strings.TrimSpace(branch.Name) == "" || branchNames[branch.Name] {
				return fmt.Errorf("%s.name: empty or duplicate branch %q", branchField, branch.Name)
			}
			branchNames[branch.Name] = true
			if branch.Base != "" && !branchNames[branch.Base] {
				return fmt.Errorf("%s.base: unknown branch %q", branchField, branch.Base)
			}
			if err := validateCommits(branchField+".commits", branch.Commits); err != nil {
				return err
			}
		}
		if err := validateCommits(field+".remote_commits", repository.RemoteCommits); err != nil {
			return err
		}
		if !branchNames[repository.CurrentBranch] {
			return fmt.Errorf("%s.current_branch: unknown branch %q", field, repository.CurrentBranch)
		}
		if repository.Upstream.Remote != "origin" {
			return fmt.Errorf("%s.upstream.remote: unknown remote %q", field, repository.Upstream.Remote)
		}
		if !branchNames[repository.Upstream.Branch] {
			return fmt.Errorf("%s.upstream.branch: unknown branch %q", field, repository.Upstream.Branch)
		}
	}
	for i, missing := range recipe.MissingEntries {
		field := fmt.Sprintf("missing_entries[%d]", i)
		if err := uniqueName(names, missing.Name, field+".name"); err != nil {
			return err
		}
		if err := validatePortablePath(field+".path", missing.Path); err != nil {
			return err
		}
		portable := filepath.ToSlash(missing.Path)
		if previous := overlappingPath(paths, portable); previous != "" {
			return fmt.Errorf("%s.path: collides with %s", field, previous)
		}
		paths[portable] = field + ".path"
		if missing.Status != "missing" {
			return fmt.Errorf("%s.status: got %q, want missing", field, missing.Status)
		}
	}
	for i, repository := range recipe.Repositories {
		if repository.Metadata == nil {
			continue
		}
		for j, related := range repository.Metadata.Related {
			if !repositoryNames[related.Repository] {
				return fmt.Errorf("repositories[%d].metadata.related_repositories[%d].repository: unknown repository %q", i, j, related.Repository)
			}
		}
	}
	return nil
}

func overlappingPath(existing map[string]string, candidate string) string {
	candidate = strings.Trim(filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate))), "/")
	for path, field := range existing {
		path = strings.Trim(filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))), "/")
		if path == candidate || strings.HasPrefix(path, candidate+"/") || strings.HasPrefix(candidate, path+"/") {
			return field
		}
	}
	return ""
}

func validateCommits(field string, commits []CommitRecipe) error {
	for i, commit := range commits {
		if strings.TrimSpace(commit.Message) == "" {
			return fmt.Errorf("%s[%d].message: must not be empty", field, i)
		}
		for path := range commit.Files {
			if err := validatePortablePath(fmt.Sprintf("%s[%d].files[%s]", field, i, path), path); err != nil {
				return err
			}
		}
	}
	return nil
}

func uniqueName(names map[string]string, name, field string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s: must not be empty", field)
	}
	if previous, exists := names[name]; exists {
		return fmt.Errorf("%s: duplicate name %q (already used by %s)", field, name, previous)
	}
	names[name] = field
	return nil
}

func validatePortablePath(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s: path must not be empty", field)
	}
	if strings.Contains(value, "\\") || filepath.IsAbs(value) || hasDriveOrUNCPath(value) {
		return fmt.Errorf("%s: path %q must be a portable relative slash path", field, value)
	}
	cleaned := filepath.Clean(filepath.FromSlash(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s: path %q escapes its namespace", field, value)
	}
	return nil
}

func hasDriveOrUNCPath(value string) bool {
	return len(value) >= 2 && ((value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z'))) || strings.HasPrefix(value, "//"))
}

func containedPath(root, portable string) (string, error) {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(rootAbsolute, filepath.FromSlash(portable)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbsolute, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root %q", portable, root)
	}
	return candidate, nil
}

func materializeRecipe(parent context.Context, recipe WorkspaceRecipe, root string, seedMode RegistrySeedMode) (*MaterializedWorkspace, error) {
	if err := validateRecipe(recipe); err != nil {
		return nil, err
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	workspace := &MaterializedWorkspace{
		Root: absoluteRoot, WorkspaceRoot: filepath.Join(absoluteRoot, "workspace"), RemotesRoot: filepath.Join(absoluteRoot, "remotes"),
		SourcesRoot: filepath.Join(absoluteRoot, "sources"), HomeRoot: filepath.Join(absoluteRoot, "home"), TempRoot: filepath.Join(absoluteRoot, "tmp"),
		CacheRoot: filepath.Join(absoluteRoot, "cache"), ConfigPath: filepath.Join(absoluteRoot, "config", "config.yaml"),
		RegistryPath: filepath.Join(absoluteRoot, "config", "registry.yaml"), Repositories: map[string]MaterializedRepository{}, MissingEntries: map[string]MaterializedMissingEntry{},
	}
	for _, path := range []string{workspace.WorkspaceRoot, workspace.RemotesRoot, workspace.SourcesRoot, workspace.HomeRoot, workspace.TempRoot, workspace.CacheRoot, filepath.Dir(workspace.ConfigPath)} {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return nil, fmt.Errorf("create scenario path %s: %w", path, err)
		}
	}
	workspace.Env, err = buildChildEnvironment(absoluteRoot, workspace.ConfigPath)
	if err != nil {
		return nil, err
	}

	layouts := make(map[string]MaterializedRepository, len(recipe.Repositories))
	gitConfigPath := environmentMap(workspace.Env)["GIT_CONFIG_GLOBAL"]
	for _, repository := range recipe.Repositories {
		checkoutPath, pathErr := containedPath(workspace.WorkspaceRoot, repository.CheckoutPath)
		if pathErr != nil {
			return nil, fmt.Errorf("repository %s checkout: %w", repository.Name, pathErr)
		}
		remotePath, pathErr := containedPath(workspace.RemotesRoot, repository.RemotePath)
		if pathErr != nil {
			return nil, fmt.Errorf("repository %s remote: %w", repository.Name, pathErr)
		}
		remoteURL := "file:///repokeeper-e2e/" + url.PathEscape(safeFixtureName(repository.Name)) + ".git"
		layouts[repository.Name] = MaterializedRepository{Name: repository.Name, CheckoutPath: checkoutPath, RemotePath: remotePath, RemoteURL: remoteURL, RepoID: gitx.NormalizeURL(remoteURL), DirtyHashes: map[string]string{}}
		rewrite := fmt.Sprintf("[url %q]\n\tinsteadOf = %s\n", fileURL(remotePath), remoteURL)
		file, openErr := os.OpenFile(gitConfigPath, os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			return nil, fmt.Errorf("open isolated git config for local URL rewrite: %w", openErr)
		}
		_, writeErr := file.WriteString(rewrite)
		closeErr := file.Close()
		if writeErr != nil {
			return nil, fmt.Errorf("write isolated Git URL rewrite: %w", writeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close isolated Git config: %w", closeErr)
		}
	}

	for index, repository := range recipe.Repositories {
		layout := layouts[repository.Name]
		sourcePath := filepath.Join(workspace.SourcesRoot, fmt.Sprintf("%02d-%s", index, safeFixtureName(repository.Name)))
		if err := runGit(parent, workspace, workspace.Root, "init bare remote "+repository.Name, "init", "--bare", layout.RemotePath); err != nil {
			return nil, err
		}
		if err := runGit(parent, workspace, workspace.Root, "init source "+repository.Name, "init", "-b", "main", sourcePath); err != nil {
			return nil, err
		}
		if err := writeFixtureFiles(sourcePath, repository.Files); err != nil {
			return nil, err
		}
		if repository.Metadata != nil {
			metadata := model.RepoMetadata{APIVersion: "repokeeper/v1", Kind: "RepoMetadata", RepoID: layout.RepoID, Name: repository.Metadata.Name, Entrypoints: repository.Metadata.Entrypoints, Paths: model.RepoMetadataPaths{Authoritative: repository.Metadata.Authoritative, LowValue: repository.Metadata.LowValue}}
			for _, relation := range repository.Metadata.Related {
				target, exists := layouts[relation.Repository]
				if !exists {
					return nil, fmt.Errorf("metadata relation target %q disappeared after validation", relation.Repository)
				}
				metadata.RelatedRepos = append(metadata.RelatedRepos, model.RepoMetadataRelatedRepo{RepoID: target.RepoID, Relationship: relation.Relationship})
			}
			data, marshalErr := yaml.Marshal(metadata)
			if marshalErr != nil {
				return nil, marshalErr
			}
			if err := os.WriteFile(filepath.Join(sourcePath, ".repokeeper-repo.yaml"), data, 0o600); err != nil {
				return nil, err
			}
		}
		if err := commitAll(parent, workspace, sourcePath, "initial fixture"); err != nil {
			return nil, err
		}
		if err := runGit(parent, workspace, sourcePath, "add origin "+repository.Name, "remote", "add", "origin", layout.RemoteURL); err != nil {
			return nil, err
		}
		if err := runGit(parent, workspace, sourcePath, "push main "+repository.Name, "push", "-u", "origin", "main"); err != nil {
			return nil, err
		}
		for _, branch := range repository.Branches {
			base := branch.Base
			if base == "" {
				base = "main"
			}
			if err := runGit(parent, workspace, sourcePath, "create branch "+branch.Name, "switch", "-c", branch.Name, base); err != nil {
				return nil, err
			}
			for _, commit := range branch.Commits {
				if err := writeFixtureFiles(sourcePath, commit.Files); err != nil {
					return nil, err
				}
				if err := commitAll(parent, workspace, sourcePath, commit.Message); err != nil {
					return nil, err
				}
			}
			if err := runGit(parent, workspace, sourcePath, "push branch "+branch.Name, "push", "-u", "origin", branch.Name); err != nil {
				return nil, err
			}
		}
		if err := runGit(parent, workspace, workspace.Root, "clone "+repository.Name, "clone", layout.RemoteURL, layout.CheckoutPath); err != nil {
			return nil, err
		}
		// Git records the rewritten local target after clone. Restore the stable
		// portable URL while the test-owned insteadOf rule keeps all I/O local.
		if err := runGit(parent, workspace, layout.CheckoutPath, "stabilize origin "+repository.Name, "config", "--local", "remote.origin.url", layout.RemoteURL); err != nil {
			return nil, err
		}
		if err := runGit(parent, workspace, layout.CheckoutPath, "checkout current branch "+repository.Name, "switch", repository.CurrentBranch); err != nil {
			return nil, err
		}
		if err := runGit(parent, workspace, layout.CheckoutPath, "set upstream "+repository.Name, "branch", "--set-upstream-to", repository.Upstream.Remote+"/"+repository.Upstream.Branch, repository.CurrentBranch); err != nil {
			return nil, err
		}
		if len(repository.RemoteCommits) > 0 {
			if err := runGit(parent, workspace, sourcePath, "switch source for remote advance "+repository.Name, "switch", repository.Upstream.Branch); err != nil {
				return nil, err
			}
			for _, commit := range repository.RemoteCommits {
				if err := writeFixtureFiles(sourcePath, commit.Files); err != nil {
					return nil, err
				}
				if err := commitAll(parent, workspace, sourcePath, commit.Message); err != nil {
					return nil, err
				}
			}
			if err := runGit(parent, workspace, sourcePath, "advance remote "+repository.Name, "push", "origin", repository.Upstream.Branch); err != nil {
				return nil, err
			}
		}
		if err := writeFixtureFiles(layout.CheckoutPath, repository.DirtyFiles); err != nil {
			return nil, err
		}
		head, err := gitOutput(parent, workspace, layout.CheckoutPath, "record HEAD "+repository.Name, "rev-parse", "HEAD")
		if err != nil {
			return nil, err
		}
		layout.BaselineHEAD = strings.TrimSpace(head)
		for path, content := range repository.DirtyFiles {
			layout.DirtyHashes[path] = hashBytes([]byte(content))
		}
		workspace.Repositories[repository.Name] = layout
	}

	reg := &registry.Registry{}
	for _, missing := range recipe.MissingEntries {
		missingPath, pathErr := containedPath(workspace.WorkspaceRoot, missing.Path)
		if pathErr != nil {
			return nil, pathErr
		}
		if _, statErr := os.Lstat(missingPath); !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("missing_entries[%s].path: expected absent path %s", missing.Name, missingPath)
		}
		reg.Entries = append(reg.Entries, registry.Entry{RepoID: missing.RepoID, CheckoutID: missing.CheckoutID, Path: missingPath, RemoteURL: missing.RemoteURL, Type: "checkout", Status: registry.StatusMissing})
		workspace.MissingEntries[missing.Name] = MaterializedMissingEntry{Name: missing.Name, Path: missingPath, RepoID: missing.RepoID}
	}
	if seedMode == SeedAllEntries {
		for _, repository := range recipe.Repositories {
			layout := workspace.Repositories[repository.Name]
			reg.Entries = append(reg.Entries, registry.Entry{RepoID: layout.RepoID, CheckoutID: safeFixtureName(repository.Name), Path: layout.CheckoutPath, RemoteURL: layout.RemoteURL, Type: "checkout", Branch: repository.CurrentBranch, Labels: cloneMap(repository.Labels), Annotations: cloneMap(repository.Annotations), Status: registry.StatusPresent})
		}
	}
	cfg := config.DefaultConfig()
	cfg.Exclude = append([]string(nil), recipe.Config.Exclude...)
	cfg.Defaults.Concurrency = 2
	cfg.Defaults.TimeoutSeconds = 10
	cfg.Registry = reg
	if recipe.Config.ExternalRegistry {
		cfg.RegistryPath = "registry.yaml"
	}
	if err := config.Save(&cfg, workspace.ConfigPath); err != nil {
		return nil, fmt.Errorf("save isolated config: %w", err)
	}
	if err := verifyReadyState(parent, workspace, recipe); err != nil {
		return nil, err
	}
	return workspace, nil
}

func writeFixtureFiles(root string, files map[string]string) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, portable := range paths {
		path, err := containedPath(root, portable)
		if err != nil {
			return err
		}
		if err := rejectSymlinkParents(root, filepath.Dir(path)); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(files[portable]), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func rejectSymlinkParents(root, parent string) error {
	relative, err := filepath.Rel(root, parent)
	if err != nil {
		return err
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination parent %s is a symlink", current)
		}
	}
	return nil
}

func runGit(parent context.Context, workspace *MaterializedWorkspace, dir, operation string, arguments ...string) error {
	result := runCommand(parent, maximumScenarioTimeout, operation, "git", arguments, dir, workspace.Env, workspace.Root, workspace.ConfigPath)
	if err := expectExit(result, 0); err != nil {
		return err
	}
	return nil
}

func gitOutput(parent context.Context, workspace *MaterializedWorkspace, dir, operation string, arguments ...string) (string, error) {
	result := runCommand(parent, maximumScenarioTimeout, operation, "git", arguments, dir, workspace.Env, workspace.Root, workspace.ConfigPath)
	if err := expectExit(result, 0); err != nil {
		return "", err
	}
	return string(result.Stdout), nil
}

func commitAll(parent context.Context, workspace *MaterializedWorkspace, dir, message string) error {
	if err := runGit(parent, workspace, dir, "stage "+message, "add", "--all"); err != nil {
		return err
	}
	return runGit(parent, workspace, dir, "commit "+message, "commit", "--no-gpg-sign", "-m", message)
}

func verifyReadyState(parent context.Context, workspace *MaterializedWorkspace, recipe WorkspaceRecipe) error {
	for _, repository := range recipe.Repositories {
		layout := workspace.Repositories[repository.Name]
		bare, err := gitOutput(parent, workspace, layout.RemotePath, "verify bare "+repository.Name, "rev-parse", "--is-bare-repository")
		if err != nil || strings.TrimSpace(bare) != "true" {
			return fmt.Errorf("remote %s is not bare: %w", repository.Name, err)
		}
		head, err := gitOutput(parent, workspace, layout.CheckoutPath, "verify HEAD "+repository.Name, "rev-parse", "HEAD")
		if err != nil || strings.TrimSpace(head) != layout.BaselineHEAD {
			return fmt.Errorf("repository %s HEAD mismatch", repository.Name)
		}
		porcelain, err := gitOutput(parent, workspace, layout.CheckoutPath, "verify worktree "+repository.Name, "status", "--porcelain=v1")
		if err != nil {
			return err
		}
		if len(repository.DirtyFiles) == 0 && strings.TrimSpace(porcelain) != "" {
			return fmt.Errorf("repository %s expected clean, got %q", repository.Name, porcelain)
		}
		if len(repository.DirtyFiles) > 0 && strings.TrimSpace(porcelain) == "" {
			return fmt.Errorf("repository %s expected dirty", repository.Name)
		}
	}
	if _, err := config.Load(workspace.ConfigPath); err != nil {
		return fmt.Errorf("reload isolated config: %w", err)
	}
	return nil
}

func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}

func safeFixtureName(value string) string {
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, value)
	return strings.Trim(value, "-")
}

func cloneMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func hashBytes(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
