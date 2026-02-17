// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=repokeeper-test",
		"GIT_AUTHOR_EMAIL=repokeeper@test.local",
		"GIT_COMMITTER_NAME=repokeeper-test",
		"GIT_COMMITTER_EMAIL=repokeeper@test.local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func writeEmptyConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

func withConfigAndCWD(t *testing.T, cfgPath string) func() {
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

func TestAddDeleteWithRegistryOverride(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	src := filepath.Join(t.TempDir(), "source")
	mustRunGit(t, filepath.Dir(src), "init", src)
	mustRunGit(t, src, "commit", "--allow-empty", "-m", "init")

	regPath := filepath.Join(t.TempDir(), "registry.yaml")
	if err := registry.Save(&registry.Registry{}, regPath); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	target := filepath.Join("repos", "repo-a")
	addOut := &bytes.Buffer{}
	addCmd.SetOut(addOut)
	addCmd.SetContext(context.Background())
	defer addCmd.SetOut(os.Stdout)
	_ = addCmd.Flags().Set("registry", regPath)
	_ = addCmd.Flags().Set("branch", "")
	_ = addCmd.Flags().Set("mirror", "false")
	if err := addCmd.RunE(addCmd, []string{target, src}); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !strings.Contains(addOut.String(), "added ") {
		t.Fatalf("expected add output, got %q", addOut.String())
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("expected one entry after add, got %d", len(reg.Entries))
	}

	delOut := &bytes.Buffer{}
	deleteCmd.SetOut(delOut)
	deleteCmd.SetContext(context.Background())
	defer deleteCmd.SetOut(os.Stdout)
	prevDeleteRegistry, _ := deleteCmd.Flags().GetString("registry")
	defer func() { _ = deleteCmd.Flags().Set("registry", prevDeleteRegistry) }()
	_ = deleteCmd.Flags().Set("registry", regPath)
	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "true")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()
	if err := deleteCmd.RunE(deleteCmd, []string{reg.Entries[0].RepoID}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	reg, err = registry.Load(regPath)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if len(reg.Entries) != 0 {
		t.Fatalf("expected zero entries after delete, got %d", len(reg.Entries))
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected repository path deleted from disk, stat err=%v", err)
	}
}

func TestAddValidationMutuallyExclusiveFlags(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	addCmd.SetContext(context.Background())
	_ = addCmd.Flags().Set("branch", "main")
	_ = addCmd.Flags().Set("mirror", "true")
	err := addCmd.RunE(addCmd, []string{"target", "https://example.invalid/repo.git"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusivity error, got %v", err)
	}
}

func TestInitCommandForceBehavior(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	_ = rootCmd.PersistentFlags().Set("config", cfgPath)
	defer func() { _ = rootCmd.PersistentFlags().Set("config", prevConfig) }()
	origWD, _ := os.Getwd()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(origWD) }()
	repoPath := filepath.Join(tmp, "repo")
	mustRunGit(t, tmp, "init", repoPath)
	mustRunGit(t, repoPath, "commit", "--allow-empty", "-m", "init")

	out := &bytes.Buffer{}
	initCmd.SetOut(out)
	initCmd.SetContext(context.Background())
	defer initCmd.SetOut(os.Stdout)
	_ = initCmd.Flags().Set("force", "false")
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file: %v", err)
	}

	err := initCmd.RunE(initCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "config already exists") {
		t.Fatalf("expected existing config error, got %v", err)
	}

	_ = initCmd.Flags().Set("force", "true")
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("forced init failed: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config after forced init: %v", err)
	}
	initialCount := len(cfg.Registry.Entries)
	if initialCount != 1 {
		t.Fatalf("expected one registry entry after forced init, got %d", initialCount)
	}

	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("second forced init failed: %v", err)
	}
	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config after second forced init: %v", err)
	}
	if got := len(cfg.Registry.Entries); got != initialCount {
		t.Fatalf("expected reinit to replace registry content (len=%d), got len=%d", initialCount, got)
	}
}

func TestScanJSONOutputAndUnsupportedFormat(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	scanCmd.SetOut(out)
	scanCmd.SetContext(context.Background())
	defer scanCmd.SetOut(os.Stdout)
	_ = scanCmd.Flags().Set("roots", "")
	_ = scanCmd.Flags().Set("exclude", "")
	_ = scanCmd.Flags().Set("follow-symlinks", "false")
	_ = scanCmd.Flags().Set("write-registry", "false")
	_ = scanCmd.Flags().Set("prune-stale", "false")
	_ = scanCmd.Flags().Set("format", "json")
	if err := scanCmd.RunE(scanCmd, nil); err != nil {
		t.Fatalf("scan json failed: %v", err)
	}
	if !strings.Contains(out.String(), "[") && !strings.Contains(out.String(), "null") {
		t.Fatalf("expected json output, got %q", out.String())
	}

	_ = scanCmd.Flags().Set("format", "yaml")
	err := scanCmd.RunE(scanCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestEditRequiresEditorConfiguration(t *testing.T) {
	cfgPath, regPath := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	prevEditor := os.Getenv("EDITOR")
	prevVisual := os.Getenv("VISUAL")
	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")
	defer func() {
		_ = os.Setenv("EDITOR", prevEditor)
		_ = os.Setenv("VISUAL", prevVisual)
	}()

	editCmd.SetContext(context.Background())
	_ = editCmd.Flags().Set("registry", regPath)
	err := editCmd.RunE(editCmd, []string{"github.com/org/repo-missing"})
	if err == nil || !strings.Contains(err.Error(), "set VISUAL or EDITOR") {
		t.Fatalf("expected editor configuration error, got %v", err)
	}
}

func TestDeleteCancelledByPrompt(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	repoPath := filepath.Join(t.TempDir(), "repo-cancel")
	mustRunGit(t, filepath.Dir(repoPath), "init", repoPath)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{{
			RepoID: "local:repo-cancel",
			Path:   repoPath,
			Status: registry.StatusPresent,
		}},
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "false")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()
	prevDeleteRegistry, _ := deleteCmd.Flags().GetString("registry")
	defer func() { _ = deleteCmd.Flags().Set("registry", prevDeleteRegistry) }()
	_ = deleteCmd.Flags().Set("registry", "")

	deleteCmd.SetContext(context.Background())
	deleteCmd.SetIn(strings.NewReader("n\n"))
	defer deleteCmd.SetIn(os.Stdin)
	if err := deleteCmd.RunE(deleteCmd, []string{"local:repo-cancel"}); err != nil {
		t.Fatalf("delete should cancel without error, got: %v", err)
	}

	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.Registry == nil || len(cfg.Registry.Entries) != 1 {
		t.Fatalf("expected entry retained after cancelled delete, got %+v", cfg.Registry)
	}
	if _, err := os.Stat(repoPath); err != nil {
		t.Fatalf("expected repository to remain on disk, stat err=%v", err)
	}
}

func TestDeleteTrackingOnlyAddsIgnoredPathAndKeepsRepo(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	repoPath := filepath.Join(t.TempDir(), "repo-track-only")
	mustRunGit(t, filepath.Dir(repoPath), "init", repoPath)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{{
			RepoID: "local:repo-track-only",
			Path:   repoPath,
			Status: registry.StatusPresent,
		}},
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	prevYes, _ := rootCmd.PersistentFlags().GetBool("yes")
	_ = rootCmd.PersistentFlags().Set("yes", "true")
	defer func() { _ = rootCmd.PersistentFlags().Set("yes", boolToFlag(prevYes)) }()

	prevTrackingOnly, _ := deleteCmd.Flags().GetBool("tracking-only")
	_ = deleteCmd.Flags().Set("tracking-only", "true")
	defer func() { _ = deleteCmd.Flags().Set("tracking-only", boolToFlag(prevTrackingOnly)) }()

	prevDeleteRegistry, _ := deleteCmd.Flags().GetString("registry")
	_ = deleteCmd.Flags().Set("registry", "")
	defer func() { _ = deleteCmd.Flags().Set("registry", prevDeleteRegistry) }()

	if err := deleteCmd.RunE(deleteCmd, []string{"local:repo-track-only"}); err != nil {
		t.Fatalf("tracking-only delete failed: %v", err)
	}

	cfg, err = config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.Registry != nil && len(cfg.Registry.Entries) != 0 {
		t.Fatalf("expected entry removed from registry, got %+v", cfg.Registry.Entries)
	}
	if !slices.Contains(cfg.IgnoredPaths, filepath.Clean(repoPath)) {
		t.Fatalf("expected ignored path %q, got %#v", filepath.Clean(repoPath), cfg.IgnoredPaths)
	}
	if _, err := os.Stat(repoPath); err != nil {
		t.Fatalf("expected repo to remain on disk for tracking-only delete, got %v", err)
	}
}
