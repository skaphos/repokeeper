// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
)

type repairDoneMsg struct {
	result engine.RepairUpstreamResult
	err    error
}

func repairUpstreamCmd(eng EngineAPI, repoID, cfgPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := eng.RepairUpstream(context.Background(), repoID, cfgPath)
		return repairDoneMsg{result: result, err: err}
	}
}

func resolveRepairTarget(m tuiModel) (repoID, targetUpstream string, err error) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return "", "", fmt.Errorf("no repo selected")
	}
	repo := list[m.cursor]

	reg := m.engine.Registry()
	cfg := m.engine.Config()
	if reg == nil {
		return "", "", fmt.Errorf("registry not available")
	}

	var entryBranch string
	for _, e := range reg.Entries {
		if e.RepoID == repo.RepoID {
			entryBranch = e.Branch
			break
		}
	}

	remote := strings.TrimSpace(repo.PrimaryRemote)
	if remote == "" {
		return repo.RepoID, "", fmt.Errorf("no remote configured for %s", repo.RepoID)
	}
	if repo.Head.Detached || strings.TrimSpace(repo.Head.Branch) == "" {
		return repo.RepoID, "", fmt.Errorf("repo %s is in detached HEAD state", repo.RepoID)
	}

	var targetBranch string
	if b := strings.TrimSpace(entryBranch); b != "" {
		targetBranch = b
	} else if up := strings.TrimSpace(repo.Tracking.Upstream); strings.Contains(up, "/") {
		parts := strings.SplitN(up, "/", 2)
		targetBranch = parts[1]
	} else if cfg != nil {
		targetBranch = strings.TrimSpace(cfg.Defaults.MainBranch)
	}
	if targetBranch == "" {
		targetBranch = strings.TrimSpace(repo.Head.Branch)
	}
	if targetBranch == "" {
		return repo.RepoID, "", fmt.Errorf("cannot determine target branch for %s", repo.RepoID)
	}

	return repo.RepoID, remote + "/" + targetBranch, nil
}

func renderRepairConfirmView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Repair Upstream"))
	b.WriteByte('\n')
	b.WriteString(renderDivider([]int{m.width - 1}))
	b.WriteByte('\n')
	b.WriteByte('\n')

	if m.repairTargetUpstream == "" {
		b.WriteString(errorTextStyle.Render("Cannot determine target upstream — no remote or branch configured."))
	} else {
		b.WriteString(fmt.Sprintf("  Repo:   %s\n", m.repairRepoID))
		b.WriteString(fmt.Sprintf("  Target: %s\n", m.repairTargetUpstream))
		b.WriteByte('\n')
		b.WriteString("  Set upstream tracking to target?")
	}

	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("y/enter: confirm  n/esc: cancel"))
	return b.String()
}
