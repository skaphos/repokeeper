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
)

func writeLabelsTestConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:   "github.com/org/repo-a",
				Path:     filepath.Join(tmp, "repo-a"),
				Status:   registry.StatusPresent,
				LastSeen: time.Now(),
				Labels: map[string]string{
					"team": "platform",
					"env":  "prod",
				},
			},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

func TestLabelCommandShowsLabels(t *testing.T) {
	cfgPath := writeLabelsTestConfig(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	labelCmd.SetOut(out)
	labelCmd.SetContext(context.Background())
	defer labelCmd.SetOut(os.Stdout)

	_ = labelCmd.Flags().Set("registry", "")
	_ = labelCmd.Flags().Set("format", "table")
	_ = labelCmd.Flags().Set("set", "")
	_ = labelCmd.Flags().Set("remove", "")
	if err := labelCmd.RunE(labelCmd, []string{"github.com/org/repo-a"}); err != nil {
		t.Fatalf("labels command failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "team=platform") || !strings.Contains(got, "env=prod") {
		t.Fatalf("expected labels in output, got: %q", got)
	}
}

func TestLabelCommandSetAndRemove(t *testing.T) {
	cfgPath := writeLabelsTestConfig(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	labelCmd.SetContext(context.Background())
	_ = labelCmd.Flags().Set("registry", "")
	_ = labelCmd.Flags().Set("format", "json")
	_ = labelCmd.Flags().Set("set", "owner=sre")
	_ = labelCmd.Flags().Set("remove", "env")
	if err := labelCmd.RunE(labelCmd, []string{"github.com/org/repo-a"}); err != nil {
		t.Fatalf("labels set/remove failed: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	entry := cfg.Registry.FindByRepoID("github.com/org/repo-a")
	if entry == nil {
		t.Fatal("expected entry")
	}
	if got := entry.Labels["owner"]; got != "sre" {
		t.Fatalf("expected owner label, got %q", got)
	}
	if _, ok := entry.Labels["env"]; ok {
		t.Fatalf("expected env label removed, got %#v", entry.Labels)
	}
}

func TestLabelCommandRejectsUnsupportedFormat(t *testing.T) {
	cfgPath := writeLabelsTestConfig(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	labelCmd.SetContext(context.Background())
	_ = labelCmd.Flags().Set("registry", "")
	_ = labelCmd.Flags().Set("format", "yaml")
	_ = labelCmd.Flags().Set("set", "")
	_ = labelCmd.Flags().Set("remove", "")
	err := labelCmd.RunE(labelCmd, []string{"github.com/org/repo-a"})
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}
