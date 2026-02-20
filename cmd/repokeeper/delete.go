// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <repo-id-or-path>",
	Short: "Remove a repository from the registry",
	Args:  cobra.ExactArgs(1),
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
		cfgRoot := config.EffectiveRoot(cfgPath)
		trackingOnly, _ := cmd.Flags().GetBool("tracking-only")

		registryOverride, _ := cmd.Flags().GetString("registry")
		if trackingOnly && registryOverride != "" {
			return fmt.Errorf("--tracking-only is not supported with --registry override (ignored paths are stored in config)")
		}
		var reg *registry.Registry
		if registryOverride != "" {
			reg, err = registry.Load(registryOverride)
			if err != nil {
				return err
			}
		} else {
			reg = cfg.Registry
			if reg == nil {
				return fmt.Errorf("registry not found in %q (run repokeeper scan first)", cfgPath)
			}
		}

		entry, err := selectRegistryEntryForDescribe(reg.Entries, args[0], cwd, []string{cfgRoot})
		if err != nil {
			return err
		}
		if !assumeYes(cmd) {
			prompt := fmt.Sprintf("Delete %s at %s and remove from registry? [y/N]: ", entry.RepoID, entry.Path)
			if trackingOnly {
				prompt = fmt.Sprintf("Stop tracking %s and ignore %s for future scans/imports? [y/N]: ", entry.RepoID, entry.Path)
			}
			confirmed, err := confirmWithPrompt(
				cmd,
				prompt,
			)
			if err != nil {
				return err
			}
			if !confirmed {
				infof(cmd, "delete cancelled")
				return nil
			}
		}

		idx := -1
		for i := range reg.Entries {
			if reg.Entries[i].RepoID == entry.RepoID && reg.Entries[i].Path == entry.Path {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("entry not found for selector %q", args[0])
		}
		if !trackingOnly {
			if err := os.RemoveAll(entry.Path); err != nil {
				return fmt.Errorf("delete repository path %q: %w", entry.Path, err)
			}
		}
		reg.Entries = append(reg.Entries[:idx], reg.Entries[idx+1:]...)

		if registryOverride != "" {
			if err := registry.Save(reg, registryOverride); err != nil {
				return err
			}
		} else {
			cfg.Registry = reg
			if trackingOnly {
				ignored := filepath.Clean(entry.Path)
				if !slices.Contains(cfg.IgnoredPaths, ignored) {
					cfg.IgnoredPaths = append(cfg.IgnoredPaths, ignored)
				}
			}
			if err := config.Save(cfg, cfgPath); err != nil {
				return err
			}
		}

		if trackingOnly {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "stopped tracking %s (%s)\n", entry.RepoID, entry.Path); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "warning: future scan/import runs will ignore this path and not add it back to the registry"); err != nil {
				return err
			}
			return nil
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "deleted %s (%s)\n", entry.RepoID, entry.Path); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().String("registry", "", "override registry file path")
	deleteCmd.Flags().Bool("tracking-only", false, "remove from registry only and ignore this path in future scan/import runs")
	rootCmd.AddCommand(deleteCmd)
}
