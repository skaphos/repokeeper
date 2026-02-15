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

	err := statusCmd.RunE(statusCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
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
