package repokeeper

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mfacenet/repokeeper/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a RepoKeeper configuration",
	Long:  "Creates a RepoKeeper config file in the current directory by default.",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		machineID, _ := cmd.Flags().GetString("machine-id")
		roots, _ := cmd.Flags().GetString("roots")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfgPath, err := config.InitConfigPath(flagConfig, cwd)
		if err != nil {
			return err
		}
		if _, err := os.Stat(cfgPath); err == nil && !force {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", cfgPath)
		}

		cfg := config.DefaultConfig()
		cfg.Roots = []string{cwd}
		if machineID != "" {
			cfg.MachineID = machineID
		} else {
			cfg.MachineID = config.GenerateMachineID()
		}

		if nonInteractive {
			if strings.TrimSpace(roots) != "" {
				cfg.Roots = splitCSV(roots)
			}
			if len(cfg.Roots) == 0 {
				return fmt.Errorf("no roots provided; use --roots")
			}
		} else {
			reader := bufio.NewReader(cmd.InOrStdin())
			cfg.MachineID = prompt(reader, cmd, "Machine ID", cfg.MachineID)
			rootInput := prompt(reader, cmd, "Roots (comma-separated)", strings.Join(cfg.Roots, ","))
			cfg.Roots = splitCSV(rootInput)
			excludeInput := prompt(reader, cmd, "Exclude patterns (comma-separated)", strings.Join(cfg.Exclude, ","))
			cfg.Exclude = splitCSV(excludeInput)
		}

		if cfg.RegistryPath == "" {
			cfg.RegistryPath = filepath.Join(filepath.Dir(cfgPath), "registry.yaml")
		}

		if err := config.Save(&cfg, cfgPath); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote config to %s\n", cfgPath)
		return nil
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config without prompting")
	initCmd.Flags().String("machine-id", "", "set machine ID non-interactively")
	initCmd.Flags().String("roots", "", "comma-separated root directories")
	initCmd.Flags().Bool("non-interactive", false, "use defaults/flags, no prompts")

	rootCmd.AddCommand(initCmd)
}

func prompt(reader *bufio.Reader, cmd *cobra.Command, label, def string) string {
	if def != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [%s]: ", label, def)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}
