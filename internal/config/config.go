// Package config handles loading, saving, and resolving the RepoKeeper
// machine configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/registry"
	"go.yaml.in/yaml/v3"
)

const (
	// LocalConfigFilename is the per-directory RepoKeeper config file.
	LocalConfigFilename = ".repokeeper.yaml"
	// ConfigAPIVersion is the current config schema apiVersion.
	ConfigAPIVersion = "skaphos.io/repokeeper/v1beta1"
	// ConfigKind is the current config schema kind.
	ConfigKind = "RepoKeeperConfig"
)

// Defaults holds default values for operations.
type Defaults struct {
	RemoteName     string `yaml:"remote_name"`
	Concurrency    int    `yaml:"concurrency"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// Config represents the machine-level RepoKeeper configuration.
type Config struct {
	APIVersion        string             `yaml:"apiVersion"`
	Kind              string             `yaml:"kind"`
	Exclude           []string           `yaml:"exclude"`
	RegistryPath      string             `yaml:"registry_path,omitempty"`
	Registry          *registry.Registry `yaml:"registry,omitempty"`
	RegistryStaleDays int                `yaml:"registry_stale_days"`
	Defaults          Defaults           `yaml:"defaults"`
	LegacyRoots       []string           `yaml:"-"`
}

// DefaultConfig returns a Config with sensible defaults applied.
func DefaultConfig() Config {
	return Config{
		APIVersion:        ConfigAPIVersion,
		Kind:              ConfigKind,
		Exclude:           []string{"**/node_modules/**", "**/.terraform/**", "**/dist/**", "**/vendor/**"},
		RegistryStaleDays: 30,
		Defaults: Defaults{
			RemoteName:     "origin",
			Concurrency:    8,
			TimeoutSeconds: 60,
		},
	}
}

// ConfigDir returns the platform-appropriate config directory path.
// It checks, in order: the override parameter, REPOKEEPER_CONFIG env var,
// and finally os.UserConfigDir()/repokeeper.
func ConfigDir(override string) (string, error) {
	if override != "" {
		if isConfigFilePath(override) {
			return filepath.Dir(override), nil
		}
		return override, nil
	}

	if env := os.Getenv("REPOKEEPER_CONFIG"); env != "" {
		if isConfigFilePath(env) {
			return filepath.Dir(env), nil
		}
		return env, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "repokeeper"), nil
}

// ConfigPath resolves the config file path from override/env/defaults.
func ConfigPath(override string) (string, error) {
	if override != "" {
		if isConfigFilePath(override) {
			return override, nil
		}
		return filepath.Join(override, "config.yaml"), nil
	}

	if env := os.Getenv("REPOKEEPER_CONFIG"); env != "" {
		if isConfigFilePath(env) {
			return env, nil
		}
		return filepath.Join(env, "config.yaml"), nil
	}

	dir, err := ConfigDir("")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// InitConfigPath resolves where "repokeeper init" should write config.
// Order: explicit override, REPOKEEPER_CONFIG, then local dotfile in cwd.
func InitConfigPath(override, cwd string) (string, error) {
	if override != "" || os.Getenv("REPOKEEPER_CONFIG") != "" {
		return ConfigPath(override)
	}

	if strings.TrimSpace(cwd) == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(cwd, LocalConfigFilename), nil
}

// ResolveConfigPath resolves config for runtime commands.
// Order: explicit override, REPOKEEPER_CONFIG, nearest local dotfile in cwd/parents,
// then global platform config path.
func ResolveConfigPath(override, cwd string) (string, error) {
	if override != "" || os.Getenv("REPOKEEPER_CONFIG") != "" {
		return ConfigPath(override)
	}

	if strings.TrimSpace(cwd) == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	localPath, err := FindNearestConfigPath(cwd)
	if err != nil {
		return "", err
	}
	if localPath != "" {
		return localPath, nil
	}

	return ConfigPath("")
}

// FindNearestConfigPath searches cwd and each parent directory for .repokeeper.yaml.
// It returns an empty string when no local config file is found.
func FindNearestConfigPath(cwd string) (string, error) {
	dir := cwd
	for {
		candidate := filepath.Join(dir, LocalConfigFilename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// Load reads the config file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	type legacyRoots struct {
		Roots []string `yaml:"roots"`
	}
	var legacy legacyRoots
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	cfg.LegacyRoots = legacy.Roots
	applyConfigGVK(&cfg)
	if err := validateConfigGVK(&cfg); err != nil {
		return nil, err
	}

	if cfg.Registry == nil && cfg.RegistryPath != "" {
		reg, err := registry.Load(ResolveRegistryPath(path, cfg.RegistryPath))
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			cfg.Registry = reg
		}
	}
	if cfg.Defaults.Concurrency == 0 {
		cfg.Defaults.Concurrency = DefaultConfig().Defaults.Concurrency
	}
	if cfg.Defaults.TimeoutSeconds == 0 {
		cfg.Defaults.TimeoutSeconds = DefaultConfig().Defaults.TimeoutSeconds
	}
	if cfg.Defaults.RemoteName == "" {
		cfg.Defaults.RemoteName = DefaultConfig().Defaults.RemoteName
	}

	return &cfg, nil
}

// ResolveRegistryPath resolves registry_path against the config file location.
// Absolute paths are returned unchanged; relative paths are joined to the
// directory containing configPath.
func ResolveRegistryPath(configPath, registryPath string) string {
	if strings.TrimSpace(registryPath) == "" {
		return ""
	}
	if filepath.IsAbs(registryPath) || strings.TrimSpace(configPath) == "" {
		return filepath.Clean(registryPath)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(configPath), registryPath))
}

// ConfigRoot returns the effective default root for a config file path.
func ConfigRoot(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return ""
	}
	return filepath.Clean(filepath.Dir(configPath))
}

// EffectiveRoot returns the inferred root for commands that need a default
// scan/display root.
func EffectiveRoot(configPath string, cfg *Config) string {
	// TODO(v0.1.0): remove legacy roots fallback once configs have migrated.
	if cfg != nil {
		for _, root := range cfg.LegacyRoots {
			root = strings.TrimSpace(root)
			if root == "" {
				continue
			}
			if filepath.IsAbs(root) {
				return filepath.Clean(root)
			}
			base := ConfigRoot(configPath)
			if base == "" {
				return filepath.Clean(root)
			}
			return filepath.Clean(filepath.Join(base, root))
		}
	}
	return ConfigRoot(configPath)
}

// Save writes the config to the given path.
func Save(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	applyConfigGVK(cfg)
	if err := validateConfigGVK(cfg); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LastUpdated is a helper to get "now" in a consistent format for timestamps.
func LastUpdated() string {
	return time.Now().Format(time.RFC3339)
}

func isConfigFilePath(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, "config.yaml") || strings.HasSuffix(lower, "config.yml") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func applyConfigGVK(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.APIVersion) == "" {
		cfg.APIVersion = ConfigAPIVersion
	}
	if strings.TrimSpace(cfg.Kind) == "" {
		cfg.Kind = ConfigKind
	}
}

func validateConfigGVK(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.APIVersion != ConfigAPIVersion {
		return fmt.Errorf("unsupported config apiVersion %q (expected %q)", cfg.APIVersion, ConfigAPIVersion)
	}
	if cfg.Kind != ConfigKind {
		return fmt.Errorf("unsupported config kind %q (expected %q)", cfg.Kind, ConfigKind)
	}
	return nil
}
