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

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type exportBundle struct {
	Version    int                `yaml:"version"`
	ExportedAt string             `yaml:"exported_at"`
	Config     config.Config      `yaml:"config"`
	Registry   *registry.Registry `yaml:"registry,omitempty"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export config (and optionally registry) for reuse on another machine",
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

		bundle := exportBundle{
			Version:    1,
			ExportedAt: time.Now().Format(time.RFC3339),
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
			_, _ = cmd.OutOrStdout().Write(data)
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
		includeRegistry, _ := cmd.Flags().GetBool("include-registry")
		preserveRegistryPath, _ := cmd.Flags().GetBool("preserve-registry-path")
		cloneRepos, _ := cmd.Flags().GetBool("clone")
		dangerouslyDeleteExisting, _ := cmd.Flags().GetBool("dangerously-delete-existing")
		fileOnly, _ := cmd.Flags().GetBool("file-only")

		if fileOnly {
			includeRegistry = false
			cloneRepos = false
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
		// Use runtime resolution semantics so import targets the same config
		// location the app would read by default.
		cfgPath, err := config.ResolveConfigPath(configOverride(cmd), cwd)
		if err != nil {
			return err
		}
		if _, err := os.Stat(cfgPath); err == nil && !force {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", cfgPath)
		}

		cfg := bundle.Config
		if includeRegistry {
			if cfg.Registry == nil && bundle.Registry != nil {
				cfg.Registry = bundle.Registry
			}
		} else {
			cfg.Registry = nil
		}
		if !preserveRegistryPath {
			cfg.RegistryPath = ""
		}
		if cloneRepos {
			if err := cloneImportedRepos(cmd, &cfg, bundle, cwd, dangerouslyDeleteExisting); err != nil {
				return err
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
	exportCmd.Flags().String("output", "repokeeper-export.yaml", "output file path or - for stdout")
	exportCmd.Flags().Bool("include-registry", true, "include registry in the export bundle")

	importCmd.Flags().Bool("force", false, "overwrite existing config file")
	importCmd.Flags().Bool("include-registry", true, "import bundled registry when present")
	importCmd.Flags().Bool("preserve-registry-path", false, "keep bundled registry_path (resolved relative to imported config file)")
	importCmd.Flags().Bool("clone", true, "clone repos from imported registry into target paths under the current directory")
	importCmd.Flags().Bool("dangerously-delete-existing", false, "dangerous: delete conflicting target repo paths before cloning")
	importCmd.Flags().Bool("file-only", false, "import config file only (disable registry import and cloning)")

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
}

func cloneImportedRepos(cmd *cobra.Command, cfg *config.Config, bundle exportBundle, cwd string, dangerouslyDeleteExisting bool) error {
	if cfg == nil || cfg.Registry == nil || len(cfg.Registry.Entries) == 0 {
		return nil
	}

	adapter := vcs.NewGitAdapter(nil)
	targets := make(map[string]registry.Entry, len(cfg.Registry.Entries))
	skippedLocal := make(map[string]registry.Entry)
	for _, entry := range cfg.Registry.Entries {
		targetRel := importTargetRelativePath(entry, nil)
		target := filepath.Clean(filepath.Join(cwd, targetRel))

		// Protect against path traversal/out-of-tree paths from malformed bundles.
		relToCWD, err := filepath.Rel(cwd, target)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relToCWD, ".."+string(filepath.Separator)) || relToCWD == ".." {
			return fmt.Errorf("refusing to clone outside current directory: %s", target)
		}
		if _, exists := targets[target]; exists {
			return fmt.Errorf("multiple repos resolve to same target path %q", target)
		}
		targets[target] = entry

		remoteURL := strings.TrimSpace(entry.RemoteURL)
		if remoteURL == "" {
			if strings.HasPrefix(strings.TrimSpace(entry.RepoID), "local:") {
				skippedLocal[target] = entry
				continue
			}
			return fmt.Errorf("cannot clone %q: missing remote_url in bundle", entry.RepoID)
		}
	}
	if !dangerouslyDeleteExisting {
		conflicts := findImportTargetConflicts(targets, skippedLocal)
		if len(conflicts) > 0 {
			var lines []string
			for _, conflict := range conflicts {
				lines = append(lines, fmt.Sprintf("%s (repo: %s)", conflict.target, conflict.entry.RepoID))
			}
			return fmt.Errorf(
				"import target conflicts detected under %s:\n- %s\nre-run with --dangerously-delete-existing to replace these paths",
				cwd,
				strings.Join(lines, "\n- "),
			)
		}
	}

	for target, entry := range targets {
		if _, skip := skippedLocal[target]; skip {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(target); err == nil {
			if !dangerouslyDeleteExisting {
				return fmt.Errorf("target path already exists: %s (use --dangerously-delete-existing to replace)", target)
			}
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("failed to remove existing path %s: %w", target, err)
			}
		} else if !os.IsNotExist(err) {
			return err
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
			return fmt.Errorf("git %s: %w", strings.Join(cloneArgs, " "), err)
		}
		entry.Path = target
		entry.Status = registry.StatusPresent
		entry.LastSeen = time.Now()
		setRegistryEntryByRepoID(cfg.Registry, entry)
	}
	for target, entry := range skippedLocal {
		entry.Path = target
		entry.Status = registry.StatusMissing
		entry.LastSeen = time.Now()
		setRegistryEntryByRepoID(cfg.Registry, entry)
		infof(cmd, "skipping local-only repo %s: missing remote_url", entry.RepoID)
	}
	return nil
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

func importTargetRelativePath(entry registry.Entry, roots []string) string {
	for _, root := range roots {
		rel, ok := relWithin(root, entry.Path)
		if ok {
			// Keep exported layout stable when the path is under a configured root.
			return rel
		}
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
