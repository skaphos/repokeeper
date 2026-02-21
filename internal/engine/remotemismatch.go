// SPDX-License-Identifier: MIT
package engine

import (
	"context"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/remotemismatch"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// RemoteMismatchReconcileMode controls how remote mismatch reconciliation is applied.
type RemoteMismatchReconcileMode = remotemismatch.ReconcileMode

// RemoteMismatchPlan describes one remote mismatch reconcile action for a repo.
type RemoteMismatchPlan = remotemismatch.Plan

const (
	// RemoteMismatchReconcileNone disables reconciliation.
	RemoteMismatchReconcileNone = remotemismatch.ReconcileNone
	// RemoteMismatchReconcileRegistry updates the registry to match the live git remote.
	RemoteMismatchReconcileRegistry = remotemismatch.ReconcileRegistry
	// RemoteMismatchReconcileGit updates the git remote to match the registry.
	RemoteMismatchReconcileGit = remotemismatch.ReconcileGit
)

// ParseRemoteMismatchReconcileMode validates and parses a reconcile mode flag value.
func ParseRemoteMismatchReconcileMode(raw string) (RemoteMismatchReconcileMode, error) {
	return remotemismatch.ParseReconcileMode(raw)
}

// BuildRemoteMismatchPlans computes reconcile plans from status data using the engine's
// registry and VCS adapter.
func (e *Engine) BuildRemoteMismatchPlans(repos []model.RepoStatus, mode RemoteMismatchReconcileMode) []RemoteMismatchPlan {
	return remotemismatch.BuildPlans(repos, e.registry, e.adapter, mode)
}

// ApplyRemoteMismatchPlans applies reconcile plans to registry and/or git remotes.
func (e *Engine) ApplyRemoteMismatchPlans(ctx context.Context, plans []RemoteMismatchPlan, mode RemoteMismatchReconcileMode) error {
	return remotemismatch.ApplyPlans(ctx, plans, e.registry, mode, vcs.NewGitAdapter(nil), nil)
}
