// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label <repo-id-or-path>",
	Short: "View or update labels for a tracked repository",
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

		setInputs, _ := cmd.Flags().GetStringArray("set")
		removeInputs, _ := cmd.Flags().GetStringArray("remove")
		setValues, err := parseMetadataAssignments(setInputs, "--set")
		if err != nil {
			return err
		}
		removeKeys, err := parseMetadataKeys(removeInputs, "--remove")
		if err != nil {
			return err
		}

		if len(setValues) > 0 || len(removeKeys) > 0 {
			entry.Labels = cloneMetadataMap(entry.Labels)
			if entry.Labels == nil {
				entry.Labels = make(map[string]string)
			}
			for key, value := range setValues {
				entry.Labels[key] = value
			}
			for _, key := range removeKeys {
				delete(entry.Labels, key)
			}
			entry.Labels = normalizeMetadataMap(entry.Labels)
			entry.LastSeen = time.Now()
			reg.Entries[idx] = entry
			reg.UpdatedAt = time.Now()

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
		}

		format, _ := cmd.Flags().GetString("format")
		switch strings.ToLower(strings.TrimSpace(format)) {
		case "json":
			payload := struct {
				RepoID string            `json:"repo_id"`
				Path   string            `json:"path"`
				Labels map[string]string `json:"labels,omitempty"`
			}{
				RepoID: entry.RepoID,
				Path:   entry.Path,
				Labels: entry.Labels,
			}
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
				return err
			}
			return nil
		case "", "table":
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "REPO: %s\n", entry.RepoID); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "PATH: %s\n", entry.Path); err != nil {
				return err
			}
			if len(entry.Labels) == 0 {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "LABELS: -"); err != nil {
					return err
				}
				return nil
			}
			keys := make([]string, 0, len(entry.Labels))
			for key := range entry.Labels {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "LABELS:"); err != nil {
				return err
			}
			for _, key := range keys {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s=%s\n", key, entry.Labels[key]); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("unsupported format %q", format)
		}
	},
}

func init() {
	labelCmd.Flags().String("registry", "", "override registry file path")
	labelCmd.Flags().StringArray("set", nil, "set label key=value (repeatable)")
	labelCmd.Flags().StringArray("remove", nil, "remove label key (repeatable)")
	labelCmd.Flags().StringP("format", "o", "table", "output format: table or json")
	rootCmd.AddCommand(labelCmd)
}
