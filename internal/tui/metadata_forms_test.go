// SPDX-License-Identifier: MIT
package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestRenderMetadataViewsAndHelpers(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-name_here")
	if err := os.MkdirAll(filepath.Join(repoPath, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoPath, "generated"), 0o755); err != nil {
		t.Fatalf("mkdir generated: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	entry := registry.Entry{RepoID: "github.com/acme/repo", Path: repoPath}
	defaults := defaultRepoMetadataForTUI(entry, nil)
	if defaults.Name != "Repo Name Here" {
		t.Fatalf("expected humanized name, got %+v", defaults)
	}
	if defaults.Entrypoints["readme"] != "README.md" {
		t.Fatalf("expected readme entrypoint, got %+v", defaults.Entrypoints)
	}
	if strings.Join(defaults.Paths.Authoritative, ",") != "docs/" {
		t.Fatalf("expected authoritative dirs, got %+v", defaults.Paths.Authoritative)
	}
	if strings.Join(defaults.Paths.LowValue, ",") != "generated/" {
		t.Fatalf("expected low-value dirs, got %+v", defaults.Paths.LowValue)
	}

	existing := &model.RepoMetadata{
		Name:        "Existing",
		RepoID:      entry.RepoID,
		Labels:      map[string]string{"team": "platform"},
		Entrypoints: map[string]string{"readme": "README.md"},
		Paths: model.RepoMetadataPaths{
			Authoritative: []string{"docs/"},
			LowValue:      []string{"generated/"},
		},
		Provides:     []string{"cli"},
		RelatedRepos: []model.RepoMetadataRelatedRepo{{RepoID: "acme/other", Relationship: "depends-on"}},
	}
	copy := defaultRepoMetadataForTUI(entry, existing)
	copy.Labels["team"] = "changed"
	copy.Entrypoints["readme"] = "HACKED.md"
	copy.Paths.Authoritative[0] = "src/"
	copy.Provides[0] = "worker"
	copy.RelatedRepos[0].RepoID = "changed"
	if existing.Labels["team"] != "platform" || existing.Entrypoints["readme"] != "README.md" {
		t.Fatalf("expected metadata defaults clone, got %+v", existing)
	}
	if existing.Paths.Authoritative[0] != "docs/" || existing.Provides[0] != "cli" || existing.RelatedRepos[0].RepoID != "acme/other" {
		t.Fatalf("expected deep clone semantics, got %+v", existing)
	}

	m := tuiModel{
		width:                    80,
		labelRepoID:              entry.RepoID,
		labelInput:               "team=platform",
		metadataRepoID:           entry.RepoID,
		metadataField:            metadataFieldLabels,
		metadataName:             "Repo Name",
		metadataRepoIDAssertion:  entry.RepoID,
		metadataLabelsInput:      "team=platform",
		metadataEntrypointsInput: "readme=README.md",
		metadataAuthoritative:    "docs/",
		metadataLowValue:         "generated/",
		metadataProvides:         "cli",
		metadataRelated:          "acme/other:depends-on",
	}
	labelView := renderLabelEditView(m)
	if !strings.Contains(labelView, "Edit Labels") || !strings.Contains(labelView, "team=platform") {
		t.Fatalf("unexpected label edit view: %q", labelView)
	}
	metadataView := renderRepoMetadataEditView(m)
	if !strings.Contains(metadataView, "Initialize Repo Metadata") || !strings.Contains(metadataView, "Related repos") {
		t.Fatalf("unexpected repo metadata view: %q", metadataView)
	}
	m.metadataExists = true
	if !strings.Contains(renderRepoMetadataEditView(m), "Edit Repo Metadata") {
		t.Fatal("expected edit title for existing metadata")
	}

	fields := repoMetadataFields(m)
	if len(fields) != metadataFieldCount || fields[metadataFieldRelated].label != "Related repos" {
		t.Fatalf("unexpected metadata fields: %+v", fields)
	}
	if got := formatRelatedReposCSV([]model.RepoMetadataRelatedRepo{{RepoID: "z/repo"}, {RepoID: "a/repo", Relationship: "documents"}}); got != "a/repo:documents,z/repo" {
		t.Fatalf("unexpected related repos csv %q", got)
	}
	if sameStringMap(map[string]string{"team": "platform"}, map[string]string{"team": "platform"}) != true {
		t.Fatal("expected sameStringMap true for identical maps")
	}
	if sameStringMap(map[string]string{"team": "platform"}, map[string]string{"team": "docs"}) != false {
		t.Fatal("expected sameStringMap false for changed values")
	}
	if got := cloneMetadataStringMap(nil); got != nil {
		t.Fatalf("expected nil map clone, got %+v", got)
	}
}

func TestEditKeyHandlersAndMetadataFieldMutationHelpers(t *testing.T) {
	t.Parallel()

	m := tuiModel{mode: viewEditLabels, labelRepoID: "acme/a", labelRepoPath: "/tmp/repo", labelInput: "team=platform", statusMsg: "stale", statusIsError: true}
	next, cmd := m.handleLabelEditKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd != nil || next.(tuiModel).labelInput != "team=platfor" {
		t.Fatalf("expected backspace label edit, got %+v", next)
	}
	next, _ = next.(tuiModel).handleLabelEditKey(tea.KeyPressMsg{Text: "m"})
	if next.(tuiModel).labelInput != "team=platform" {
		t.Fatalf("expected appended label input, got %+v", next)
	}
	next, _ = next.(tuiModel).handleLabelEditKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm := next.(tuiModel)
	if nm.mode != viewDetail || nm.labelRepoID != "" || nm.statusMsg != "" || nm.statusIsError {
		t.Fatalf("expected escaped label editor reset, got %+v", nm)
	}

	m = tuiModel{mode: viewEditRepoMetadata, metadataField: metadataFieldName, metadataName: "Repo", metadataRepoIDAssertion: "acme/a", metadataLabelsInput: "team=platform", metadataEntrypointsInput: "readme=README.md", metadataAuthoritative: "docs/", metadataLowValue: "generated/", metadataProvides: "cli", metadataRelated: "acme/b"}
	for field, suffix := range map[int]string{
		metadataFieldName:            "!",
		metadataFieldRepoIDAssertion: "!",
		metadataFieldLabels:          "!",
		metadataFieldEntrypoints:     "!",
		metadataFieldAuthoritative:   "!",
		metadataFieldLowValue:        "!",
		metadataFieldProvides:        "!",
		metadataFieldRelated:         "!",
	} {
		m.metadataField = field
		m = appendRepoMetadataField(m, suffix)
		m = trimRepoMetadataField(m)
	}
	if m.metadataName != "Repo" || m.metadataRepoIDAssertion != "acme/a" || m.metadataLabelsInput != "team=platform" || m.metadataEntrypointsInput != "readme=README.md" || m.metadataAuthoritative != "docs/" || m.metadataLowValue != "generated/" || m.metadataProvides != "cli" || m.metadataRelated != "acme/b" {
		t.Fatalf("expected append+trim round trip to preserve values, got %+v", m)
	}

	m.metadataField = metadataFieldName
	next, cmd = m.handleRepoMetadataEditKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if cmd != nil || next.(tuiModel).metadataField != metadataFieldName {
		t.Fatalf("expected up key to clamp at first field, got %+v", next)
	}
	next, _ = next.(tuiModel).handleRepoMetadataEditKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if next.(tuiModel).metadataField != metadataFieldRepoIDAssertion {
		t.Fatalf("expected down key to advance field, got %+v", next)
	}
	next, _ = next.(tuiModel).handleRepoMetadataEditKey(tea.KeyPressMsg{Text: "x"})
	if next.(tuiModel).metadataRepoIDAssertion != "acme/ax" {
		t.Fatalf("expected text append on active field, got %+v", next)
	}
	next, _ = next.(tuiModel).handleRepoMetadataEditKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if next.(tuiModel).metadataRepoIDAssertion != "acme/a" {
		t.Fatalf("expected backspace trim on active field, got %+v", next)
	}
	next, cmd = next.(tuiModel).handleRepoMetadataEditKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || next.(tuiModel).metadataField != metadataFieldLabels {
		t.Fatalf("expected enter to move to next field before save, got %+v", next)
	}
	next, _ = next.(tuiModel).handleRepoMetadataEditKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	nm = next.(tuiModel)
	if nm.mode != viewDetail || nm.metadataRepoID != "" || nm.metadataLabelsInput != "" || nm.statusMsg != "" || nm.statusIsError {
		t.Fatalf("expected escaped metadata editor reset, got %+v", nm)
	}
}
