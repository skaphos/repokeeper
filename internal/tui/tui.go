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
	m := newModel(eng, reg, cfgPath)
	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
