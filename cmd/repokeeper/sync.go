package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run safe fetch/prune on registered repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		debugf(cmd, "starting sync")
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
				return fmt.Errorf("registry not found at %s (run repokeeper scan first)", regPath)
			}
			return err
		}

		only, _ := cmd.Flags().GetString("only")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		timeout, _ := cmd.Flags().GetInt("timeout")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		updateLocal, _ := cmd.Flags().GetBool("update-local")
		format, _ := cmd.Flags().GetString("format")

		if concurrency == 0 {
			concurrency = cfg.Defaults.Concurrency
		}
		if timeout == 0 {
			timeout = cfg.Defaults.TimeoutSeconds
		}

		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil))
		results, err := eng.Sync(cmd.Context(), engine.SyncOptions{
			Filter:      engine.FilterKind(only),
			Concurrency: concurrency,
			Timeout:     timeout,
			DryRun:      dryRun,
			UpdateLocal: updateLocal,
		})
		if err != nil {
			return err
		}
		// Keep sync output stable across runs regardless of goroutine completion order.
		sort.SliceStable(results, func(i, j int) bool {
			if results[i].RepoID == results[j].RepoID {
				return results[i].Action < results[j].Action
			}
			return results[i].RepoID < results[j].RepoID
		})

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			writeSyncTable(cmd, results)
		default:
			return fmt.Errorf("unsupported format %q", format)
		}
		for _, res := range results {
			if !res.OK {
				// Missing repos are warning-level; operational failures are error-level.
				if res.Error == "missing" {
					raiseExitCode(1)
					continue
				}
				raiseExitCode(2)
				continue
			}
			if strings.HasPrefix(res.Error, "skipped-local-update:") {
				raiseExitCode(1)
			}
		}
		infof(cmd, "sync completed: %d repos", len(results))
		return nil
	},
}

func init() {
	syncCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, missing")
	syncCmd.Flags().Int("concurrency", 0, "max concurrent repo operations (default: min(8, NumCPU))")
	syncCmd.Flags().Int("timeout", 60, "timeout in seconds per repo")
	syncCmd.Flags().Bool("dry-run", false, "print intended operations without executing")
	syncCmd.Flags().Bool("update-local", false, "after fetch, run pull --rebase only for clean branches tracking */main")
	syncCmd.Flags().String("format", "table", "output format: table or json")

	rootCmd.AddCommand(syncCmd)
}

func writeSyncTable(cmd *cobra.Command, results []engine.SyncResult) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPO\tOK\tERROR_CLASS\tERROR\tACTION")
	for _, res := range results {
		ok := "yes"
		if !res.OK {
			ok = "no"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", res.RepoID, ok, res.ErrorClass, res.Error, res.Action)
	}
	_ = w.Flush()
}
