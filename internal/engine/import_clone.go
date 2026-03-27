// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/pathutil"
	"github.com/skaphos/repokeeper/internal/registry"
)

type ImportCloneOptions struct {
	CWD                       string
	BundleRoot                string
	DangerouslyDeleteExisting bool
	ResolveTargetRelativePath func(entry registry.Entry, root string) string
}

type ImportCloneTarget struct {
	Path  string
	Entry registry.Entry
}

type ImportCloneSkip struct {
	Path   string
	Entry  registry.Entry
	Reason string
}

type ImportClonePlan struct {
	CWD                       string
	DangerouslyDeleteExisting bool
	Clones                    []ImportCloneTarget
	Skipped                   []ImportCloneSkip
}

type ImportCloneCallbacks struct {
	OnStart    SyncStartCallback
	OnComplete SyncResultCallback
}

func (e *Engine) PlanImportClones(entries []registry.Entry, opts ImportCloneOptions) (ImportClonePlan, error) {
	if e.registry == nil {
		return ImportClonePlan{}, nil
	}
	if entries == nil {
		entries = e.registry.Entries
	}
	if len(entries) == 0 {
		return ImportClonePlan{}, nil
	}

	cwd := filepath.Clean(strings.TrimSpace(opts.CWD))
	if cwd == "" {
		cwd = "."
	}

	ignored := ignoredImportPathSet(e.cfg)
	targets := make(map[string]ImportCloneTarget, len(entries))
	skipped := make(map[string]ImportCloneSkip)

	for _, entry := range entries {
		targetRel := strings.TrimSpace(entry.Path)
		if opts.ResolveTargetRelativePath != nil {
			targetRel = opts.ResolveTargetRelativePath(entry, opts.BundleRoot)
		}
		target := filepath.Clean(filepath.Join(cwd, targetRel))
		targetKey := pathutil.CanonicalNormalize(target)

		relToCWD, err := filepath.Rel(cwd, target)
		if err != nil {
			return ImportClonePlan{}, err
		}
		if strings.HasPrefix(relToCWD, ".."+string(filepath.Separator)) || relToCWD == ".." {
			return ImportClonePlan{}, fmt.Errorf("refusing to clone outside current directory: %q", target)
		}
		if _, exists := targets[targetKey]; exists {
			return ImportClonePlan{}, fmt.Errorf("multiple repos resolve to same target path %q", target)
		}
		targets[targetKey] = ImportCloneTarget{Path: target, Entry: entry}

		if ignored[targetKey] {
			skipped[targetKey] = ImportCloneSkip{Path: target, Entry: entry, Reason: "path is ignored by local config"}
			continue
		}

		if entry.Status == registry.StatusMissing {
			skipped[targetKey] = ImportCloneSkip{Path: target, Entry: entry, Reason: "marked missing in bundle"}
			continue
		}

		if strings.TrimSpace(entry.RemoteURL) == "" {
			skipped[targetKey] = ImportCloneSkip{Path: target, Entry: entry, Reason: "no remote URL configured"}
			continue
		}

		if entry.Type != "mirror" && strings.TrimSpace(entry.Branch) == "" {
			skipped[targetKey] = ImportCloneSkip{Path: target, Entry: entry, Reason: "no upstream branch configured"}
			continue
		}
	}

	if !opts.DangerouslyDeleteExisting {
		conflicts := findImportCloneConflicts(targets, skipped)
		if len(conflicts) > 0 {
			lines := make([]string, 0, len(conflicts))
			for _, conflict := range conflicts {
				lines = append(lines, fmt.Sprintf("%s (repo: %s)", conflict.target, conflict.entry.RepoID))
			}
			return ImportClonePlan{}, fmt.Errorf(
				"import target conflicts detected under %s:\n- %s\nre-run with --dangerously-delete-existing to replace these paths",
				cwd,
				strings.Join(lines, "\n- "),
			)
		}
	}

	keys := make([]string, 0, len(targets))
	for key := range targets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	plan := ImportClonePlan{
		CWD:                       cwd,
		DangerouslyDeleteExisting: opts.DangerouslyDeleteExisting,
		Clones:                    make([]ImportCloneTarget, 0, len(targets)-len(skipped)),
		Skipped:                   make([]ImportCloneSkip, 0, len(skipped)),
	}
	for _, key := range keys {
		if skip, ok := skipped[key]; ok {
			plan.Skipped = append(plan.Skipped, skip)
			continue
		}
		plan.Clones = append(plan.Clones, targets[key])
	}

	return plan, nil
}

func (e *Engine) ExecuteImportClones(ctx context.Context, plan ImportClonePlan, callbacks ImportCloneCallbacks) ([]SyncResult, error) {
	failures := make([]SyncResult, 0)

	for _, target := range plan.Clones {
		entry := target.Entry
		result := SyncResult{RepoID: entry.RepoID, Path: target.Path, Action: "git clone"}
		if callbacks.OnStart != nil {
			callbacks.OnStart(result)
		}
		if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
			return failures, err
		}
		if _, err := os.Stat(target.Path); err == nil {
			if !plan.DangerouslyDeleteExisting {
				return failures, fmt.Errorf("target path already exists: %q (use --dangerously-delete-existing to replace)", target.Path)
			}
			if err := os.RemoveAll(target.Path); err != nil {
				return failures, fmt.Errorf("failed to remove existing path %q: %w", target.Path, err)
			}
		} else if !os.IsNotExist(err) {
			return failures, err
		}

		if err := e.adapter.Clone(ctx, strings.TrimSpace(entry.RemoteURL), target.Path, strings.TrimSpace(entry.Branch), entry.Type == "mirror"); err != nil {
			result.OK = false
			result.ErrorClass = e.classifier.ClassifyError(err)
			result.Error = importCloneFailureMessage(result.ErrorClass)
			identity := entry
			entry.Path = target.Path
			entry.Status = registry.StatusMissing
			entry.LastSeen = time.Now()
			e.setImportRegistryEntry(identity, entry)
			failures = append(failures, result)
			if callbacks.OnComplete != nil {
				callbacks.OnComplete(result)
			}
			continue
		}

		identity := entry
		entry.Path = target.Path
		entry.Status = registry.StatusPresent
		entry.LastSeen = time.Now()
		e.setImportRegistryEntry(identity, entry)
		result.OK = true
		if callbacks.OnComplete != nil {
			callbacks.OnComplete(result)
		}
	}

	for _, skip := range plan.Skipped {
		entry := skip.Entry
		if skip.Reason == "path is ignored by local config" {
			e.removeImportRegistryEntry(entry)
			continue
		}
		identity := entry
		entry.Path = skip.Path
		if entry.Status == "" || skip.Reason == "no remote URL configured" || skip.Reason == "no upstream branch configured" {
			entry.Status = registry.StatusMissing
		}
		entry.LastSeen = time.Now()
		e.setImportRegistryEntry(identity, entry)
	}

	return failures, nil
}

type importCloneConflict struct {
	target string
	entry  registry.Entry
}

func findImportCloneConflicts(targets map[string]ImportCloneTarget, skipped map[string]ImportCloneSkip) []importCloneConflict {
	conflicts := make([]importCloneConflict, 0)
	for key, plan := range targets {
		if _, skip := skipped[key]; skip {
			continue
		}
		if _, err := os.Stat(plan.Path); err == nil {
			conflicts = append(conflicts, importCloneConflict{target: plan.Path, entry: plan.Entry})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].target < conflicts[j].target
	})
	return conflicts
}

func ignoredImportPathSet(cfg *config.Config) map[string]bool {
	if cfg == nil {
		return make(map[string]bool)
	}
	pathSet := pathutil.IgnoredPathSet(cfg.IgnoredPaths, pathutil.CanonicalNormalize)
	out := make(map[string]bool, len(pathSet))
	for k := range pathSet {
		out[k] = true
	}
	return out
}

func importCloneFailureMessage(errorClass string) string {
	switch strings.TrimSpace(errorClass) {
	case "auth":
		return "import-clone-auth"
	case "network":
		return "import-clone-network"
	case "timeout":
		return "import-clone-timeout"
	case "corrupt":
		return "import-clone-corrupt"
	case "missing_remote":
		return "import-clone-missing-remote"
	default:
		return "import-clone-failed"
	}
}

func (e *Engine) setImportRegistryEntry(identity registry.Entry, entry registry.Entry) {
	e.registryMu.Lock()
	defer e.registryMu.Unlock()
	if e.registry == nil {
		e.registry = &registry.Registry{}
	}

	if match := e.registry.FindByRepoIDAndCheckoutID(identity.RepoID, strings.TrimSpace(identity.CheckoutID)); strings.TrimSpace(identity.CheckoutID) != "" && match != nil {
		*match = entry
		return
	}
	if path := strings.TrimSpace(identity.Path); path != "" {
		if match := e.registry.FindEntry(identity.RepoID, path); match != nil && strings.TrimSpace(match.Path) == path {
			*match = entry
			return
		}
	}
	repoMatches := e.registry.FindEntriesByRepoID(identity.RepoID)
	if len(repoMatches) == 1 {
		for i := range e.registry.Entries {
			if e.registry.Entries[i].RepoID == identity.RepoID {
				e.registry.Entries[i] = entry
				return
			}
		}
	}
	e.registry.Entries = append(e.registry.Entries, entry)
}

func (e *Engine) removeImportRegistryEntry(identity registry.Entry) {
	e.registryMu.Lock()
	defer e.registryMu.Unlock()
	if e.registry == nil {
		return
	}

	removeIndex := -1
	if checkoutID := strings.TrimSpace(identity.CheckoutID); checkoutID != "" {
		for i := range e.registry.Entries {
			if e.registry.Entries[i].RepoID == identity.RepoID && e.registry.Entries[i].CheckoutID == checkoutID {
				removeIndex = i
				break
			}
		}
	}
	if removeIndex < 0 {
		path := strings.TrimSpace(identity.Path)
		if path != "" {
			for i := range e.registry.Entries {
				if e.registry.Entries[i].RepoID == identity.RepoID && strings.TrimSpace(e.registry.Entries[i].Path) == path {
					removeIndex = i
					break
				}
			}
		}
	}
	if removeIndex < 0 {
		repoMatches := e.registry.FindEntriesByRepoID(identity.RepoID)
		if len(repoMatches) == 1 {
			for i := range e.registry.Entries {
				if e.registry.Entries[i].RepoID == identity.RepoID {
					removeIndex = i
					break
				}
			}
		}
	}
	if removeIndex < 0 {
		return
	}

	out := e.registry.Entries[:0]
	for i, entry := range e.registry.Entries {
		if i == removeIndex {
			continue
		}
		out = append(out, entry)
	}
	e.registry.Entries = out
}
