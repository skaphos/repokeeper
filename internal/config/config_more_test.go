// SPDX-License-Identifier: MIT
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func canonicalPath(path string) string {
	cleaned := filepath.Clean(path)

	if _, err := os.Stat(cleaned); err == nil {
		if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
			return filepath.Clean(resolved)
		}
	}

	dir := filepath.Dir(cleaned)
	base := filepath.Base(cleaned)
	if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Clean(filepath.Join(resolvedDir, base))
	}

	return cleaned
}

func TestInitConfigPathUsesGetwdWhenCWDMissing(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	path, err := InitConfigPath("", "")
	if err != nil {
		t.Fatalf("InitConfigPath: %v", err)
	}
	want := filepath.Join(tmp, LocalConfigFilename)
	if canonicalPath(path) != canonicalPath(want) {
		t.Fatalf("unexpected init config path %q (want %q)", path, want)
	}
}

func TestResolveConfigPathUsesGetwdWhenCWDMissing(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	localCfg := filepath.Join(tmp, LocalConfigFilename)
	if err := os.WriteFile(localCfg, []byte("exclude: []\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	path, err := ResolveConfigPath("", "")
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if canonicalPath(path) != canonicalPath(localCfg) {
		t.Fatalf("expected local config path %q, got %q", localCfg, path)
	}
}

func TestFindNearestConfigPathErrorsOnInvalidCWD(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := findNearestConfigPath(filePath)
	if err == nil {
		t.Fatal("expected error for invalid cwd file path")
	}
}

func TestFindNearestConfigPathIgnoresDirectoryNamedLikeConfig(t *testing.T) {
	tmp := t.TempDir()
	// A directory (not a regular file) named .repokeeper.yaml must not be
	// treated as a config file.
	if err := os.Mkdir(filepath.Join(tmp, LocalConfigFilename), 0o755); err != nil {
		t.Fatalf("mkdir config-named dir: %v", err)
	}
	got, err := findNearestConfigPath(tmp)
	if err != nil {
		t.Fatalf("findNearestConfigPath: %v", err)
	}
	if got != "" {
		t.Fatalf("expected no config match for a directory, got %q", got)
	}
}

func TestFindNearestConfigPathDoesNotWalkIntoSharedDir(t *testing.T) {
	// Simulate a planted config in a shared, world-writable ancestor. The walk
	// must not ascend into it and adopt the planted file.
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o777); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	// Make it world-writable + sticky regardless of umask (mimics /tmp).
	if err := os.Chmod(shared, 0o777|os.ModeSticky); err != nil {
		t.Fatalf("chmod shared: %v", err)
	}
	planted := filepath.Join(shared, LocalConfigFilename)
	if err := os.WriteFile(planted, []byte("exclude: []\n"), 0o644); err != nil {
		t.Fatalf("write planted config: %v", err)
	}
	child := filepath.Join(shared, "victim")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir victim: %v", err)
	}

	got, err := findNearestConfigPath(child)
	if err != nil {
		t.Fatalf("findNearestConfigPath: %v", err)
	}
	if got != "" {
		t.Fatalf("expected walk to stop before adopting planted config, got %q", got)
	}
}

func TestFindNearestConfigPathStopsAtHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home directory available")
	}
	// From within home the walk must never return a path above home.
	got, err := findNearestConfigPath(home)
	if err != nil {
		t.Fatalf("findNearestConfigPath: %v", err)
	}
	if got != "" {
		if rel, relErr := filepath.Rel(home, got); relErr == nil && strings.HasPrefix(rel, "..") {
			t.Fatalf("walk escaped above home: %q", got)
		}
	}
}

func TestResolveRegistryPathBlank(t *testing.T) {
	if got := ResolveRegistryPath("/tmp/config.yaml", "   "); got != "" {
		t.Fatalf("expected blank registry path, got %q", got)
	}
}

func TestSaveNilConfigErrors(t *testing.T) {
	if err := Save(nil, filepath.Join(t.TempDir(), "cfg.yaml")); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestSaveErrorsWhenParentIsFile(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	cfg := DefaultConfig()
	err := Save(&cfg, filepath.Join(blocker, "config.yaml"))
	if err == nil {
		t.Fatal("expected save error when parent path is file")
	}
}

func TestLoadInvalidYAMLErrors(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "broken.yaml")
	if err := os.WriteFile(cfgPath, []byte(":\n"), 0o644); err != nil {
		t.Fatalf("write invalid yaml: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected yaml parse error")
	}
}

func TestValidateSavedConfigGVKErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIVersion = "example/v1"
	if err := validateSavedConfigGVK(&cfg); err == nil || !strings.Contains(err.Error(), "unsupported config apiVersion") {
		t.Fatalf("expected apiVersion validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Kind = "WrongKind"
	if err := validateSavedConfigGVK(&cfg); err == nil || !strings.Contains(err.Error(), "unsupported config kind") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
}

func TestConfigDirVariants(t *testing.T) {
	got, err := configDir(filepath.Join("/tmp", "cfg", "config.yaml"))
	if err != nil {
		t.Fatalf("ConfigDir override file: %v", err)
	}
	if got != filepath.Join("/tmp", "cfg") {
		t.Fatalf("expected file-dir path, got %q", got)
	}

	got, err = configDir(filepath.Join("/tmp", "cfgdir"))
	if err != nil {
		t.Fatalf("ConfigDir override dir: %v", err)
	}
	if got != filepath.Join("/tmp", "cfgdir") {
		t.Fatalf("expected override dir path, got %q", got)
	}

	if err := os.Setenv("REPOKEEPER_CONFIG", filepath.Join("/tmp", "envcfg", "config.yaml")); err != nil {
		t.Fatalf("set env: %v", err)
	}
	got, err = configDir("")
	if err != nil {
		t.Fatalf("ConfigDir env file: %v", err)
	}
	if got != filepath.Join("/tmp", "envcfg") {
		t.Fatalf("expected env file dir, got %q", got)
	}

	if err := os.Setenv("REPOKEEPER_CONFIG", filepath.Join("/tmp", "envcfgdir")); err != nil {
		t.Fatalf("set env: %v", err)
	}
	got, err = configDir("")
	if err != nil {
		t.Fatalf("ConfigDir env dir: %v", err)
	}
	if got != filepath.Join("/tmp", "envcfgdir") {
		t.Fatalf("expected env dir path, got %q", got)
	}
	_ = os.Unsetenv("REPOKEEPER_CONFIG")
}

func TestValidationAndRootEdgeCases(t *testing.T) {
	if err := validateLoadedConfigGVK(nil); err == nil {
		t.Fatal("expected nil config validation error")
	}
	if ConfigRoot("   ") != "" {
		t.Fatal("expected blank config root for empty input")
	}
	if got := ResolveRegistryPath("", "/tmp/reg.yaml"); got != filepath.Clean("/tmp/reg.yaml") {
		t.Fatalf("expected absolute registry path passthrough, got %q", got)
	}
	if got := ResolveRegistryPath("", "relative.yaml"); got != "relative.yaml" {
		t.Fatalf("expected relative passthrough with blank config path, got %q", got)
	}
}
