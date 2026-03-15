// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/editor"
	"github.com/skaphos/repokeeper/internal/registry"
	"go.yaml.in/yaml/v3"
)

type editReadyMsg struct {
	editorCmd     *exec.Cmd
	tmpPath       string
	originalEntry registry.Entry
	entryIdx      int
	reg           *registry.Registry
	cfg           *config.Config
	cfgPath       string
}

type editDoneMsg struct {
	repoID string
	saved  bool
	err    error
}

func prepareEditCmd(m tuiModel) tea.Cmd {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return nil
	}

	repo := list[m.cursor]
	reg := m.engine.Registry()
	cfg := m.engine.Config()
	cfgPath := m.cfgPath

	return func() tea.Msg {
		if reg == nil {
			return editDoneMsg{err: fmt.Errorf("registry not available")}
		}

		var entry registry.Entry
		entryIdx := -1
		for i, e := range reg.Entries {
			if e.RepoID == repo.RepoID {
				entry = e
				entryIdx = i
				break
			}
		}
		if entryIdx < 0 {
			return editDoneMsg{err: fmt.Errorf("registry entry not found for %s", repo.RepoID)}
		}

		editorParts, err := editor.ResolveEditorCommand()
		if err != nil {
			return editDoneMsg{err: err}
		}

		tmpFile, err := os.CreateTemp("", "repokeeper-edit-*.yaml")
		if err != nil {
			return editDoneMsg{err: err}
		}
		tmpPath := tmpFile.Name()

		data, err := yaml.Marshal(entry)
		if err != nil {
			_ = os.Remove(tmpPath)
			return editDoneMsg{err: err}
		}
		if _, err := tmpFile.Write(data); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return editDoneMsg{err: err}
		}
		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return editDoneMsg{err: err}
		}

		cmd := exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return editReadyMsg{
			editorCmd:     cmd,
			tmpPath:       tmpPath,
			originalEntry: entry,
			entryIdx:      entryIdx,
			reg:           reg,
			cfg:           cfg,
			cfgPath:       cfgPath,
		}
	}
}

func handleEditReady(msg editReadyMsg) (tea.Model, tea.Cmd) {
	return nil, tea.ExecProcess(msg.editorCmd, func(err error) tea.Msg {
		defer func() { _ = os.Remove(msg.tmpPath) }()

		if err != nil {
			return editDoneMsg{repoID: msg.originalEntry.RepoID, err: err}
		}

		data, readErr := os.ReadFile(msg.tmpPath)
		if readErr != nil {
			return editDoneMsg{repoID: msg.originalEntry.RepoID, err: readErr}
		}

		var edited registry.Entry
		if unmarshalErr := yaml.Unmarshal(data, &edited); unmarshalErr != nil {
			return editDoneMsg{repoID: msg.originalEntry.RepoID, err: fmt.Errorf("invalid yaml: %w", unmarshalErr)}
		}

		if reflect.DeepEqual(msg.originalEntry, edited) {
			return editDoneMsg{repoID: msg.originalEntry.RepoID, saved: false}
		}

		if validateErr := validateEditEntry(edited, msg.reg, msg.entryIdx); validateErr != nil {
			return editDoneMsg{repoID: msg.originalEntry.RepoID, err: validateErr}
		}

		if edited.LastSeen.IsZero() {
			edited.LastSeen = time.Now()
		}
		msg.reg.Entries[msg.entryIdx] = edited
		msg.reg.UpdatedAt = time.Now()
		msg.cfg.Registry = msg.reg

		if saveErr := config.Save(msg.cfg, msg.cfgPath); saveErr != nil {
			return editDoneMsg{repoID: edited.RepoID, err: saveErr}
		}

		return editDoneMsg{repoID: edited.RepoID, saved: true}
	})
}

func validateEditEntry(entry registry.Entry, reg *registry.Registry, index int) error {
	if strings.TrimSpace(entry.RepoID) == "" {
		return fmt.Errorf("invalid entry: repo_id is required")
	}
	entryPath := strings.TrimSpace(entry.Path)
	if entryPath == "" {
		return fmt.Errorf("invalid entry: path is required")
	}
	if !filepath.IsAbs(entryPath) {
		return fmt.Errorf("invalid entry: path must be absolute, got %q", entry.Path)
	}
	if entry.Status == "" {
		return fmt.Errorf("invalid entry: status is required")
	}
	switch entry.Status {
	case registry.StatusPresent, registry.StatusMissing, registry.StatusMoved:
	default:
		return fmt.Errorf("invalid entry: unsupported status %q", entry.Status)
	}
	typ := strings.TrimSpace(entry.Type)
	if typ != "" && typ != "checkout" && typ != "mirror" {
		return fmt.Errorf("invalid entry: unsupported type %q", entry.Type)
	}
	for key := range entry.Labels {
		if err := validateEntryKey(strings.TrimSpace(key), "label"); err != nil {
			return err
		}
	}
	for key := range entry.Annotations {
		if err := validateEntryKey(strings.TrimSpace(key), "annotation"); err != nil {
			return err
		}
	}
	if reg != nil {
		for i := range reg.Entries {
			if i == index {
				continue
			}
			if reg.Entries[i].RepoID == entry.RepoID {
				return fmt.Errorf("invalid entry: repo_id %q already exists", entry.RepoID)
			}
		}
	}
	return nil
}

func validateEntryKey(key, field string) error {
	if key == "" {
		return fmt.Errorf("invalid %s key: cannot be empty", field)
	}
	if strings.ContainsAny(key, " \t\r\n=") {
		return fmt.Errorf("invalid %s key %q: keys cannot contain whitespace or '='", field, key)
	}
	return nil
}
