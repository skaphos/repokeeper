// SPDX-License-Identifier: MIT
// Package mcpserver exposes RepoKeeper operations as an MCP (Model Context
// Protocol) server. It is a thin adapter over the engine layer, following the
// same pattern as internal/tui.
package mcpserver

import (
	"context"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// EngineAPI abstracts engine operations so the MCP server can be tested
// without a concrete engine or live repositories. Extends the TUI pattern
// with Scan and Adapter accessors needed by MCP mutation tools.
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
	RepairUpstream(ctx context.Context, repoID, cfgPath string) (engine.RepairUpstreamResult, error)
	ResetRepo(ctx context.Context, repoID, cfgPath string) error
	DeleteRepo(ctx context.Context, repoID, cfgPath string, deleteFiles bool) error
	CloneAndRegister(ctx context.Context, remoteURL, targetPath, cfgPath string, mirror bool) error
	Scan(ctx context.Context, opts engine.ScanOptions) ([]model.RepoStatus, error)
	Registry() *registry.Registry
	Config() *config.Config
	Adapter() vcs.Adapter
}

// Compile-time check: *engine.Engine must satisfy EngineAPI.
var _ EngineAPI = (*engine.Engine)(nil)
