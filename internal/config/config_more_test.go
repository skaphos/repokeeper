package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if path != filepath.Join(tmp, LocalConfigFilename) {
		t.Fatalf("unexpected init config path %q", path)
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
	if path != localCfg {
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
