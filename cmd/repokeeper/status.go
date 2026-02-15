package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/remotemismatch"
	"github.com/skaphos/repokeeper/internal/strutil"
	"github.com/skaphos/repokeeper/internal/tableutil"
	"github.com/skaphos/repokeeper/internal/termstyle"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

type divergedAdvice struct {
	RepoID            string `json:"repo_id"`
	Path              string `json:"path"`
	Branch            string `json:"branch"`
	Upstream          string `json:"upstream"`
	Reason            string `json:"reason"`
	RecommendedAction string `json:"recommended_action"`
}

type remoteMismatchReconcileMode = remotemismatch.ReconcileMode

const (
	remoteMismatchReconcileNone     = remotemismatch.ReconcileNone
	remoteMismatchReconcileRegistry = remotemismatch.ReconcileRegistry
	remoteMismatchReconcileGit      = remotemismatch.ReconcileGit
)

type remoteMismatchPlan = remotemismatch.Plan

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report repo health for all registered repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		debugf(cmd, "starting status")
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

		roots, _ := cmd.Flags().GetString("roots")
		format, _ := cmd.Flags().GetString("format")
		only, _ := cmd.Flags().GetString("only")
		fieldSelector, _ := cmd.Flags().GetString("field-selector")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")
		reconcileModeRaw, _ := cmd.Flags().GetString("reconcile-remote-mismatch")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		filter, err := resolveRepoFilter(only, fieldSelector)
		if err != nil {
			return err
		}
		reconcileMode, err := parseRemoteMismatchReconcileMode(reconcileModeRaw)
		if err != nil {
			return err
		}

		adapter := vcs.NewGitAdapter(nil)
		eng := engine.New(cfg, reg, adapter)

		if roots != "" {
			debugf(cmd, "rescanning roots override")
			_, err := eng.Scan(cmd.Context(), engine.ScanOptions{
				Roots: strutil.SplitCSV(roots),
			})
			if err != nil {
				return err
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
		}

		report, err := eng.Status(cmd.Context(), engine.StatusOptions{
			Filter:      filter,
			Concurrency: 0,
			Timeout:     0,
		})
		if err != nil {
			return err
		}
		plans := buildRemoteMismatchPlans(report.Repos, reg, adapter, reconcileMode)
		if len(plans) > 0 {
			logOutputWriteFailure(cmd, "status remote mismatch plan", writeRemoteMismatchPlan(cmd, plans, cwd, []string{cfgRoot}, dryRun || reconcileMode == remoteMismatchReconcileNone))
		}
		if reconcileMode != remoteMismatchReconcileNone && !dryRun {
			if !assumeYes(cmd) {
				confirmed, err := confirmWithPrompt(cmd, "Proceed with remote mismatch reconciliation? [y/N]: ")
				if err != nil {
					return err
				}
				if !confirmed {
					infof(cmd, "remote mismatch reconcile cancelled")
					return nil
				}
			}
			if err := applyRemoteMismatchPlans(cmd, plans, reg, reconcileMode); err != nil {
				return err
			}
			if reconcileMode == remoteMismatchReconcileRegistry {
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
			report, err = eng.Status(cmd.Context(), engine.StatusOptions{
				Filter:      filter,
				Concurrency: 0,
				Timeout:     0,
			})
			if err != nil {
				return err
			}
		}

		switch strings.ToLower(format) {
		case "json":
			setColorOutputMode(cmd, format)
			output := any(report)
			if filter == engine.FilterDiverged {
				output = struct {
					*model.StatusReport
					Diverged []divergedAdvice `json:"diverged"`
				}{
					StatusReport: report,
					Diverged:     buildDivergedAdvice(report.Repos),
				}
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			logOutputWriteFailure(cmd, "status json", err)
		case "table":
			setColorOutputMode(cmd, format)
			if filter == engine.FilterDiverged {
				logOutputWriteFailure(cmd, "status diverged table", writeDivergedStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, false))
				break
			}
			logOutputWriteFailure(cmd, "status table", writeStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, false))
		case "wide":
			setColorOutputMode(cmd, format)
			if filter == engine.FilterDiverged {
				logOutputWriteFailure(cmd, "status diverged wide", writeDivergedStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, true))
				break
			}
			logOutputWriteFailure(cmd, "status wide", writeStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, true))
		default:
			return fmt.Errorf("unsupported format %q", format)
		}

		if statusHasWarningsOrErrors(report, reg) {
			raiseExitCode(cmd, 1)
		}
		infof(cmd, "status completed: %d repos", len(report.Repos))
		return nil
	},
}

func init() {
	statusCmd.Flags().String("roots", "", "additional roots to scan (optional)")
	statusCmd.Flags().String("registry", "", "override registry file path")
	addFormatFlag(statusCmd, "output format: table, wide, or json")
	addRepoFilterFlags(statusCmd)
	statusCmd.Flags().String("reconcile-remote-mismatch", "none", "optional reconcile mode for remote mismatch: none, registry, git")
	statusCmd.Flags().Bool("dry-run", true, "preview reconcile actions without modifying registry or git remotes")
	addNoHeadersFlag(statusCmd)

}

func writeStatusTable(cmd *cobra.Command, report *model.StatusReport, cwd string, roots []string, noHeaders bool, wide bool) error {
	w := tableutil.New(cmd.OutOrStdout(), true)
	showBranch := true
	showDirty := true
	if !wide {
		width, hasWidth := tableWidth(cmd)
		switch {
		case hasWidth && width < tinyTableWidth:
			showBranch = false
			showDirty = false
		case hasWidth && width < narrowTableWidth:
			showDirty = false
		}
	}
	headers := "PATH"
	if showBranch {
		headers += "\tBRANCH"
	}
	if showDirty {
		headers += "\tDIRTY"
	}
	headers += "\tTRACKING"
	if wide {
		headers = "PATH\tBRANCH\tDIRTY\tTRACKING\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND\tERROR_CLASS"
	}
	if err := tableutil.PrintHeaders(w, noHeaders, headers); err != nil {
		return err
	}
	pathMax := adaptiveCellLimit(cmd, 0, 48, 32)
	branchMax := adaptiveCellLimit(cmd, 0, 24, 16)
	for _, repo := range report.Repos {
		branch := repo.Head.Branch
		if repo.Head.Detached {
			branch = "detached:" + branch
		}
		if repo.Type == "mirror" {
			branch = "-"
		}
		path := formatCell(displayRepoPath(repo.Path, cwd, roots), false, pathMax)
		branch = formatCell(branch, false, branchMax)
		dirty := "-"
		if repo.Worktree != nil {
			if repo.Worktree.Dirty {
				dirty = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "yes", termstyle.Warn)
			} else {
				dirty = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "no", termstyle.Healthy)
			}
		}
		tracking := displayTrackingStatus(repo.Tracking.Status)
		if repo.Type == "mirror" {
			// Mirrors are bare repos; tracking labels are not meaningful per branch.
			tracking = termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "mirror", termstyle.Info)
		}
		if !wide {
			row := []string{path}
			if showBranch {
				row = append(row, branch)
			}
			if showDirty {
				row = append(row, dirty)
			}
			row = append(row, tracking)
			if _, err := fmt.Fprintf(w, "%s\n", strings.Join(row, "\t")); err != nil {
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
		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			branch,
			dirty,
			tracking,
			repo.PrimaryRemote,
			repo.Tracking.Upstream,
			ahead,
			behind,
			repo.ErrorClass,
		); err != nil {
			return err
		}
	}
	return w.Flush()
}

func writeDivergedStatusTable(cmd *cobra.Command, report *model.StatusReport, cwd string, roots []string, noHeaders bool, wide bool) error {
	adviceByPath := make(map[string]divergedAdvice, len(report.Repos))
	for _, advice := range buildDivergedAdvice(report.Repos) {
		adviceByPath[advice.Path] = advice
	}

	w := tableutil.New(cmd.OutOrStdout(), true)
	headers := "PATH\tBRANCH\tTRACKING\tREASON\tRECOMMENDED_ACTION"
	if wide {
		headers = "PATH\tBRANCH\tTRACKING\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND\tREASON\tRECOMMENDED_ACTION"
	}
	if err := tableutil.PrintHeaders(w, noHeaders, headers); err != nil {
		return err
	}
	pathMax := adaptiveCellLimit(cmd, 0, 48, 32)
	branchMax := adaptiveCellLimit(cmd, 0, 24, 16)
	reasonMax := adaptiveCellLimit(cmd, 0, 36, 24)
	actionMax := adaptiveCellLimit(cmd, 0, 36, 24)
	for _, repo := range report.Repos {
		advice, ok := adviceByPath[repo.Path]
		if !ok {
			continue
		}
		branch := repo.Head.Branch
		if repo.Head.Detached {
			branch = "detached:" + branch
		}
		path := formatCell(displayRepoPath(repo.Path, cwd, roots), false, pathMax)
		branch = formatCell(branch, false, branchMax)
		tracking := displayTrackingStatus(repo.Tracking.Status)
		reason := formatCell(advice.Reason, false, reasonMax)
		action := formatCell(advice.RecommendedAction, false, actionMax)
		if !wide {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", path, branch, tracking, reason, action); err != nil {
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
		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			branch,
			tracking,
			repo.PrimaryRemote,
			repo.Tracking.Upstream,
			ahead,
			behind,
			reason,
			action,
		); err != nil {
			return err
		}
	}
	return w.Flush()
}

func displayTrackingStatus(status model.TrackingStatus) string {
	switch status {
	case model.TrackingEqual:
		return termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, "up to date", termstyle.Healthy)
	case model.TrackingDiverged:
		return termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, string(status), termstyle.Error)
	case model.TrackingGone:
		return termstyle.Colorize(runtimeStateFor(rootCmd).colorOutputEnabled, string(status), termstyle.Error)
	default:
		return string(status)
	}
}

func displayRepoPath(repoPath, cwd string, roots []string) string {
	if repoPath == "" {
		return repoPath
	}
	// Prefer paths relative to CWD, then configured roots, then absolute fallback.
	if rel, ok := relWithin(cwd, repoPath); ok {
		return rel
	}
	for _, root := range roots {
		if rel, ok := relWithin(root, repoPath); ok {
			return rel
		}
	}
	return repoPath
}

func formatCell(value string, wrap bool, max int) string {
	if wrap || max <= 0 {
		return value
	}
	return truncateASCII(value, max)
}

func truncateASCII(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func statusHasWarningsOrErrors(report *model.StatusReport, reg *registry.Registry) bool {
	for _, repo := range report.Repos {
		if repo.Error != "" || repo.Tracking.Status == model.TrackingGone || (repo.Worktree != nil && repo.Worktree.Dirty) {
			return true
		}
	}
	for _, entry := range reg.Entries {
		if entry.Status == registry.StatusMissing || entry.Status == registry.StatusMoved {
			return true
		}
	}
	return false
}

func writeStatusDetails(cmd *cobra.Command, repo model.RepoStatus, cwd string, roots []string) error {
	// Detail output is intentionally color-free and key/value stable for scripting.
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "PATH: %s\n", displayRepoPath(repo.Path, cwd, roots)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "PATH_ABS: %s\n", repo.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "REPO: %s\n", repo.RepoID); err != nil {
		return err
	}
	if repo.Type != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "TYPE: %s\n", repo.Type); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "BARE: %t\n", repo.Bare); err != nil {
		return err
	}
	branch := repo.Head.Branch
	if repo.Head.Detached {
		branch = "detached:" + branch
	}
	if repo.Type == "mirror" {
		branch = "-"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "BRANCH: %s\n", branch); err != nil {
		return err
	}
	dirty := "-"
	if repo.Worktree != nil {
		if repo.Worktree.Dirty {
			dirty = "yes"
		} else {
			dirty = "no"
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "DIRTY: %s\n", dirty); err != nil {
		return err
	}
	tracking := displayTrackingStatusNoColor(repo.Tracking.Status)
	if repo.Type == "mirror" {
		tracking = "mirror"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "TRACKING: %s\n", tracking); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "UPSTREAM: %s\n", repo.Tracking.Upstream); err != nil {
		return err
	}
	ahead := "-"
	if repo.Tracking.Ahead != nil {
		ahead = fmt.Sprintf("%d", *repo.Tracking.Ahead)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "AHEAD: %s\n", ahead); err != nil {
		return err
	}
	behind := "-"
	if repo.Tracking.Behind != nil {
		behind = fmt.Sprintf("%d", *repo.Tracking.Behind)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "BEHIND: %s\n", behind); err != nil {
		return err
	}
	if repo.ErrorClass != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ERROR_CLASS: %s\n", repo.ErrorClass); err != nil {
			return err
		}
	}
	if repo.Error != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ERROR: %s\n", repo.Error); err != nil {
			return err
		}
	}
	return nil
}

func buildDivergedAdvice(repos []model.RepoStatus) []divergedAdvice {
	advice := make([]divergedAdvice, 0, len(repos))
	for _, repo := range repos {
		if repo.Tracking.Status != model.TrackingDiverged {
			continue
		}
		reason, action := divergedReasonAndAction(repo)
		advice = append(advice, divergedAdvice{
			RepoID:            repo.RepoID,
			Path:              repo.Path,
			Branch:            repo.Head.Branch,
			Upstream:          repo.Tracking.Upstream,
			Reason:            reason,
			RecommendedAction: action,
		})
	}
	return advice
}

func divergedReasonAndAction(repo model.RepoStatus) (string, string) {
	if repo.Tracking.Status != model.TrackingDiverged {
		return "", ""
	}
	if repo.Worktree != nil && repo.Worktree.Dirty {
		return "local and upstream histories diverged with uncommitted changes", "commit or stash changes, then resolve with manual rebase/merge"
	}
	if repo.Tracking.Ahead != nil && repo.Tracking.Behind != nil {
		return fmt.Sprintf("branch is %d ahead and %d behind upstream", *repo.Tracking.Ahead, *repo.Tracking.Behind), "resolve manually, or run reconcile with --update-local --force if acceptable"
	}
	return "local and upstream histories diverged", "resolve manually, or run reconcile with --update-local --force if acceptable"
}

func displayTrackingStatusNoColor(status model.TrackingStatus) string {
	if status == model.TrackingEqual {
		return "up to date"
	}
	return string(status)
}

func relWithin(base, target string) (string, bool) {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(target) == "" {
		return "", false
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil || rel == "." || rel == ".." {
		return "", false
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func parseRemoteMismatchReconcileMode(raw string) (remoteMismatchReconcileMode, error) {
	return remotemismatch.ParseReconcileMode(raw)
}

func buildRemoteMismatchPlans(repos []model.RepoStatus, reg *registry.Registry, adapter vcs.Adapter, mode remoteMismatchReconcileMode) []remoteMismatchPlan {
	return remotemismatch.BuildPlans(repos, reg, adapter, mode)
}

func writeRemoteMismatchPlan(cmd *cobra.Command, plans []remoteMismatchPlan, cwd string, roots []string, dryRun bool) error {
	if len(plans) == 0 {
		return nil
	}
	modeLabel := "planned"
	if !dryRun {
		modeLabel = "applying"
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Remote mismatch reconcile (%s):\n", modeLabel); err != nil {
		return err
	}
	rows := make([][]string, 0, len(plans))
	for _, plan := range plans {
		rows = append(rows, []string{
			displayRepoPath(plan.Path, cwd, roots),
			plan.Action,
			plan.PrimaryRemote,
			plan.RepoRemoteURL,
			plan.RegistryURL,
			plan.RepoID,
		})
	}
	return cliio.WriteTable(
		cmd.ErrOrStderr(),
		false,
		false,
		[]string{"PATH", "ACTION", "PRIMARY_REMOTE", "GIT_REMOTE_URL", "REGISTRY_REMOTE_URL", "REPO"},
		rows,
	)
}

func applyRemoteMismatchPlans(cmd *cobra.Command, plans []remoteMismatchPlan, reg *registry.Registry, mode remoteMismatchReconcileMode) error {
	return remotemismatch.ApplyPlans(cmd.Context(), plans, reg, mode, vcs.NewGitAdapter(nil), nil)
}
