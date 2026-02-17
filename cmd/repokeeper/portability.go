// SPDX-License-Identifier: MIT
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
	"github.com/skaphos/repokeeper/internal/gitx"
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
		var exportedRegistry *registry.Registry
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
			exportedRegistry = cloneRegistry(cfgCopy.Registry)
			if inferredRoot := inferRegistrySharedRoot(exportedRegistry); inferredRoot != "" {
				bundle.Root = inferredRoot
			}
			exportedRegistry = prepareRegistryForExport(exportedRegistry, bundle.Root)
			adapter := vcs.NewGitAdapter(nil)
			populateExportBranches(cmd.Context(), exportedRegistry, adapter.Head, adapter.TrackingStatus)
		}
		// Export bundles carry registry at the top level only.
		// config.registry is always omitted to avoid duplicated payloads.
		cfgCopy.Registry = nil
		bundle.Registry = exportedRegistry
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
	if entries == nil {
		entries = cfg.Registry.Entries
	}
	if len(entries) == 0 {
		return nil, nil
	}

	adapter := vcs.NewGitAdapter(nil)
	ignored := ignoredPathSet(cfg)
	targets := make(map[string]registry.Entry, len(entries))
	skipped := make(map[string]registry.Entry)
	skipReasons := make(map[string]string)
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

		if ignored[target] {
			skipped[target] = entry
			skipReasons[target] = "path is ignored by local config"
			continue
		}

		if entry.Status == registry.StatusMissing {
			skipped[target] = entry
			skipReasons[target] = "marked missing in bundle"
			continue
		}

		remoteURL := strings.TrimSpace(entry.RemoteURL)
		if remoteURL == "" {
			// @todo(milestone-8): Preserve upstream-missing intent from source scan/export
			// so import/reconcile classify these as explicit "skipped no upstream"
			// instead of falling through to clone-time failures.
			skipped[target] = entry
			skipReasons[target] = "no remote URL configured"
			continue
		}

		if entry.Type != "mirror" && strings.TrimSpace(entry.Branch) == "" {
			skipped[target] = entry
			skipReasons[target] = "no upstream branch configured"
			continue
		}
	}
	if !dangerouslyDeleteExisting {
		conflicts := findImportTargetConflicts(targets, skipped)
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
		if _, skip := skipped[target]; skip {
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

		if err := adapter.Clone(cmd.Context(), strings.TrimSpace(entry.RemoteURL), target, strings.TrimSpace(entry.Branch), entry.Type == "mirror"); err != nil {
			result.OK = false
			result.ErrorClass = gitx.ClassifyError(err)
			result.Error = importCloneFailureMessage(result.ErrorClass)
			entry.Path = target
			entry.Status = registry.StatusMissing
			entry.LastSeen = time.Now()
			setRegistryEntryByRepoID(cfg.Registry, entry)
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
	for target, entry := range skipped {
		if skipReasons[target] == "path is ignored by local config" {
			removeRegistryEntryByRepoID(cfg.Registry, entry.RepoID)
			infof(cmd, "skipping import for %s: %s", entry.RepoID, skipReasons[target])
			continue
		}
		entry.Path = target
		if entry.Status == "" ||
			skipReasons[target] == "no remote URL configured" ||
			skipReasons[target] == "no upstream branch configured" {
			entry.Status = registry.StatusMissing
		}
		entry.LastSeen = time.Now()
		setRegistryEntryByRepoID(cfg.Registry, entry)
		infof(cmd, "skipping import clone for %s: %s", entry.RepoID, skipReasons[target])
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
		// Keep exported layout stable when the path is under the exported config root.
		return rel
	}
	if rel, ok := relFromRootBasename(root, entry.Path); ok {
		// If roots differ across machines but share a stable folder name
		// (for example ".../workspace/project"), keep the project-relative suffix.
		return rel
	}
	if rel, ok := relativeFromAbsolutePath(entry.Path); ok {
		// Preserve layout details from absolute bundle paths instead of flattening.
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

func prepareRegistryForExport(reg *registry.Registry, root string) *registry.Registry {
	if reg == nil {
		return nil
	}
	out := cloneRegistry(reg)
	out.UpdatedAt = time.Time{}
	for i := range out.Entries {
		entry := out.Entries[i]
		entry.LastSeen = time.Time{}
		entry.Path = exportEntryPath(entry.Path, root)
		out.Entries[i] = entry
	}
	return out
}

func exportEntryPath(path, root string) string {
	if rel, ok := relWithin(root, path); ok {
		return rel
	}
	if rel, ok := cleanRelativePath(path); ok {
		return rel
	}
	return filepath.Clean(path)
}

func cleanRelativePath(path string) (string, bool) {
	raw := normalizePathLikeInput(path)
	cleaned := filepath.Clean(raw)
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) || filepath.IsAbs(cleaned) {
		return "", false
	}
	rel := filepath.ToSlash(cleaned)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func inferRegistrySharedRoot(reg *registry.Registry) string {
	if reg == nil || len(reg.Entries) == 0 {
		return ""
	}
	absCount := 0
	firstAbsPath := ""
	var root string
	for _, entry := range reg.Entries {
		path := filepath.Clean(strings.TrimSpace(entry.Path))
		if path == "" || !filepath.IsAbs(path) {
			continue
		}
		absCount++
		if firstAbsPath == "" {
			firstAbsPath = path
		}
		if root == "" {
			root = path
			continue
		}
		root = commonPathRoot(root, path)
		if root == "" || root == string(filepath.Separator) {
			return root
		}
	}
	if absCount == 1 && firstAbsPath != "" {
		return filepath.Dir(firstAbsPath)
	}
	return root
}

func commonPathRoot(left, right string) string {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	if left == "" || right == "" {
		return ""
	}
	sep := string(filepath.Separator)
	leftParts := strings.Split(left, sep)
	rightParts := strings.Split(right, sep)
	limit := len(leftParts)
	if len(rightParts) < limit {
		limit = len(rightParts)
	}
	shared := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		if leftParts[i] != rightParts[i] {
			break
		}
		shared = append(shared, leftParts[i])
	}
	if len(shared) == 0 {
		return ""
	}
	if len(shared) == 1 && shared[0] == "" {
		return sep
	}
	return filepath.Clean(strings.Join(shared, sep))
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
		if ignored[filepath.Clean(entry.Path)] {
			continue
		}
		target := filepath.Clean(filepath.Join(cwd, importTargetRelativePath(entry, bundle.Root)))
		if ignored[target] {
			continue
		}
		kept = append(kept, entry)
	}
	cfg.Registry.Entries = kept
}

func ignoredPathSet(cfg *config.Config) map[string]bool {
	out := make(map[string]bool)
	if cfg == nil {
		return out
	}
	for _, p := range cfg.IgnoredPaths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out[filepath.Clean(p)] = true
	}
	return out
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
	trackingFn func(context.Context, string) (model.Tracking, error),
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
		if trackingFn != nil {
			tracking, err := trackingFn(ctx, path)
			if err == nil && strings.TrimSpace(tracking.Upstream) == "" {
				// Preserve "no upstream configured" intent in export bundles so
				// import/reconcile can skip checkout attempts deterministically.
				reg.Entries[i].Branch = ""
				continue
			}
		}
		reg.Entries[i].Branch = branch
	}
}

func importCloneFailureMessage(errorClass string) string {
	switch strings.TrimSpace(errorClass) {
	case "auth":
		return "import-clone-auth"
	case "network":
		return "import-clone-network"
	case "timeout":
		return "import-clone-timeout"
	case "corrupt":
		return "import-clone-corrupt"
	case "missing_remote":
		return "import-clone-missing-remote"
	default:
		return "import-clone-failed"
	}
}

func normalizePathLikeInput(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	// Treat backslashes as separators to keep imported bundles portable
	// across operating systems.
	return strings.ReplaceAll(trimmed, "\\", string(filepath.Separator))
}

func relFromRootBasename(root, target string) (string, bool) {
	rootClean := filepath.Clean(normalizePathLikeInput(root))
	targetClean := filepath.Clean(normalizePathLikeInput(target))
	if rootClean == "" || targetClean == "" || !filepath.IsAbs(targetClean) {
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
	cleaned := filepath.Clean(normalizePathLikeInput(path))
	if cleaned == "" || !filepath.IsAbs(cleaned) {
		return "", false
	}
	withoutVolume := strings.TrimPrefix(cleaned, filepath.VolumeName(cleaned))
	withoutLeadingSeps := strings.TrimLeft(withoutVolume, `/\`)
	if withoutLeadingSeps == "" {
		return "", false
	}
	return cleanRelativePath(withoutLeadingSeps)
}
