// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/sortutil"
	"github.com/skaphos/repokeeper/internal/strutil"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan roots for repositories and update the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		debugf(cmd, "starting scan")
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
		debugf(cmd, "using config %s", cfgPath)

		reg := cfg.Registry
		if reg == nil {
			reg = &registry.Registry{}
		}

		roots, _ := cmd.Flags().GetString("roots")
		exclude, _ := cmd.Flags().GetString("exclude")
		followSymlinks, _ := cmd.Flags().GetBool("follow-symlinks")
		writeRegistry, _ := cmd.Flags().GetBool("write-registry")
		pruneStale, _ := cmd.Flags().GetBool("prune-stale")
		format, _ := cmd.Flags().GetString("format")
		mode, err := parseOutputMode(format)
		if err != nil {
			return err
		}
		noHeaders, _ := cmd.Flags().GetBool("no-headers")

		adapter, err := selectedAdapterForCommand(cmd)
		if err != nil {
			return err
		}
		eng := engine.New(cfg, reg, adapter)
		scanRoots := strutil.SplitCSV(roots)
		if len(scanRoots) == 0 {
			scanRoots = []string{config.EffectiveRoot(cfgPath)}
		}

		statuses, err := eng.Scan(cmd.Context(), engine.ScanOptions{
			Roots:          scanRoots,
			Exclude:        strutil.SplitCSV(exclude),
			FollowSymlinks: followSymlinks,
		})
		if err != nil {
			return err
		}

		if pruneStale {
			reg.PruneStale(time.Duration(cfg.RegistryStaleDays) * 24 * time.Hour)
		}
		// Keep persisted registry ordering deterministic for stable diffs/output.
		sortutil.SortRegistryEntries(reg.Entries)

		if writeRegistry {
			cfg.Registry = reg
			if err := config.Save(cfg, cfgPath); err != nil {
				return err
			}
		}

		switch mode.kind {
		case outputKindJSON:
			data, err := json.MarshalIndent(statuses, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			logOutputWriteFailure(cmd, "scan json", err)
		case outputKindCustomColumns:
			logOutputWriteFailure(cmd, "scan custom-columns", writeCustomColumnsOutput(cmd, statuses, mode.expr, noHeaders))
		case outputKindTable:
			logOutputWriteFailure(cmd, "scan table", writeScanTable(cmd, statuses, noHeaders))
		default:
			return fmt.Errorf("unsupported format %q", format)
		}

		if hasRegistryWarnings(reg) {
			// Missing/moved entries are warning-level conditions for scan/status flows.
			raiseExitCode(cmd, 1)
		}
		infof(cmd, "scan completed: %d repos", len(statuses))
		return nil
	},
}

func writeScanTable(cmd *cobra.Command, statuses []model.RepoStatus, noHeaders bool) error {
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		bare := "no"
		if status.Bare {
			bare = "yes"
		}
		rows = append(rows, []string{status.RepoID, status.Path, bare, status.PrimaryRemote})
	}
	return cliio.WriteTable(cmd.OutOrStdout(), false, noHeaders, []string{"REPO", "PATH", "BARE", "PRIMARY_REMOTE"}, rows)
}

func hasRegistryWarnings(reg *registry.Registry) bool {
	for _, entry := range reg.Entries {
		if entry.Status == registry.StatusMissing || entry.Status == registry.StatusMoved {
			return true
		}
	}
	return false
}

func init() {
	scanCmd.Flags().String("roots", "", "comma-separated root directories to scan")
	scanCmd.Flags().String("exclude", "", "comma-separated glob patterns to exclude")
	scanCmd.Flags().Bool("follow-symlinks", false, "follow symbolic links during scan")
	scanCmd.Flags().Bool("write-registry", true, "write discovered repos to registry")
	scanCmd.Flags().Bool("prune-stale", false, "remove registry entries marked missing beyond stale threshold")
	addFormatFlag(scanCmd, "output format: table or json")
	addNoHeadersFlag(scanCmd)
	addVCSFlag(scanCmd)

	rootCmd.AddCommand(scanCmd)
}
