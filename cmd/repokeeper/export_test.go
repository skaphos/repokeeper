// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"go.yaml.in/yaml/v3"
)

func TestPrepareRegistryForExportStripsTimestampsAndRelativizesPaths(t *testing.T) {
	now := time.Now()
	reg := &registry.Registry{
		UpdatedAt: now,
		Entries: []registry.Entry{
			{
				RepoID:    "github.com/org/repo-a",
				Path:      "/source/root/team/repo-a",
				RemoteURL: "git@github.com:org/repo-a.git",
				LastSeen:  now,
				Status:    registry.StatusPresent,
			},
			{
				RepoID:    "github.com/org/repo-missing",
				Path:      "/source/root/team/repo-missing",
				RemoteURL: "git@github.com:org/repo-missing.git",
				LastSeen:  now,
				Status:    registry.StatusMissing,
			},
			{
				RepoID:    "github.com/org/repo-moved",
				Path:      "/source/root/team/repo-moved",
				RemoteURL: "git@github.com:org/repo-moved.git",
				LastSeen:  now,
				Status:    registry.StatusMoved,
			},
		},
	}

	got := prepareRegistryForExport(reg, "/source/root")
	if got == nil || len(got.Entries) != 1 {
		t.Fatalf("expected one exported entry, got %+v", got)
	}
	if !got.UpdatedAt.IsZero() {
		t.Fatalf("expected updated_at stripped, got %v", got.UpdatedAt)
	}
	if !got.Entries[0].LastSeen.IsZero() {
		t.Fatalf("expected last_seen stripped, got %v", got.Entries[0].LastSeen)
	}
	if got.Entries[0].Path != "team/repo-a" {
		t.Fatalf("expected relative export path, got %q", got.Entries[0].Path)
	}
}

func TestPrepareRegistryForExportDropsNonPresentEntries(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r-present", Path: "/repos/present", Status: registry.StatusPresent},
			{RepoID: "r-missing", Path: "/repos/missing", Status: registry.StatusMissing},
			{RepoID: "r-moved", Path: "/repos/moved", Status: registry.StatusMoved},
		},
	}

	got := prepareRegistryForExport(reg, "/repos")
	if got == nil {
		t.Fatal("expected non-nil exported registry")
	}
	if len(got.Entries) != 1 {
		t.Fatalf("expected only present entries to export, got %+v", got.Entries)
	}
	if got.Entries[0].RepoID != "r-present" {
		t.Fatalf("expected present repo exported, got %+v", got.Entries[0])
	}
}

func TestInferRegistrySharedRootIgnoresNonPresentEntries(t *testing.T) {
	presentRoot := filepath.Join(string(filepath.Separator), "workspace", "repos", "team", "repo-a")
	missingPath := filepath.Join(string(filepath.Separator), "elsewhere", "legacy", "repo-gone")

	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r-present", Path: presentRoot, Status: registry.StatusPresent},
			{RepoID: "r-missing", Path: missingPath, Status: registry.StatusMissing},
		},
	}

	got := inferRegistrySharedRoot(reg)
	want := filepath.Dir(presentRoot)
	if got != want {
		t.Fatalf("expected inferred root %q, got %q", want, got)
	}
}

func TestPopulateExportBranches(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/repos/r1", Status: registry.StatusPresent},
			{RepoID: "r2", Path: "/repos/r2", Status: registry.StatusPresent, Branch: "keep-me"},
			{RepoID: "r3", Path: "/repos/r3", Status: registry.StatusMissing, Branch: "stale"},
			{RepoID: "r4", Path: "/repos/r4", Type: "mirror", Status: registry.StatusPresent, Branch: "mirror-branch"},
		},
	}

	populateExportBranches(context.Background(), reg, func(_ context.Context, path string) (model.Head, error) {
		switch path {
		case "/repos/r1":
			return model.Head{Branch: "feature/a"}, nil
		case "/repos/r2":
			return model.Head{}, errors.New("head failed")
		case "/repos/r4":
			return model.Head{Branch: "should-not-apply"}, nil
		default:
			return model.Head{}, nil
		}
	}, nil)

	if got, want := reg.Entries[0].Branch, "feature/a"; got != want {
		t.Fatalf("expected branch %q, got %q", want, got)
	}
	if got, want := reg.Entries[1].Branch, "keep-me"; got != want {
		t.Fatalf("expected existing branch %q to remain, got %q", want, got)
	}
	if got, want := reg.Entries[2].Branch, "stale"; got != want {
		t.Fatalf("expected missing entry branch %q to remain, got %q", want, got)
	}
	if got, want := reg.Entries[3].Branch, "mirror-branch"; got != want {
		t.Fatalf("expected mirror branch %q to remain, got %q", want, got)
	}
}

func TestPopulateExportBranchesClearsNoUpstreamBranches(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/repos/r1", Status: registry.StatusPresent, Branch: "old"},
			{RepoID: "r2", Path: "/repos/r2", Status: registry.StatusPresent, Branch: "keep"},
		},
	}

	populateExportBranches(
		context.Background(),
		reg,
		func(_ context.Context, path string) (model.Head, error) {
			switch path {
			case "/repos/r1":
				return model.Head{Branch: "main"}, nil
			case "/repos/r2":
				return model.Head{Branch: "release"}, nil
			default:
				return model.Head{}, nil
			}
		},
		func(_ context.Context, path string) (model.Tracking, error) {
			switch path {
			case "/repos/r1":
				return model.Tracking{}, nil
			case "/repos/r2":
				return model.Tracking{Upstream: "origin/release"}, nil
			default:
				return model.Tracking{}, nil
			}
		},
	)

	if got := reg.Entries[0].Branch; got != "" {
		t.Fatalf("expected branch to be cleared when upstream missing, got %q", got)
	}
	if got, want := reg.Entries[1].Branch, "release"; got != want {
		t.Fatalf("expected branch %q for upstream-tracked repo, got %q", want, got)
	}
}

func TestPopulateExportBranchesWithTrackingErrorFallsBackToHead(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/repos/r1", Status: registry.StatusPresent, Branch: "old"},
		},
	}

	populateExportBranches(
		context.Background(),
		reg,
		func(_ context.Context, _ string) (model.Head, error) {
			return model.Head{Branch: "main"}, nil
		},
		func(_ context.Context, _ string) (model.Tracking, error) {
			return model.Tracking{}, errors.New("tracking failed")
		},
	)

	if got, want := reg.Entries[0].Branch, "main"; got != want {
		t.Fatalf("expected tracking error to keep head branch %q, got %q", want, got)
	}
}

func TestPopulateExportBranchesNoTrackingFunction(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/repos/r1", Status: registry.StatusPresent, Branch: "old"},
		},
	}

	populateExportBranches(context.Background(), reg, func(_ context.Context, _ string) (model.Head, error) {
		return model.Head{Branch: "feature/a"}, nil
	}, nil)

	if got, want := reg.Entries[0].Branch, "feature/a"; got != want {
		t.Fatalf("expected branch %q, got %q", want, got)
	}
}

func TestCloneRegistry(t *testing.T) {
	if cloneRegistry(nil) != nil {
		t.Fatal("expected nil clone for nil registry")
	}

	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/r1", Status: registry.StatusPresent},
		},
	}
	cloned := cloneRegistry(reg)
	if cloned == nil || len(cloned.Entries) != 1 {
		t.Fatalf("unexpected clone: %#v", cloned)
	}
	cloned.Entries[0].RepoID = "changed"
	if reg.Entries[0].RepoID != "r1" {
		t.Fatalf("expected deep-copied entries slice, got %#v", reg.Entries[0])
	}
}

func TestExportCommandRunEToStdoutWithoutRegistry(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	exportCmd.SetOut(out)
	exportCmd.SetContext(context.Background())
	defer exportCmd.SetOut(nil)

	_ = exportCmd.Flags().Set("include-registry", "false")
	_ = exportCmd.Flags().Set("output", "-")

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("export run failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "version: 1") || !strings.Contains(got, "config:") {
		t.Fatalf("expected exported yaml output, got: %q", got)
	}
}

func TestExportCommandRunELoadsRegistryFromRegistryPath(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	regPath := filepath.Join(tmp, "registry.yaml")

	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: filepath.Join(tmp, "repo-a"), Status: registry.StatusPresent},
		},
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Registry = nil
	cfg.RegistryPath = "registry.yaml"
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	exportCmd.SetOut(out)
	exportCmd.SetContext(context.Background())
	defer exportCmd.SetOut(nil)
	_ = exportCmd.Flags().Set("include-registry", "true")
	_ = exportCmd.Flags().Set("output", "-")

	if err := exportCmd.RunE(exportCmd, nil); err != nil {
		t.Fatalf("export run failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "repo_id: github.com/org/repo-a") {
		t.Fatalf("expected exported registry entry, got: %q", got)
	}
	if !strings.Contains(got, "path: repo-a") {
		t.Fatalf("expected root-relative exported path, got: %q", got)
	}
	if strings.Contains(got, "last_seen:") {
		t.Fatalf("did not expect last_seen in export output, got: %q", got)
	}
	if strings.Contains(got, "updated_at:") {
		t.Fatalf("did not expect updated_at in export output, got: %q", got)
	}

	var exported exportBundle
	if err := yaml.Unmarshal([]byte(got), &exported); err != nil {
		t.Fatalf("unmarshal exported bundle: %v", err)
	}
	if exported.Config.Registry != nil {
		t.Fatalf("expected config.registry omitted in export, got %+v", exported.Config.Registry)
	}
	if exported.Registry == nil || len(exported.Registry.Entries) != 1 {
		t.Fatalf("expected top-level registry in export bundle, got %+v", exported.Registry)
	}
}

func TestExportCommandRunEWritesFile(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	outputFile := filepath.Join(t.TempDir(), "bundle.yaml")
	exportCmd.SetContext(context.Background())
	_ = exportCmd.Flags().Set("include-registry", "false")
	_ = exportCmd.Flags().Set("output", "-")

	if err := exportCmd.RunE(exportCmd, []string{outputFile}); err != nil {
		t.Fatalf("export run failed: %v", err)
	}
	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("expected export file at %s: %v", outputFile, err)
	}
}
