package repokeeper

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <repo-id-or-path>",
	Short: "Edit repository metadata and tracking configuration",
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
		setUpstream, _ := cmd.Flags().GetString("set-upstream")
		setUpstream = strings.TrimSpace(setUpstream)
		if setUpstream == "" {
			return fmt.Errorf("--set-upstream is required")
		}
		if err := validateUpstreamRef(setUpstream); err != nil {
			return err
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
				return fmt.Errorf("registry not found in %s (run repokeeper scan first)", cfgPath)
			}
		}

		entry, err := selectRegistryEntryForDescribe(reg.Entries, args[0], cwd, []string{cfgRoot})
		if err != nil {
			return err
		}
		if entry.Status == registry.StatusMissing {
			return fmt.Errorf("cannot set upstream for missing repository %q at %s", entry.RepoID, entry.Path)
		}

		adapter := vcs.NewGitAdapter(nil)
		head, err := adapter.Head(cmd.Context(), entry.Path)
		if err != nil {
			return err
		}
		if head.Detached || strings.TrimSpace(head.Branch) == "" {
			return fmt.Errorf("cannot set upstream on detached HEAD for %q", entry.RepoID)
		}

		if err := adapter.SetUpstream(cmd.Context(), entry.Path, setUpstream, head.Branch); err != nil {
			return fmt.Errorf("git branch --set-upstream-to %s %s: %w", setUpstream, head.Branch, err)
		}

		entry.Branch = trackingBranchFromUpstream(setUpstream)
		entry.LastSeen = time.Now()
		entry.Status = registry.StatusPresent
		for i := range reg.Entries {
			if reg.Entries[i].RepoID == entry.RepoID && reg.Entries[i].Path == entry.Path {
				reg.Entries[i] = entry
				break
			}
		}

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

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "set upstream for %s (%s) to %s\n", entry.RepoID, entry.Path, setUpstream)
		return nil
	},
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
	editCmd.Flags().String("set-upstream", "", "set upstream reference (example: origin/main)")
	rootCmd.AddCommand(editCmd)
}
