// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <repo-id-or-path> <new-path>",
	Short: "Move a tracked repository directory and update its registry path",
	Args:  cobra.ExactArgs(2),
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
		idx := findRegistryEntryIndex(reg.Entries, entry)
		if idx < 0 {
			return fmt.Errorf("entry not found for selector %q", args[0])
		}

		targetAbs, err := filepath.Abs(filepath.Join(cwd, args[1]))
		if err != nil {
			return err
		}
		targetAbs = filepath.Clean(targetAbs)
		if targetAbs == filepath.Clean(entry.Path) {
			return fmt.Errorf("target path is unchanged: %q", targetAbs)
		}

		if _, err := os.Stat(entry.Path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("cannot move %q: source path does not exist", entry.Path)
			}
			return fmt.Errorf("cannot inspect source path %q: %w", entry.Path, err)
		}
		if _, err := os.Stat(targetAbs); err == nil {
			return fmt.Errorf("cannot move to %q: target already exists", targetAbs)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot inspect target path %q: %w", targetAbs, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return fmt.Errorf("prepare target parent %q: %w", filepath.Dir(targetAbs), err)
		}

		oldPath := entry.Path
		if err := os.Rename(oldPath, targetAbs); err != nil {
			return fmt.Errorf("failed to move repository directory from %q to %q: %w", oldPath, targetAbs, err)
		}

		entry.Path = targetAbs
		entry.Status = registry.StatusPresent
		entry.LastSeen = time.Now()
		reg.Entries[idx] = entry
		reg.UpdatedAt = time.Now()

		saveErr := func() error {
			if registryOverride != "" {
				return registry.Save(reg, registryOverride)
			}
			cfg.Registry = reg
			return config.Save(cfg, cfgPath)
		}()
		if saveErr != nil {
			if rollbackErr := os.Rename(targetAbs, oldPath); rollbackErr != nil {
				return fmt.Errorf(
					"saved registry update failed after move (%v), and rollback failed moving %q back to %q: %v",
					saveErr,
					targetAbs,
					oldPath,
					rollbackErr,
				)
			}
			reg.Entries[idx].Path = oldPath
			reg.Entries[idx].Status = registry.StatusPresent
			return fmt.Errorf("registry update failed; reverted filesystem move from %q back to %q: %w", targetAbs, oldPath, saveErr)
		}

		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "moved %s from %s to %s\n", entry.RepoID, oldPath, targetAbs); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	moveCmd.Flags().String("registry", "", "override registry file path")
	rootCmd.AddCommand(moveCmd)
}
