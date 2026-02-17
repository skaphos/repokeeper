// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var editCmd = &cobra.Command{
	Use:   "edit <repo-id-or-path>",
	Short: "Edit a single repository registry entry in your editor",
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
		idx := findRegistryEntryIndex(reg.Entries, entry)
		if idx < 0 {
			return fmt.Errorf("entry not found for selector %q", args[0])
		}

		edited, changed, err := editRegistryEntryWithEditor(cmd, entry)
		if err != nil {
			return err
		}
		if !changed {
			infof(cmd, "no changes for %s", entry.RepoID)
			return nil
		}
		if err := validateEditedRegistryEntry(edited, reg, idx); err != nil {
			return err
		}
		if edited.LastSeen.IsZero() {
			edited.LastSeen = time.Now()
		}
		reg.Entries[idx] = edited
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

		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "updated %s (%s)\n", edited.RepoID, edited.Path); err != nil {
			return err
		}
		return nil
	},
}

func findRegistryEntryIndex(entries []registry.Entry, target registry.Entry) int {
	for i := range entries {
		if entries[i].RepoID == target.RepoID && entries[i].Path == target.Path {
			return i
		}
	}
	return -1
}

func editRegistryEntryWithEditor(cmd *cobra.Command, entry registry.Entry) (registry.Entry, bool, error) {
	editorParts, err := resolveEditorCommand()
	if err != nil {
		return registry.Entry{}, false, err
	}
	tmpFile, err := os.CreateTemp("", "repokeeper-edit-*.yaml")
	if err != nil {
		return registry.Entry{}, false, err
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	data, err := yaml.Marshal(entry)
	if err != nil {
		return registry.Entry{}, false, err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return registry.Entry{}, false, err
	}
	if err := tmpFile.Close(); err != nil {
		return registry.Entry{}, false, err
	}

	run := exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
	run.Stdin = cmd.InOrStdin()
	run.Stdout = cmd.OutOrStdout()
	run.Stderr = cmd.ErrOrStderr()
	if err := run.Run(); err != nil {
		return registry.Entry{}, false, err
	}

	editedData, err := os.ReadFile(tmpPath)
	if err != nil {
		return registry.Entry{}, false, err
	}
	var edited registry.Entry
	if err := yaml.Unmarshal(editedData, &edited); err != nil {
		return registry.Entry{}, false, fmt.Errorf("invalid edited yaml: %w", err)
	}
	if reflect.DeepEqual(entry, edited) {
		return entry, false, nil
	}
	return edited, true, nil
}

func resolveEditorCommand() ([]string, error) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return nil, fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}
	return parts, nil
}

func validateEditedRegistryEntry(entry registry.Entry, reg *registry.Registry, index int) error {
	if strings.TrimSpace(entry.RepoID) == "" {
		return fmt.Errorf("invalid entry: repo_id is required")
	}
	entryPath := strings.TrimSpace(entry.Path)
	if entryPath == "" {
		return fmt.Errorf("invalid entry: path is required")
	}
	if !filepath.IsAbs(entryPath) {
		return fmt.Errorf("invalid entry: path must be absolute, got %q", entry.Path)
	}
	if entry.Status == "" {
		return fmt.Errorf("invalid entry: status is required")
	}
	switch entry.Status {
	case registry.StatusPresent, registry.StatusMissing, registry.StatusMoved:
	default:
		return fmt.Errorf("invalid entry: unsupported status %q", entry.Status)
	}
	typ := strings.TrimSpace(entry.Type)
	if typ != "" && typ != "checkout" && typ != "mirror" {
		return fmt.Errorf("invalid entry: unsupported type %q", entry.Type)
	}
	for key := range entry.Labels {
		if err := validateMetadataKey(strings.TrimSpace(key), "label"); err != nil {
			return err
		}
	}
	for key := range entry.Annotations {
		if err := validateMetadataKey(strings.TrimSpace(key), "annotation"); err != nil {
			return err
		}
	}
	if reg != nil {
		for i := range reg.Entries {
			if i == index {
				continue
			}
			if reg.Entries[i].RepoID == entry.RepoID {
				return fmt.Errorf("invalid entry: repo_id %q already exists", entry.RepoID)
			}
		}
	}
	return nil
}

func trackingBranchFromUpstream(upstream string) string {
	trimmed := strings.TrimSpace(upstream)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

func validateUpstreamRef(upstream string) error {
	trimmed := strings.TrimSpace(upstream)
	if trimmed == "" {
		return fmt.Errorf("--set-upstream is required")
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("invalid --set-upstream %q: expected remote/branch", upstream)
	}
	if strings.Contains(parts[0], "/") {
		return fmt.Errorf("invalid --set-upstream %q: expected remote/branch", upstream)
	}
	return nil
}

func init() {
	editCmd.Flags().String("registry", "", "override registry file path")
	rootCmd.AddCommand(editCmd)
}
