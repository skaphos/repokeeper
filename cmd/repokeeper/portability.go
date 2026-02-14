package repokeeper

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
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
		cfgPath, err := config.ResolveConfigPath(flagConfig, cwd)
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
				reg, err := registry.Load(cfgCopy.RegistryPath)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				if err == nil {
					cfgCopy.Registry = reg
				}
			}
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
	Use:   "import",
	Short: "Import an exported config bundle",
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath, _ := cmd.Flags().GetString("input")
		force, _ := cmd.Flags().GetBool("force")
		includeRegistry, _ := cmd.Flags().GetBool("include-registry")
		preserveRegistryPath, _ := cmd.Flags().GetBool("preserve-registry-path")

		if inputPath == "" {
			return fmt.Errorf("input path is required")
		}
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return err
		}
		var bundle exportBundle
		if err := yaml.Unmarshal(data, &bundle); err != nil {
			return err
		}

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

	importCmd.Flags().String("input", "", "path to exported bundle file")
	importCmd.Flags().Bool("force", false, "overwrite existing config file")
	importCmd.Flags().Bool("include-registry", true, "import bundled registry when present")
	importCmd.Flags().Bool("preserve-registry-path", false, "keep bundled registry_path instead of rewriting beside imported config")

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
}
