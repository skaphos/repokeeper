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
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func TestImportTargetRelativePath(t *testing.T) {
	entry := registry.Entry{
		RepoID: "github.com/org/repo-a",
		Path:   "/source/root/team/repo-a",
	}

	if got := importTargetRelativePath(entry, "/source/root"); got != "team/repo-a" {
		t.Fatalf("expected root-relative target, got %q", got)
	}

	if got := importTargetRelativePath(entry, "/other/workspace"); got != "source/root/team/repo-a" {
		t.Fatalf("expected absolute-path fallback preserving layout, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-z", Path: ""}
	if got := importTargetRelativePath(entry, ""); got != "repo-z" {
		t.Fatalf("expected repo-id fallback, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-r", Path: "team/repo-r"}
	if got := importTargetRelativePath(entry, "/source/root"); got != "team/repo-r" {
		t.Fatalf("expected already-relative path preserved, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-w", Path: `team\repo-w`}
	if got := importTargetRelativePath(entry, "/source/root"); got != "team/repo-w" {
		t.Fatalf("expected windows separators normalized, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-x", Path: "/mnt/cache/sdp/team/repo-x"}
	if got := importTargetRelativePath(entry, "/home/user/sdp"); got != "team/repo-x" {
		t.Fatalf("expected root-basename fallback preserving project-relative suffix, got %q", got)
	}
}

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

func TestCloneImportedReposReportsSpecificTargetConflicts(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/repo-a",
					Path:      "/source/root/team/repo-a",
					RemoteURL: "git@github.com:org/repo-a.git",
					Branch:    "main",
					Status:    registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Root: "/source/root"}
	target := filepath.Join(cwd, "team", "repo-a")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := cloneImportedRepos(cmd, cfg, bundle, cwd, false)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "import target conflicts detected") {
		t.Fatalf("expected conflict summary error, got: %v", err)
	}
	if !strings.Contains(err.Error(), target) {
		t.Fatalf("expected target path in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "github.com/org/repo-a") {
		t.Fatalf("expected repo id in error, got: %v", err)
	}
}

func TestCloneImportedReposSkipsLocalEntriesWithoutRemoteURL(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:   "local:/source/root/team/repo-a",
					Path:     "/source/root/team/repo-a",
					LastSeen: time.Now(),
					Status:   registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Root: "/source/root"}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := cloneImportedRepos(cmd, cfg, bundle, cwd, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Registry.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(cfg.Registry.Entries))
	}
	entry := cfg.Registry.Entries[0]
	if entry.Status != registry.StatusMissing {
		t.Fatalf("expected local entry to be missing after skip, got %q", entry.Status)
	}
	if got, want := entry.Path, filepath.Join(cwd, "team", "repo-a"); got != want {
		t.Fatalf("expected rewritten path %q, got %q", want, got)
	}
}

func TestCloneImportedReposSkipsNonLocalMissingRemoteURL(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID: "github.com/org/repo-a",
					Path:   "/source/root/team/repo-a",
					Status: registry.StatusPresent,
				},
			},
		},
	}
	bundle := exportBundle{Config: config.Config{}}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := cloneImportedRepos(cmd, cfg, bundle, cwd, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Registry.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(cfg.Registry.Entries))
	}
	entry := cfg.Registry.Entries[0]
	if entry.Path == "" {
		t.Fatal("expected skipped entry path to be rewritten under import root")
	}
}

func TestCloneImportedReposSkipsMissingAndNoUpstreamEntries(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/missing",
					Path:      "/source/root/team/missing",
					RemoteURL: "git@github.com:org/missing.git",
					Branch:    "main",
					Status:    registry.StatusMissing,
				},
				{
					RepoID:    "github.com/org/no-upstream",
					Path:      "/source/root/team/no-upstream",
					RemoteURL: "git@github.com:org/no-upstream.git",
					Status:    registry.StatusPresent,
				},
			},
		},
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := cloneImportedRepos(cmd, cfg, exportBundle{Root: "/source/root"}, cwd, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := cfg.Registry.FindByRepoID("github.com/org/missing"); got == nil || got.Status != registry.StatusMissing {
		t.Fatalf("expected missing repo to stay missing, got %+v", got)
	}
	if got := cfg.Registry.FindByRepoID("github.com/org/no-upstream"); got == nil || got.Path != filepath.Join(cwd, "team", "no-upstream") {
		t.Fatalf("expected no-upstream repo path rewrite under import root, got %+v", got)
	} else if got.Status != registry.StatusMissing {
		t.Fatalf("expected no-upstream repo to remain missing until explicitly repaired, got %+v", got)
	}
}

func TestCloneImportedReposMarksFailedCloneAsMissing(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/fails",
					Path:      "/source/root/team/fails",
					RemoteURL: "/does/not/exist",
					Branch:    "main",
					Status:    registry.StatusPresent,
				},
			},
		},
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	failures, err := cloneImportedReposWithProgress(cmd, cfg, exportBundle{Root: "/source/root"}, cwd, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("expected one clone failure, got %d", len(failures))
	}
	if got, want := failures[0].ErrorClass, "unknown"; got != want {
		t.Fatalf("expected classified error %q, got %q", want, got)
	}
	if got, want := failures[0].Error, "import-clone-failed"; got != want {
		t.Fatalf("expected normalized clone error %q, got %q", want, got)
	}
	entry := cfg.Registry.FindByRepoID("github.com/org/fails")
	if entry == nil {
		t.Fatal("expected registry entry after failed clone")
	}
	if entry.Status != registry.StatusMissing {
		t.Fatalf("expected failed clone to mark entry missing, got %q", entry.Status)
	}
}

func TestDropIgnoredImportEntriesRemovesIgnoredTargets(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		IgnoredPaths: []string{filepath.Join(cwd, "team", "repo-a")},
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID: "github.com/org/repo-a",
					Path:   "/source/root/team/repo-a",
					Status: registry.StatusPresent,
				},
				{
					RepoID: "github.com/org/repo-b",
					Path:   "/source/root/team/repo-b",
					Status: registry.StatusPresent,
				},
			},
		},
	}

	dropIgnoredImportEntries(cfg, exportBundle{Root: "/source/root"}, cwd)

	if got := cfg.Registry.FindByRepoID("github.com/org/repo-a"); got != nil {
		t.Fatalf("expected ignored import entry to be removed, got %+v", got)
	}
	if got := cfg.Registry.FindByRepoID("github.com/org/repo-b"); got == nil {
		t.Fatal("expected non-ignored import entry to be retained")
	}
}

func TestImportCommandArgsValidation(t *testing.T) {
	if importCmd.Args == nil {
		t.Fatal("expected import command args validator")
	}
	if err := importCmd.Args(importCmd, []string{"a.yaml", "b.yaml"}); err == nil {
		t.Fatal("expected too-many-args validation error")
	}
	if err := importCmd.Args(importCmd, []string{}); err != nil {
		t.Fatalf("expected zero args to be valid (stdin), got: %v", err)
	}
	if err := importCmd.Args(importCmd, []string{"bundle.yaml"}); err != nil {
		t.Fatalf("expected one arg to be valid, got: %v", err)
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

func TestCloneImportedReposRejectsUnsafeTargets(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/repo-a",
					Path:      "..",
					RemoteURL: "git@github.com:org/repo-a.git",
					Status:    registry.StatusPresent,
				},
			},
		},
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := cloneImportedRepos(cmd, cfg, exportBundle{}, cwd, false)
	if err == nil || !strings.Contains(err.Error(), "refusing to clone outside current directory") {
		t.Fatalf("expected path traversal protection error, got: %v", err)
	}
}

func TestCloneImportedReposRejectsDuplicateTargets(t *testing.T) {
	cwd := t.TempDir()
	cfg := &config.Config{
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "github.com/org/repo-a",
					Path:      "repo",
					RemoteURL: "git@github.com:org/repo-a.git",
					Status:    registry.StatusPresent,
				},
				{
					RepoID:    "github.com/org/repo-b",
					Path:      "./repo",
					RemoteURL: "git@github.com:org/repo-b.git",
					Status:    registry.StatusPresent,
				},
			},
		},
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := cloneImportedRepos(cmd, cfg, exportBundle{}, cwd, false)
	if err == nil || !strings.Contains(err.Error(), "multiple repos resolve to same target path") {
		t.Fatalf("expected duplicate target error, got: %v", err)
	}
}

func TestCloneImportedReposNoopWithoutRegistry(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := cloneImportedRepos(cmd, nil, exportBundle{}, t.TempDir(), false); err != nil {
		t.Fatalf("expected nil cfg to no-op, got: %v", err)
	}
	if err := cloneImportedRepos(cmd, &config.Config{}, exportBundle{}, t.TempDir(), false); err != nil {
		t.Fatalf("expected nil registry to no-op, got: %v", err)
	}
	if err := cloneImportedRepos(cmd, &config.Config{Registry: &registry.Registry{}}, exportBundle{}, t.TempDir(), false); err != nil {
		t.Fatalf("expected empty registry to no-op, got: %v", err)
	}
}

func TestSetRegistryEntryByRepoID(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/r1"},
		},
	}

	setRegistryEntryByRepoID(reg, registry.Entry{RepoID: "r1", Path: "/updated"})
	if got := reg.Entries[0].Path; got != "/updated" {
		t.Fatalf("expected existing entry update, got %q", got)
	}

	setRegistryEntryByRepoID(reg, registry.Entry{RepoID: "r2", Path: "/r2"})
	if len(reg.Entries) != 2 {
		t.Fatalf("expected append for new repo id, got len=%d", len(reg.Entries))
	}

	// Ensure nil registry is safe.
	setRegistryEntryByRepoID(nil, registry.Entry{RepoID: "ignored"})
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

func TestImportCommandRunEFileOnlyFromStdin(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	if err := rootCmd.PersistentFlags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("config", prevConfig) }()

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	bundle := exportBundle{
		Version: 1,
		Config:  config.DefaultConfig(),
		Registry: &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "github.com/org/repo", Path: "/tmp/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent},
			},
		},
	}
	data, err := yaml.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	in := bytes.NewBuffer(data)
	importCmd.SetIn(in)
	importCmd.SetContext(context.Background())
	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "true")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()
	_ = importCmd.Flags().Set("force", "true")
	_ = importCmd.Flags().Set("file-only", "true")
	_ = importCmd.Flags().Set("include-registry", "true")
	_ = importCmd.Flags().Set("preserve-registry-path", "true")

	if err := importCmd.RunE(importCmd, []string{"-"}); err != nil {
		t.Fatalf("import run failed: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load imported config: %v", err)
	}
	if cfg.Registry != nil {
		t.Fatalf("expected file-only import to omit registry, got %+v", cfg.Registry)
	}
}

func TestImportCommandRunERejectsBlankBundleArg(t *testing.T) {
	importCmd.SetContext(context.Background())
	err := importCmd.RunE(importCmd, []string{"   "})
	if err == nil || !strings.Contains(err.Error(), "bundle-file cannot be empty") {
		t.Fatalf("expected blank bundle arg error, got: %v", err)
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

func TestImportCommandRunERequiresForceWhenConfigExists(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	bundle := exportBundle{Version: 1, Config: config.DefaultConfig()}
	data, err := yaml.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	importCmd.SetIn(bytes.NewBuffer(data))
	importCmd.SetContext(context.Background())
	_ = importCmd.Flags().Set("force", "false")
	_ = importCmd.Flags().Set("mode", "replace")
	_ = importCmd.Flags().Set("file-only", "true")

	err = importCmd.RunE(importCmd, []string{"-"})
	if err == nil || !strings.Contains(err.Error(), "config already exists") {
		t.Fatalf("expected force-required error, got: %v", err)
	}
}

func TestImportCommandRunEMergeDoesNotRequireForceWhenConfigExists(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	bundle := exportBundle{Version: 1, Config: config.DefaultConfig()}
	data, err := yaml.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	importCmd.SetIn(bytes.NewBuffer(data))
	importCmd.SetContext(context.Background())
	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "true")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()
	_ = importCmd.Flags().Set("force", "false")
	_ = importCmd.Flags().Set("mode", "merge")
	_ = importCmd.Flags().Set("file-only", "true")

	if err := importCmd.RunE(importCmd, []string{"-"}); err != nil {
		t.Fatalf("expected merge import without force to succeed, got: %v", err)
	}
}

func TestImportCommandDefaultsToLocalConfigPath(t *testing.T) {
	tmp := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	_ = rootCmd.PersistentFlags().Set("config", "")
	defer func() { _ = rootCmd.PersistentFlags().Set("config", prevConfig) }()

	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "true")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()

	bundle := exportBundle{Version: 1, Config: config.DefaultConfig()}
	data, err := yaml.Marshal(&bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	importCmd.SetIn(bytes.NewBuffer(data))
	importCmd.SetContext(context.Background())
	_ = importCmd.Flags().Set("mode", "merge")
	_ = importCmd.Flags().Set("file-only", "true")

	if err := importCmd.RunE(importCmd, []string{"-"}); err != nil {
		t.Fatalf("import run failed: %v", err)
	}

	localCfg := filepath.Join(tmp, ".repokeeper.yaml")
	if _, err := os.Stat(localCfg); err != nil {
		t.Fatalf("expected local config at %s: %v", localCfg, err)
	}
}

func TestMergeImportedRegistryPolicyTable(t *testing.T) {
	mkCfg := func() *config.Config {
		return &config.Config{
			Registry: &registry.Registry{
				Entries: []registry.Entry{
					{RepoID: "github.com/org/repo", Path: "/local/repo", RemoteURL: "git@github.com:org/repo.git", Branch: "main", Type: "checkout", Status: registry.StatusPresent},
				},
			},
		}
	}
	incoming := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo", Path: "/bundle/repo", RemoteURL: "git@github.com:org/repo.git", Branch: "feature/a", Type: "checkout", Status: registry.StatusPresent},
			{RepoID: "github.com/org/new", Path: "/bundle/new", RemoteURL: "git@github.com:org/new.git", Branch: "main", Type: "checkout", Status: registry.StatusPresent},
		},
	}

	cfg := mkCfg()
	mergeImportedRegistry(cfg, importModeMerge, true, incoming, importConflictPolicyBundle)
	if got := cfg.Registry.FindByRepoID("github.com/org/repo").Path; got != "/bundle/repo" {
		t.Fatalf("expected bundle policy to overwrite path, got %q", got)
	}
	if got := cfg.Registry.FindByRepoID("github.com/org/new"); got == nil {
		t.Fatal("expected new repo appended in merge mode")
	}

	cfg = mkCfg()
	mergeImportedRegistry(cfg, importModeMerge, true, incoming, importConflictPolicyLocal)
	if got := cfg.Registry.FindByRepoID("github.com/org/repo").Path; got != "/local/repo" {
		t.Fatalf("expected local policy to keep local path, got %q", got)
	}

	cfg = mkCfg()
	mergeImportedRegistry(cfg, importModeMerge, true, incoming, importConflictPolicySkip)
	if got := cfg.Registry.FindByRepoID("github.com/org/repo").Path; got != "/local/repo" {
		t.Fatalf("expected skip policy to keep local path, got %q", got)
	}
}

func TestParseImportModeAndConflictPolicy(t *testing.T) {
	if mode, err := parseImportMode("merge"); err != nil || mode != importModeMerge {
		t.Fatalf("expected merge mode, got %q (%v)", mode, err)
	}
	if mode, err := parseImportMode("replace"); err != nil || mode != importModeReplace {
		t.Fatalf("expected replace mode, got %q (%v)", mode, err)
	}
	if _, err := parseImportMode("weird"); err == nil {
		t.Fatal("expected invalid mode to error")
	}

	if policy, err := parseImportConflictPolicy("bundle"); err != nil || policy != importConflictPolicyBundle {
		t.Fatalf("expected bundle policy, got %q (%v)", policy, err)
	}
	if policy, err := parseImportConflictPolicy("local"); err != nil || policy != importConflictPolicyLocal {
		t.Fatalf("expected local policy, got %q (%v)", policy, err)
	}
	if _, err := parseImportConflictPolicy("oops"); err == nil {
		t.Fatal("expected invalid policy to error")
	}
}

func TestSelectMergeCloneEntriesPolicy(t *testing.T) {
	local := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: "/local/a", RemoteURL: "git@github.com:org/repo-a.git", Branch: "main"},
			{RepoID: "github.com/org/repo-b", Path: "/local/b", RemoteURL: "git@github.com:org/repo-b.git", Branch: "main"},
		},
	}
	bundled := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: "/bundle/a", RemoteURL: "git@github.com:org/repo-a.git", Branch: "feature/a"},
			{RepoID: "github.com/org/repo-b", Path: "/local/b", RemoteURL: "git@github.com:org/repo-b.git", Branch: "main"},
			{RepoID: "github.com/org/repo-c", Path: "/bundle/c", RemoteURL: "git@github.com:org/repo-c.git", Branch: "main"},
		},
	}

	skip := selectMergeCloneEntries(local, bundled, importConflictPolicySkip)
	if len(skip) != 1 || skip[0].RepoID != "github.com/org/repo-c" {
		t.Fatalf("expected skip policy to clone only new repo, got %+v", skip)
	}

	localPolicy := selectMergeCloneEntries(local, bundled, importConflictPolicyLocal)
	if len(localPolicy) != 1 || localPolicy[0].RepoID != "github.com/org/repo-c" {
		t.Fatalf("expected local policy to clone only new repo, got %+v", localPolicy)
	}

	bundlePolicy := selectMergeCloneEntries(local, bundled, importConflictPolicyBundle)
	if len(bundlePolicy) != 2 {
		t.Fatalf("expected bundle policy to clone new+conflicted repos, got %+v", bundlePolicy)
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
