package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/sortutil"
	"github.com/skaphos/repokeeper/internal/strutil"
	"github.com/skaphos/repokeeper/internal/tableutil"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan roots for git repos and update the registry",
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
		noHeaders, _ := cmd.Flags().GetBool("no-headers")

		adapter := vcs.NewGitAdapter(nil)
		eng := engine.New(cfg, reg, adapter)
		scanRoots := strutil.SplitCSV(roots)
		if len(scanRoots) == 0 {
			scanRoots = []string{config.EffectiveRoot(cfgPath, cfg)}
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

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(statuses, "", "  ")
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
				return err
			}
		case "table":
			if err := writeScanTable(cmd, statuses, noHeaders); err != nil {
				return err
			}
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
	w := tableutil.New(cmd.OutOrStdout(), false)
	tableutil.PrintHeaders(w, noHeaders, "REPO\tPATH\tBARE\tPRIMARY_REMOTE")
	for _, status := range statuses {
		bare := "no"
		if status.Bare {
			bare = "yes"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", status.RepoID, status.Path, bare, status.PrimaryRemote); err != nil {
			return err
		}
	}
	return w.Flush()
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
	scanCmd.Flags().StringP("format", "o", "table", "output format: table or json")
	scanCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")

	rootCmd.AddCommand(scanCmd)
}
