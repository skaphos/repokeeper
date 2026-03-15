// SPDX-License-Identifier: MIT
package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	addFieldURL    = 0
	addFieldPath   = 1
	addFieldMirror = 2
)

type addDoneMsg struct {
	repoID string
	err    error
}

func cloneAndRegisterCmd(eng EngineAPI, url, path, cfgPath string, mirror bool) tea.Cmd {
	return func() tea.Msg {
		err := eng.CloneAndRegister(context.Background(), url, path, cfgPath, mirror)
		if err != nil {
			return addDoneMsg{repoID: repoNameFromURL(url), err: err}
		}
		reg := eng.Registry()
		repoID := repoNameFromURL(url)
		if reg != nil && len(reg.Entries) > 0 {
			repoID = reg.Entries[len(reg.Entries)-1].RepoID
		}
		return addDoneMsg{repoID: repoID, err: nil}
	}
}

func repoNameFromURL(rawURL string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(rawURL), ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	if colon := strings.LastIndex(name, ":"); colon >= 0 {
		name = name[colon+1:]
	}
	return name
}

func defaultClonePath(rawURL string) string {
	return repoNameFromURL(rawURL)
}

func renderAddView(m tuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add Repository"))
	b.WriteByte('\n')
	b.WriteString(" " + renderDivider([]int{m.width - 2}))
	b.WriteByte('\n')
	b.WriteByte('\n')

	field := func(label, value string, active bool) {
		prefix := "  "
		if active {
			prefix = "▶ "
		}
		placeholder := ""
		if value == "" {
			placeholder = loadingStyle.Render("(type here)")
		}
		v := value
		if active && value != "" {
			v = value + "█"
		}
		if v == "" {
			v = placeholder
		}
		b.WriteString(fmt.Sprintf("%s%-20s %s\n", prefix, label+":", v))
	}

	field("URL", m.addURL, m.addField == addFieldURL)
	b.WriteByte('\n')

	pathVal := m.addPath
	if pathVal == "" && m.addField > addFieldURL {
		pathVal = defaultClonePath(m.addURL)
	}
	field("Checkout location", pathVal, m.addField == addFieldPath)
	b.WriteByte('\n')

	mirrorPrefix := "  "
	if m.addField == addFieldMirror {
		mirrorPrefix = "▶ "
	}
	mirrorVal := "No"
	if m.addMirror {
		mirrorVal = "Yes"
	}
	b.WriteString(fmt.Sprintf("%s%-20s %s\n", mirrorPrefix, "Mirror clone:", mirrorVal))
	if m.addField == addFieldMirror {
		b.WriteString("                       (space to toggle)\n")
	}
	b.WriteByte('\n')

	if m.addField == addFieldMirror {
		b.WriteString(statusBarStyle.Render("space: toggle mirror  enter: confirm  esc: cancel"))
	} else {
		b.WriteString(statusBarStyle.Render("enter: next field  esc: cancel"))
	}
	return b.String()
}

func resolvedAddPath(m tuiModel) string {
	if m.addPath != "" {
		return m.addPath
	}
	name := defaultClonePath(m.addURL)
	if name == "" {
		return ""
	}
	home, _ := filepath.Abs(".")
	return filepath.Join(home, name)
}
