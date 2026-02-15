package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/strutil"
	"github.com/skaphos/repokeeper/internal/tableutil"
	"github.com/skaphos/repokeeper/internal/termstyle"
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
		cfgPath, err := config.ResolveConfigPath(configOverride(cmd), cwd)
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
			return fmt.Errorf("registry not found in %q (run repokeeper scan first)", cfgPath)
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
		mode, err := parseOutputMode(format)
		if err != nil {
			return err
		}
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

		adapter, err := selectedAdapterForCommand(cmd)
		if err != nil {
			return err
		}
		eng := engine.New(cfg, reg, adapter)
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
			ProtectedBranches:    strutil.SplitCSV(protectedBranchesRaw),
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
		logOutputWriteFailure(cmd, "sync plan", writeSyncPlan(cmd, plan, cwd, []string{cfgRoot}))
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
		streamResults := shouldStreamSyncResults(cmd, dryRun, mode.kind)
		if !dryRun {
			var streamWriter *syncProgressWriter
			if streamResults {
				streamWriter = newSyncProgressWriter(cmd, cwd, []string{cfgRoot})
				if err != nil {
					return err
				}
			}

			results, err = eng.ExecuteSyncPlanWithCallbacks(cmd.Context(), plan, engine.SyncOptions{
				Concurrency:     concurrency,
				Timeout:         timeout,
				ContinueOnError: continueOnError,
			}, func(res engine.SyncResult) {
				if streamWriter == nil {
					return
				}
				if streamErr := streamWriter.StartResult(res); streamErr != nil {
					logOutputWriteFailure(cmd, "sync stream start", streamErr)
				}
			}, func(res engine.SyncResult) {
				if streamWriter == nil {
					return
				}
				if streamErr := streamWriter.WriteResult(res); streamErr != nil {
					logOutputWriteFailure(cmd, "sync stream row", streamErr)
				}
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

		switch mode.kind {
		case outputKindJSON:
			setColorOutputMode(cmd, string(mode.kind))
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			logOutputWriteFailure(cmd, "sync json", err)
		case outputKindCustomColumns:
			setColorOutputMode(cmd, string(mode.kind))
			logOutputWriteFailure(cmd, "sync custom-columns", writeCustomColumnsOutput(cmd, results, mode.expr, noHeaders))
		case outputKindTable:
			setColorOutputMode(cmd, string(mode.kind))
			if !streamResults {
				logOutputWriteFailure(cmd, "sync table", writeSyncTable(cmd, results, nil, cwd, []string{cfgRoot}, wrap, noHeaders, false))
			}
		case outputKindWide:
			setColorOutputMode(cmd, string(mode.kind))
			if !streamResults {
				logOutputWriteFailure(cmd, "sync wide", writeSyncTable(cmd, results, nil, cwd, []string{cfgRoot}, wrap, noHeaders, true))
			}
		default:
			return fmt.Errorf("unsupported format %q", format)
		}
		for _, res := range results {
			if !res.OK {
				// Missing repos are warning-level; operational failures are error-level.
				if res.Error == engine.SyncErrorMissing {
					raiseExitCode(cmd, 1)
					continue
				}
				raiseExitCode(cmd, 2)
				continue
			}
			if strings.HasPrefix(res.Error, engine.SyncErrorSkippedLocalUpdatePrefix) {
				raiseExitCode(cmd, 1)
			}
		}
		logOutputWriteFailure(cmd, "sync failure summary", writeSyncFailureSummary(cmd, results, cwd, []string{cfgRoot}))
		infof(cmd, "sync completed: %d repos", len(results))
		return nil
	},
}

func init() {
	addRepoFilterFlags(syncCmd)
	syncCmd.Flags().Int("concurrency", 0, "max concurrent repo operations (default: min(8, NumCPU))")
	syncCmd.Flags().Int("timeout", 0, "timeout in seconds per repo (0 uses config default)")
	syncCmd.Flags().Bool("continue-on-error", true, "continue syncing remaining repos after a per-repo failure")
	syncCmd.Flags().Bool("dry-run", false, "print intended operations without executing")
	syncCmd.Flags().Bool("update-local", false, "after fetch, run pull --rebase for the checked-out tracking branch when safe")
	syncCmd.Flags().Bool("push-local", false, "when used with --update-local, push branches that are ahead of upstream")
	syncCmd.Flags().Bool("rebase-dirty", false, "when used with --update-local, stash local changes before rebase and pop afterwards")
	syncCmd.Flags().Bool("force", false, "when used with --update-local, allow rebase even when branch tracking state is diverged")
	syncCmd.Flags().String("protected-branches", "", "comma-separated branch patterns to protect from auto-rebase during --update-local (default: none)")
	syncCmd.Flags().Bool("allow-protected-rebase", false, "when used with --update-local, allow rebase on branches matched by --protected-branches")
	syncCmd.Flags().Bool("checkout-missing", false, "clone missing repos from registry remote_url back to their registered paths")
	addFormatFlag(syncCmd, "output format: table, wide, or json")
	addNoHeadersFlag(syncCmd)
	syncCmd.Flags().Bool("wrap", false, "allow table columns to wrap instead of truncating")
	addVCSFlag(syncCmd)

}

func writeSyncPlan(cmd *cobra.Command, plan []engine.SyncResult, cwd string, roots []string) error {
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Planned sync operations:"); err != nil {
		return err
	}
	rows := make([][]string, 0, len(plan))
	for _, res := range plan {
		rows = append(rows, []string{
			displayRepoPath(res.Path, cwd, roots),
			describeSyncAction(res),
			res.RepoID,
		})
	}
	return cliio.WriteTable(cmd.ErrOrStderr(), false, false, []string{"PATH", "ACTION", "REPO"}, rows)
}

func confirmSyncExecution(cmd *cobra.Command) (bool, error) {
	return confirmWithPrompt(cmd, "Proceed with local updates? [y/N]: ")
}

func confirmWithPrompt(cmd *cobra.Command, prompt string) (bool, error) {
	return cliio.PromptYesNo(cmd.ErrOrStderr(), cmd.InOrStdin(), prompt)
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

type syncTableMode int

const (
	syncTableModeFull syncTableMode = iota
	syncTableModeCompact
	syncTableModeTiny
)

type syncProgressWriter struct {
	cmd   *cobra.Command
	cwd   string
	roots []string

	supportsInPlace bool
	mu              sync.Mutex
	running         map[string]*syncProgressState
}

type syncProgressState struct {
	displayPath string
	dots        int
	lastLen     int
	stop        chan struct{}
	done        chan struct{}
}

func shouldStreamSyncResults(cmd *cobra.Command, dryRun bool, kind outputKind) bool {
	if dryRun {
		return false
	}
	if kind != outputKindTable && kind != outputKindWide {
		return false
	}
	if cmd == nil || cmd.Name() != "repos" || cmd.Parent() == nil {
		return false
	}
	return cmd.Parent().Name() == "reconcile"
}

func newSyncProgressWriter(cmd *cobra.Command, cwd string, roots []string) *syncProgressWriter {
	supportsInPlace := false
	if file, ok := cmd.OutOrStdout().(*os.File); ok {
		supportsInPlace = isTerminalFD(int(file.Fd()))
	}
	return &syncProgressWriter{
		cmd:             cmd,
		cwd:             cwd,
		roots:           roots,
		supportsInPlace: supportsInPlace,
		running:         make(map[string]*syncProgressState),
	}
}

func (s *syncProgressWriter) StartResult(res engine.SyncResult) error {
	path := displayRepoPath(res.Path, s.cwd, s.roots)
	state := &syncProgressState{
		displayPath: path,
		dots:        1,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}

	s.mu.Lock()
	if _, exists := s.running[res.Path]; exists {
		s.mu.Unlock()
		return nil
	}
	s.running[res.Path] = state
	if err := s.writeProgressLine(state, strings.Repeat(".", state.dots), false); err != nil {
		delete(s.running, res.Path)
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()

	go s.runDots(res.Path, state)
	return nil
}

func (s *syncProgressWriter) runDots(path string, state *syncProgressState) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer close(state.done)

	for {
		select {
		case <-state.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			if _, exists := s.running[path]; !exists {
				s.mu.Unlock()
				return
			}
			state.dots++
			_ = s.writeProgressLine(state, strings.Repeat(".", state.dots), false)
			s.mu.Unlock()
		}
	}
}

func (s *syncProgressWriter) WriteResult(res engine.SyncResult) error {
	path := displayRepoPath(res.Path, s.cwd, s.roots)

	var done <-chan struct{}
	s.mu.Lock()
	if state, exists := s.running[res.Path]; exists {
		delete(s.running, res.Path)
		close(state.stop)
		done = state.done
		path = state.displayPath
	}
	s.mu.Unlock()
	if done != nil {
		<-done
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	action := describeSyncAction(res)
	state := &syncProgressState{displayPath: path}
	if err := s.writeProgressLine(state, syncProgressMessage(s.cmd, res), true); err != nil {
		return err
	}
	if !res.OK && !strings.HasPrefix(action, "skip") && !isQuiet(s.cmd) && strings.TrimSpace(res.Error) != "" {
		_, err := fmt.Fprintf(s.cmd.ErrOrStderr(), "error: %s: %s\n", path, res.Error)
		return err
	}
	return nil
}

func (s *syncProgressWriter) writeProgressLine(state *syncProgressState, message string, newline bool) error {
	line := fmt.Sprintf("%s %s", state.displayPath, message)
	if s.supportsInPlace {
		padding := ""
		if state.lastLen > len(line) {
			padding = strings.Repeat(" ", state.lastLen-len(line))
		}
		if newline {
			if _, err := fmt.Fprintf(s.cmd.OutOrStdout(), "\r%s%s\n", line, padding); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(s.cmd.OutOrStdout(), "\r%s%s", line, padding); err != nil {
				return err
			}
		}
		state.lastLen = len(line)
		return nil
	}
	if newline {
		_, err := fmt.Fprintf(s.cmd.OutOrStdout(), "%s\n", line)
		return err
	}
	_, err := fmt.Fprintf(s.cmd.OutOrStdout(), "%s\n", line)
	return err
}

func syncProgressMessage(cmd *cobra.Command, res engine.SyncResult) string {
	action := describeSyncAction(res)
	if strings.HasPrefix(action, "skip") {
		return termstyle.Colorize(runtimeStateFor(cmd).colorOutputEnabled, action, termstyle.Warn)
	}
	if !res.OK {
		out := "failed"
		if res.ErrorClass != "" {
			out = fmt.Sprintf("failed (%s)", res.ErrorClass)
		}
		return termstyle.Colorize(runtimeStateFor(cmd).colorOutputEnabled, out, termstyle.Error)
	}
	out := "updated!"
	if action != "" {
		out += " (" + action + ")"
	}
	return termstyle.Colorize(runtimeStateFor(cmd).colorOutputEnabled, out, termstyle.Healthy)
}

func syncTableModeFor(cmd *cobra.Command, wide bool) syncTableMode {
	if wide {
		return syncTableModeFull
	}
	mode := syncTableModeFull
	width, hasWidth := tableWidth(cmd)
	switch {
	case hasWidth && width < tinyTableWidth:
		mode = syncTableModeTiny
	case hasWidth && width < narrowTableWidth:
		mode = syncTableModeCompact
	}
	return mode
}

func syncTableHeaders(mode syncTableMode, wide bool) string {
	headers := "PATH\tACTION\tBRANCH\tDIRTY\tTRACKING\tOK\tERROR_CLASS\tERROR\tREPO"
	if mode == syncTableModeCompact {
		headers = "PATH\tACTION\tOK\tERROR\tREPO"
	}
	if mode == syncTableModeTiny {
		headers = "PATH\tACTION\tOK\tERROR"
	}
	if wide {
		headers += "\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND"
	}
	return headers
}

func writeSyncTable(cmd *cobra.Command, results []engine.SyncResult, report *model.StatusReport, cwd string, roots []string, wrap bool, noHeaders bool, wide bool) error {
	statusByPath := make(map[string]model.RepoStatus, len(results))
	if report != nil {
		for _, repo := range report.Repos {
			// Sync results are keyed by path, so table enrichment uses the same key.
			statusByPath[repo.Path] = repo
		}
	}

	w := tableutil.New(cmd.OutOrStdout(), true)
	mode := syncTableModeFor(cmd, wide)
	headers := syncTableHeaders(mode, wide)
	if err := tableutil.PrintHeaders(w, noHeaders, headers); err != nil {
		return err
	}
	pathMax := adaptiveCellLimit(cmd, 0, 48, 32)
	actionMax := adaptiveCellLimit(cmd, 0, 22, 16)
	branchMax := adaptiveCellLimit(cmd, 0, 24, 16)
	repoMax := adaptiveCellLimit(cmd, 0, 32, 20)
	for _, res := range results {
		ok := "yes"
		if !res.OK {
			ok = "no"
		}
		repo, found := statusByPath[res.Path]
		branch := "-"
		dirty := "-"
		tracking := string(model.TrackingNone)
		path := formatCell(displayRepoPath(res.Path, cwd, roots), wrap, pathMax)
		if found {
			path = formatCell(displayRepoPath(repo.Path, cwd, roots), wrap, pathMax)
			branch = repo.Head.Branch
			if repo.Head.Detached {
				branch = "detached:" + branch
			}
			if repo.Type == "mirror" {
				branch = "-"
			}
			if repo.Worktree != nil {
				if repo.Worktree.Dirty {
					dirty = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "yes", termstyle.Warn)
				} else {
					dirty = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "no", termstyle.Healthy)
				}
			}
			tracking = displayTrackingStatus(repo.Tracking.Status)
			if repo.Type == "mirror" {
				// Mirror repos do not have branch tracking semantics in the same way
				// as a non-bare checkout.
				tracking = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "mirror", termstyle.Info)
			}
		}
		action := formatCell(describeSyncAction(res), wrap, actionMax)
		branch = formatCell(branch, wrap, branchMax)
		repoID := formatCell(res.RepoID, wrap, repoMax)
		if !wide {
			switch mode {
			case syncTableModeTiny:
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					path,
					action,
					ok,
					formatCell(res.Error, wrap, 28)); err != nil {
					return err
				}
				continue
			case syncTableModeCompact:
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					path,
					action,
					ok,
					formatCell(res.Error, wrap, 32),
					repoID); err != nil {
					return err
				}
				continue
			default:
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				path,
				action,
				branch,
				dirty,
				tracking,
				ok,
				res.ErrorClass,
				formatCell(res.Error, wrap, 36),
				repoID); err != nil {
				return err
			}
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
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			action,
			branch,
			dirty,
			tracking,
			ok,
			res.ErrorClass,
			formatCell(res.Error, wrap, 36),
			repoID,
			primaryRemote,
			upstream,
			ahead,
			behind); err != nil {
			return err
		}
	}
	return w.Flush()
}

func describeSyncAction(res engine.SyncResult) string {
	action := strings.TrimSpace(res.Action)

	// Prefer explicit skip reasons from the engine over heuristic action parsing.
	if strings.HasPrefix(res.Error, engine.SyncErrorSkippedLocalUpdatePrefix) {
		reason := strings.TrimSpace(strings.TrimPrefix(res.Error, engine.SyncErrorSkippedLocalUpdatePrefix))
		if reason == "already up to date" {
			return "fetch"
		}
		if reason == "" {
			return "skip local update"
		}
		return "skip local update (" + reason + ")"
	}
	if res.Error == engine.SyncErrorSkippedNoUpstream {
		return "skip no upstream"
	}
	if res.Error == engine.SyncErrorSkipped {
		return "skip"
	}
	if res.Error == engine.SyncErrorMissing {
		return "skip missing"
	}

	normalized := strings.ToLower(action)
	// Collapse verbose git command strings into stable, human-readable summaries.
	switch {
	case strings.Contains(normalized, "stash") && strings.Contains(normalized, "rebase"):
		return "stash & rebase"
	case strings.Contains(normalized, "hg pull"):
		return "fetch"
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

func writeSyncFailureSummary(cmd *cobra.Command, results []engine.SyncResult, cwd string, roots []string) error {
	failed := make([]engine.SyncResult, 0, len(results))
	for _, res := range results {
		if res.OK {
			continue
		}
		failed = append(failed, res)
	}
	if len(failed) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Failed sync operations:"); err != nil {
		return err
	}
	rows := make([][]string, 0, len(failed))
	for _, res := range failed {
		rows = append(rows, []string{
			displayRepoPath(res.Path, cwd, roots),
			describeSyncAction(res),
			res.ErrorClass,
			res.Error,
			res.RepoID,
		})
	}
	return cliio.WriteTable(cmd.ErrOrStderr(), false, false, []string{"PATH", "ACTION", "ERROR_CLASS", "ERROR", "REPO"}, rows)
}
