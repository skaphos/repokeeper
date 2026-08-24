// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

type RepositorySnapshot struct {
	HEAD      string
	Porcelain string
	Refs      string
	Files     map[string]string
}

type WorkspaceSnapshot struct {
	ConfigHash   string
	RegistryHash string
	Repositories map[string]RepositorySnapshot
}

func reloadWorkspaceState(workspace *MaterializedWorkspace) (*config.Config, *registry.Registry, error) {
	cfg, err := config.Load(workspace.ConfigPath)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Registry == nil {
		return nil, nil, fmt.Errorf("config %s has nil registry", workspace.ConfigPath)
	}
	return cfg, cfg.Registry, nil
}

func captureWorkspaceSnapshot(ctx context.Context, workspace *MaterializedWorkspace) (WorkspaceSnapshot, error) {
	snapshot := WorkspaceSnapshot{Repositories: map[string]RepositorySnapshot{}}
	configData, err := os.ReadFile(workspace.ConfigPath)
	if err != nil {
		return snapshot, err
	}
	snapshot.ConfigHash = hashBytes(configData)
	if data, readErr := os.ReadFile(workspace.RegistryPath); readErr == nil {
		snapshot.RegistryHash = hashBytes(data)
	}
	for name, repository := range workspace.Repositories {
		head, err := gitOutput(ctx, workspace, repository.CheckoutPath, "snapshot HEAD "+name, "rev-parse", "HEAD")
		if err != nil {
			return snapshot, err
		}
		porcelain, err := gitOutput(ctx, workspace, repository.CheckoutPath, "snapshot porcelain "+name, "status", "--porcelain=v1")
		if err != nil {
			return snapshot, err
		}
		refs, err := gitOutput(ctx, workspace, repository.CheckoutPath, "snapshot refs "+name, "for-each-ref", "--format=%(refname)=%(objectname)")
		if err != nil {
			return snapshot, err
		}
		files := map[string]string{}
		for path := range repository.DirtyHashes {
			data, readErr := os.ReadFile(filepath.Join(repository.CheckoutPath, filepath.FromSlash(path)))
			if readErr != nil {
				return snapshot, readErr
			}
			files[path] = hashBytes(data)
		}
		snapshot.Repositories[name] = RepositorySnapshot{HEAD: strings.TrimSpace(head), Porcelain: strings.TrimSpace(porcelain), Refs: strings.TrimSpace(refs), Files: files}
	}
	return snapshot, nil
}

func assertContained(root string, paths ...string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	for _, path := range paths {
		absolute, absErr := filepath.Abs(path)
		if absErr != nil {
			return absErr
		}
		relative, relErr := filepath.Rel(root, absolute)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("path %s is outside scenario root %s", path, root)
		}
	}
	return nil
}

func semanticPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(relative)
}

func sortedRegistryPaths(reg *registry.Registry, root string) []string {
	paths := make([]string, 0, len(reg.Entries))
	for _, entry := range reg.Entries {
		paths = append(paths, semanticPath(root, entry.Path))
	}
	sort.Strings(paths)
	return paths
}
