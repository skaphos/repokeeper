package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report repo health for all registered repositories",
	// @todo(milestone6): retain as legacy alias once `get repos` is added; remove
	// duplicated output/filter wiring here after kubectl-style command migration.
	RunE: func(cmd *cobra.Command, args []string) error {
		debugf(cmd, "starting status")
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

		adapter := vcs.NewGitAdapter(nil)
		eng := engine.New(cfg, reg, adapter)

		if roots != "" {
			debugf(cmd, "rescanning roots override")
			_, err := eng.Scan(cmd.Context(), engine.ScanOptions{
				Roots: splitCSV(roots),
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
			Filter:      engine.FilterKind(only),
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

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			writeStatusTable(cmd, report, cwd, []string{cfgRoot})
		default:
			return fmt.Errorf("unsupported format %q", format)
		}

		if statusHasWarningsOrErrors(report, reg) {
			raiseExitCode(1)
		}
		infof(cmd, "status completed: %d repos", len(report.Repos))
		return nil
	},
}

func init() {
	statusCmd.Flags().String("roots", "", "additional roots to scan (optional)")
	statusCmd.Flags().String("registry", "", "override registry file path")
	statusCmd.Flags().String("format", "table", "output format: table or json")
	statusCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing")

	rootCmd.AddCommand(statusCmd)
}

func writeStatusTable(cmd *cobra.Command, report *model.StatusReport, cwd string, roots []string) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.StripEscape)
	_, _ = fmt.Fprintln(w, "PATH\tBRANCH\tDIRTY\tTRACKING")
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
				dirty = colorize("yes", ansiBrown)
			} else {
				dirty = colorize("no", ansiGreen)
			}
		}
		tracking := displayTrackingStatus(repo.Tracking.Status)
		if repo.Type == "mirror" {
			tracking = colorize("mirror", ansiBlue)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			path,
			branch,
			dirty,
			tracking)
	}
	_ = w.Flush()
}

const (
	ansiReset = "\x1b[0m"
	ansiGreen = "\x1b[32m"
	ansiBrown = "\x1b[33m"
	ansiRed   = "\x1b[31m"
	ansiBlue  = "\x1b[34m"
)

func colorize(value, color string) string {
	if flagNoColor || value == "" || color == "" {
		return value
	}
	// Hide ANSI sequences from tabwriter width calculations so columns align.
	esc := string([]byte{tabwriter.Escape})
	return esc + color + esc + value + esc + ansiReset + esc
}

func displayTrackingStatus(status model.TrackingStatus) string {
	switch status {
	case model.TrackingEqual:
		return colorize("up to date", ansiGreen)
	case model.TrackingDiverged:
		return colorize(string(status), ansiRed)
	case model.TrackingGone:
		return colorize(string(status), ansiRed)
	default:
		return string(status)
	}
}

func displayRepoPath(repoPath, cwd string, roots []string) string {
	if repoPath == "" {
		return repoPath
	}
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
