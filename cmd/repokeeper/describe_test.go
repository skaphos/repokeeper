package repokeeper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

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

	prevConfig := flagConfig
	flagConfig = cfgPath
	defer func() { flagConfig = prevConfig }()

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
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

	prevConfig := flagConfig
	flagConfig = cfgPath
	defer func() { flagConfig = prevConfig }()

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
