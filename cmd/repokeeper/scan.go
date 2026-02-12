package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mfacenet/repokeeper/internal/config"
	"github.com/mfacenet/repokeeper/internal/engine"
	"github.com/mfacenet/repokeeper/internal/model"
	"github.com/mfacenet/repokeeper/internal/registry"
	"github.com/mfacenet/repokeeper/internal/vcs"
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
		cfgPath, err := config.ResolveConfigPath(flagConfig, cwd)
		if err != nil {
			return err
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		debugf(cmd, "using config %s", cfgPath)

		regPath := cfg.RegistryPath
		reg, err := registry.Load(regPath)
		if err != nil {
			if os.IsNotExist(err) {
				reg = &registry.Registry{MachineID: cfg.MachineID}
			} else {
				return err
			}
		}

		roots, _ := cmd.Flags().GetString("roots")
		exclude, _ := cmd.Flags().GetString("exclude")
		followSymlinks, _ := cmd.Flags().GetBool("follow-symlinks")
		writeRegistry, _ := cmd.Flags().GetBool("write-registry")
		pruneStale, _ := cmd.Flags().GetBool("prune-stale")
		format, _ := cmd.Flags().GetString("format")

		adapter := vcs.NewGitAdapter(nil)
		eng := engine.New(cfg, reg, adapter)

		statuses, err := eng.Scan(cmd.Context(), engine.ScanOptions{
			Roots:          splitCSV(roots),
			Exclude:        splitCSV(exclude),
			FollowSymlinks: followSymlinks,
		})
		if err != nil {
			return err
		}

		if pruneStale {
			reg.PruneStale(time.Duration(cfg.RegistryStaleDays) * 24 * time.Hour)
		}
		// Keep persisted registry ordering deterministic for stable diffs/output.
		sort.SliceStable(reg.Entries, func(i, j int) bool {
			if reg.Entries[i].RepoID == reg.Entries[j].RepoID {
				return reg.Entries[i].Path < reg.Entries[j].Path
			}
			return reg.Entries[i].RepoID < reg.Entries[j].RepoID
		})

		if writeRegistry {
			if err := registry.Save(reg, regPath); err != nil {
				return err
			}
		}

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(statuses, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			writeScanTable(cmd, statuses)
		default:
			return fmt.Errorf("unsupported format %q", format)
		}

		if hasRegistryWarnings(reg) {
			// Missing/moved entries are warning-level conditions for scan/status flows.
			raiseExitCode(1)
		}
		infof(cmd, "scan completed: %d repos", len(statuses))
		return nil
	},
}

func writeScanTable(cmd *cobra.Command, statuses []model.RepoStatus) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPO\tPATH\tBARE\tPRIMARY_REMOTE")
	for _, status := range statuses {
		bare := "no"
		if status.Bare {
			bare = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", status.RepoID, status.Path, bare, status.PrimaryRemote)
	}
	_ = w.Flush()
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
	scanCmd.Flags().String("format", "table", "output format: table or json")

	rootCmd.AddCommand(scanCmd)
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
