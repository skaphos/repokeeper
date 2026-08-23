// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func (e *Engine) ResetRepo(ctx context.Context, repoID, cfgPath string) error {
	e.registryMu.Lock()
	reg := e.registry
	e.registryMu.Unlock()

	if reg == nil {
		return fmt.Errorf("registry not available")
	}
	entries := reg.FindEntriesByRepoID(repoID)
	if len(entries) == 0 {
		return fmt.Errorf("repo %q not found in registry", repoID)
	}
	if len(entries) > 1 {
		return fmt.Errorf("repo %q is ambiguous: found %d local checkouts; re-run with an exact checkout selector", repoID, len(entries))
	}
	entry := entries[0]
	if entry.Status == registry.StatusMissing {
		return fmt.Errorf("repo %q path is missing on disk", repoID)
	}

	if err := e.adapter.ResetHard(ctx, entry.Path); err != nil {
		return fmt.Errorf("git reset --hard HEAD: %w", err)
	}
	if err := e.adapter.CleanFD(ctx, entry.Path); err != nil {
		return fmt.Errorf("git clean -f -d: %w", err)
	}
	return nil
}

func (e *Engine) DeleteRepo(ctx context.Context, repo, cfgPath string, deleteFiles bool) error {
	e.registryMu.Lock()
	reg := e.registry
	e.registryMu.Unlock()

	if reg == nil {
		return fmt.Errorf("registry not available")
	}
	entry, err := resolveDeleteEntry(reg, repo)
	if err != nil {
		return err
	}
	entryPath := entry.Path
	if deleteFiles && entryPath != "" {
		if err := validateRemoveAllTarget(entryPath); err != nil {
			return err
		}
	}

	newEntries := make([]registry.Entry, 0, len(reg.Entries))
	for _, en := range reg.Entries {
		if en.RepoID == entry.RepoID && filepath.Clean(en.Path) == filepath.Clean(entryPath) {
			continue
		}
		newEntries = append(newEntries, en)
	}

	e.registryMu.Lock()
	reg.Entries = newEntries
	reg.UpdatedAt = time.Now()
	e.registryMu.Unlock()

	if err := e.persistRegistry(cfgPath); err != nil {
		return fmt.Errorf("saving registry after delete: %w", err)
	}

	if deleteFiles && entryPath != "" {
		if err := safeRemoveAll(entryPath); err != nil {
			return err
		}
	}
	return nil
}

func resolveDeleteEntry(reg *registry.Registry, repo string) (registry.Entry, error) {
	if filepath.IsAbs(repo) {
		wantPath := filepath.Clean(repo)
		var pathMatches []registry.Entry
		for _, entry := range reg.Entries {
			if filepath.Clean(entry.Path) == wantPath {
				pathMatches = append(pathMatches, entry)
			}
		}
		if len(pathMatches) == 1 {
			return pathMatches[0], nil
		}
		if len(pathMatches) > 1 {
			return registry.Entry{}, fmt.Errorf("repository path %q is ambiguous: found %d registry entries", repo, len(pathMatches))
		}
	}

	var checkoutMatches []registry.Entry
	for _, entry := range reg.Entries {
		if entry.CheckoutID == repo {
			checkoutMatches = append(checkoutMatches, entry)
		}
	}
	if len(checkoutMatches) == 1 {
		return checkoutMatches[0], nil
	}
	if len(checkoutMatches) > 1 {
		return registry.Entry{}, fmt.Errorf("checkout_id %q is ambiguous: found %d registry entries", repo, len(checkoutMatches))
	}

	entries := reg.FindEntriesByRepoID(repo)
	switch len(entries) {
	case 0:
		return registry.Entry{}, fmt.Errorf("repo %q not found in registry", repo)
	case 1:
		return entries[0], nil
	default:
		return registry.Entry{}, fmt.Errorf("repo %q is ambiguous: found %d local checkouts; re-run with an exact checkout selector", repo, len(entries))
	}
}

// safeRemoveAll wraps os.RemoveAll with defensive checks to prevent accidental
// deletion of non-absolute paths, filesystem roots, or non-directory targets.
func safeRemoveAll(path string) error {
	if err := validateRemoveAllTarget(path); err != nil {
		return err
	}
	clean := filepath.Clean(path)
	if err := os.RemoveAll(clean); err != nil {
		return fmt.Errorf("deleting %q from disk: %w", clean, err)
	}
	return nil
}

func validateRemoveAllTarget(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to delete non-absolute path %q", path)
	}
	clean := filepath.Clean(path)
	// Reject filesystem roots: a path is a root when its parent equals itself.
	if clean == filepath.Dir(clean) {
		return fmt.Errorf("refusing to delete filesystem root %q", clean)
	}
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %q: %w", clean, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to delete non-directory path %q", clean)
	}
	return nil
}

func (e *Engine) CloneAndRegister(ctx context.Context, remoteURL, targetPath, cfgPath string, mirror bool) error {
	// Trim once up front and reuse: clone, normalization, and the stored entry
	// must all see the same URL, or a whitespace-padded input (e.g. from the
	// TUI/MCP server) could clone/normalize one value while the registry records
	// another.
	remoteURL = strings.TrimSpace(remoteURL)
	if err := e.adapter.Clone(ctx, remoteURL, targetPath, "", mirror); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	repoID := e.adapter.NormalizeURL(remoteURL)
	if repoID == "" {
		repoID = "local:" + filepath.ToSlash(targetPath)
	}

	repoType := "checkout"
	if mirror {
		repoType = "mirror"
	}

	entry := registry.Entry{
		RepoID:    repoID,
		Path:      targetPath,
		Type:      repoType,
		RemoteURL: remoteURL,
		Status:    registry.StatusPresent,
		LastSeen:  time.Now(),
	}
	if !mirror {
		entry.Branch = repoDefaultBranch(ctx, e.adapter, targetPath)
	}

	e.upsertRegistryEntry(entry)

	e.registryMu.Lock()
	reg := e.registry
	e.registryMu.Unlock()
	if reg == nil {
		return fmt.Errorf("registry not available")
	}
	if err := e.persistRegistry(cfgPath); err != nil {
		return fmt.Errorf("saving registry after clone: %w", err)
	}
	return nil
}

func (e *Engine) persistRegistry(cfgPath string) error {
	e.registryMu.Lock()
	cfg := e.cfg
	reg := e.registry
	e.registryMu.Unlock()
	if cfg == nil {
		return fmt.Errorf("config not available")
	}

	if fresh, err := config.Load(cfgPath); err == nil {
		cfg = fresh
	} else if !os.IsNotExist(err) {
		return err
	}
	cfg.Registry = reg
	if err := config.Save(cfg, cfgPath); err != nil {
		return err
	}

	e.registryMu.Lock()
	e.cfg = cfg
	e.registryMu.Unlock()
	return nil
}

func repoDefaultBranch(ctx context.Context, adapter vcs.Adapter, targetPath string) string {
	head, err := adapter.Head(ctx, targetPath)
	if err != nil || head.Detached {
		return ""
	}
	return strings.TrimSpace(head.Branch)
}
