// SPDX-License-Identifier: MIT
// Package tui provides the interactive Bubble Tea terminal UI.
package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func Run(ctx context.Context, cfg *config.Config, reg *registry.Registry, cfgPath string) error {
	if cfg == nil {
		return fmt.Errorf("tui: config is required")
	}
	adapter, err := vcs.NewAdapterForSelection("git")
	if err != nil {
		return fmt.Errorf("tui: selecting VCS adapter: %w", err)
	}
	eng := engine.New(cfg, reg, adapter, vcs.NewGitErrorClassifier(), vcs.NewGitURLNormalizer(), nil)
	return RunWithEngine(ctx, eng, reg, cfgPath)
}

func RunWithEngine(ctx context.Context, eng EngineAPI, reg *registry.Registry, cfgPath string) error {
	m := newModel(ctx, eng, reg, cfgPath)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	// m.program is a pointer shared by every value-copy of the model
	// (including the one tea.NewProgram just captured), so storing into it
	// here makes the running program visible to Update-goroutine Cmds such
	// as executeSyncCmd without needing a pointer-receiver model.
	m.program.Store(p)
	_, err := p.Run()
	return err
}
