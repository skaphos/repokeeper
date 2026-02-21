// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/pathutil"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type importMode string

const (
	importModeMerge   importMode = "merge"
	importModeReplace importMode = "replace"
)

type importConflictPolicy string

const (
	importConflictPolicySkip   importConflictPolicy = "skip"
	importConflictPolicyBundle importConflictPolicy = "bundle"
	importConflictPolicyLocal  importConflictPolicy = "local"
)

var importCmd = &cobra.Command{
	Use:   "import [bundle-file|-]",
	Short: "Import an exported config bundle",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		modeRaw, _ := cmd.Flags().GetString("mode")
		mode, err := parseImportMode(modeRaw)
		if err != nil {
			return err
		}
		onConflictRaw, _ := cmd.Flags().GetString("on-conflict")
		onConflict, err := parseImportConflictPolicy(onConflictRaw)
		if err != nil {
			return err
		}
		includeRegistry, _ := cmd.Flags().GetBool("include-registry")
		preserveRegistryPath, _ := cmd.Flags().GetBool("preserve-registry-path")
		dangerouslyDeleteExisting, _ := cmd.Flags().GetBool("dangerously-delete-existing")
		fileOnly, _ := cmd.Flags().GetBool("file-only")
		cloneRepos := !fileOnly

		if fileOnly {
			includeRegistry = false
			preserveRegistryPath = false
		}

		inputPath := "-"
		if len(args) == 1 {
			inputPath = strings.TrimSpace(args[0])
		}
		if inputPath == "" {
			return fmt.Errorf("bundle-file cannot be empty")
		}
		var data []byte
		if inputPath == "-" {
			stdinData, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			data = stdinData
		} else {
			fileData, err := os.ReadFile(inputPath)
			if err != nil {
				return err
			}
			data = fileData
		}
		var bundle exportBundle
		if err := yaml.Unmarshal(data, &bundle); err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfgPath, err := config.InitConfigPath(configOverride(cmd), cwd)
		if err != nil {
			return err
		}
		existingCfg, hasExistingCfg, err := loadExistingConfig(cfgPath)
		if err != nil {
			return err
		}
		if mode == importModeReplace && hasExistingCfg && !force {
			return fmt.Errorf("config already exists at %q (use --force to overwrite)", cfgPath)
		}

		cfg := prepareImportedConfig(mode, existingCfg, hasExistingCfg, bundle.Config)
		mergeImportedRegistry(&cfg, mode, includeRegistry, bundle.Registry, onConflict)
		dropIgnoredImportEntries(&cfg, bundle, cwd)
		if !preserveRegistryPath && mode == importModeReplace {
			cfg.RegistryPath = ""
		}
		if !assumeYes(cmd) {
			confirmed, err := confirmWithPrompt(cmd, "Proceed with import changes? [y/N]: ")
			if err != nil {
				return err
			}
			if !confirmed {
				infof(cmd, "import cancelled")
				return nil
			}
		}
		if cloneRepos {
			progress := newSyncProgressWriter(cmd, cwd, nil)
			var failures []engine.SyncResult
			if mode == importModeMerge && hasExistingCfg {
				entriesToClone := selectMergeCloneEntries(existingCfg.Registry, bundle.Registry, onConflict)
				var err error
				failures, err = cloneImportedEntriesWithProgress(cmd, &cfg, bundle, cwd, dangerouslyDeleteExisting, entriesToClone, progress)
				if err != nil {
					return err
				}
			} else {
				var err error
				failures, err = cloneImportedReposWithProgress(cmd, &cfg, bundle, cwd, dangerouslyDeleteExisting, progress)
				if err != nil {
					return err
				}
			}
			if len(failures) > 0 {
				raiseExitCode(cmd, 2)
				if err := writeImportCloneFailureSummary(cmd, failures, cwd); err != nil {
					return err
				}
				infof(cmd, "import clone completed with %d failures", len(failures))
			}
		}
		if err := config.Save(&cfg, cfgPath); err != nil {
			return err
		}
		infof(cmd, "imported config to %s", cfgPath)
		return nil
	},
}

func init() {
	importCmd.Flags().Bool("force", false, "overwrite existing config file")
	importCmd.Flags().String("mode", string(importModeMerge), "import mode: merge or replace")
	importCmd.Flags().String("on-conflict", string(importConflictPolicyBundle), "when mode=merge and repo_id exists locally: skip, bundle, or local")
	importCmd.Flags().Bool("include-registry", true, "import bundled registry when present")
	importCmd.Flags().Bool("preserve-registry-path", false, "keep bundled registry_path (resolved relative to imported config file)")
	importCmd.Flags().Bool("dangerously-delete-existing", false, "dangerous: delete conflicting target repo paths before cloning")
	importCmd.Flags().Bool("file-only", false, "import config file only (disable registry import and cloning)")

	rootCmd.AddCommand(importCmd)
}

func parseImportMode(raw string) (importMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(importModeMerge):
		return importModeMerge, nil
	case string(importModeReplace):
		return importModeReplace, nil
	default:
		return "", fmt.Errorf("invalid --mode %q (supported: merge,replace)", raw)
	}
}

func parseImportConflictPolicy(raw string) (importConflictPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(importConflictPolicyBundle):
		return importConflictPolicyBundle, nil
	case string(importConflictPolicySkip):
		return importConflictPolicySkip, nil
	case string(importConflictPolicyLocal):
		return importConflictPolicyLocal, nil
	default:
		return "", fmt.Errorf("invalid --on-conflict %q (supported: skip,bundle,local)", raw)
	}
}

func loadExistingConfig(cfgPath string) (config.Config, bool, error) {
	var empty config.Config
	if _, err := os.Stat(cfgPath); err != nil {
		if os.IsNotExist(err) {
			return empty, false, nil
		}
		return empty, false, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return empty, false, err
	}
	return *cfg, true, nil
}

func prepareImportedConfig(mode importMode, existing config.Config, hasExisting bool, bundled config.Config) config.Config {
	if mode == importModeMerge && hasExisting {
		return existing
	}
	return bundled
}

func mergeImportedRegistry(
	cfg *config.Config,
	mode importMode,
	includeRegistry bool,
	bundled *registry.Registry,
	policy importConflictPolicy,
) {
	if cfg == nil {
		return
	}
	if !includeRegistry {
		if mode == importModeReplace {
			cfg.Registry = nil
		}
		return
	}
	if mode == importModeReplace {
		cfg.Registry = cloneRegistry(bundled)
		return
	}
	if cfg.Registry == nil {
		cfg.Registry = &registry.Registry{}
	}
	if bundled == nil {
		return
	}
	for _, incoming := range bundled.Entries {
		existing := cfg.Registry.FindByRepoID(incoming.RepoID)
		if existing == nil {
			cfg.Registry.Entries = append(cfg.Registry.Entries, incoming)
			continue
		}
		if !registryEntriesConflict(*existing, incoming) {
			*existing = incoming
			continue
		}
		switch policy {
		case importConflictPolicyBundle:
			*existing = incoming
		case importConflictPolicySkip, importConflictPolicyLocal:
		}
	}
	cfg.Registry.UpdatedAt = time.Now()
}

func registryEntriesConflict(local, incoming registry.Entry) bool {
	return strings.TrimSpace(local.Path) != strings.TrimSpace(incoming.Path) ||
		strings.TrimSpace(local.RemoteURL) != strings.TrimSpace(incoming.RemoteURL) ||
		strings.TrimSpace(local.Branch) != strings.TrimSpace(incoming.Branch) ||
		strings.TrimSpace(local.Type) != strings.TrimSpace(incoming.Type) ||
		!stringMapsEqual(local.Labels, incoming.Labels) ||
		!stringMapsEqual(local.Annotations, incoming.Annotations)
}

func stringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func cloneImportedRepos(cmd *cobra.Command, cfg *config.Config, bundle exportBundle, cwd string, dangerouslyDeleteExisting bool) error {
	_, err := cloneImportedReposWithProgress(cmd, cfg, bundle, cwd, dangerouslyDeleteExisting, nil)
	return err
}

func cloneImportedReposWithProgress(cmd *cobra.Command, cfg *config.Config, bundle exportBundle, cwd string, dangerouslyDeleteExisting bool, progress *syncProgressWriter) ([]engine.SyncResult, error) {
	return cloneImportedEntriesWithProgress(cmd, cfg, bundle, cwd, dangerouslyDeleteExisting, nil, progress)
}

func cloneImportedEntriesWithProgress(
	cmd *cobra.Command,
	cfg *config.Config,
	bundle exportBundle,
	cwd string,
	dangerouslyDeleteExisting bool,
	entries []registry.Entry,
	progress *syncProgressWriter,
) ([]engine.SyncResult, error) {
	if cfg == nil || cfg.Registry == nil {
		return nil, nil
	}

	eng := engine.New(cfg, cfg.Registry, vcs.NewGitAdapter(nil), vcs.NewGitErrorClassifier(), vcs.NewGitURLNormalizer(), nil)
	plan, err := eng.PlanImportClones(entries, engine.ImportCloneOptions{
		CWD:                       cwd,
		BundleRoot:                bundle.Root,
		DangerouslyDeleteExisting: dangerouslyDeleteExisting,
		ResolveTargetRelativePath: importTargetRelativePath,
	})
	if err != nil {
		return nil, err
	}

	failures, err := eng.ExecuteImportClones(cmd.Context(), plan, engine.ImportCloneCallbacks{
		OnStart: func(result engine.SyncResult) {
			if progress != nil {
				_ = progress.StartResult(result)
			}
		},
		OnComplete: func(result engine.SyncResult) {
			if progress != nil {
				_ = progress.WriteResult(result)
			}
		},
	})
	if err != nil {
		return failures, err
	}
	for _, skipped := range plan.Skipped {
		if skipped.Reason == "path is ignored by local config" {
			infof(cmd, "skipping import for %s: %s", skipped.Entry.RepoID, skipped.Reason)
			continue
		}
		infof(cmd, "skipping import clone for %s: %s", skipped.Entry.RepoID, skipped.Reason)
	}
	return failures, nil
}

func writeImportCloneFailureSummary(cmd *cobra.Command, failures []engine.SyncResult, cwd string) error {
	if len(failures) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Failed import clone operations:"); err != nil {
		return err
	}
	rows := make([][]string, 0, len(failures))
	for _, res := range failures {
		rows = append(rows, []string{
			displayRepoPath(res.Path, cwd, nil),
			res.ErrorClass,
			res.Error,
			res.RepoID,
		})
	}
	return cliio.WriteTable(cmd.ErrOrStderr(), false, false, []string{"PATH", "ERROR_CLASS", "ERROR", "REPO"}, rows)
}

func selectMergeCloneEntries(local, bundled *registry.Registry, policy importConflictPolicy) []registry.Entry {
	if bundled == nil || len(bundled.Entries) == 0 {
		return nil
	}
	selected := make([]registry.Entry, 0, len(bundled.Entries))
	for _, incoming := range bundled.Entries {
		if local == nil {
			selected = append(selected, incoming)
			continue
		}
		existing := local.FindByRepoID(incoming.RepoID)
		if existing == nil {
			selected = append(selected, incoming)
			continue
		}
		if policy == importConflictPolicyBundle && registryEntriesConflict(*existing, incoming) {
			selected = append(selected, incoming)
		}
	}
	return selected
}

func setRegistryEntryByRepoID(reg *registry.Registry, entry registry.Entry) {
	if reg == nil {
		return
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == entry.RepoID {
			reg.Entries[i] = entry
			return
		}
	}
	reg.Entries = append(reg.Entries, entry)
}

func removeRegistryEntryByRepoID(reg *registry.Registry, repoID string) {
	if reg == nil {
		return
	}
	out := reg.Entries[:0]
	for _, entry := range reg.Entries {
		if entry.RepoID == repoID {
			continue
		}
		out = append(out, entry)
	}
	reg.Entries = out
}

func importTargetRelativePath(entry registry.Entry, root string) string {
	if rel, ok := cleanRelativePath(entry.Path); ok {
		return rel
	}
	if rel, ok := relWithin(root, entry.Path); ok {
		return rel
	}
	if rel, ok := relFromRootBasename(root, entry.Path); ok {
		return rel
	}
	if rel, ok := relativeFromAbsolutePath(entry.Path); ok {
		return rel
	}

	base := filepath.Base(entry.Path)
	if base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}

	repoID := strings.TrimSpace(entry.RepoID)
	if repoID == "" {
		return "repo"
	}
	parts := strings.Split(repoID, "/")
	name := parts[len(parts)-1]
	if strings.TrimSpace(name) == "" {
		return "repo"
	}
	return name
}

func relFromRootBasename(root, target string) (string, bool) {
	rootClean := filepath.Clean(normalizePathLikeInput(root))
	targetRaw := normalizePathLikeInput(target)
	targetClean := filepath.Clean(targetRaw)
	if rootClean == "" || targetClean == "" || !isAbsoluteLikePath(targetRaw, targetClean) {
		return "", false
	}
	rootBase := filepath.Base(rootClean)
	if rootBase == "" || rootBase == "." || rootBase == string(filepath.Separator) {
		return "", false
	}
	pathWithoutVolume := strings.TrimPrefix(targetClean, filepath.VolumeName(targetClean))
	parts := strings.Split(filepath.ToSlash(pathWithoutVolume), "/")
	lastIdx := -1
	for i, part := range parts {
		if part == rootBase {
			lastIdx = i
		}
	}
	if lastIdx < 0 || lastIdx+1 >= len(parts) {
		return "", false
	}
	return cleanRelativePath(strings.Join(parts[lastIdx+1:], "/"))
}

func relativeFromAbsolutePath(path string) (string, bool) {
	raw := normalizePathLikeInput(path)
	cleaned := filepath.Clean(raw)
	if cleaned == "" || !isAbsoluteLikePath(raw, cleaned) {
		return "", false
	}
	withoutVolume := strings.TrimPrefix(cleaned, filepath.VolumeName(cleaned))
	withoutLeadingSeps := strings.TrimLeft(withoutVolume, `/\`)
	if withoutLeadingSeps == "" {
		return "", false
	}
	return cleanRelativePath(withoutLeadingSeps)
}

func dropIgnoredImportEntries(cfg *config.Config, bundle exportBundle, cwd string) {
	if cfg == nil || cfg.Registry == nil {
		return
	}
	ignored := ignoredPathSet(cfg)
	if len(ignored) == 0 {
		return
	}
	kept := make([]registry.Entry, 0, len(cfg.Registry.Entries))
	for _, entry := range cfg.Registry.Entries {
		if ignored[pathutil.CanonicalNormalize(entry.Path)] {
			continue
		}
		target := filepath.Clean(filepath.Join(cwd, importTargetRelativePath(entry, bundle.Root)))
		if ignored[pathutil.CanonicalNormalize(target)] {
			continue
		}
		kept = append(kept, entry)
	}
	cfg.Registry.Entries = kept
}

func ignoredPathSet(cfg *config.Config) map[string]bool {
	if cfg == nil {
		return make(map[string]bool)
	}
	pathSet := pathutil.IgnoredPathSet(cfg.IgnoredPaths, pathutil.CanonicalNormalize)
	out := make(map[string]bool, len(pathSet))
	for k := range pathSet {
		out[k] = true
	}
	return out
}
