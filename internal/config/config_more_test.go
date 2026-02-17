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

	_, err := FindNearestConfigPath(filePath)
	if err == nil {
		t.Fatal("expected error for invalid cwd file path")
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
	got, err := ConfigDir(filepath.Join("/tmp", "cfg", "config.yaml"))
	if err != nil {
		t.Fatalf("ConfigDir override file: %v", err)
	}
	if got != filepath.Join("/tmp", "cfg") {
		t.Fatalf("expected file-dir path, got %q", got)
	}

	got, err = ConfigDir(filepath.Join("/tmp", "cfgdir"))
	if err != nil {
		t.Fatalf("ConfigDir override dir: %v", err)
	}
	if got != filepath.Join("/tmp", "cfgdir") {
		t.Fatalf("expected override dir path, got %q", got)
	}

	if err := os.Setenv("REPOKEEPER_CONFIG", filepath.Join("/tmp", "envcfg", "config.yaml")); err != nil {
		t.Fatalf("set env: %v", err)
	}
	got, err = ConfigDir("")
	if err != nil {
		t.Fatalf("ConfigDir env file: %v", err)
	}
	if got != filepath.Join("/tmp", "envcfg") {
		t.Fatalf("expected env file dir, got %q", got)
	}

	if err := os.Setenv("REPOKEEPER_CONFIG", filepath.Join("/tmp", "envcfgdir")); err != nil {
		t.Fatalf("set env: %v", err)
	}
	got, err = ConfigDir("")
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
