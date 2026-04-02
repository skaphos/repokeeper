// SPDX-License-Identifier: MIT
package repokeeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
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
			Version:    currentExportBundleVersion,
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

func init() {
	exportCmd.Flags().String("output", "-", "output file path or - for stdout")
	exportCmd.Flags().Bool("include-registry", true, "include registry in the export bundle")

	rootCmd.AddCommand(exportCmd)
}

func prepareRegistryForExport(reg *registry.Registry, root string) *registry.Registry {
	if reg == nil {
		return nil
	}
	out := cloneRegistry(reg)
	out.UpdatedAt = time.Time{}
	filtered := make([]registry.Entry, 0, len(out.Entries))
	for i := range out.Entries {
		entry := out.Entries[i]
		if entry.Status != registry.StatusPresent {
			continue
		}
		entry.LastSeen = time.Time{}
		entry.RepoMetadataFile = ""
		entry.RepoMetadataError = ""
		entry.RepoMetadataFingerprint = ""
		entry.RepoMetadata = nil
		entry.Path = exportEntryPath(entry.Path, root)
		filtered = append(filtered, entry)
	}
	out.Entries = filtered
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

func inferRegistrySharedRoot(reg *registry.Registry) string {
	if reg == nil || len(reg.Entries) == 0 {
		return ""
	}
	absCount := 0
	firstAbsPath := ""
	var root string
	for _, entry := range reg.Entries {
		if entry.Status != registry.StatusPresent {
			continue
		}
		path := filepath.Clean(strings.TrimSpace(entry.Path))
		if path == "" || !isAbsoluteLikePath(entry.Path, path) {
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
