// SPDX-License-Identifier: MIT
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type viewMode int

const (
	viewList viewMode = iota
	viewDetail
	viewSyncPlan
	viewProgress
)

type statusReportMsg struct {
	report *model.StatusReport
	err    error
}

type tuiModel struct {
	engine  EngineAPI
	cfgPath string

	mode          viewMode
	width, height int

	repos         []model.RepoStatus
	filteredRepos []model.RepoStatus
	cursor        int
	offset        int
	loading       bool
	err           error

	filterMode bool
	filterText string

	selected map[string]bool

	syncPlan     []engine.SyncResult
	syncResults  []engine.SyncResult
	syncProgress map[string]engine.SyncResult
	syncDone     bool
	syncErr      error
}

func newModel(eng EngineAPI, reg *registry.Registry, cfgPath string) tuiModel {
	var repos []model.RepoStatus
	if reg != nil {
		repos = make([]model.RepoStatus, 0, len(reg.Entries))
		for _, entry := range reg.Entries {
			repos = append(repos, registryEntryToPartialStatus(entry))
		}
	}
	return tuiModel{
		engine:  eng,
		cfgPath: cfgPath,
		repos:   repos,
		loading: true,
		mode:    viewList,
	}
}

func registryEntryToPartialStatus(entry registry.Entry) model.RepoStatus {
	s := model.RepoStatus{
		RepoID: entry.RepoID,
		Path:   entry.Path,
		Type:   entry.Type,
	}
	if entry.Branch != "" {
		s.Head = model.Head{Branch: entry.Branch}
	}
	if entry.Labels != nil {
		s.Labels = make(map[string]string, len(entry.Labels))
		for k, v := range entry.Labels {
			s.Labels[k] = v
		}
	}
	if entry.Annotations != nil {
		s.Annotations = make(map[string]string, len(entry.Annotations))
		for k, v := range entry.Annotations {
			s.Annotations[k] = v
		}
	}
	if entry.Status == registry.StatusMissing {
		s.Error = "path missing"
		s.ErrorClass = "missing"
	}
	return s
}

func (m tuiModel) visibleList() []model.RepoStatus {
	if m.filterText != "" {
		return m.filteredRepos
	}
	return m.repos
}

func (m tuiModel) Init() tea.Cmd {
	return loadStatusCmd(m.engine)
}

func loadStatusCmd(eng EngineAPI) tea.Cmd {
	return func() tea.Msg {
		report, err := eng.Status(context.Background(), engine.StatusOptions{
			Filter: engine.FilterAll,
		})
		return statusReportMsg{report: report, err: err}
	}
}
