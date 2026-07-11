// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case statusReportMsg:
		return m.handleStatusReport(msg)

	case syncPlanMsg:
		return m.handleSyncPlan(msg)

	case syncProgressMsg:
		return m.handleSyncProgress(msg)

	case syncDoneMsg:
		return m.handleSyncDone(msg)

	case repoStatusMsg:
		return m.handleRepoStatus(msg)

	case editReadyMsg:
		return m.handleEditReady(msg)

	case editDoneMsg:
		return m.handleEditDone(msg)

	case repairDoneMsg:
		return m.handleRepairDone(msg)

	case resetDoneMsg:
		return m.handleResetDone(msg)

	case deleteDoneMsg:
		return m.handleDeleteDone(msg)

	case addDoneMsg:
		return m.handleAddDone(msg)

	case labelEditDoneMsg:
		return m.handleLabelEditDone(msg)

	case repoMetadataEditDoneMsg:
		return m.handleRepoMetadataEditDone(msg)
	}
	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.mode {
	case viewSyncPlan:
		return m.handleSyncPlanKey(msg)
	case viewProgress:
		return m.handleSyncProgressKey(msg)
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewRepairConfirm:
		return m.handleRepairConfirmKey(msg)
	case viewResetConfirm:
		return m.handleResetConfirmKey(msg)
	case viewDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	case viewAdd:
		return m.handleAddKey(msg)
	case viewEditLabels:
		return m.handleLabelEditKey(msg)
	case viewEditRepoMetadata:
		return m.handleRepoMetadataEditKey(msg)
	default:
		if m.filterMode {
			return m.handleFilterKey(msg)
		}
		return m.handleListKey(msg)
	}
}

func (m tuiModel) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "backspace":
		m.mode = viewList
		return m, nil
	case "l":
		return startLabelEdit(m)
	case "i":
		return startRepoMetadataEdit(m)
	}
	return m, nil
}

func (m tuiModel) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "j", "down":
		return m.moveCursor(1), nil

	case "k", "up":
		return m.moveCursor(-1), nil

	case "/":
		m.filterMode = true
		return m, nil

	case "f5":
		return startStatusRefresh(m)

	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.filteredRepos = nil
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	case "space":
		return m.toggleSelection(), nil

	case "a":
		return m.toggleSelectAll(), nil

	case "s":
		return m.startSync()

	case "enter":
		list := m.visibleList()
		if len(list) > 0 && m.cursor < len(list) {
			m.mode = viewDetail
		}
		return m, nil

	case "e":
		return m, prepareEditCmd(m)

	case "r":
		return m.startRepair()

	case "ctrl+x":
		return m.startReset()

	case "ctrl+d":
		return m.startDelete()

	case "n":
		return m.startAdd()
	}
	return m, nil
}

func (m tuiModel) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterText = ""
		m.filteredRepos = nil
		m.cursor = 0
		m.offset = 0
		return m, nil

	case "enter":
		m.filterMode = false
		return m, nil

	case "backspace":
		if len(m.filterText) > 0 {
			runes := []rune(m.filterText)
			m.filterText = string(runes[:len(runes)-1])
			m.filteredRepos = filterRows(m.repos, m.filterText)
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	default:
		if t := msg.Text; t != "" {
			m.filterText += t
			m.filteredRepos = filterRows(m.repos, m.filterText)
			m.cursor = 0
			m.offset = 0
		}
	}
	return m, nil
}

var syncPlanOptionCount = 2

func (m tuiModel) handleSyncPlanKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = viewList
		m.syncPlan = nil
		m.modalCursor = 0
		return m, nil
	}
	left, right := isModalNav(msg)
	if left {
		m = modalMoveLeft(m)
		return m, nil
	}
	if right {
		m = modalMoveRight(m, syncPlanOptionCount)
		return m, nil
	}
	if msg.String() == "enter" {
		if m.modalCursor == 0 {
			m.mode = viewList
			m.syncPlan = nil
			m.modalCursor = 0
			return m, nil
		}
		if len(m.syncPlan) == 0 {
			m.mode = viewList
			m.modalCursor = 0
			return m, nil
		}
		m.mode = viewProgress
		m.syncProgress = make(map[string]engine.SyncResult)
		m.syncDone = false
		m.syncErr = nil
		m.modalCursor = 0
		return m, executeSyncCmd(m)
	}
	return m, nil
}

func (m tuiModel) handleSyncProgressKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.syncDone {
		m.mode = viewList
		m.syncPlan = nil
		m.syncProgress = nil
		m.syncDone = false
		m.syncErr = nil
		return startStatusRefresh(m)
	}
	return m, nil
}

// startStatusRefresh puts the model into a loading state and returns the
// command that re-inspects every registry entry, recording how many
// repoStatusMsg deliveries to expect so handleRepoStatus clears loading at
// the right time instead of after the first streamed message.
func startStatusRefresh(m tuiModel) (tea.Model, tea.Cmd) {
	m.loading = true
	if reg := m.engine.Registry(); reg != nil {
		m.pendingInspections = len(reg.Entries)
	}
	return m, refreshStatusCmd(m.context(), m.engine)
}

func (m tuiModel) moveCursor(delta int) tuiModel {
	list := m.visibleList()
	n := len(list)
	if n == 0 {
		return m
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	visible := visibleRows(m)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	return m
}

func (m tuiModel) toggleSelection() tuiModel {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m
	}
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	id := list[m.cursor].RepoID
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		m.selected[id] = true
	}
	return m
}

func (m tuiModel) toggleSelectAll() tuiModel {
	list := m.visibleList()
	if len(list) == 0 {
		return m
	}
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	allSelected := len(m.selected) == len(list)
	if allSelected {
		m.selected = make(map[string]bool)
	} else {
		for _, r := range list {
			m.selected[r.RepoID] = true
		}
	}
	return m
}

func (m tuiModel) startSync() (tea.Model, tea.Cmd) {
	m.mode = viewSyncPlan
	m.syncPlan = nil
	m.loading = true
	m.modalCursor = 0

	repoIDs := m.selected
	if len(repoIDs) == 0 {
		list := m.visibleList()
		if m.cursor < len(list) {
			repoIDs = map[string]bool{list[m.cursor].RepoID: true}
		}
	}
	return m, buildSyncPlanCmd(m.context(), m.engine, repoIDs)
}

func executeSyncCmd(m tuiModel) tea.Cmd {
	ctx := m.context()
	plan := m.syncPlan
	eng := m.engine
	progRef := m.program
	return func() tea.Msg {
		var prog *tea.Program
		if progRef != nil {
			prog = progRef.Load()
		}
		onStart := func(r engine.SyncResult) {
			if prog != nil {
				prog.Send(syncProgressMsg{result: r, started: true})
			}
		}
		onComplete := func(r engine.SyncResult) {
			if prog != nil {
				prog.Send(syncProgressMsg{result: r, started: false})
			}
		}
		results, err := eng.ExecuteSyncPlanWithCallbacks(
			ctx,
			plan,
			engine.SyncOptions{ContinueOnError: true},
			onStart,
			onComplete,
		)
		return syncDoneMsg{results: results, err: err}
	}
}

func (m tuiModel) handleStatusReport(msg statusReportMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	if msg.report != nil {
		m.repos = msg.report.Repos
		if m.filterText != "" {
			m.filteredRepos = filterRows(m.repos, m.filterText)
		}
	}
	m.cursor = 0
	m.offset = 0
	return m, nil
}

func (m tuiModel) handleSyncPlan(msg syncPlanMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.syncErr = msg.err
		m.mode = viewList
		return m, nil
	}
	m.syncPlan = msg.plan
	return m, nil
}

func (m tuiModel) handleSyncProgress(msg syncProgressMsg) (tea.Model, tea.Cmd) {
	if m.syncProgress == nil {
		m.syncProgress = make(map[string]engine.SyncResult)
	}
	r := msg.result
	r.Planned = msg.started
	m.syncProgress[syncResultIdentityKey(r)] = r
	return m, nil
}

func (m tuiModel) handleSyncDone(msg syncDoneMsg) (tea.Model, tea.Cmd) {
	m.syncErr = msg.err
	m.syncDone = true
	if m.syncProgress == nil {
		m.syncProgress = make(map[string]engine.SyncResult)
	}
	for _, r := range msg.results {
		m.syncProgress[syncResultIdentityKey(r)] = r
	}
	return m, nil
}

func (m tuiModel) handleRepoStatus(msg repoStatusMsg) (tea.Model, tea.Cmd) {
	if index := findRepoStatusIndex(m.repos, msg.status); index >= 0 {
		m.repos[index] = msg.status
	} else if repoStatusStillTracked(m.engine, msg.status) {
		m.repos = append(m.repos, msg.status)
	} else {
		// A late repoStatusMsg for a row that's no longer in m.repos (e.g. it
		// was deleted while its inspection was still in flight) and is no
		// longer in the registry either: drop it instead of resurrecting a
		// ghost row.
		m.pendingInspections--
		if m.pendingInspections <= 0 {
			m.loading = false
		}
		return m, nil
	}
	if m.engine != nil {
		writeRepoMetadataSnapshot(m.engine.Registry(), msg.status)
	}
	if m.filterText != "" {
		m.filteredRepos = filterRows(m.repos, m.filterText)
	}
	m.pendingInspections--
	if m.pendingInspections <= 0 {
		m.loading = false
	}
	return m, nil
}

// repoStatusStillTracked reports whether status still corresponds to a
// registry entry. It is permissive (true) when the engine or registry is
// unavailable so genuinely new rows (e.g. from the add flow, where the
// registry already contains the entry before streamStatusCmd is dispatched)
// keep appearing normally.
func repoStatusStillTracked(eng EngineAPI, status model.RepoStatus) bool {
	if eng == nil {
		return true
	}
	reg := eng.Registry()
	if reg == nil {
		return true
	}
	return reg.FindEntry(status.RepoID, status.Path) != nil
}

func writeRepoMetadataSnapshot(reg *registry.Registry, status model.RepoStatus) {
	if reg == nil {
		return
	}
	idx := reg.FindEntryIndex(status.RepoID, status.Path)
	if idx < 0 {
		return
	}
	entry := reg.Entries[idx]
	registry.StoreRepoMetadataStatus(&entry, status)
	reg.Entries[idx] = entry
}

func (m tuiModel) startReset() (tea.Model, tea.Cmd) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m, nil
	}
	repo := list[m.cursor]
	if m.engine != nil && repoIDRowCount(m.engine.Registry(), repo.RepoID) > 1 {
		m.statusMsg = fmt.Sprintf("repo_id %q is ambiguous across multiple checkouts; resolve with 'e' before resetting", repo.RepoID)
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewResetConfirm
	m.resetRepoID = repo.RepoID
	m.modalCursor = 0
	return m, nil
}

var resetOptionCount = 2

func (m tuiModel) handleResetConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = viewList
		m.resetRepoID = ""
		m.modalCursor = 0
		return m, nil
	}
	left, right := isModalNav(msg)
	if left {
		m = modalMoveLeft(m)
		return m, nil
	}
	if right {
		m = modalMoveRight(m, resetOptionCount)
		return m, nil
	}
	if msg.String() == "enter" {
		if m.modalCursor == 1 {
			repoID := m.resetRepoID
			m.mode = viewList
			m.resetRepoID = ""
			m.modalCursor = 0
			return m, resetRepoCmd(m.context(), m.engine, repoID, m.cfgPath)
		}
		m.mode = viewList
		m.resetRepoID = ""
		m.modalCursor = 0
	}
	return m, nil
}

func (m tuiModel) handleResetDone(msg resetDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "reset error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "reset: " + msg.repoID
	m.statusIsError = false
	return startStatusRefresh(m)
}

func (m tuiModel) startDelete() (tea.Model, tea.Cmd) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return m, nil
	}
	repo := list[m.cursor]
	if m.engine != nil && repoIDRowCount(m.engine.Registry(), repo.RepoID) > 1 {
		m.statusMsg = fmt.Sprintf("repo_id %q is ambiguous across multiple checkouts; resolve with 'e' before deleting", repo.RepoID)
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewDeleteConfirm
	m.deleteRepoID = repo.RepoID
	m.deleteRepoPath = repo.Path
	m.modalCursor = 0
	return m, nil
}

var deleteOptionCount = 3

func (m tuiModel) handleDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = viewList
		m.deleteRepoID = ""
		m.deleteRepoPath = ""
		m.modalCursor = 0
		return m, nil
	}
	left, right := isModalNav(msg)
	if left {
		m = modalMoveLeft(m)
		return m, nil
	}
	if right {
		m = modalMoveRight(m, deleteOptionCount)
		return m, nil
	}
	if msg.String() == "enter" {
		switch m.modalCursor {
		case 1:
			repoID := m.deleteRepoID
			m.mode = viewList
			m.deleteRepoID = ""
			m.deleteRepoPath = ""
			m.modalCursor = 0
			m.cursor = 0
			m.offset = 0
			return m, deleteRepoCmd(m.context(), m.engine, repoID, m.cfgPath, false)
		case 2:
			repoID := m.deleteRepoID
			m.mode = viewList
			m.deleteRepoID = ""
			m.deleteRepoPath = ""
			m.modalCursor = 0
			m.cursor = 0
			m.offset = 0
			return m, deleteRepoCmd(m.context(), m.engine, repoID, m.cfgPath, true)
		default:
			m.mode = viewList
			m.deleteRepoID = ""
			m.deleteRepoPath = ""
			m.modalCursor = 0
		}
	}
	return m, nil
}

func (m tuiModel) handleDeleteDone(msg deleteDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "delete error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "deleted: " + msg.repoID
	m.statusIsError = false
	m.repos = removeRepoByID(m.repos, msg.repoID)
	if m.filterText != "" {
		m.filteredRepos = removeRepoByID(m.filteredRepos, msg.repoID)
	}
	if m.cursor >= len(m.visibleList()) && m.cursor > 0 {
		m.cursor--
	}
	return m, nil
}

func removeRepoByID(repos []model.RepoStatus, repoID string) []model.RepoStatus {
	out := repos[:0]
	for _, r := range repos {
		if r.RepoID != repoID {
			out = append(out, r)
		}
	}
	return out
}

func (m tuiModel) startAdd() (tea.Model, tea.Cmd) {
	m.mode = viewAdd
	m.addURL = ""
	m.addPath = ""
	m.addMirror = false
	m.addField = addFieldURL
	return m, nil
}

func (m tuiModel) handleAddKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewList
		m.addURL = ""
		m.addPath = ""
		m.addMirror = false
		m.addField = addFieldURL
		return m, nil

	case "enter":
		switch m.addField {
		case addFieldURL:
			if strings.TrimSpace(m.addURL) == "" {
				return m, nil
			}
			m.addField = addFieldPath
		case addFieldPath:
			m.addField = addFieldMirror
		case addFieldMirror:
			url := strings.TrimSpace(m.addURL)
			path := resolvedAddPath(m)
			if url == "" || path == "" {
				m.statusMsg = "URL and path are required"
				m.statusIsError = true
				m.mode = viewList
				return m, nil
			}
			m.mode = viewList
			return m, cloneAndRegisterCmd(m.context(), m.engine, url, path, m.cfgPath, m.addMirror)
		}
		return m, nil

	case "backspace":
		switch m.addField {
		case addFieldURL:
			if len(m.addURL) > 0 {
				r := []rune(m.addURL)
				m.addURL = string(r[:len(r)-1])
			}
		case addFieldPath:
			if len(m.addPath) > 0 {
				r := []rune(m.addPath)
				m.addPath = string(r[:len(r)-1])
			}
		}
		return m, nil

	case "space":
		if m.addField == addFieldMirror {
			m.addMirror = !m.addMirror
		} else if t := msg.Text; t != "" {
			m = m.addFieldAppend(t)
		}
		return m, nil

	default:
		if t := msg.Text; t != "" {
			m = m.addFieldAppend(t)
		}
	}
	return m, nil
}

func (m tuiModel) addFieldAppend(text string) tuiModel {
	switch m.addField {
	case addFieldURL:
		m.addURL += text
	case addFieldPath:
		m.addPath += text
	}
	return m
}

func (m tuiModel) handleAddDone(msg addDoneMsg) (tea.Model, tea.Cmd) {
	m.addURL = ""
	m.addPath = ""
	m.addMirror = false
	m.addField = addFieldURL
	if msg.err != nil {
		m.statusMsg = "add error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusMsg = "added: " + msg.repoID
	m.statusIsError = false
	return startStatusRefresh(m)
}

func (m tuiModel) handleLabelEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewDetail
		m.labelRepoID = ""
		m.labelRepoPath = ""
		m.labelInput = ""
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil
	case "enter":
		return m, saveLabelEditCmd(m)
	case "backspace":
		if len(m.labelInput) > 0 {
			runes := []rune(m.labelInput)
			m.labelInput = string(runes[:len(runes)-1])
		}
		return m, nil
	default:
		if t := msg.Text; t != "" {
			m.labelInput += t
		}
	}
	return m, nil
}

func (m tuiModel) handleRepoMetadataEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m = resetRepoMetadataEditState(m)
		m.mode = viewDetail
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil
	case "up":
		if m.metadataField > 0 {
			m.metadataField--
		}
		return m, nil
	case "down":
		if m.metadataField < metadataFieldCount-1 {
			m.metadataField++
		}
		return m, nil
	case "enter":
		if m.metadataField == metadataFieldCount-1 {
			return m, saveRepoMetadataEditCmd(m)
		}
		m.metadataField++
		return m, nil
	case "backspace":
		m = trimRepoMetadataField(m)
		return m, nil
	default:
		if t := msg.Text; t != "" {
			m = appendRepoMetadataField(m, t)
		}
	}
	return m, nil
}

func appendRepoMetadataField(m tuiModel, text string) tuiModel {
	switch m.metadataField {
	case metadataFieldName:
		m.metadataName += text
	case metadataFieldRepoIDAssertion:
		m.metadataRepoIDAssertion += text
	case metadataFieldLabels:
		m.metadataLabelsInput += text
	case metadataFieldEntrypoints:
		m.metadataEntrypointsInput += text
	case metadataFieldAuthoritative:
		m.metadataAuthoritative += text
	case metadataFieldLowValue:
		m.metadataLowValue += text
	case metadataFieldProvides:
		m.metadataProvides += text
	case metadataFieldRelated:
		m.metadataRelated += text
	}
	return m
}

func trimRepoMetadataField(m tuiModel) tuiModel {
	trim := func(value string) string {
		if len(value) == 0 {
			return value
		}
		runes := []rune(value)
		return string(runes[:len(runes)-1])
	}
	switch m.metadataField {
	case metadataFieldName:
		m.metadataName = trim(m.metadataName)
	case metadataFieldRepoIDAssertion:
		m.metadataRepoIDAssertion = trim(m.metadataRepoIDAssertion)
	case metadataFieldLabels:
		m.metadataLabelsInput = trim(m.metadataLabelsInput)
	case metadataFieldEntrypoints:
		m.metadataEntrypointsInput = trim(m.metadataEntrypointsInput)
	case metadataFieldAuthoritative:
		m.metadataAuthoritative = trim(m.metadataAuthoritative)
	case metadataFieldLowValue:
		m.metadataLowValue = trim(m.metadataLowValue)
	case metadataFieldProvides:
		m.metadataProvides = trim(m.metadataProvides)
	case metadataFieldRelated:
		m.metadataRelated = trim(m.metadataRelated)
	}
	return m
}

func resetRepoMetadataEditState(m tuiModel) tuiModel {
	m.metadataRepoID = ""
	m.metadataRepoPath = ""
	m.metadataField = metadataFieldName
	m.metadataName = ""
	m.metadataRepoIDAssertion = ""
	m.metadataLabelsInput = ""
	m.metadataEntrypointsInput = ""
	m.metadataAuthoritative = ""
	m.metadataLowValue = ""
	m.metadataProvides = ""
	m.metadataRelated = ""
	m.metadataExists = false
	return m
}

func (m tuiModel) startRepair() (tea.Model, tea.Cmd) {
	repoID, target, err := resolveRepairTarget(m)
	if err != nil {
		m.statusMsg = err.Error()
		m.statusIsError = true
		return m, nil
	}
	if repoIDRowCount(m.engine.Registry(), repoID) > 1 {
		m.statusMsg = fmt.Sprintf("repo_id %q is ambiguous across multiple checkouts; resolve with 'e' before repairing", repoID)
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewRepairConfirm
	m.repairRepoID = repoID
	m.repairTargetUpstream = target
	m.modalCursor = 0
	return m, nil
}

var repairOptionCount = 2

func (m tuiModel) handleRepairConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || (msg.String() == "enter" && m.repairTargetUpstream == "") {
		m.mode = viewList
		m.repairRepoID = ""
		m.repairTargetUpstream = ""
		m.modalCursor = 0
		return m, nil
	}
	left, right := isModalNav(msg)
	if left {
		m = modalMoveLeft(m)
		return m, nil
	}
	if right {
		m = modalMoveRight(m, repairOptionCount)
		return m, nil
	}
	if msg.String() == "enter" {
		if m.modalCursor == 0 {
			m.mode = viewList
			m.repairRepoID = ""
			m.repairTargetUpstream = ""
			m.modalCursor = 0
			return m, nil
		}
		repoID := m.repairRepoID
		m.mode = viewList
		m.repairRepoID = ""
		m.repairTargetUpstream = ""
		m.modalCursor = 0
		return m, repairUpstreamCmd(m.context(), m.engine, repoID, m.cfgPath)
	}
	return m, nil
}

func (m tuiModel) handleEditReady(msg editReadyMsg) (tea.Model, tea.Cmd) {
	_, cmd := handleEditReady(msg)
	return m, cmd
}

func (m tuiModel) handleEditDone(msg editDoneMsg) (tea.Model, tea.Cmd) {
	m.mode = viewList
	if msg.err != nil {
		m.statusMsg = "edit error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	if !msg.saved {
		m.statusMsg = "no changes"
		m.statusIsError = false
		return m, nil
	}
	m.statusMsg = "updated " + msg.repoID
	m.statusIsError = false
	return startStatusRefresh(m)
}

func (m tuiModel) handleRepairDone(msg repairDoneMsg) (tea.Model, tea.Cmd) {
	m.repairRepoID = ""
	m.repairTargetUpstream = ""
	if msg.err != nil {
		m.statusMsg = "repair error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.statusIsError = false
	if msg.result.Action != "repaired" {
		m.statusMsg = msg.result.Action + ": " + msg.result.RepoID
		return m, nil
	}
	m.statusMsg = "repaired: " + msg.result.RepoID
	return startStatusRefresh(m)
}

func refreshStatusCmd(ctx context.Context, eng EngineAPI) tea.Cmd {
	reg := eng.Registry()
	if reg != nil && len(reg.Entries) > 0 {
		return streamStatusCmd(ctx, eng, reg.Entries)
	}
	return loadStatusCmd(ctx, eng)
}
