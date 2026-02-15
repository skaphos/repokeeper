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
		fieldSelector, _ := cmd.Flags().GetString("field-selector")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		timeout, _ := cmd.Flags().GetInt("timeout")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		updateLocal, _ := cmd.Flags().GetBool("update-local")
		pushLocal, _ := cmd.Flags().GetBool("push-local")
		rebaseDirty, _ := cmd.Flags().GetBool("rebase-dirty")
		force, _ := cmd.Flags().GetBool("force")
		protectedBranchesRaw, _ := cmd.Flags().GetString("protected-branches")
		allowProtectedRebase, _ := cmd.Flags().GetBool("allow-protected-rebase")
		checkoutMissing, _ := cmd.Flags().GetBool("checkout-missing")
		format, _ := cmd.Flags().GetString("format")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")
		wrap, _ := cmd.Flags().GetBool("wrap")
		if rebaseDirty && !updateLocal {
			return fmt.Errorf("--rebase-dirty requires --update-local")
		}
		if pushLocal && !updateLocal {
			return fmt.Errorf("--push-local requires --update-local")
		}
		filter, err := resolveRepoFilter(only, fieldSelector)
		if err != nil {
			return err
		}

		if concurrency == 0 {
			concurrency = cfg.Defaults.Concurrency
		}
		if timeout == 0 {
			timeout = cfg.Defaults.TimeoutSeconds
		}

		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil))
		plan, err := eng.Sync(cmd.Context(), engine.SyncOptions{
			Filter:               filter,
			Concurrency:          concurrency,
			Timeout:              timeout,
			ContinueOnError:      continueOnError,
			DryRun:               true,
			UpdateLocal:          updateLocal,
			PushLocal:            pushLocal,
			RebaseDirty:          rebaseDirty,
			Force:                force,
			ProtectedBranches:    splitCSV(protectedBranchesRaw),
			AllowProtectedRebase: allowProtectedRebase,
			CheckoutMissing:      checkoutMissing,
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
		if !yes && syncPlanNeedsConfirmation(plan) {
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
				Filter:               filter,
				Concurrency:          concurrency,
				Timeout:              timeout,
				ContinueOnError:      continueOnError,
				DryRun:               false,
				UpdateLocal:          updateLocal,
				PushLocal:            pushLocal,
				RebaseDirty:          rebaseDirty,
				Force:                force,
				ProtectedBranches:    splitCSV(protectedBranchesRaw),
				AllowProtectedRebase: allowProtectedRebase,
				CheckoutMissing:      checkoutMissing,
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
			setColorOutputMode(cmd, format)
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			setColorOutputMode(cmd, format)
			report, err := eng.Status(cmd.Context(), engine.StatusOptions{
				Filter:      engine.FilterAll,
				Concurrency: concurrency,
				Timeout:     timeout,
			})
			if err != nil {
				return err
			}
			writeSyncTable(cmd, results, report, cwd, []string{cfgRoot}, wrap, noHeaders, false)
		case "wide":
			setColorOutputMode(cmd, format)
			report, err := eng.Status(cmd.Context(), engine.StatusOptions{
				Filter:      engine.FilterAll,
				Concurrency: concurrency,
				Timeout:     timeout,
			})
			if err != nil {
				return err
			}
			writeSyncTable(cmd, results, report, cwd, []string{cfgRoot}, wrap, noHeaders, true)
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
		writeSyncFailureSummary(cmd, results, cwd, []string{cfgRoot})
		infof(cmd, "sync completed: %d repos", len(results))
		return nil
	},
}

func init() {
	syncCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing")
	syncCmd.Flags().String("field-selector", "", "field selector (phase 1): tracking.status=diverged|gone, worktree.dirty=true|false, repo.error=true, repo.missing=true, remote.mismatch=true")
	syncCmd.Flags().Int("concurrency", 0, "max concurrent repo operations (default: min(8, NumCPU))")
	syncCmd.Flags().Int("timeout", 60, "timeout in seconds per repo")
	syncCmd.Flags().Bool("continue-on-error", true, "continue syncing remaining repos after a per-repo failure")
	syncCmd.Flags().Bool("dry-run", false, "print intended operations without executing")
	syncCmd.Flags().Bool("yes", false, "accept sync plan and execute without confirmation")
	syncCmd.Flags().Bool("update-local", false, "after fetch, run pull --rebase only for clean branches tracking */main")
	syncCmd.Flags().Bool("push-local", false, "when used with --update-local, push branches that are ahead of upstream")
	syncCmd.Flags().Bool("rebase-dirty", false, "when used with --update-local, stash local changes before rebase and pop afterwards")
	syncCmd.Flags().Bool("force", false, "when used with --update-local, allow rebase even when branch tracking state is diverged")
	syncCmd.Flags().String("protected-branches", "main,master,release/*", "comma-separated branch patterns to protect from auto-rebase during --update-local")
	syncCmd.Flags().Bool("allow-protected-rebase", false, "when used with --update-local, allow rebase on branches matched by --protected-branches")
	syncCmd.Flags().Bool("checkout-missing", false, "clone missing repos from registry remote_url back to their registered paths")
	syncCmd.Flags().StringP("format", "o", "table", "output format: table, wide, or json")
	syncCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")
	syncCmd.Flags().Bool("wrap", false, "allow table columns to wrap instead of truncating")

}

func writeSyncPlan(cmd *cobra.Command, plan []engine.SyncResult, cwd string, roots []string) {
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Planned sync operations:")
	w := tabwriter.NewWriter(cmd.ErrOrStderr(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tACTION\tREPO")
	for _, res := range plan {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\n",
			displayRepoPath(res.Path, cwd, roots),
			describeSyncAction(res),
			res.RepoID,
		)
	}
	_ = w.Flush()
}

func confirmSyncExecution(cmd *cobra.Command) (bool, error) {
	_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Proceed with local updates? [y/N]: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	return choice == "y" || choice == "yes", nil
}

func syncPlanNeedsConfirmation(plan []engine.SyncResult) bool {
	for _, res := range plan {
		if syncResultNeedsConfirmation(res) {
			return true
		}
	}
	return false
}

func syncResultNeedsConfirmation(res engine.SyncResult) bool {
	// Confirmation is reserved for operations that mutate local state.
	action := strings.ToLower(strings.TrimSpace(res.Action))
	if strings.Contains(action, "pull --rebase") || strings.Contains(action, "stash push") {
		return true
	}
	if strings.Contains(action, "git clone") {
		return true
	}
	return false
}

func writeSyncTable(cmd *cobra.Command, results []engine.SyncResult, report *model.StatusReport, cwd string, roots []string, wrap bool, noHeaders bool, wide bool) {
	statusByPath := make(map[string]model.RepoStatus, len(results))
	if report != nil {
		for _, repo := range report.Repos {
			// Sync results are keyed by path, so table enrichment uses the same key.
			statusByPath[repo.Path] = repo
		}
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.StripEscape)
	if !noHeaders {
		headers := "PATH\tACTION\tBRANCH\tDIRTY\tTRACKING\tOK\tERROR_CLASS\tERROR\tREPO"
		if wide {
			headers += "\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND"
		}
		_, _ = fmt.Fprintln(w, headers)
	}
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
				// Mirror repos do not have branch tracking semantics in the same way
				// as a non-bare checkout.
				tracking = colorize("mirror", ansiBlue)
			}
		}
		if !wide {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				path,
				describeSyncAction(res),
				branch,
				dirty,
				tracking,
				ok,
				res.ErrorClass,
				formatCell(res.Error, wrap, 36),
				res.RepoID)
			continue
		}

		ahead := "-"
		if repo.Tracking.Ahead != nil {
			ahead = fmt.Sprintf("%d", *repo.Tracking.Ahead)
		}
		behind := "-"
		if repo.Tracking.Behind != nil {
			behind = fmt.Sprintf("%d", *repo.Tracking.Behind)
		}
		primaryRemote := ""
		upstream := ""
		if found {
			primaryRemote = repo.PrimaryRemote
			upstream = repo.Tracking.Upstream
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			describeSyncAction(res),
			branch,
			dirty,
			tracking,
			ok,
			res.ErrorClass,
			formatCell(res.Error, wrap, 36),
			res.RepoID,
			primaryRemote,
			upstream,
			ahead,
			behind)
	}
	_ = w.Flush()
}

func describeSyncAction(res engine.SyncResult) string {
	action := strings.TrimSpace(res.Action)

	// Prefer explicit skip reasons from the engine over heuristic action parsing.
	if strings.HasPrefix(res.Error, "skipped-local-update:") {
		reason := strings.TrimSpace(strings.TrimPrefix(res.Error, "skipped-local-update:"))
		if reason == "" {
			return "skip local update"
		}
		return "skip local update (" + reason + ")"
	}
	if res.Error == "skipped-no-upstream" {
		return "skip no upstream"
	}
	if res.Error == "skipped" {
		return "skip"
	}
	if res.Error == "missing" {
		return "skip missing"
	}

	normalized := strings.ToLower(action)
	// Collapse verbose git command strings into stable, human-readable summaries.
	switch {
	case strings.Contains(normalized, "stash") && strings.Contains(normalized, "rebase"):
		return "stash & rebase"
	case strings.Contains(normalized, "fetch --all") && strings.Contains(normalized, "git push"):
		return "fetch + push"
	case strings.Contains(normalized, "fetch --all") && strings.Contains(normalized, "pull --rebase"):
		return "fetch + rebase"
	case strings.Contains(normalized, "git push"):
		return "push"
	case strings.Contains(normalized, "pull --rebase"):
		return "rebase"
	case strings.Contains(normalized, "fetch --all"):
		return "fetch"
	case strings.Contains(normalized, "git clone --mirror"):
		return "checkout missing (mirror)"
	case strings.Contains(normalized, "git clone"):
		return "checkout missing"
	}

	if action == "" {
		if res.OK {
			return "fetch"
		}
		return "-"
	}
	return action
}

func writeSyncFailureSummary(cmd *cobra.Command, results []engine.SyncResult, cwd string, roots []string) {
	failed := make([]engine.SyncResult, 0, len(results))
	for _, res := range results {
		if res.OK {
			continue
		}
		failed = append(failed, res)
	}
	if len(failed) == 0 {
		return
	}

	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Failed sync operations:")
	w := tabwriter.NewWriter(cmd.ErrOrStderr(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tACTION\tERROR_CLASS\tERROR\tREPO")
	for _, res := range failed {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\n",
			displayRepoPath(res.Path, cwd, roots),
			describeSyncAction(res),
			res.ErrorClass,
			res.Error,
			res.RepoID,
		)
	}
	_ = w.Flush()
}
