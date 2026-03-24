// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
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
