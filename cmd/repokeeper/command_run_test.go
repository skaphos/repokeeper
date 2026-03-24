// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

func writeTestConfigAndRegistry(t *testing.T) (cfgPath string, regPath string) {
	t.Helper()
	tmp := t.TempDir()

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

	cfgPath = filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = reg
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	regPath = filepath.Join(tmp, "registry.yaml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	return cfgPath, regPath
}

func withTestConfig(t *testing.T, cfgPath string) func() {
	t.Helper()
	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	if err := rootCmd.PersistentFlags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(filepath.Dir(cfgPath)); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	return func() {
		_ = rootCmd.PersistentFlags().Set("config", prevConfig)
		_ = os.Chdir(origWD)
	}
}

func withAssumeYes(t *testing.T, yes bool) func() {
	t.Helper()
	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	if err := rootCmd.PersistentFlags().Set("yes", map[bool]string{true: "true", false: "false"}[yes]); err != nil {
		t.Fatalf("set yes flag: %v", err)
	}
	return func() {
		_ = rootCmd.PersistentFlags().Set("yes", map[bool]string{true: "true", false: "false"}[prevYes])
	}
}

func TestStatusRunEUnsupportedFormat(t *testing.T) {
	cfgPath, regPath := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)

	_ = statusCmd.Flags().Set("registry", regPath)
	_ = statusCmd.Flags().Set("format", "yaml")
	_ = statusCmd.Flags().Set("only", "all")
	_ = statusCmd.Flags().Set("selector", "")

	err := statusCmd.RunE(statusCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestStatusRunECustomColumns(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)

	_ = statusCmd.Flags().Set("format", "custom-columns=REPO:.repo_id,ERROR:.error_class")
	_ = statusCmd.Flags().Set("only", "missing")
	_ = statusCmd.Flags().Set("field-selector", "")
	_ = statusCmd.Flags().Set("selector", "")
	_ = statusCmd.Flags().Set("registry", "")
	if err := statusCmd.RunE(statusCmd, nil); err != nil {
		t.Fatalf("status custom-columns failed: %v", err)
	}
	if !strings.Contains(out.String(), "REPO") || !strings.Contains(out.String(), "ERROR") {
		t.Fatalf("expected custom-column headers, got %q", out.String())
	}
	if !strings.Contains(out.String(), "github.com/org/repo-missing") || !strings.Contains(out.String(), "missing") {
		t.Fatalf("expected custom-column data, got %q", out.String())
	}
}

func TestStatusRunELabelSelectorFilter(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/repo-prod",
				Path:     filepath.Join(tmp, "missing-prod"),
				Status:   registry.StatusMissing,
				LastSeen: time.Now(),
				Labels:   map[string]string{"env": "prod"},
			},
			{
				RepoID:   "github.com/org/repo-dev",
				Path:     filepath.Join(tmp, "missing-dev"),
				Status:   registry.StatusMissing,
				LastSeen: time.Now(),
				Labels:   map[string]string{"env": "dev"},
			},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)

	_ = statusCmd.Flags().Set("format", "json")
	_ = statusCmd.Flags().Set("only", "missing")
	_ = statusCmd.Flags().Set("field-selector", "")
	_ = statusCmd.Flags().Set("selector", "env=prod")
	_ = statusCmd.Flags().Set("registry", "")
	if err := statusCmd.RunE(statusCmd, nil); err != nil {
		t.Fatalf("status with label selector failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "repo-prod") {
		t.Fatalf("expected prod repo in output, got: %q", got)
	}
	if strings.Contains(got, "repo-dev") {
		t.Fatalf("did not expect dev repo in output, got: %q", got)
	}
}

func TestStatusRunEInvalidVCSSelection(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)
	prevFormat, _ := statusCmd.Flags().GetString("format")
	prevOnly, _ := statusCmd.Flags().GetString("only")
	prevFieldSelector, _ := statusCmd.Flags().GetString("field-selector")
	prevSelector, _ := statusCmd.Flags().GetString("selector")
	prevVCS, _ := statusCmd.Flags().GetString("vcs")
	defer func() {
		_ = statusCmd.Flags().Set("format", prevFormat)
		_ = statusCmd.Flags().Set("only", prevOnly)
		_ = statusCmd.Flags().Set("field-selector", prevFieldSelector)
		_ = statusCmd.Flags().Set("selector", prevSelector)
		_ = statusCmd.Flags().Set("vcs", prevVCS)
	}()

	_ = statusCmd.Flags().Set("format", "json")
	_ = statusCmd.Flags().Set("only", "all")
	_ = statusCmd.Flags().Set("field-selector", "")
	_ = statusCmd.Flags().Set("selector", "")
	_ = statusCmd.Flags().Set("vcs", "git,svn")
	err := statusCmd.RunE(statusCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported vcs") {
		t.Fatalf("expected unsupported vcs error, got %v", err)
	}
}

func TestSyncRunEJSONMissingFilter(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	syncCmd.SetOut(out)
	syncCmd.SetErr(errOut)
	defer syncCmd.SetOut(os.Stdout)
	defer syncCmd.SetErr(os.Stderr)

	_ = syncCmd.Flags().Set("only", "missing")
	_ = syncCmd.Flags().Set("dry-run", "true")
	_ = syncCmd.Flags().Set("yes", "true")
	_ = syncCmd.Flags().Set("format", "json")

	if err := syncCmd.RunE(syncCmd, nil); err != nil {
		t.Fatalf("sync run failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"RepoID\": \"github.com/org/repo-missing\"") {
		t.Fatalf("expected missing repo in json output, got: %q", out.String())
	}
}

func TestRepairUpstreamRunEUnsupportedFormat(t *testing.T) {
	cfgPath, regPath := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	repairUpstreamCmd.SetOut(out)
	repairUpstreamCmd.SetErr(errOut)
	defer repairUpstreamCmd.SetOut(os.Stdout)
	defer repairUpstreamCmd.SetErr(os.Stderr)

	_ = repairUpstreamCmd.Flags().Set("registry", regPath)
	_ = repairUpstreamCmd.Flags().Set("dry-run", "true")
	_ = repairUpstreamCmd.Flags().Set("format", "yaml")

	err := repairUpstreamCmd.RunE(repairUpstreamCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestSyncRunEValidationFlags(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	syncCmd.SetOut(out)
	syncCmd.SetErr(errOut)
	defer syncCmd.SetOut(os.Stdout)
	defer syncCmd.SetErr(os.Stderr)

	_ = syncCmd.Flags().Set("only", "missing")
	_ = syncCmd.Flags().Set("dry-run", "true")
	_ = syncCmd.Flags().Set("yes", "true")
	_ = syncCmd.Flags().Set("format", "json")
	_ = syncCmd.Flags().Set("update-local", "false")
	_ = syncCmd.Flags().Set("rebase-dirty", "true")

	err := syncCmd.RunE(syncCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--rebase-dirty requires --update-local") {
		t.Fatalf("expected rebase-dirty validation error, got %v", err)
	}

	_ = syncCmd.Flags().Set("rebase-dirty", "false")
	_ = syncCmd.Flags().Set("push-local", "true")
	err = syncCmd.RunE(syncCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--push-local requires --update-local") {
		t.Fatalf("expected push-local validation error, got %v", err)
	}
}

func TestSyncRunEUnsupportedFormat(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	syncCmd.SetOut(out)
	syncCmd.SetErr(errOut)
	defer syncCmd.SetOut(os.Stdout)
	defer syncCmd.SetErr(os.Stderr)

	_ = syncCmd.Flags().Set("only", "missing")
	_ = syncCmd.Flags().Set("dry-run", "true")
	_ = syncCmd.Flags().Set("yes", "true")
	_ = syncCmd.Flags().Set("format", "yaml")
	_ = syncCmd.Flags().Set("update-local", "false")
	_ = syncCmd.Flags().Set("rebase-dirty", "false")
	_ = syncCmd.Flags().Set("push-local", "false")

	err := syncCmd.RunE(syncCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestDescribeRunEPaths(t *testing.T) {
	cfgPath, regPath := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	describeRepoCmd.SetOut(out)
	describeRepoCmd.SetErr(errOut)
	describeRepoCmd.SetContext(context.Background())
	defer describeRepoCmd.SetOut(os.Stdout)
	defer describeRepoCmd.SetErr(os.Stderr)

	_ = describeRepoCmd.Flags().Set("registry", regPath)
	_ = describeRepoCmd.Flags().Set("format", "table")
	err := describeRepoCmd.RunE(describeRepoCmd, []string{"github.com/org/repo-missing"})
	if err != nil {
		t.Fatalf("describe run failed: %v", err)
	}
	if !strings.Contains(out.String(), "ERROR_CLASS: missing") {
		t.Fatalf("expected table details output, got %q", out.String())
	}

	_ = describeRepoCmd.Flags().Set("registry", filepath.Join(t.TempDir(), "missing-registry.yaml"))
	err = describeRepoCmd.RunE(describeRepoCmd, []string{"github.com/org/repo-missing"})
	if err == nil {
		t.Fatal("expected missing registry file error")
	}
}

func TestStatusUsesNearestConfigFromNestedCWD(t *testing.T) {
	tmp := t.TempDir()
	parentCfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	childRoot := filepath.Join(tmp, "workspace")
	childCfgPath := filepath.Join(childRoot, ".repokeeper.yaml")
	nested := filepath.Join(childRoot, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	parentCfg := config.DefaultConfig()
	parentCfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/parent-missing",
				Path:     filepath.Join(tmp, "parent-missing"),
				Status:   registry.StatusMissing,
				LastSeen: time.Now(),
			},
		},
	}
	if err := config.Save(&parentCfg, parentCfgPath); err != nil {
		t.Fatalf("save parent config: %v", err)
	}

	childCfg := config.DefaultConfig()
	childCfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/child-missing",
				Path:     filepath.Join(childRoot, "child-missing"),
				Status:   registry.StatusMissing,
				LastSeen: time.Now(),
			},
		},
	}
	if err := config.Save(&childCfg, childCfgPath); err != nil {
		t.Fatalf("save child config: %v", err)
	}

	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	if err := rootCmd.PersistentFlags().Set("config", ""); err != nil {
		t.Fatalf("clear config flag: %v", err)
	}
	prevEnv, hadEnv := os.LookupEnv("REPOKEEPER_CONFIG")
	if err := os.Unsetenv("REPOKEEPER_CONFIG"); err != nil {
		t.Fatalf("unset REPOKEEPER_CONFIG: %v", err)
	}
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	defer func() {
		_ = rootCmd.PersistentFlags().Set("config", prevConfig)
		if hadEnv {
			_ = os.Setenv("REPOKEEPER_CONFIG", prevEnv)
		} else {
			_ = os.Unsetenv("REPOKEEPER_CONFIG")
		}
		_ = os.Chdir(origWD)
	}()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)
	_ = statusCmd.Flags().Set("registry", "")
	_ = statusCmd.Flags().Set("format", "json")
	_ = statusCmd.Flags().Set("only", "missing")
	_ = statusCmd.Flags().Set("field-selector", "")
	_ = statusCmd.Flags().Set("selector", "")
	_ = statusCmd.Flags().Set("reconcile-remote-mismatch", "none")
	_ = statusCmd.Flags().Set("dry-run", "true")
	_ = statusCmd.Flags().Set("no-headers", "false")

	if err := statusCmd.RunE(statusCmd, nil); err != nil {
		t.Fatalf("status from nested cwd failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\"repo_id\": \"github.com/org/child-missing\"") {
		t.Fatalf("expected child config repo in output, got: %q", got)
	}
	if strings.Contains(got, "\"repo_id\": \"github.com/org/parent-missing\"") {
		t.Fatalf("did not expect parent config repo in output, got: %q", got)
	}
}

func TestStatusRunEIncludesRepoMetadata(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-with-meta")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	metadataPath := filepath.Join(repoPath, ".repokeeper-repo.yaml")
	metadata := "apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Repo With Meta\nlabels:\n  role: docs\nentrypoints:\n  readme: README.md\npaths:\n  authoritative:\n    - docs/\nprovides:\n  - guides\n"
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-with-meta", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	before, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata before status: %v", err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	statusCmd.SetOut(out)
	statusCmd.SetErr(errOut)
	defer statusCmd.SetOut(os.Stdout)
	defer statusCmd.SetErr(os.Stderr)
	_ = statusCmd.Flags().Set("registry", "")
	_ = statusCmd.Flags().Set("format", "json")
	_ = statusCmd.Flags().Set("only", "all")
	_ = statusCmd.Flags().Set("field-selector", "")
	_ = statusCmd.Flags().Set("selector", "")
	_ = statusCmd.Flags().Set("reconcile-remote-mismatch", "none")
	_ = statusCmd.Flags().Set("dry-run", "true")
	_ = statusCmd.Flags().Set("no-headers", "false")

	if err := statusCmd.RunE(statusCmd, nil); err != nil {
		t.Fatalf("status with repo metadata failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"\"repo_metadata_file\"", "\"repo_metadata\"", "\"name\": \"Repo With Meta\"", "\"role\": \"docs\"", "\"guides\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected status output to contain %q, got: %q", want, got)
		}
	}
	after, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata after status: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("expected status to leave repo metadata file unchanged")
	}
}

func TestDescribeRunEIncludesRepoMetadata(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-with-meta")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	metadataPath := filepath.Join(repoPath, ".repokeeper-repo.yaml")
	if err := os.WriteFile(metadataPath, []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Repo With Meta\nprovides:\n  - guides\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-with-meta", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	describeRepoCmd.SetOut(out)
	describeRepoCmd.SetErr(errOut)
	defer describeRepoCmd.SetOut(os.Stdout)
	defer describeRepoCmd.SetErr(os.Stderr)
	_ = describeRepoCmd.Flags().Set("registry", "")
	_ = describeRepoCmd.Flags().Set("format", "table")

	if err := describeRepoCmd.RunE(describeRepoCmd, []string{"github.com/org/repo-with-meta"}); err != nil {
		t.Fatalf("describe with repo metadata failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"REPO_METADATA_FILE:", "REPO_METADATA_NAME: Repo With Meta", "REPO_METADATA_PROVIDES: guides"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected describe output to contain %q, got: %q", want, got)
		}
	}
}

func TestIndexRunEPreviewOnlyDoesNotWrite(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, true)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader(""))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "false")
	_ = indexCmd.Flags().Set("force", "false")

	if err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"}); err != nil {
		t.Fatalf("index preview failed: %v", err)
	}
	if !strings.Contains(out.String(), "# Repo metadata preview") {
		t.Fatalf("expected preview output, got %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".repokeeper-repo.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo metadata file to be written, got err=%v", err)
	}
}

func TestIndexRunEWriteDeclinedDoesNotWrite(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, false)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader("\n\n\n\n\n\n\nno\n"))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "true")
	_ = indexCmd.Flags().Set("force", "false")

	if err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"}); err != nil {
		t.Fatalf("index declined write failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".repokeeper-repo.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected declined write to leave no file, got err=%v", err)
	}
	if !strings.Contains(errOut.String(), "Write repo metadata") {
		t.Fatalf("expected write confirmation prompt, got %q", errOut.String())
	}
}

func TestIndexRunEFailsEarlyWhenMetadataExistsWithoutForce(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".repokeeper-repo.yaml"), []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Existing\n"), 0o644); err != nil {
		t.Fatalf("write existing metadata: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, false)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader("\n\n\n\n\n\n\n"))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "true")
	_ = indexCmd.Flags().Set("force", "false")

	err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"})
	if err == nil || !strings.Contains(err.Error(), "use --force to overwrite") {
		t.Fatalf("expected overwrite guidance error, got %v", err)
	}
	if strings.Contains(out.String(), "# Repo metadata preview") {
		t.Fatalf("expected no preview output before early failure, got %q", out.String())
	}
	if strings.Contains(errOut.String(), "Write repo metadata") {
		t.Fatalf("expected no confirmation prompt before early failure, got %q", errOut.String())
	}
}

func TestIndexRunEWritesWithYes(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(filepath.Join(repoPath, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir repo docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, true)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader(""))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "true")
	_ = indexCmd.Flags().Set("force", "false")

	if err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"}); err != nil {
		t.Fatalf("index write failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, ".repokeeper-repo.yaml"))
	if err != nil {
		t.Fatalf("read written metadata: %v", err)
	}
	got := string(data)
	for _, want := range []string{"apiVersion: repokeeper/v1", "kind: RepoMetadata", "repo_id: github.com/org/repo-index", "readme: README.md", "docs/"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected written metadata to contain %q, got: %q", want, got)
		}
	}
	if !strings.Contains(out.String(), "wrote repo metadata") {
		t.Fatalf("expected success output, got %q", out.String())
	}
}

func TestIndexRunERejectsRepoIDMismatch(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, false)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader("\nother/repo\n\n\n\n\n\n\n"))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "false")
	_ = indexCmd.Flags().Set("force", "false")

	err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"})
	if err == nil || !strings.Contains(err.Error(), "must match tracked repo_id") {
		t.Fatalf("expected repo_id mismatch error, got %v", err)
	}
}

func TestIndexRunEForceResolvesDualMetadataFiles(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo-index")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".repokeeper-repo.yaml"), []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden metadata: %v", err)
	}
	legacyPath := filepath.Join(repoPath, "repokeeper.yaml")
	if err := os.WriteFile(legacyPath, []byte("apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy metadata: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "github.com/org/repo-index", Path: repoPath, Status: registry.StatusPresent, LastSeen: time.Now()}}}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()
	yesCleanup := withAssumeYes(t, true)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	indexCmd.SetOut(out)
	indexCmd.SetErr(errOut)
	defer indexCmd.SetOut(os.Stdout)
	defer indexCmd.SetErr(os.Stderr)
	indexCmd.SetIn(strings.NewReader(""))
	defer indexCmd.SetIn(os.Stdin)
	_ = indexCmd.Flags().Set("write", "true")
	_ = indexCmd.Flags().Set("force", "true")

	if err := indexCmd.RunE(indexCmd, []string{"github.com/org/repo-index"}); err != nil {
		t.Fatalf("force index write failed: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy metadata file removed, got err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoPath, ".repokeeper-repo.yaml"))
	if err != nil {
		t.Fatalf("read preferred metadata: %v", err)
	}
	if !strings.Contains(string(data), "repo_id: github.com/org/repo-index") {
		t.Fatalf("expected force write to refresh preferred metadata, got %q", string(data))
	}
}

func TestScanRunEIncludesRepoMetadataWithoutWritingIt(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "scan-repo")
	if out, err := exec.Command("git", "init", repoPath).CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v %s", err, string(out))
	}
	metadataPath := filepath.Join(repoPath, ".repokeeper-repo.yaml")
	metadata := "apiVersion: repokeeper/v1\nkind: RepoMetadata\nname: Scan Repo\nlabels:\n  role: tooling\n"
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	before, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata before scan: %v", err)
	}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	scanCmd.SetOut(out)
	scanCmd.SetErr(errOut)
	scanCmd.SetContext(context.Background())
	defer scanCmd.SetOut(os.Stdout)
	defer scanCmd.SetErr(os.Stderr)
	_ = scanCmd.Flags().Set("roots", tmp)
	_ = scanCmd.Flags().Set("exclude", "")
	_ = scanCmd.Flags().Set("follow-symlinks", "false")
	_ = scanCmd.Flags().Set("write-registry", "false")
	_ = scanCmd.Flags().Set("prune-stale", "false")
	_ = scanCmd.Flags().Set("format", "json")
	_ = scanCmd.Flags().Set("no-headers", "false")

	if err := scanCmd.RunE(scanCmd, nil); err != nil {
		t.Fatalf("scan with metadata failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\"repo_metadata\"") || !strings.Contains(got, "Scan Repo") {
		t.Fatalf("expected scan output to include repo metadata, got %q", got)
	}
	after, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata after scan: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("expected scan to leave repo metadata file unchanged")
	}
}
