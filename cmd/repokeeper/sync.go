package repokeeper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
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
		cfgRoot := config.EffectiveRoot(cfgPath, cfg)
		debugf(cmd, "using config %s", cfgPath)

		reg := cfg.Registry
		if reg == nil {
			return fmt.Errorf("registry not found in %s (run repokeeper scan first)", cfgPath)
		}

		only, _ := cmd.Flags().GetString("only")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		timeout, _ := cmd.Flags().GetInt("timeout")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		updateLocal, _ := cmd.Flags().GetBool("update-local")
		checkoutMissing, _ := cmd.Flags().GetBool("checkout-missing")
		format, _ := cmd.Flags().GetString("format")
		wrap, _ := cmd.Flags().GetBool("wrap")

		if concurrency == 0 {
			concurrency = cfg.Defaults.Concurrency
		}
		if timeout == 0 {
			timeout = cfg.Defaults.TimeoutSeconds
		}

		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil))
		plan, err := eng.Sync(cmd.Context(), engine.SyncOptions{
			Filter:          engine.FilterKind(only),
			Concurrency:     concurrency,
			Timeout:         timeout,
			DryRun:          true,
			UpdateLocal:     updateLocal,
			CheckoutMissing: checkoutMissing,
		})
		if err != nil {
			return err
		}
		// Keep sync output stable across runs regardless of goroutine completion order.
		sort.SliceStable(plan, func(i, j int) bool {
			if plan[i].RepoID == plan[j].RepoID {
				return plan[i].Action < plan[j].Action
			}
			return plan[i].RepoID < plan[j].RepoID
		})
		writeSyncPlan(cmd, plan, cwd, []string{cfgRoot})
		if !yes {
			confirmed, err := confirmSyncExecution(cmd)
			if err != nil {
				return err
			}
			if !confirmed {
				infof(cmd, "sync cancelled")
				return nil
			}
		}

		results := plan
		if !dryRun {
			results, err = eng.Sync(cmd.Context(), engine.SyncOptions{
				Filter:          engine.FilterKind(only),
				Concurrency:     concurrency,
				Timeout:         timeout,
				DryRun:          false,
				UpdateLocal:     updateLocal,
				CheckoutMissing: checkoutMissing,
			})
			if err != nil {
				return err
			}
			sort.SliceStable(results, func(i, j int) bool {
				if results[i].RepoID == results[j].RepoID {
					return results[i].Action < results[j].Action
				}
				return results[i].RepoID < results[j].RepoID
			})
		}

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			report, err := eng.Status(cmd.Context(), engine.StatusOptions{
				Filter:      engine.FilterAll,
				Concurrency: concurrency,
				Timeout:     timeout,
			})
			if err != nil {
				return err
			}
			writeSyncTable(cmd, results, report, cwd, []string{cfgRoot}, wrap)
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
	syncCmd.Flags().Bool("yes", false, "accept sync plan and execute without confirmation")
	syncCmd.Flags().Bool("update-local", false, "after fetch, run pull --rebase only for clean branches tracking */main")
	syncCmd.Flags().Bool("checkout-missing", false, "clone missing repos from registry remote_url back to their registered paths")
	syncCmd.Flags().String("format", "table", "output format: table or json")
	syncCmd.Flags().Bool("wrap", false, "allow table columns to wrap instead of truncating")

	rootCmd.AddCommand(syncCmd)
}

func writeSyncPlan(cmd *cobra.Command, plan []engine.SyncResult, cwd string, roots []string) {
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Planned sync operations:")
	w := tabwriter.NewWriter(cmd.ErrOrStderr(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tREPO\tACTION")
	for _, res := range plan {
		action := strings.TrimSpace(res.Action)
		if action == "" {
			if res.Error == "missing" {
				action = "skip missing repo"
			} else {
				action = "git fetch --all --prune --prune-tags --no-recurse-submodules"
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", displayRepoPath(res.Path, cwd, roots), res.RepoID, action)
	}
	_ = w.Flush()
}

func confirmSyncExecution(cmd *cobra.Command) (bool, error) {
	_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Proceed with sync? [y/N]: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	return choice == "y" || choice == "yes", nil
}

func writeSyncTable(cmd *cobra.Command, results []engine.SyncResult, report *model.StatusReport, cwd string, roots []string, wrap bool) {
	statusByPath := make(map[string]model.RepoStatus, len(results))
	if report != nil {
		for _, repo := range report.Repos {
			statusByPath[repo.Path] = repo
		}
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.StripEscape)
	_, _ = fmt.Fprintln(w, "PATH\tBRANCH\tDIRTY\tTRACKING\tOK\tERROR_CLASS\tERROR\tACTION")
	for _, res := range results {
		ok := "yes"
		if !res.OK {
			ok = "no"
		}
		repo, found := statusByPath[res.Path]
		branch := "-"
		dirty := "-"
		tracking := string(model.TrackingNone)
		path := res.Path
		if found {
			path = displayRepoPath(repo.Path, cwd, roots)
			branch = repo.Head.Branch
			if repo.Head.Detached {
				branch = "detached:" + branch
			}
			if repo.Type == "mirror" {
				branch = "-"
			}
			if repo.Worktree != nil {
				if repo.Worktree.Dirty {
					dirty = colorize("yes", ansiBrown)
				} else {
					dirty = colorize("no", ansiGreen)
				}
			}
			tracking = displayTrackingStatus(repo.Tracking.Status)
			if repo.Type == "mirror" {
				tracking = colorize("mirror", ansiBlue)
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			branch,
			dirty,
			tracking,
			ok,
			res.ErrorClass,
			formatCell(res.Error, wrap, 36),
			formatCell(res.Action, wrap, 48))
	}
	_ = w.Flush()
}
