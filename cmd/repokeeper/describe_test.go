// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func withConfigFlag(t *testing.T, cfgPath string) func() {
	t.Helper()
	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	if err := rootCmd.PersistentFlags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}
	return func() {
		_ = rootCmd.PersistentFlags().Set("config", prevConfig)
	}
}

func TestDescribeRepoSubcommandExists(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"describe", "repo", "example"})
	if err != nil {
		t.Fatalf("expected describe repo command to resolve, got error: %v", err)
	}
	if cmd == nil || cmd.Name() != "repo" {
		t.Fatalf("expected repo subcommand, got %#v", cmd)
	}
}

func TestSelectRegistryEntryForDescribe(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "github.com/org/repo-a", Path: "/tmp/work/repo-a"},
		{RepoID: "github.com/org/repo-b", Path: "/tmp/root/repo-b"},
	}

	byID, err := selectRegistryEntryForDescribe(entries, "github.com/org/repo-a", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected id selector to match, got error: %v", err)
	}
	if byID.Path != "/tmp/work/repo-a" {
		t.Fatalf("unexpected id match: %#v", byID)
	}

	byCWD, err := selectRegistryEntryForDescribe(entries, "repo-a", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected cwd-relative selector to match, got error: %v", err)
	}
	if byCWD.RepoID != "github.com/org/repo-a" {
		t.Fatalf("unexpected cwd match: %#v", byCWD)
	}

	byRoot, err := selectRegistryEntryForDescribe(entries, "repo-b", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected root-relative selector to match, got error: %v", err)
	}
	if byRoot.RepoID != "github.com/org/repo-b" {
		t.Fatalf("unexpected root match: %#v", byRoot)
	}
}

func TestRunDescribeRepoWithMissingRegistryEntry(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	regPath := filepath.Join(tmp, "registry.yaml")
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/repo-missing",
				Path:     filepath.Join(tmp, "missing-repo"),
				Status:   registry.StatusMissing,
				LastSeen: time.Now(),
			},
		},
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(out)
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")
	if err := cmd.Flags().Set("registry", regPath); err != nil {
		t.Fatalf("set registry flag: %v", err)
	}
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format flag: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	if err := runDescribeRepo(cmd, []string{"github.com/org/repo-missing"}); err != nil {
		t.Fatalf("runDescribeRepo: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\"repo_id\": \"github.com/org/repo-missing\"") {
		t.Fatalf("expected repo id in json output, got: %q", got)
	}
	if !strings.Contains(got, "\"error\": \"path missing\"") {
		t.Fatalf("expected missing-path error in json output, got: %q", got)
	}
}

func TestRunDescribeRepoUnsupportedFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	regPath := filepath.Join(tmp, "registry.yaml")
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo", Path: filepath.Join(tmp, "repo"), Status: registry.StatusMissing, LastSeen: time.Now()},
		},
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")
	_ = cmd.Flags().Set("registry", regPath)
	_ = cmd.Flags().Set("format", "yaml")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	err = runDescribeRepo(cmd, []string{"github.com/org/repo"})
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got: %v", err)
	}
}

func TestSelectRegistryEntryForDescribeErrorsAndHelpers(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "github.com/org/repo-a", Path: "/tmp/work/repo-a"},
		{RepoID: "github.com/org/repo-b", Path: "/tmp/root/repo-b"},
		{RepoID: "github.com/org/repo-c", Path: "/tmp/work/dup"},
		{RepoID: "github.com/org/repo-d", Path: "/tmp/root/dup"},
	}

	if _, err := selectRegistryEntryForDescribe(entries, "   ", "/tmp/work", []string{"/tmp/root"}); err == nil {
		t.Fatal("expected empty selector to error")
	}
	if _, err := selectRegistryEntryForDescribe(entries, "missing", "/tmp/work", []string{"/tmp/root"}); err == nil {
		t.Fatal("expected unknown selector to error")
	}
	if _, err := selectRegistryEntryForDescribe(entries, "dup", "/tmp/work", []string{"/tmp/root"}); err == nil {
		t.Fatal("expected ambiguous selector to error")
	}

	if got, ok := canonicalPathForMatch("   "); ok || got != "" {
		t.Fatalf("expected blank canonical path to fail, got %q, %t", got, ok)
	}

	if pathWithinBase("/tmp/root/repo", "/tmp/root") != true {
		t.Fatal("expected path to be within base")
	}
	if pathWithinBase("/tmp/root/../other/repo", "/tmp/root") != false {
		t.Fatal("expected normalized traversal path to be out of base")
	}
	if pathWithinBase("/tmp/other/repo", "/tmp/root") != false {
		t.Fatal("expected outside path to be out of base")
	}

	left := filepath.Clean("/tmp/work/repo-a")
	right := filepath.Clean("/tmp/work/repo-a")
	if !samePathForMatch(left, right) {
		t.Fatalf("expected same paths to match: %q vs %q", left, right)
	}
}

func TestSelectRegistryEntryForDescribeRejectsPathTraversal(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "github.com/org/repo-a", Path: "/tmp/other/repo-a"},
	}
	if _, err := selectRegistryEntryForDescribe(entries, "../other/repo-a", "/tmp/root/work", []string{"/tmp/root"}); err == nil {
		t.Fatal("expected traversal selector to be rejected")
	}
}

func TestRunDescribeRepoErrorsForMissingConfig(t *testing.T) {
	tmp := t.TempDir()
	restoreConfig := withConfigFlag(t, filepath.Join(tmp, ".repokeeper.yaml"))
	defer restoreConfig()

	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	err = runDescribeRepo(cmd, []string{"github.com/org/repo"})
	if err == nil {
		t.Fatal("expected missing config error")
	}
}

func TestRunDescribeRepoErrorsWithoutRegistry(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = nil
	cfg.RegistryPath = ""
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	err = runDescribeRepo(cmd, []string{"github.com/org/repo"})
	if err == nil || !strings.Contains(err.Error(), "registry not found") {
		t.Fatalf("expected missing registry error, got: %v", err)
	}
}

func TestRunDescribeRepoRegistryOverrideLoadError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	regPath := filepath.Join(tmp, "registry.yaml")
	if err := os.WriteFile(regPath, []byte("not: [valid"), 0o644); err != nil {
		t.Fatalf("write invalid registry: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")
	_ = cmd.Flags().Set("registry", regPath)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	err = runDescribeRepo(cmd, []string{"github.com/org/repo"})
	if err == nil {
		t.Fatal("expected registry load error")
	}
}

func TestRunDescribeRepoSelectorNotFound(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	regPath := filepath.Join(tmp, "registry.yaml")
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-a", Path: filepath.Join(tmp, "repo-a"), Status: registry.StatusMissing},
		},
	}
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")
	_ = cmd.Flags().Set("registry", regPath)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	err = runDescribeRepo(cmd, []string{"github.com/org/unknown"})
	if err == nil || !strings.Contains(err.Error(), "repo not found for selector") {
		t.Fatalf("expected selector-not-found error, got: %v", err)
	}
}

func TestRunDescribeRepoInspectErrorPopulatesOutput(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID: "github.com/org/repo-present",
				Path:   filepath.Join(tmp, "not-a-repo"),
				Status: registry.StatusPresent,
			},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	restoreConfig := withConfigFlag(t, cfgPath)
	defer restoreConfig()

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(out)
	cmd.Flags().String("registry", "", "")
	cmd.Flags().String("format", "table", "")
	_ = cmd.Flags().Set("format", "json")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	if err := runDescribeRepo(cmd, []string{"github.com/org/repo-present"}); err != nil {
		t.Fatalf("expected inspect errors to be reported in output, got: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\"error_class\":") || !strings.Contains(got, "\"error\":") {
		t.Fatalf("expected error fields in output, got: %q", got)
	}
}
