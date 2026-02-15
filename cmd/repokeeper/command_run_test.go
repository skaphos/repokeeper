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
	prevConfig := flagConfig
	flagConfig = cfgPath

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(filepath.Dir(cfgPath)); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	return func() {
		flagConfig = prevConfig
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
