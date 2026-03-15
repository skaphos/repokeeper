// SPDX-License-Identifier: MIT
package tui

import (
	"context"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

// EngineAPI abstracts engine operations so the TUI can be tested without
// a concrete engine or live repositories.
type EngineAPI interface {
	Status(ctx context.Context, opts engine.StatusOptions) (*model.StatusReport, error)
	Sync(ctx context.Context, opts engine.SyncOptions) ([]engine.SyncResult, error)
	ExecuteSyncPlanWithCallbacks(
		ctx context.Context,
		plan []engine.SyncResult,
		opts engine.SyncOptions,
		onStart engine.SyncStartCallback,
		onComplete engine.SyncResultCallback,
	) ([]engine.SyncResult, error)
	InspectRepo(ctx context.Context, path string) (*model.RepoStatus, error)
	Registry() *registry.Registry
	Config() *config.Config
}

var _ EngineAPI = (*engine.Engine)(nil)
