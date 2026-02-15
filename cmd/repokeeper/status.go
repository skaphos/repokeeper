package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/strutil"
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

type remoteMismatchReconcileMode string

const (
	remoteMismatchReconcileNone     remoteMismatchReconcileMode = "none"
	remoteMismatchReconcileRegistry remoteMismatchReconcileMode = "registry"
	remoteMismatchReconcileGit      remoteMismatchReconcileMode = "git"
)

type remoteMismatchPlan struct {
	RepoID        string
	Path          string
	PrimaryRemote string
	RepoRemoteURL string
	RegistryURL   string
	EntryIndex    int
	Action        string
}

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
				return fmt.Errorf("registry not found in %s (run repokeeper scan first)", cfgPath)
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
			Concurrency: cfg.Defaults.Concurrency,
			Timeout:     cfg.Defaults.TimeoutSeconds,
		})
		if err != nil {
			return err
		}
		// Sort once here so both table and JSON output stay predictable.
		sort.SliceStable(report.Repos, func(i, j int) bool {
			if report.Repos[i].RepoID == report.Repos[j].RepoID {
				return report.Repos[i].Path < report.Repos[j].Path
			}
			return report.Repos[i].RepoID < report.Repos[j].RepoID
		})
		plans := buildRemoteMismatchPlans(report.Repos, reg, adapter, reconcileMode)
		if len(plans) > 0 {
			writeRemoteMismatchPlan(cmd, plans, cwd, []string{cfgRoot}, dryRun || reconcileMode == remoteMismatchReconcileNone)
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
				Concurrency: cfg.Defaults.Concurrency,
				Timeout:     cfg.Defaults.TimeoutSeconds,
			})
			if err != nil {
				return err
			}
			sort.SliceStable(report.Repos, func(i, j int) bool {
				if report.Repos[i].RepoID == report.Repos[j].RepoID {
					return report.Repos[i].Path < report.Repos[j].Path
				}
				return report.Repos[i].RepoID < report.Repos[j].RepoID
			})
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
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			setColorOutputMode(cmd, format)
			if filter == engine.FilterDiverged {
				writeDivergedStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, false)
				break
			}
			writeStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, false)
		case "wide":
			setColorOutputMode(cmd, format)
			if filter == engine.FilterDiverged {
				writeDivergedStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, true)
				break
			}
			writeStatusTable(cmd, report, cwd, []string{cfgRoot}, noHeaders, true)
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
	statusCmd.Flags().StringP("format", "o", "table", "output format: table, wide, or json")
	statusCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing")
	statusCmd.Flags().String("field-selector", "", "field selector (phase 1): tracking.status=diverged|gone, worktree.dirty=true|false, repo.error=true, repo.missing=true, remote.mismatch=true")
	statusCmd.Flags().String("reconcile-remote-mismatch", "none", "optional reconcile mode for remote mismatch: none, registry, git")
	statusCmd.Flags().Bool("dry-run", true, "preview reconcile actions without modifying registry or git remotes")
	statusCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")

}

func writeStatusTable(cmd *cobra.Command, report *model.StatusReport, cwd string, roots []string, noHeaders bool, wide bool) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.StripEscape)
	if !noHeaders {
		headers := "PATH\tBRANCH\tDIRTY\tTRACKING"
		if wide {
			headers += "\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND\tERROR_CLASS"
		}
		_, _ = fmt.Fprintln(w, headers)
	}
	for _, repo := range report.Repos {
		branch := repo.Head.Branch
		if repo.Head.Detached {
			branch = "detached:" + branch
		}
		if repo.Type == "mirror" {
			branch = "-"
		}
		path := displayRepoPath(repo.Path, cwd, roots)
		dirty := "-"
		if repo.Worktree != nil {
			if repo.Worktree.Dirty {
				dirty = colorize("yes", ansiWarn)
			} else {
				dirty = colorize("no", ansiHealthy)
			}
		}
		tracking := displayTrackingStatus(repo.Tracking.Status)
		if repo.Type == "mirror" {
			// Mirrors are bare repos; tracking labels are not meaningful per branch.
			tracking = colorize("mirror", ansiInfo)
		}
		if !wide {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				path,
				branch,
				dirty,
				tracking)
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
		_, _ = fmt.Fprintf(
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
		)
	}
	_ = w.Flush()
}

func writeDivergedStatusTable(cmd *cobra.Command, report *model.StatusReport, cwd string, roots []string, noHeaders bool, wide bool) {
	adviceByPath := make(map[string]divergedAdvice, len(report.Repos))
	for _, advice := range buildDivergedAdvice(report.Repos) {
		adviceByPath[advice.Path] = advice
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.StripEscape)
	if !noHeaders {
		headers := "PATH\tBRANCH\tTRACKING\tREASON\tRECOMMENDED_ACTION"
		if wide {
			headers = "PATH\tBRANCH\tTRACKING\tPRIMARY_REMOTE\tUPSTREAM\tAHEAD\tBEHIND\tREASON\tRECOMMENDED_ACTION"
		}
		_, _ = fmt.Fprintln(w, headers)
	}
	for _, repo := range report.Repos {
		advice, ok := adviceByPath[repo.Path]
		if !ok {
			continue
		}
		branch := repo.Head.Branch
		if repo.Head.Detached {
			branch = "detached:" + branch
		}
		path := displayRepoPath(repo.Path, cwd, roots)
		tracking := displayTrackingStatus(repo.Tracking.Status)
		if !wide {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", path, branch, tracking, advice.Reason, advice.RecommendedAction)
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
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			path,
			branch,
			tracking,
			repo.PrimaryRemote,
			repo.Tracking.Upstream,
			ahead,
			behind,
			advice.Reason,
			advice.RecommendedAction,
		)
	}
	_ = w.Flush()
}

const (
	ansiReset = "\x1b[0m"
	ansiGreen = "\x1b[32m"
	ansiBrown = "\x1b[33m"
	ansiRed   = "\x1b[31m"
	ansiBlue  = "\x1b[34m"

	// Semantic color aliases for consistent status/sync styling.
	ansiHealthy = ansiGreen
	ansiWarn    = ansiBrown
	ansiError   = ansiRed
	ansiInfo    = ansiBlue
)

func colorize(value, color string) string {
	if !runtimeStateFor(rootCmd).colorOutputEnabled || value == "" || color == "" {
		return value
	}
	// Hide ANSI sequences from tabwriter width calculations so columns align.
	esc := string([]byte{tabwriter.Escape})
	return esc + color + esc + value + esc + ansiReset + esc
}

func displayTrackingStatus(status model.TrackingStatus) string {
	switch status {
	case model.TrackingEqual:
		return colorize("up to date", ansiHealthy)
	case model.TrackingDiverged:
		return colorize(string(status), ansiError)
	case model.TrackingGone:
		return colorize(string(status), ansiError)
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

func writeStatusDetails(cmd *cobra.Command, repo model.RepoStatus, cwd string, roots []string) {
	// Detail output is intentionally color-free and key/value stable for scripting.
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PATH: %s\n", displayRepoPath(repo.Path, cwd, roots))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PATH_ABS: %s\n", repo.Path)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "REPO: %s\n", repo.RepoID)
	if repo.Type != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "TYPE: %s\n", repo.Type)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "BARE: %t\n", repo.Bare)
	branch := repo.Head.Branch
	if repo.Head.Detached {
		branch = "detached:" + branch
	}
	if repo.Type == "mirror" {
		branch = "-"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "BRANCH: %s\n", branch)
	dirty := "-"
	if repo.Worktree != nil {
		if repo.Worktree.Dirty {
			dirty = "yes"
		} else {
			dirty = "no"
		}
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "DIRTY: %s\n", dirty)
	tracking := displayTrackingStatusNoColor(repo.Tracking.Status)
	if repo.Type == "mirror" {
		tracking = "mirror"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "TRACKING: %s\n", tracking)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "UPSTREAM: %s\n", repo.Tracking.Upstream)
	ahead := "-"
	if repo.Tracking.Ahead != nil {
		ahead = fmt.Sprintf("%d", *repo.Tracking.Ahead)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "AHEAD: %s\n", ahead)
	behind := "-"
	if repo.Tracking.Behind != nil {
		behind = fmt.Sprintf("%d", *repo.Tracking.Behind)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "BEHIND: %s\n", behind)
	if repo.ErrorClass != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ERROR_CLASS: %s\n", repo.ErrorClass)
	}
	if repo.Error != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ERROR: %s\n", repo.Error)
	}
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
	mode := remoteMismatchReconcileMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case "", remoteMismatchReconcileNone:
		return remoteMismatchReconcileNone, nil
	case remoteMismatchReconcileRegistry, remoteMismatchReconcileGit:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported --reconcile-remote-mismatch value %q (expected none, registry, or git)", raw)
	}
}

func buildRemoteMismatchPlans(repos []model.RepoStatus, reg *registry.Registry, adapter vcs.Adapter, mode remoteMismatchReconcileMode) []remoteMismatchPlan {
	if reg == nil || adapter == nil || mode == remoteMismatchReconcileNone {
		return nil
	}
	plans := make([]remoteMismatchPlan, 0)
	for _, repo := range repos {
		entryIndex := findRegistryEntryIndexForStatus(reg, repo)
		if entryIndex < 0 {
			continue
		}
		entry := reg.Entries[entryIndex]
		registryURL := strings.TrimSpace(entry.RemoteURL)
		if registryURL == "" || strings.TrimSpace(repo.RepoID) == "" {
			continue
		}
		if adapter.NormalizeURL(registryURL) == repo.RepoID {
			continue
		}
		repoRemoteURL := primaryRemoteURL(repo)
		action := ""
		switch mode {
		case remoteMismatchReconcileRegistry:
			if repoRemoteURL == "" {
				continue
			}
			action = "set registry remote_url to live git remote"
		case remoteMismatchReconcileGit:
			if strings.TrimSpace(repo.PrimaryRemote) == "" {
				continue
			}
			action = "set git remote URL to registry remote_url"
		}
		plans = append(plans, remoteMismatchPlan{
			RepoID:        repo.RepoID,
			Path:          repo.Path,
			PrimaryRemote: repo.PrimaryRemote,
			RepoRemoteURL: repoRemoteURL,
			RegistryURL:   registryURL,
			EntryIndex:    entryIndex,
			Action:        action,
		})
	}
	return plans
}

func findRegistryEntryIndexForStatus(reg *registry.Registry, repo model.RepoStatus) int {
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == repo.RepoID && reg.Entries[i].Path == repo.Path {
			return i
		}
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == repo.RepoID {
			return i
		}
	}
	return -1
}

func primaryRemoteURL(repo model.RepoStatus) string {
	for _, remote := range repo.Remotes {
		if remote.Name == repo.PrimaryRemote {
			return strings.TrimSpace(remote.URL)
		}
	}
	return ""
}

func writeRemoteMismatchPlan(cmd *cobra.Command, plans []remoteMismatchPlan, cwd string, roots []string, dryRun bool) {
	if len(plans) == 0 {
		return
	}
	modeLabel := "planned"
	if !dryRun {
		modeLabel = "applying"
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Remote mismatch reconcile (%s):\n", modeLabel)
	w := tabwriter.NewWriter(cmd.ErrOrStderr(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tACTION\tPRIMARY_REMOTE\tGIT_REMOTE_URL\tREGISTRY_REMOTE_URL\tREPO")
	for _, plan := range plans {
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			displayRepoPath(plan.Path, cwd, roots),
			plan.Action,
			plan.PrimaryRemote,
			plan.RepoRemoteURL,
			plan.RegistryURL,
			plan.RepoID,
		)
	}
	_ = w.Flush()
}

func applyRemoteMismatchPlans(cmd *cobra.Command, plans []remoteMismatchPlan, reg *registry.Registry, mode remoteMismatchReconcileMode) error {
	if len(plans) == 0 {
		return nil
	}
	switch mode {
	case remoteMismatchReconcileRegistry:
		for _, plan := range plans {
			if plan.EntryIndex < 0 || plan.EntryIndex >= len(reg.Entries) {
				continue
			}
			reg.Entries[plan.EntryIndex].RemoteURL = plan.RepoRemoteURL
			reg.Entries[plan.EntryIndex].LastSeen = time.Now()
		}
	case remoteMismatchReconcileGit:
		runner := &gitx.GitRunner{}
		for _, plan := range plans {
			if strings.TrimSpace(plan.PrimaryRemote) == "" {
				continue
			}
			if _, err := runner.Run(cmd.Context(), plan.Path, "remote", "set-url", plan.PrimaryRemote, plan.RegistryURL); err != nil {
				return fmt.Errorf("git remote set-url %s %s (%s): %w", plan.PrimaryRemote, plan.RegistryURL, plan.Path, err)
			}
		}
	}
	return nil
}
