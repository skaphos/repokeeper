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
		if _, err := os.Stat(cfgPath); err == nil && !force {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", cfgPath)
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote config to %s\n", cfgPath)
		return nil
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config without prompting")

	rootCmd.AddCommand(initCmd)
}
