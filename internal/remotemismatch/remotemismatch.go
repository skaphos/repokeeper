// SPDX-License-Identifier: MIT
package remotemismatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// ReconcileMode controls how remote mismatch reconciliation is applied.
type ReconcileMode string

const (
	ReconcileNone     ReconcileMode = "none"
	ReconcileRegistry ReconcileMode = "registry"
	ReconcileGit      ReconcileMode = "git"
)

// Plan describes one remote mismatch reconcile action for a repo.
type Plan struct {
	RepoID        string
	Path          string
	PrimaryRemote string
	RepoRemoteURL string
	RegistryURL   string
	EntryIndex    int
	Action        string
}

// ParseReconcileMode validates and parses a reconcile mode flag value.
func ParseReconcileMode(raw string) (ReconcileMode, error) {
	mode := ReconcileMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case "", ReconcileNone:
		return ReconcileNone, nil
	case ReconcileRegistry, ReconcileGit:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported --reconcile-remote-mismatch value %q (expected none, registry, or git)", raw)
	}
}

// BuildPlans computes reconcile plans from status data and registry state.
func BuildPlans(repos []model.RepoStatus, reg *registry.Registry, adapter vcs.Adapter, mode ReconcileMode) []Plan {
	if reg == nil || adapter == nil || mode == ReconcileNone {
		return nil
	}
	plans := make([]Plan, 0)
	for _, repo := range repos {
		entryIndex := findRegistryEntryIndexForStatus(reg, repo)
		if entryIndex < 0 {
			continue
		}
		entry := reg.Entries[entryIndex]
		registryURL := strings.TrimSpace(entry.RemoteURL)
		if registryURL == "" || strings.TrimSpace(repo.RepoID) == "" {
			continue
		}
		if adapter.NormalizeURL(registryURL) == repo.RepoID {
			continue
		}
		repoRemoteURL := primaryRemoteURL(repo)
		action := ""
		switch mode {
		case ReconcileRegistry:
			if repoRemoteURL == "" {
				continue
			}
			action = "set registry remote_url to live git remote"
		case ReconcileGit:
			if strings.TrimSpace(repo.PrimaryRemote) == "" {
				continue
			}
			action = "set git remote URL to registry remote_url"
		}
		plans = append(plans, Plan{
			RepoID:        repo.RepoID,
			Path:          repo.Path,
			PrimaryRemote: repo.PrimaryRemote,
			RepoRemoteURL: repoRemoteURL,
			RegistryURL:   registryURL,
			EntryIndex:    entryIndex,
			Action:        action,
		})
	}
	return plans
}

// ApplyPlans applies plans to registry and/or git remotes based on mode.
func ApplyPlans(ctx context.Context, plans []Plan, reg *registry.Registry, mode ReconcileMode, adapter vcs.Adapter, now func() time.Time) error {
	if len(plans) == 0 {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	switch mode {
	case ReconcileRegistry:
		for _, plan := range plans {
			if reg == nil {
				continue
			}
			if plan.EntryIndex < 0 || plan.EntryIndex >= len(reg.Entries) {
				continue
			}
			reg.Entries[plan.EntryIndex].RemoteURL = plan.RepoRemoteURL
			reg.Entries[plan.EntryIndex].LastSeen = now()
		}
	case ReconcileGit:
		if adapter == nil {
			return fmt.Errorf("adapter is required for git remote reconciliation")
		}
		for _, plan := range plans {
			if strings.TrimSpace(plan.PrimaryRemote) == "" {
				continue
			}
			if err := adapter.SetRemoteURL(ctx, plan.Path, plan.PrimaryRemote, plan.RegistryURL); err != nil {
				return fmt.Errorf("git remote set-url %q %q (%q): %w", plan.PrimaryRemote, plan.RegistryURL, plan.Path, err)
			}
		}
	}
	return nil
}

func findRegistryEntryIndexForStatus(reg *registry.Registry, repo model.RepoStatus) int {
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == repo.RepoID && reg.Entries[i].Path == repo.Path {
			return i
		}
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == repo.RepoID {
			return i
		}
	}
	return -1
}

func primaryRemoteURL(repo model.RepoStatus) string {
	for _, remote := range repo.Remotes {
		if remote.Name == repo.PrimaryRemote {
			return strings.TrimSpace(remote.URL)
		}
	}
	return ""
}
