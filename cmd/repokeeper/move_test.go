// SPDX-License-Identifier: MIT
package repokeeper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestMoveCommandMovesDirectoryAndUpdatesRegistry(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	source := filepath.Join(t.TempDir(), "repo-move-source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{{
			RepoID: "local:repo-move",
			Path:   source,
			Status: registry.StatusPresent,
		}},
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	moveCmd.SetContext(context.Background())
	_ = moveCmd.Flags().Set("registry", "")
	target := filepath.Join("moved", "repo-move")
	if err := moveCmd.RunE(moveCmd, []string{"local:repo-move", target}); err != nil {
		t.Fatalf("move command failed: %v", err)
	}

	targetAbs := filepath.Join(filepath.Dir(cfgPath), target)
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("expected source path moved away, stat err=%v", err)
	}
	if _, err := os.Stat(targetAbs); err != nil {
		t.Fatalf("expected target path after move: %v", err)
	}

	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	got := cfg.Registry.FindByRepoID("local:repo-move")
	if got == nil || got.Path != filepath.Clean(targetAbs) {
		t.Fatalf("expected moved registry path %q, got %+v", filepath.Clean(targetAbs), got)
	}
}

func TestMoveCommandFailureKeepsRegistryUnchanged(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	source := filepath.Join(t.TempDir(), "repo-move-fail")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	target := filepath.Join(filepath.Dir(cfgPath), "already", "exists")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{{
			RepoID: "local:repo-move-fail",
			Path:   source,
			Status: registry.StatusPresent,
		}},
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	moveCmd.SetContext(context.Background())
	_ = moveCmd.Flags().Set("registry", "")
	err = moveCmd.RunE(moveCmd, []string{"local:repo-move-fail", filepath.Join("already", "exists")})
	if err == nil || !strings.Contains(err.Error(), "target already exists") {
		t.Fatalf("expected clear target-exists error, got: %v", err)
	}

	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	got := cfg.Registry.FindByRepoID("local:repo-move-fail")
	if got == nil || got.Path != source {
		t.Fatalf("expected registry to remain unchanged at %q, got %+v", source, got)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("expected source path to remain after failed move: %v", err)
	}
}

func TestMoveCommandRollsBackFilesystemWhenRegistrySaveFails(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	source := filepath.Join(t.TempDir(), "repo-move-rollback")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	targetDir := filepath.Join(filepath.Dir(cfgPath), "rollback-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}

	regDir := filepath.Join(t.TempDir(), "registry-dir")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		t.Fatalf("mkdir registry dir: %v", err)
	}
	regPath := filepath.Join(regDir, "registry.yaml")
	if err := registry.Save(&registry.Registry{
		Entries: []registry.Entry{{
			RepoID: "local:repo-move-rollback",
			Path:   source,
			Status: registry.StatusPresent,
		}},
	}, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	if err := os.Chmod(regPath, 0o444); err != nil {
		t.Fatalf("chmod registry file: %v", err)
	}
	if err := os.Chmod(regDir, 0o555); err != nil {
		t.Fatalf("chmod registry dir: %v", err)
	}

	moveCmd.SetContext(context.Background())
	prevRegistry, _ := moveCmd.Flags().GetString("registry")
	_ = moveCmd.Flags().Set("registry", regPath)
	defer func() {
		_ = os.Chmod(regDir, 0o755)
		_ = os.Chmod(regPath, 0o644)
		_ = moveCmd.Flags().Set("registry", prevRegistry)
	}()

	err := moveCmd.RunE(moveCmd, []string{"local:repo-move-rollback", filepath.Join("rollback-target", "repo-move-rollback")})
	if err == nil || !strings.Contains(err.Error(), "registry update failed; reverted filesystem move") {
		t.Fatalf("expected rollback error, got: %v", err)
	}

	targetPath := filepath.Join(targetDir, "repo-move-rollback")
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("expected source path restored after rollback: %v", err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("expected target path absent after rollback, stat err=%v", err)
	}
}
