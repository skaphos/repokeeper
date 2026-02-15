package repokeeper

import (
	"fmt"
	"os"

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
		cfgRoot := config.EffectiveRoot(cfgPath, cfg)

		registryOverride, _ := cmd.Flags().GetString("registry")
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
		reg.Entries = append(reg.Entries[:idx], reg.Entries[idx+1:]...)

		if registryOverride != "" {
			if err := registry.Save(reg, registryOverride); err != nil {
				return err
			}
		} else {
			cfg.Registry = reg
			if err := config.Save(cfg, cfgPath); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "deleted %s (%s)\n", entry.RepoID, entry.Path); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().String("registry", "", "override registry file path")
	rootCmd.AddCommand(deleteCmd)
}
