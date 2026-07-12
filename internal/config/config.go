// SPDX-License-Identifier: MIT
// Package config handles loading, saving, and resolving the RepoKeeper
// machine configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skaphos/repokeeper/internal/pathutil"
	"github.com/skaphos/repokeeper/internal/registry"
	"go.yaml.in/yaml/v3"
)

const (
	// LocalConfigFilename is the per-directory RepoKeeper config file.
	LocalConfigFilename = ".repokeeper.yaml"
	// LegacyConfigAPIVersion is the implicit schema version used when legacy
	// configs omit apiVersion/kind.
	LegacyConfigAPIVersion = "skaphos.io/repokeeper/v1alpha1"
	// ConfigAPIVersion is the current config schema apiVersion.
	ConfigAPIVersion = "skaphos.io/repokeeper/v1beta1"
	// ConfigKind is the current config schema kind.
	ConfigKind = "RepoKeeperConfig"
)

// Defaults holds default values for operations.
type Defaults struct {
	RemoteName     string `yaml:"remote_name"`
	MainBranch     string `yaml:"main_branch"`
	Concurrency    int    `yaml:"concurrency"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// BranchPolicy configures branch retention and protection for prune-safety
// classification. It is machine-local operator policy (ADR-0005, ADR-0015) and
// affects classification/planning only, never execution.
type BranchPolicy struct {
	// ProtectedPatterns are glob patterns (path.Match) for branches that must
	// never be prune candidates. This is a distinct set from the rebase
	// --protected-branches knob.
	ProtectedPatterns []string `yaml:"protected_patterns"`
	// BaseBranch, when non-empty, is a global override for the merge-into-base
	// reference. Empty means the base is resolved per repository.
	BaseBranch string `yaml:"base_branch,omitempty"`
	// StaleDays escalates an unintegrated branch older than this many days to
	// needs_review. 0 disables staleness escalation.
	StaleDays int `yaml:"stale_days"`
	// RequireMerged, when true, trusts only reachability as merge proof, so a
	// patch-equivalent-only branch is surfaced for review rather than as a
	// probably_safe prune candidate.
	RequireMerged bool `yaml:"require_merged"`
}

// Config represents the machine-level RepoKeeper configuration.
type Config struct {
	APIVersion        string             `yaml:"apiVersion"`
	Kind              string             `yaml:"kind"`
	Exclude           []string           `yaml:"exclude"`
	IgnoredPaths      []string           `yaml:"ignored_paths,omitempty"`
	RegistryPath      string             `yaml:"registry_path,omitempty"`
	Registry          *registry.Registry `yaml:"registry,omitempty"`
	RegistryStaleDays int                `yaml:"registry_stale_days"`
	Defaults          Defaults           `yaml:"defaults"`
	BranchPolicy      BranchPolicy       `yaml:"branch_policy"`
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
			MainBranch:     "main",
			Concurrency:    8,
			TimeoutSeconds: 60,
		},
		BranchPolicy: BranchPolicy{
			ProtectedPatterns: []string{"main", "master", "release/*"},
			StaleDays:         0,
			RequireMerged:     true,
		},
	}
}

// configDir returns the platform-appropriate config directory path.
// It checks, in order: the override parameter, REPOKEEPER_CONFIG env var,
// and finally os.UserConfigDir()/repokeeper.
func configDir(override string) (string, error) {
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

	dir, err := configDir("")
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

	localPath, err := findNearestConfigPath(cwd)
	if err != nil {
		return "", err
	}
	if localPath != "" {
		return localPath, nil
	}

	return ConfigPath("")
}

// findNearestConfigPath searches cwd and each parent directory for a
// .repokeeper.yaml regular file. It returns an empty string when none is found.
//
// The upward walk is bounded so an attacker cannot plant a config in a shared
// ancestor (e.g. /tmp/.repokeeper.yaml) and have it silently adopted:
//   - the walk stops at the user's home directory (inclusive); and
//   - it never ascends into a world-writable or sticky directory such as /tmp.
//
// The candidate must be a regular file; a directory named .repokeeper.yaml is
// ignored.
func findNearestConfigPath(cwd string) (string, error) {
	info, err := os.Stat(cwd)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cwd is not a directory: %s", cwd)
	}

	home, _ := os.UserHomeDir()

	dir := cwd
	for {
		candidate := filepath.Join(dir, LocalConfigFilename)
		if fi, err := os.Lstat(candidate); err == nil {
			if fi.Mode().IsRegular() {
				return candidate, nil
			}
			// A non-regular match (directory, symlink, socket) is not a config
			// file; ignore it and keep walking upward.
		} else if !os.IsNotExist(err) {
			return "", err
		}

		// Boundary: never search above the user's home directory.
		if home != "" && sameDirPath(dir, home) {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached the filesystem root
		}
		// Do not ascend into a shared/world-writable directory (e.g. /tmp),
		// where a config file could be planted by another user.
		if isSharedDir(parent) {
			return "", nil
		}
		dir = parent
	}
}

// sameDirPath reports whether two directory paths refer to the same location
// after cleaning (and case-folding on Windows).
func sameDirPath(a, b string) bool {
	left := filepath.Clean(a)
	right := filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

// isSharedDir reports whether dir is world-writable or sticky, which marks it
// as a shared location (like /tmp) that config discovery must not walk into.
func isSharedDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode.Perm()&0o002 != 0 || mode&os.ModeSticky != 0
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
	loadGVKState(&cfg, data)
	// Validate before applying defaults so schema/version errors are surfaced clearly.
	if err := validateLoadedConfigGVK(&cfg); err != nil {
		return nil, err
	}
	if err := validateBranchPolicy(&cfg); err != nil {
		return nil, err
	}

	if cfg.Registry == nil && cfg.RegistryPath != "" {
		// A missing registry file is not fatal for first-run/new-config flows.
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
		// Persisted zero-values are treated as "unset" and backfilled from defaults.
		cfg.Defaults.Concurrency = DefaultConfig().Defaults.Concurrency
	}
	if cfg.Defaults.TimeoutSeconds == 0 {
		cfg.Defaults.TimeoutSeconds = DefaultConfig().Defaults.TimeoutSeconds
	}
	if cfg.Defaults.RemoteName == "" {
		cfg.Defaults.RemoteName = DefaultConfig().Defaults.RemoteName
	}
	if cfg.Defaults.MainBranch == "" {
		cfg.Defaults.MainBranch = DefaultConfig().Defaults.MainBranch
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
// scan/display root. Root selection is path-derived only; config-based root
// fallbacks were retired.
func EffectiveRoot(configPath string) string {
	return ConfigRoot(configPath)
}

// Save writes the config to the given path.
//
// When RegistryPath is set, the registry is persisted to that external file
// rather than being inlined into the config document. Inlining it would
// duplicate the registry into config.yaml and orphan the external file that
// Load reads back from, so the round-trip would not be stable.
func Save(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	upgradeConfigGVK(cfg)
	if err := validateSavedConfigGVK(cfg); err != nil {
		return err
	}
	if err := validateBranchPolicy(cfg); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Marshal a shallow copy so the caller's Config is never mutated, and so the
	// external registry is not embedded into config.yaml.
	toWrite := *cfg
	if strings.TrimSpace(cfg.RegistryPath) != "" {
		if cfg.Registry != nil {
			regPath := ResolveRegistryPath(path, cfg.RegistryPath)
			if regPath == "" {
				return fmt.Errorf("registry_path %q resolved to empty path", cfg.RegistryPath)
			}
			if err := registry.Save(cfg.Registry, regPath); err != nil {
				return err
			}
		}
		toWrite.Registry = nil
	}

	data, err := yaml.Marshal(&toWrite)
	if err != nil {
		return err
	}
	// Atomic write so a crash mid-write cannot destroy the sole-copy config.
	return pathutil.WriteFileAtomic(path, data, 0o644)
}

func isConfigFilePath(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, "config.yaml") || strings.HasSuffix(lower, "config.yml") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func loadGVKState(cfg *Config, raw []byte) {
	if cfg == nil {
		return
	}
	type gvkPresence struct {
		APIVersion *string `yaml:"apiVersion"`
		Kind       *string `yaml:"kind"`
	}
	var present gvkPresence
	if err := yaml.Unmarshal(raw, &present); err != nil {
		return
	}
	if present.APIVersion == nil && present.Kind == nil {
		cfg.APIVersion = LegacyConfigAPIVersion
		cfg.Kind = ConfigKind
		return
	}
	if strings.TrimSpace(cfg.APIVersion) == "" {
		cfg.APIVersion = ConfigAPIVersion
	}
	if strings.TrimSpace(cfg.Kind) == "" {
		cfg.Kind = ConfigKind
	}
}

func upgradeConfigGVK(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.APIVersion = ConfigAPIVersion
	cfg.Kind = ConfigKind
}

func validateLoadedConfigGVK(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.APIVersion == LegacyConfigAPIVersion && cfg.Kind == ConfigKind {
		return nil
	}
	if cfg.APIVersion != ConfigAPIVersion {
		return fmt.Errorf("unsupported config apiVersion %q (expected %q)", cfg.APIVersion, ConfigAPIVersion)
	}
	if cfg.Kind != ConfigKind {
		return fmt.Errorf("unsupported config kind %q (expected %q)", cfg.Kind, ConfigKind)
	}
	return nil
}

func validateSavedConfigGVK(cfg *Config) error {
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

// validateBranchPolicy fails closed on branch-policy inputs that would silently
// weaken prune protection: malformed protected globs, an over-broad "*" pattern,
// a glob-shaped base branch, or a negative stale window.
func validateBranchPolicy(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	bp := cfg.BranchPolicy
	if bp.StaleDays < 0 {
		return fmt.Errorf("branch_policy.stale_days must not be negative, got %d", bp.StaleDays)
	}
	for _, pattern := range bp.ProtectedPatterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		if p == "*" {
			return fmt.Errorf("branch_policy.protected_patterns must not contain %q (it is overly broad: path.Match %q protects all top-level branches; list them explicitly instead)", pattern, "*")
		}
		if _, err := path.Match(p, "example"); err != nil {
			return fmt.Errorf("branch_policy.protected_patterns has invalid glob %q: %w", pattern, err)
		}
	}
	if base := strings.TrimSpace(bp.BaseBranch); base != "" && strings.ContainsAny(base, "*?[") {
		return fmt.Errorf("branch_policy.base_branch %q must be a branch name, not a glob", base)
	}
	return nil
}
