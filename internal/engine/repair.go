// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

// RepairUpstreamResult holds the outcome of a single upstream repair operation.
type RepairUpstreamResult struct {
	RepoID          string
	Path            string
	LocalBranch     string
	CurrentUpstream string
	TargetUpstream  string
	Action          string
	OK              bool
	Error           string
}

// RepairUpstream inspects a single repo by repoID, resolves the target upstream
// tracking reference, and sets it via the VCS adapter. The registry and config
// are updated on success.
func (e *Engine) RepairUpstream(ctx context.Context, repoID, cfgPath string) (RepairUpstreamResult, error) {
	e.registryMu.Lock()
	reg := e.registry
	cfg := e.cfg
	e.registryMu.Unlock()

	if reg == nil {
		return RepairUpstreamResult{}, fmt.Errorf("registry not available")
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
		return RepairUpstreamResult{}, fmt.Errorf("repo %q not found in registry", repoID)
	}

	res := RepairUpstreamResult{
		RepoID: entry.RepoID,
		Path:   entry.Path,
		OK:     true,
		Action: "unchanged",
	}

	if entry.Status == registry.StatusMissing {
		res.Action = "skip missing"
		return res, nil
	}

	status, err := e.InspectRepo(ctx, entry.Path)
	if err != nil {
		res.OK = false
		res.Action = "failed"
		res.Error = err.Error()
		return res, nil
	}

	res.LocalBranch = status.Head.Branch
	res.CurrentUpstream = strings.TrimSpace(status.Tracking.Upstream)

	if status.Error != "" {
		res.Action = "skip status error"
		res.Error = status.Error
		return res, nil
	}
	if status.Head.Detached || strings.TrimSpace(status.Head.Branch) == "" {
		res.Action = "skip detached"
		return res, nil
	}

	remote := strings.TrimSpace(status.PrimaryRemote)
	if remote == "" {
		res.Action = "skip no remote"
		return res, nil
	}

	targetBranch := repairResolveTargetBranch(entry, *status, cfg)
	if targetBranch == "" {
		res.Action = "skip no branch"
		return res, nil
	}

	targetUpstream := remote + "/" + targetBranch
	res.TargetUpstream = targetUpstream

	if !repairNeedsUpstream(*status, targetUpstream) {
		return res, nil
	}

	if err := e.adapter.SetUpstream(ctx, entry.Path, targetUpstream, status.Head.Branch); err != nil {
		res.OK = false
		res.Action = "failed"
		res.Error = err.Error()
		return res, nil
	}

	res.Action = "repaired"

	entry.Branch = targetBranch
	entry.LastSeen = time.Now()
	entry.Status = registry.StatusPresent
	e.upsertRegistryEntry(entry)

	e.registryMu.Lock()
	defer e.registryMu.Unlock()
	cfg.Registry = e.registry
	if err := config.Save(cfg, cfgPath); err != nil {
		return res, fmt.Errorf("repair succeeded but config save failed: %w", err)
	}

	return res, nil
}

func repairResolveTargetBranch(entry registry.Entry, status model.RepoStatus, cfg *config.Config) string {
	if b := strings.TrimSpace(entry.Branch); b != "" {
		return b
	}
	upstream := strings.TrimSpace(status.Tracking.Upstream)
	if upstream != "" {
		parts := strings.SplitN(upstream, "/", 2)
		if len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}
	if cfg != nil {
		if b := strings.TrimSpace(cfg.Defaults.MainBranch); b != "" {
			return b
		}
	}
	return strings.TrimSpace(status.Head.Branch)
}

func repairNeedsUpstream(status model.RepoStatus, target string) bool {
	current := strings.TrimSpace(status.Tracking.Upstream)
	t := strings.TrimSpace(target)
	if t == "" {
		return false
	}
	if current != t {
		return true
	}
	return status.Tracking.Status == model.TrackingNone
}
