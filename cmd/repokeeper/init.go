// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"os"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/sortutil"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a RepoKeeper configuration",
	Long:  "Creates a RepoKeeper config file in the current directory by default.",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfgPath, err := config.InitConfigPath(configOverride(cmd), cwd)
		if err != nil {
			return err
		}
		if _, err := os.Stat(cfgPath); err == nil {
			if !force {
				return fmt.Errorf("config already exists at %q (use --force to overwrite)", cfgPath)
			}
			// Ensure forced init replaces the existing config file rather than
			// preserving any prior on-disk content.
			if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove existing config %q: %w", cfgPath, err)
			}
		}

		cfg := config.DefaultConfig()
		cfg.RegistryPath = ""
		cfg.Registry = &registry.Registry{}

		if err := config.Save(&cfg, cfgPath); err != nil {
			return err
		}

		eng := engine.New(&cfg, cfg.Registry, vcs.NewGitAdapter(nil))
		if _, err := eng.Scan(cmd.Context(), engine.ScanOptions{Roots: []string{config.ConfigRoot(cfgPath)}}); err != nil {
			return err
		}
		sortutil.SortRegistryEntries(cfg.Registry.Entries)
		if err := config.Save(&cfg, cfgPath); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Wrote config to %s\n", cfgPath); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config without prompting")

	rootCmd.AddCommand(initCmd)
}
