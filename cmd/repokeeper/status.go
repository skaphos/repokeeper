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
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report repo health for all registered repositories",
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
		debugf(cmd, "using config %s", cfgPath)

		registryOverride, _ := cmd.Flags().GetString("registry")
		regPath := cfg.RegistryPath
		if registryOverride != "" {
			regPath = registryOverride
		}

		reg, err := registry.Load(regPath)
		if err != nil {
			return err
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
			if err := registry.Save(reg, regPath); err != nil {
				return err
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
			writeStatusTable(cmd, report)
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
	statusCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, missing")

	rootCmd.AddCommand(statusCmd)
}

func writeStatusTable(cmd *cobra.Command, report *model.StatusReport) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPO\tPATH\tBRANCH\tDIRTY\tTRACKING\tAHEAD\tBEHIND\tERROR_CLASS\tERROR")
	for _, repo := range report.Repos {
		branch := repo.Head.Branch
		if repo.Head.Detached {
			branch = "detached:" + branch
		}
		dirty := "-"
		if repo.Worktree != nil {
			if repo.Worktree.Dirty {
				dirty = "yes"
			} else {
				dirty = "no"
			}
		}
		tracking := string(repo.Tracking.Status)
		ahead := "-"
		behind := "-"
		if repo.Tracking.Ahead != nil {
			ahead = fmt.Sprintf("%d", *repo.Tracking.Ahead)
		}
		if repo.Tracking.Behind != nil {
			behind = fmt.Sprintf("%d", *repo.Tracking.Behind)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			repo.RepoID, repo.Path, branch, dirty, tracking, ahead, behind, repo.ErrorClass, repo.Error)
	}
	_ = w.Flush()
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
