package repokeeper

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type exportBundle struct {
	Version    int                `yaml:"version"`
	ExportedAt string             `yaml:"exported_at"`
	Root       string             `yaml:"root,omitempty"`
	Config     config.Config      `yaml:"config"`
	Registry   *registry.Registry `yaml:"registry,omitempty"`
}

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

var exportCmd = &cobra.Command{
	Use:   "export [output-file|-]",
	Short: "Export config (and optionally registry) for reuse on another machine",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfgPath, err := config.ResolveConfigPath(configOverride(cmd), cwd)
		if err != nil {
			return err
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		includeRegistry, _ := cmd.Flags().GetBool("include-registry")
		outputPath, _ := cmd.Flags().GetString("output")
		if len(args) == 1 {
			outputPath = strings.TrimSpace(args[0])
		}
		if outputPath == "" {
			return fmt.Errorf("output path cannot be empty")
		}

		bundle := exportBundle{
			Version:    1,
			ExportedAt: time.Now().Format(time.RFC3339),
			Root:       config.ConfigRoot(cfgPath),
		}
		cfgCopy := *cfg
		if includeRegistry {
			if cfgCopy.Registry == nil && cfgCopy.RegistryPath != "" {
				reg, err := registry.Load(config.ResolveRegistryPath(cfgPath, cfgCopy.RegistryPath))
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				if err == nil {
					cfgCopy.Registry = reg
				}
			}
			cfgCopy.Registry = cloneRegistry(cfgCopy.Registry)
			populateExportBranches(cmd.Context(), cfgCopy.Registry, vcs.NewGitAdapter(nil).Head)
			bundle.Registry = cfgCopy.Registry
		} else {
			cfgCopy.Registry = nil
			bundle.Registry = nil
		}
		bundle.Config = cfgCopy

		data, err := yaml.Marshal(&bundle)
		if err != nil {
			return err
		}
		if outputPath == "-" {
			if _, err := cmd.OutOrStdout().Write(data); err != nil {
				return err
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, data, 0o644); err != nil {
			return err
		}
		infof(cmd, "exported bundle to %s", outputPath)
		return nil
	},
}

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
		// Import writes local workspace config by default so migration bundles
		// land in the current directory unless explicitly overridden.
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
	exportCmd.Flags().String("output", "-", "output file path or - for stdout")
	exportCmd.Flags().Bool("include-registry", true, "include registry in the export bundle")

	importCmd.Flags().Bool("force", false, "overwrite existing config file")
	importCmd.Flags().String("mode", string(importModeMerge), "import mode: merge or replace")
	importCmd.Flags().String("on-conflict", string(importConflictPolicyBundle), "when mode=merge and repo_id exists locally: skip, bundle, or local")
	importCmd.Flags().Bool("include-registry", true, "import bundled registry when present")
	importCmd.Flags().Bool("preserve-registry-path", false, "keep bundled registry_path (resolved relative to imported config file)")
	importCmd.Flags().Bool("dangerously-delete-existing", false, "dangerous: delete conflicting target repo paths before cloning")
	importCmd.Flags().Bool("file-only", false, "import config file only (disable registry import and cloning)")

	rootCmd.AddCommand(exportCmd)
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
	// Merge mode: keep existing registry and merge bundled entries by repo_id.
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
			// Keep local entry as-is.
		}
	}
	cfg.Registry.UpdatedAt = time.Now()
}

func registryEntriesConflict(local, incoming registry.Entry) bool {
	return strings.TrimSpace(local.Path) != strings.TrimSpace(incoming.Path) ||
		strings.TrimSpace(local.RemoteURL) != strings.TrimSpace(incoming.RemoteURL) ||
		strings.TrimSpace(local.Branch) != strings.TrimSpace(incoming.Branch) ||
		strings.TrimSpace(local.Type) != strings.TrimSpace(incoming.Type)
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
	if entries == nil {
		entries = cfg.Registry.Entries
	}
	if len(entries) == 0 {
		return nil, nil
	}

	adapter := vcs.NewGitAdapter(nil)
	targets := make(map[string]registry.Entry, len(entries))
	skippedLocal := make(map[string]registry.Entry)
	for _, entry := range entries {
		targetRel := importTargetRelativePath(entry, bundle.Root)
		target := filepath.Clean(filepath.Join(cwd, targetRel))

		// Protect against path traversal/out-of-tree paths from malformed bundles.
		relToCWD, err := filepath.Rel(cwd, target)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relToCWD, ".."+string(filepath.Separator)) || relToCWD == ".." {
			return nil, fmt.Errorf("refusing to clone outside current directory: %q", target)
		}
		if _, exists := targets[target]; exists {
			return nil, fmt.Errorf("multiple repos resolve to same target path %q", target)
		}
		targets[target] = entry

		remoteURL := strings.TrimSpace(entry.RemoteURL)
		if remoteURL == "" {
			if strings.HasPrefix(strings.TrimSpace(entry.RepoID), "local:") {
				skippedLocal[target] = entry
				continue
			}
			return nil, fmt.Errorf("cannot clone %q: missing remote_url in bundle", entry.RepoID)
		}
	}
	if !dangerouslyDeleteExisting {
		conflicts := findImportTargetConflicts(targets, skippedLocal)
		if len(conflicts) > 0 {
			var lines []string
			for _, conflict := range conflicts {
				lines = append(lines, fmt.Sprintf("%s (repo: %s)", conflict.target, conflict.entry.RepoID))
			}
			return nil, fmt.Errorf(
				"import target conflicts detected under %s:\n- %s\nre-run with --dangerously-delete-existing to replace these paths",
				cwd,
				strings.Join(lines, "\n- "),
			)
		}
	}

	failures := make([]engine.SyncResult, 0)
	for target, entry := range targets {
		if _, skip := skippedLocal[target]; skip {
			continue
		}
		result := engine.SyncResult{RepoID: entry.RepoID, Path: target, Action: "git clone"}
		if progress != nil {
			_ = progress.StartResult(result)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return failures, err
		}
		if _, err := os.Stat(target); err == nil {
			if !dangerouslyDeleteExisting {
				return failures, fmt.Errorf("target path already exists: %q (use --dangerously-delete-existing to replace)", target)
			}
			if err := os.RemoveAll(target); err != nil {
				return failures, fmt.Errorf("failed to remove existing path %q: %w", target, err)
			}
		} else if !os.IsNotExist(err) {
			return failures, err
		}

		cloneArgs := []string{"clone"}
		if entry.Type == "mirror" {
			cloneArgs = append(cloneArgs, "--mirror")
		} else if strings.TrimSpace(entry.Branch) != "" {
			// Preserve the exported branch so imported checkouts land on the same branch.
			cloneArgs = append(cloneArgs, "--branch", strings.TrimSpace(entry.Branch), "--single-branch")
		}
		cloneArgs = append(cloneArgs, strings.TrimSpace(entry.RemoteURL), target)
		if err := adapter.Clone(cmd.Context(), strings.TrimSpace(entry.RemoteURL), target, strings.TrimSpace(entry.Branch), entry.Type == "mirror"); err != nil {
			result.OK = false
			result.ErrorClass = "unknown"
			result.Error = fmt.Sprintf("git %q: %v", strings.Join(cloneArgs, " "), err)
			failures = append(failures, result)
			if progress != nil {
				_ = progress.WriteResult(result)
			}
			continue
		}
		entry.Path = target
		entry.Status = registry.StatusPresent
		entry.LastSeen = time.Now()
		setRegistryEntryByRepoID(cfg.Registry, entry)
		result.OK = true
		if progress != nil {
			_ = progress.WriteResult(result)
		}
	}
	for target, entry := range skippedLocal {
		entry.Path = target
		entry.Status = registry.StatusMissing
		entry.LastSeen = time.Now()
		setRegistryEntryByRepoID(cfg.Registry, entry)
		infof(cmd, "skipping local-only repo %s: missing remote_url", entry.RepoID)
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

type importConflict struct {
	target string
	entry  registry.Entry
}

func findImportTargetConflicts(targets map[string]registry.Entry, skippedLocal map[string]registry.Entry) []importConflict {
	conflicts := make([]importConflict, 0)
	for target, entry := range targets {
		if _, skip := skippedLocal[target]; skip {
			// Local-only entries are intentionally skipped and should not block import.
			continue
		}
		if _, err := os.Stat(target); err == nil {
			conflicts = append(conflicts, importConflict{target: target, entry: entry})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].target < conflicts[j].target
	})
	return conflicts
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

func importTargetRelativePath(entry registry.Entry, root string) string {
	if rel, ok := relWithin(root, entry.Path); ok {
		// Keep exported layout stable when the path is under the exported config root.
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

func cloneRegistry(reg *registry.Registry) *registry.Registry {
	if reg == nil {
		return nil
	}
	clone := *reg
	clone.Entries = append([]registry.Entry(nil), reg.Entries...)
	return &clone
}

func populateExportBranches(
	ctx context.Context,
	reg *registry.Registry,
	headFn func(context.Context, string) (model.Head, error),
) {
	if reg == nil || headFn == nil {
		return
	}
	for i := range reg.Entries {
		entry := reg.Entries[i]
		if entry.Status != registry.StatusPresent || entry.Type == "mirror" {
			continue
		}
		path := strings.TrimSpace(entry.Path)
		if path == "" {
			continue
		}
		head, err := headFn(ctx, path)
		if err != nil || head.Detached {
			// Detached heads are not stable branch selections for import replay.
			continue
		}
		branch := strings.TrimSpace(head.Branch)
		if branch == "" {
			continue
		}
		reg.Entries[i].Branch = branch
	}
}
