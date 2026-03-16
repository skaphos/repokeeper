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
	var entry registry.Entry
	found := false
	for _, en := range reg.Entries {
		if en.RepoID == repoID {
			entry = en
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("repo %q not found in registry", repoID)
	}
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

func (e *Engine) DeleteRepo(ctx context.Context, repoID, cfgPath string, deleteFiles bool) error {
	e.registryMu.Lock()
	reg := e.registry
	cfg := e.cfg
	e.registryMu.Unlock()

	if reg == nil {
		return fmt.Errorf("registry not available")
	}
	var entryPath string
	newEntries := make([]registry.Entry, 0, len(reg.Entries))
	for _, en := range reg.Entries {
		if en.RepoID == repoID {
			entryPath = en.Path
			continue
		}
		newEntries = append(newEntries, en)
	}
	if entryPath == "" {
		return fmt.Errorf("repo %q not found in registry", repoID)
	}

	e.registryMu.Lock()
	reg.Entries = newEntries
	reg.UpdatedAt = time.Now()
	e.registryMu.Unlock()

	cfg.Registry = reg
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving registry after delete: %w", err)
	}

	if deleteFiles && entryPath != "" {
		if err := safeRemoveAll(entryPath); err != nil {
			return err
		}
	}
	return nil
}

// safeRemoveAll wraps os.RemoveAll with defensive checks to prevent accidental
// deletion of non-absolute paths, filesystem roots, or non-directory targets.
func safeRemoveAll(path string) error {
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
	if err := os.RemoveAll(clean); err != nil {
		return fmt.Errorf("deleting %q from disk: %w", clean, err)
	}
	return nil
}

func (e *Engine) CloneAndRegister(ctx context.Context, remoteURL, targetPath, cfgPath string, mirror bool) error {
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
		RepoID:   repoID,
		Path:     targetPath,
		Type:     repoType,
		Status:   registry.StatusPresent,
		LastSeen: time.Now(),
	}
	if !mirror {
		entry.Branch = repoDefaultBranch(ctx, e.adapter, targetPath)
	}

	e.upsertRegistryEntry(entry)

	e.registryMu.Lock()
	reg := e.registry
	cfg := e.cfg
	e.registryMu.Unlock()
	cfg.Registry = reg
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving registry after clone: %w", err)
	}
	return nil
}

func repoDefaultBranch(ctx context.Context, adapter vcs.Adapter, targetPath string) string {
	head, err := adapter.Head(ctx, targetPath)
	if err != nil || head.Detached {
		return ""
	}
	return strings.TrimSpace(head.Branch)
}
