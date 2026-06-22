// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/repometa"
)

func TestFormatAssignmentDefaultsSortsKeys(t *testing.T) {
	t.Parallel()

	got := formatAssignmentDefaults(map[string]string{"zeta": "last", "alpha": "first"})
	if got != "alpha=first,zeta=last" {
		t.Fatalf("expected sorted assignments, got %q", got)
	}
}

func TestParseIndexAssignments(t *testing.T) {
	t.Parallel()

	got, err := parseIndexAssignments("zeta=last, alpha=first")
	if err != nil {
		t.Fatalf("parse assignments: %v", err)
	}
	if got["alpha"] != "first" || got["zeta"] != "last" {
		t.Fatalf("unexpected assignments: %#v", got)
	}

	empty, err := parseIndexAssignments(" , ")
	if err != nil {
		t.Fatalf("parse empty assignments: %v", err)
	}
	if empty != nil {
		t.Fatalf("expected nil assignments for empty input, got %#v", empty)
	}

	if _, err := parseIndexAssignments("missing-value"); err == nil {
		t.Fatal("expected malformed assignments to fail")
	}
}

func TestParseIndexListSortsAndTrims(t *testing.T) {
	t.Parallel()

	got := parseIndexList(" zeta , alpha ,, beta ")
	want := []string{"alpha", "beta", "zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected sorted list %v, got %v", want, got)
	}
}

func TestFormatRelatedRepoDefaultsSortsValues(t *testing.T) {
	t.Parallel()

	got := formatRelatedRepoDefaults([]model.RepoMetadataRelatedRepo{{RepoID: "z/repo", Relationship: "depends-on"}, {RepoID: "a/repo"}})
	if got != "a/repo,z/repo:depends-on" {
		t.Fatalf("expected sorted related repos, got %q", got)
	}
}

func TestIndexQuestionerUsesDefaultsAndParsers(t *testing.T) {
	t.Parallel()

	defaults := map[string]string{"team": "platform"}
	defaultList := []string{"docs/", "src/"}
	defaultRelated := []model.RepoMetadataRelatedRepo{{RepoID: "acme/docs", Relationship: "documents"}}

	var prompts bytes.Buffer
	q := newIndexQuestioner(strings.NewReader("\n\n\n"), &prompts)

	assignments, err := q.askAssignments("Labels", defaults)
	if err != nil {
		t.Fatalf("ask assignments with defaults: %v", err)
	}
	assignments["team"] = "changed"
	if defaults["team"] != "platform" {
		t.Fatalf("expected assignments defaults to be cloned, got %#v", defaults)
	}

	list, err := q.askList("Authoritative", defaultList)
	if err != nil {
		t.Fatalf("ask list with defaults: %v", err)
	}
	list[0] = "changed"
	if defaultList[0] != "docs/" {
		t.Fatalf("expected list defaults to be copied, got %v", defaultList)
	}

	related, err := q.askRelatedRepos("Related", defaultRelated)
	if err != nil {
		t.Fatalf("ask related repos with defaults: %v", err)
	}
	related[0].RepoID = "changed"
	if defaultRelated[0].RepoID != "acme/docs" {
		t.Fatalf("expected related defaults to be copied, got %#v", defaultRelated)
	}

	out := prompts.String()
	for _, want := range []string{
		"Labels [team=platform]: ",
		"Authoritative [docs/,src/]: ",
		"Related [acme/docs:documents]: ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected prompt %q in %q", want, out)
		}
	}

	q = newIndexQuestioner(strings.NewReader("zeta=last, alpha=first\n zeta , alpha \nz/repo:depends-on, a/repo\n"), &bytes.Buffer{})
	assignments, err = q.askAssignments("Labels", nil)
	if err != nil {
		t.Fatalf("ask assignments parse: %v", err)
	}
	if assignments["alpha"] != "first" || assignments["zeta"] != "last" {
		t.Fatalf("unexpected parsed assignments: %#v", assignments)
	}

	list, err = q.askList("Authoritative", nil)
	if err != nil {
		t.Fatalf("ask list parse: %v", err)
	}
	if strings.Join(list, ",") != "alpha,zeta" {
		t.Fatalf("unexpected parsed list: %v", list)
	}

	related, err = q.askRelatedRepos("Related", nil)
	if err != nil {
		t.Fatalf("ask related parse: %v", err)
	}
	if len(related) != 2 || related[0].RepoID != "a/repo" || related[1].Relationship != "depends-on" {
		t.Fatalf("unexpected parsed related repos: %#v", related)
	}
}

func TestGuessRepoMetadataDefaultsDetectsRepoStructure(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "repo-name_here")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	for _, dir := range []string{"docs", "src", "generated", "vendor"} {
		if err := os.MkdirAll(filepath.Join(repoPath, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	got := guessRepoMetadataDefaults(registry.Entry{RepoID: "github.com/acme/repo-name", Path: repoPath}, nil)
	if got.Name != "Repo Name Here" {
		t.Fatalf("expected guessed name, got %#v", got)
	}
	if got.RepoID != "github.com/acme/repo-name" {
		t.Fatalf("expected repo id to carry through, got %#v", got)
	}
	if got.Entrypoints["readme"] != "README.md" {
		t.Fatalf("expected readme entrypoint, got %#v", got.Entrypoints)
	}
	if strings.Join(got.Paths.Authoritative, ",") != "docs/,src/" {
		t.Fatalf("expected authoritative defaults, got %v", got.Paths.Authoritative)
	}
	if strings.Join(got.Paths.LowValue, ",") != "generated/,vendor/" {
		t.Fatalf("expected low-value defaults, got %v", got.Paths.LowValue)
	}
	if fallbackMetadataPath(repoPath, "") != filepath.Join(repoPath, repometa.PreferredFilename) {
		t.Fatalf("expected preferred fallback path, got %q", fallbackMetadataPath(repoPath, ""))
	}
	if humanizeRepoName("repo-name_here") != "Repo Name Here" {
		t.Fatalf("expected humanized repo name, got %q", humanizeRepoName("repo-name_here"))
	}
}

func TestGuessRepoMetadataDefaultsClonesExistingMetadata(t *testing.T) {
	t.Parallel()

	existing := &model.RepoMetadata{
		Name:        "Existing",
		RepoID:      "github.com/acme/repo",
		Labels:      map[string]string{"team": "platform"},
		Entrypoints: map[string]string{"readme": "README.md"},
		Paths: model.RepoMetadataPaths{
			Authoritative: []string{"docs/"},
			LowValue:      []string{"generated/"},
		},
		Provides:     []string{"api"},
		RelatedRepos: []model.RepoMetadataRelatedRepo{{RepoID: "acme/docs", Relationship: "documents"}},
	}

	got := guessRepoMetadataDefaults(registry.Entry{RepoID: existing.RepoID, Path: t.TempDir()}, existing)
	got.Labels["team"] = "changed"
	got.Entrypoints["readme"] = "HACKED.md"
	got.Paths.Authoritative[0] = "src/"
	got.Paths.LowValue[0] = "dist/"
	got.Provides[0] = "worker"
	got.RelatedRepos[0].RepoID = "acme/other"

	if existing.Labels["team"] != "platform" {
		t.Fatalf("expected labels clone, got %#v", existing.Labels)
	}
	if existing.Entrypoints["readme"] != "README.md" {
		t.Fatalf("expected entrypoints clone, got %#v", existing.Entrypoints)
	}
	if existing.Paths.Authoritative[0] != "docs/" || existing.Paths.LowValue[0] != "generated/" {
		t.Fatalf("expected paths clone, got %#v", existing.Paths)
	}
	if existing.Provides[0] != "api" {
		t.Fatalf("expected provides clone, got %#v", existing.Provides)
	}
	if existing.RelatedRepos[0].RepoID != "acme/docs" {
		t.Fatalf("expected related repos slice clone, got %#v", existing.RelatedRepos)
	}
}

func TestUnifiedDiffEmptyWhenIdentical(t *testing.T) {
	t.Parallel()
	content := []byte("name: Repo\nrepoID: acme/repo\n")
	got, err := unifiedDiff(content, content, "/repo/.repokeeper-repo.yaml")
	if err != nil {
		t.Fatalf("unified diff: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty diff for identical content, got %q", got)
	}
}

func TestUnifiedDiffReportsChanges(t *testing.T) {
	t.Parallel()
	current := []byte("name: Old Name\nrepoID: acme/repo\n")
	proposed := []byte("name: New Name\nrepoID: acme/repo\n")
	got, err := unifiedDiff(current, proposed, "/repo/.repokeeper-repo.yaml")
	if err != nil {
		t.Fatalf("unified diff: %v", err)
	}
	for _, want := range []string{"(current)", "(proposed)", "-name: Old Name", "+name: New Name"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diff to contain %q, got:\n%s", want, got)
		}
	}
}

func TestWriteMetadataPreviewNewFile(t *testing.T) {
	t.Parallel()
	proposed, err := repometa.Render(&model.RepoMetadata{Name: "Repo", RepoID: "acme/repo"})
	if err != nil {
		t.Fatalf("render proposal: %v", err)
	}
	var out bytes.Buffer
	target := filepath.Join(t.TempDir(), repometa.PreferredFilename)
	if err := writeMetadataPreview(&out, target, proposed, repometa.ErrNotFound); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "# Repo metadata preview") {
		t.Fatalf("expected full preview header, got:\n%s", got)
	}
	if !strings.Contains(got, string(proposed)) {
		t.Fatalf("expected full proposed file in preview, got:\n%s", got)
	}
}

func TestWriteMetadataPreviewUnchanged(t *testing.T) {
	t.Parallel()
	proposed, err := repometa.Render(&model.RepoMetadata{Name: "Repo", RepoID: "acme/repo"})
	if err != nil {
		t.Fatalf("render proposal: %v", err)
	}
	target := filepath.Join(t.TempDir(), repometa.PreferredFilename)
	if err := os.WriteFile(target, proposed, 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	var out bytes.Buffer
	if err := writeMetadataPreview(&out, target, proposed, nil); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "# Repo metadata unchanged") {
		t.Fatalf("expected unchanged note, got:\n%s", got)
	}
	if strings.Contains(got, "@@") {
		t.Fatalf("did not expect a diff body for unchanged content, got:\n%s", got)
	}
}

func TestWriteMetadataPreviewShowsDiff(t *testing.T) {
	t.Parallel()
	current, err := repometa.Render(&model.RepoMetadata{Name: "Old Name", RepoID: "acme/repo"})
	if err != nil {
		t.Fatalf("render current: %v", err)
	}
	proposed, err := repometa.Render(&model.RepoMetadata{Name: "New Name", RepoID: "acme/repo"})
	if err != nil {
		t.Fatalf("render proposal: %v", err)
	}
	target := filepath.Join(t.TempDir(), repometa.PreferredFilename)
	if err := os.WriteFile(target, current, 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	var out bytes.Buffer
	if err := writeMetadataPreview(&out, target, proposed, nil); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "# Repo metadata diff") {
		t.Fatalf("expected diff header, got:\n%s", got)
	}
	for _, want := range []string{"-name: Old Name", "+name: New Name"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diff to contain %q, got:\n%s", want, got)
		}
	}
}

func TestWriteMetadataPreviewDiffsUnparseableExisting(t *testing.T) {
	t.Parallel()
	proposed, err := repometa.Render(&model.RepoMetadata{Name: "Repo", RepoID: "acme/repo"})
	if err != nil {
		t.Fatalf("render proposal: %v", err)
	}
	target := filepath.Join(t.TempDir(), repometa.PreferredFilename)
	// Existing on-disk file that is present but not valid metadata.
	if err := os.WriteFile(target, []byte("name: [unterminated\n"), 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	var out bytes.Buffer
	loadErr := errors.New("parse .repokeeper-repo.yaml: yaml: bad indentation")
	if err := writeMetadataPreview(&out, target, proposed, loadErr); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	got := out.String()
	for _, want := range []string{"# Repo metadata diff", "could not be parsed", "+name: Repo"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diff of unparseable existing file to contain %q, got:\n%s", want, got)
		}
	}
}
