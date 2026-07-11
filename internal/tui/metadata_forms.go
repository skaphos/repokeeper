// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/repometa"
)

const (
	metadataFieldName = iota
	metadataFieldRepoIDAssertion
	metadataFieldLabels
	metadataFieldEntrypoints
	metadataFieldAuthoritative
	metadataFieldLowValue
	metadataFieldProvides
	metadataFieldRelated
	metadataFieldCount
)

// labelEditDoneMsg carries the outcome of the (pure, registry-free) parsing
// step run in saveLabelEditCmd's Cmd goroutine. The actual registry mutation
// and config.Save happen afterward in handleLabelEditDone, which runs on the
// single Update goroutine, so the shared *registry.Registry is never touched
// concurrently with the inspect goroutines started by streamStatusCmd.
type labelEditDoneMsg struct {
	repoID   string
	repoPath string
	labels   map[string]string
	err      error
}

// repoMetadataEditDoneMsg carries the outcome of saveRepoMetadataEditCmd.
// refreshed is the freshly-computed repo metadata snapshot; applying it to
// the registry entry and persisting config happen in
// handleRepoMetadataEditDone on the Update goroutine, for the same reason as
// labelEditDoneMsg above.
type repoMetadataEditDoneMsg struct {
	repoID    string
	repoPath  string
	refreshed model.RepoStatus
	saved     bool
	err       error
}

func startLabelEdit(m tuiModel) (tea.Model, tea.Cmd) {
	repo, ok := currentVisibleRepo(m)
	if !ok {
		m.statusMsg = "no repository selected"
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewEditLabels
	m.labelRepoID = repo.RepoID
	m.labelRepoPath = repo.Path
	m.labelInput = formatStringMapCSV(repo.Labels)
	m.statusMsg = ""
	m.statusIsError = false
	return m, nil
}

func startRepoMetadataEdit(m tuiModel) (tea.Model, tea.Cmd) {
	repo, ok := currentVisibleRepo(m)
	if !ok {
		m.statusMsg = "no repository selected"
		m.statusIsError = true
		return m, nil
	}
	if repo.RepoMetadataError != "" {
		m.statusMsg = "repo metadata error: " + repo.RepoMetadataError
		m.statusIsError = true
		return m, nil
	}
	entry, _, err := currentRegistryEntry(m)
	if err != nil {
		m.statusMsg = err.Error()
		m.statusIsError = true
		return m, nil
	}
	proposal := defaultRepoMetadataForTUI(entry, repo.RepoMetadata)
	m.mode = viewEditRepoMetadata
	m.metadataRepoID = repo.RepoID
	m.metadataRepoPath = repo.Path
	m.metadataField = metadataFieldName
	m.metadataExists = repo.RepoMetadata != nil && repo.RepoMetadataFile != ""
	m.metadataName = proposal.Name
	m.metadataRepoIDAssertion = proposal.RepoID
	m.metadataLabelsInput = formatStringMapCSV(proposal.Labels)
	m.metadataEntrypointsInput = formatStringMapCSV(proposal.Entrypoints)
	m.metadataAuthoritative = strings.Join(proposal.Paths.Authoritative, ",")
	m.metadataLowValue = strings.Join(proposal.Paths.LowValue, ",")
	m.metadataProvides = strings.Join(proposal.Provides, ",")
	m.metadataRelated = formatRelatedReposCSV(proposal.RelatedRepos)
	m.statusMsg = ""
	m.statusIsError = false
	return m, nil
}

// saveLabelEditCmd only parses the input the user typed. It deliberately
// never touches the shared *registry.Registry or calls config.Save: those
// happen in handleLabelEditDone on the Update goroutine, so this Cmd
// goroutine cannot race with the registry reads done by in-flight
// inspection Cmds (stream.go).
func saveLabelEditCmd(m tuiModel) tea.Cmd {
	repoID := m.labelRepoID
	repoPath := m.labelRepoPath
	input := m.labelInput
	return func() tea.Msg {
		labels, err := parseStringMapCSV(input, "label")
		if err != nil {
			return labelEditDoneMsg{repoID: repoID, err: err}
		}
		return labelEditDoneMsg{repoID: repoID, repoPath: repoPath, labels: labels}
	}
}

func (m tuiModel) handleLabelEditDone(msg labelEditDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "label error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m.mode = viewDetail
	m.labelRepoID = ""
	m.labelRepoPath = ""
	m.labelInput = ""

	reg := m.engine.Registry()
	if reg == nil {
		m.statusMsg = "label error: registry not available"
		m.statusIsError = true
		return m, nil
	}
	cfg := m.engine.Config()
	if cfg == nil {
		m.statusMsg = "label error: config not available"
		m.statusIsError = true
		return m, nil
	}
	if strings.TrimSpace(m.cfgPath) == "" {
		m.statusMsg = "label error: config path not available"
		m.statusIsError = true
		return m, nil
	}
	index := registryEntryIndexByIdentity(reg, msg.repoID, msg.repoPath)
	if index < 0 {
		m.statusMsg = fmt.Sprintf("label error: registry entry not found for %s", msg.repoID)
		m.statusIsError = true
		return m, nil
	}
	if sameStringMap(reg.Entries[index].Labels, msg.labels) {
		m.statusMsg = "no label changes"
		m.statusIsError = false
		return m, nil
	}
	reg.Entries[index].Labels = msg.labels
	reg.Entries[index].LastSeen = time.Now()
	reg.UpdatedAt = time.Now()
	cfg.Registry = reg
	if err := config.Save(cfg, m.cfgPath); err != nil {
		m.statusMsg = "label error: " + err.Error()
		m.statusIsError = true
		return m, nil
	}

	m.statusMsg = "updated labels for " + msg.repoID
	m.statusIsError = false
	return startStatusRefresh(m)
}

// saveRepoMetadataEditCmd resolves the target registry entry and copies the
// fields it needs (entry.Path, entry.RepoID) synchronously on the Update
// goroutine before returning the Cmd, so the goroutine below only reads its
// own local copies plus does file I/O for the repo's own metadata file. The
// registry mutation and config.Save happen afterward in
// handleRepoMetadataEditDone on the Update goroutine, for the same
// concurrency-safety reason as saveLabelEditCmd above.
func saveRepoMetadataEditCmd(m tuiModel) tea.Cmd {
	repoID := m.metadataRepoID
	repoPath := m.metadataRepoPath
	name := m.metadataName
	repoAssertion := m.metadataRepoIDAssertion
	labelsInput := m.metadataLabelsInput
	entrypointsInput := m.metadataEntrypointsInput
	authoritativeInput := m.metadataAuthoritative
	lowValueInput := m.metadataLowValue
	providesInput := m.metadataProvides
	relatedInput := m.metadataRelated
	force := m.metadataExists

	reg := m.engine.Registry()
	cfgAvailable := m.engine.Config() != nil
	cfgPath := m.cfgPath

	var entry registry.Entry
	entryFound := false
	if reg != nil {
		if index := registryEntryIndexByIdentity(reg, repoID, repoPath); index >= 0 {
			entry = reg.Entries[index]
			entryFound = true
		}
	}

	return func() tea.Msg {
		if reg == nil {
			return repoMetadataEditDoneMsg{repoID: repoID, err: fmt.Errorf("registry not available")}
		}
		if !cfgAvailable {
			return repoMetadataEditDoneMsg{repoID: repoID, err: fmt.Errorf("config not available")}
		}
		if strings.TrimSpace(cfgPath) == "" {
			return repoMetadataEditDoneMsg{repoID: repoID, err: fmt.Errorf("config path not available")}
		}
		if !entryFound {
			return repoMetadataEditDoneMsg{repoID: repoID, err: fmt.Errorf("registry entry not found for %s", repoID)}
		}
		labels, err := parseStringMapCSV(labelsInput, "label")
		if err != nil {
			return repoMetadataEditDoneMsg{repoID: repoID, err: err}
		}
		entrypoints, err := parseStringMapCSV(entrypointsInput, "entrypoint")
		if err != nil {
			return repoMetadataEditDoneMsg{repoID: repoID, err: err}
		}
		related, err := parseRelatedReposCSV(relatedInput)
		if err != nil {
			return repoMetadataEditDoneMsg{repoID: repoID, err: err}
		}
		proposal := &model.RepoMetadata{
			Name:        strings.TrimSpace(name),
			RepoID:      strings.TrimSpace(repoAssertion),
			Labels:      labels,
			Entrypoints: entrypoints,
			Paths: model.RepoMetadataPaths{
				Authoritative: parseCSV(authoritativeInput),
				LowValue:      parseCSV(lowValueInput),
			},
			Provides:     parseCSV(providesInput),
			RelatedRepos: related,
		}
		if proposal.RepoID != "" && proposal.RepoID != entry.RepoID {
			return repoMetadataEditDoneMsg{repoID: repoID, err: fmt.Errorf("repo metadata repo_id %q must match tracked repo_id %q", proposal.RepoID, entry.RepoID)}
		}
		if _, err := repometa.Save(entry.Path, proposal, force); err != nil {
			return repoMetadataEditDoneMsg{repoID: repoID, err: err}
		}
		refreshed := model.RepoStatus{RepoID: entry.RepoID, Path: entry.Path}
		registry.SeedRepoMetadataStatus(entry, &refreshed)
		repometa.Apply(&refreshed)
		return repoMetadataEditDoneMsg{repoID: repoID, repoPath: repoPath, refreshed: refreshed, saved: true}
	}
}

func (m tuiModel) handleRepoMetadataEditDone(msg repoMetadataEditDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "repo metadata error: " + msg.err.Error()
		m.statusIsError = true
		return m, nil
	}
	m = resetRepoMetadataEditState(m)
	m.mode = viewDetail
	if !msg.saved {
		m.statusMsg = "no repo metadata changes"
		m.statusIsError = false
		return m, nil
	}

	reg := m.engine.Registry()
	if reg == nil {
		m.statusMsg = "repo metadata error: registry not available"
		m.statusIsError = true
		return m, nil
	}
	cfg := m.engine.Config()
	if cfg == nil {
		m.statusMsg = "repo metadata error: config not available"
		m.statusIsError = true
		return m, nil
	}
	index := registryEntryIndexByIdentity(reg, msg.repoID, msg.repoPath)
	if index < 0 {
		m.statusMsg = fmt.Sprintf("repo metadata error: registry entry not found for %s", msg.repoID)
		m.statusIsError = true
		return m, nil
	}
	entry := reg.Entries[index]
	registry.StoreRepoMetadataStatus(&entry, msg.refreshed)
	reg.Entries[index] = entry
	cfg.Registry = reg
	if err := config.Save(cfg, m.cfgPath); err != nil {
		m.statusMsg = "repo metadata error: " + err.Error()
		m.statusIsError = true
		return m, nil
	}

	m.statusMsg = "updated repo metadata for " + msg.repoID
	m.statusIsError = false
	return startStatusRefresh(m)
}

func renderLabelEditView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Edit Labels"))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  Repository: %s\n\n", m.labelRepoID)
	if m.statusMsg != "" {
		style := loadingStyle
		if m.statusIsError {
			style = errorTextStyle
		}
		b.WriteString("  " + style.Render(m.statusMsg))
		b.WriteString("\n\n")
	}
	value := m.labelInput
	if value == "" {
		value = loadingStyle.Render("(empty clears labels)")
	} else {
		value += "█"
	}
	fmt.Fprintf(&b, "▶ %-20s %s\n\n", "Labels (k=v,...)"+":", value)
	b.WriteString(statusBarStyle.Render("enter: save  esc: cancel  backspace: delete"))
	return b.String()
}

func renderRepoMetadataEditView(m tuiModel) string {
	var b strings.Builder
	title := "Initialize Repo Metadata"
	if m.metadataExists {
		title = "Edit Repo Metadata"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  Repository: %s\n\n", m.metadataRepoID)
	if m.statusMsg != "" {
		style := loadingStyle
		if m.statusIsError {
			style = errorTextStyle
		}
		b.WriteString("  " + style.Render(m.statusMsg))
		b.WriteString("\n\n")
	}
	for idx, field := range repoMetadataFields(m) {
		prefix := "  "
		if idx == m.metadataField {
			prefix = "▶ "
		}
		value := field.value
		if value == "" {
			value = loadingStyle.Render("(empty)")
		} else if idx == m.metadataField {
			value += "█"
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", prefix, field.label+":", value)
		if idx < metadataFieldCount-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(statusBarStyle.Render("enter: next/save  ↑/↓: move  esc: cancel  backspace: delete"))
	return b.String()
}

type repoMetadataFieldView struct {
	label string
	value string
}

func repoMetadataFields(m tuiModel) []repoMetadataFieldView {
	return []repoMetadataFieldView{
		{label: "Repository name", value: m.metadataName},
		{label: "Repo ID assertion", value: m.metadataRepoIDAssertion},
		{label: "Labels", value: m.metadataLabelsInput},
		{label: "Entrypoints", value: m.metadataEntrypointsInput},
		{label: "Authoritative paths", value: m.metadataAuthoritative},
		{label: "Low-value paths", value: m.metadataLowValue},
		{label: "Provides", value: m.metadataProvides},
		{label: "Related repos", value: m.metadataRelated},
	}
}

func currentVisibleRepo(m tuiModel) (model.RepoStatus, bool) {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return model.RepoStatus{}, false
	}
	return list[m.cursor], true
}

func currentRegistryEntry(m tuiModel) (registry.Entry, int, error) {
	repo, ok := currentVisibleRepo(m)
	if !ok {
		return registry.Entry{}, -1, fmt.Errorf("no repository selected")
	}
	reg := m.engine.Registry()
	if reg == nil {
		return registry.Entry{}, -1, fmt.Errorf("registry not available")
	}
	index := registryEntryIndexByIdentity(reg, repo.RepoID, repo.Path)
	if index >= 0 {
		return reg.Entries[index], index, nil
	}
	return registry.Entry{}, -1, fmt.Errorf("registry entry not found for %s", repo.RepoID)
}

func registryEntryIndexByIdentity(reg *registry.Registry, repoID, path string) int {
	if reg == nil {
		return -1
	}
	normalizedPath := filepath.Clean(strings.TrimSpace(path))
	if normalizedPath != "" {
		for i, entry := range reg.Entries {
			if filepath.Clean(strings.TrimSpace(entry.Path)) == normalizedPath {
				return i
			}
		}
	}
	for i, entry := range reg.Entries {
		if entry.RepoID == repoID {
			return i
		}
	}
	return -1
}

func formatStringMapCSV(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func parseStringMapCSV(raw, field string) (map[string]string, error) {
	parts := parseCSV(raw)
	if len(parts) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid %s value %q: expected key=value", field, part)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if err := validateEntryKey(key, field); err != nil {
			return nil, err
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func formatRelatedReposCSV(values []model.RepoMetadataRelatedRepo) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Relationship) == "" {
			parts = append(parts, value.RepoID)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s", value.RepoID, value.Relationship))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func parseRelatedReposCSV(raw string) ([]model.RepoMetadataRelatedRepo, error) {
	parts := parseCSV(raw)
	if len(parts) == 0 {
		return nil, nil
	}
	out := make([]model.RepoMetadataRelatedRepo, 0, len(parts))
	for _, part := range parts {
		repoID, relationship, _ := strings.Cut(part, ":")
		repoID = strings.TrimSpace(repoID)
		relationship = strings.TrimSpace(relationship)
		if repoID == "" {
			return nil, fmt.Errorf("related repos entries require repo_id")
		}
		out = append(out, model.RepoMetadataRelatedRepo{RepoID: repoID, Relationship: relationship})
	}
	return out, nil
}

func defaultRepoMetadataForTUI(entry registry.Entry, existing *model.RepoMetadata) *model.RepoMetadata {
	if existing != nil {
		copy := *existing
		copy.Labels = cloneMetadataStringMap(copy.Labels)
		copy.Entrypoints = cloneMetadataStringMap(copy.Entrypoints)
		copy.Paths.Authoritative = append([]string(nil), copy.Paths.Authoritative...)
		copy.Paths.LowValue = append([]string(nil), copy.Paths.LowValue...)
		copy.Provides = append([]string(nil), copy.Provides...)
		copy.RelatedRepos = append([]model.RepoMetadataRelatedRepo(nil), copy.RelatedRepos...)
		return &copy
	}
	metadata := &model.RepoMetadata{RepoID: entry.RepoID}
	if base := filepath.Base(entry.Path); base != "" && base != "." {
		metadata.Name = humanizeRepoNameForTUI(base)
	}
	metadata.Entrypoints = make(map[string]string)
	if readme := detectReadmeEntrypointForTUI(entry.Path); readme != "" {
		metadata.Entrypoints["readme"] = readme
	}
	metadata.Paths.Authoritative = detectNamedDirsForTUI(entry.Path, []string{"docs", "src", "cmd", "internal", "pkg", "app", "lib", "templates", "scripts", "examples"})
	metadata.Paths.LowValue = detectNamedDirsForTUI(entry.Path, []string{"generated", "dist", "build", "archive", ".github", "vendor", "node_modules"})
	return metadata
}

func humanizeRepoNameForTUI(name string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ")
	parts := strings.Fields(replacer.Replace(name))
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func detectReadmeEntrypointForTUI(repoRoot string) string {
	for _, candidate := range []string{"README.md", "README.rst", "README.txt", "README"} {
		if _, err := os.Stat(filepath.Join(repoRoot, candidate)); err == nil {
			return candidate
		}
	}
	return ""
}

func detectNamedDirsForTUI(repoRoot string, preferred []string) []string {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil
	}
	available := make(map[string]bool, len(entries))
	for _, entry := range entries {
		available[entry.Name()] = true
	}
	out := make([]string, 0, len(preferred))
	for _, name := range preferred {
		if available[name] {
			out = append(out, name+"/")
		}
	}
	return out
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func cloneMetadataStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
