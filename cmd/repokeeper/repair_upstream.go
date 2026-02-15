package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

type repairUpstreamResult struct {
	RepoID          string `json:"repo_id"`
	Path            string `json:"path"`
	LocalBranch     string `json:"local_branch"`
	CurrentUpstream string `json:"current_upstream"`
	TargetUpstream  string `json:"target_upstream"`
	Action          string `json:"action"`
	OK              bool   `json:"ok"`
	ErrorClass      string `json:"error_class,omitempty"`
	Error           string `json:"error,omitempty"`
}

var repairUpstreamCmd = &cobra.Command{
	Use:   "repair-upstream",
	Short: "Repair missing or mismatched upstream tracking for registered repos",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		registryOverride, _ := cmd.Flags().GetString("registry")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		only, _ := cmd.Flags().GetString("only")
		format, _ := cmd.Flags().GetString("format")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")

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

		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil))
		report, err := eng.Status(cmd.Context(), engine.StatusOptions{
			Filter:      engine.FilterAll,
			Concurrency: cfg.Defaults.Concurrency,
			Timeout:     cfg.Defaults.TimeoutSeconds,
		})
		if err != nil {
			return err
		}

		statusByPath := make(map[string]model.RepoStatus, len(report.Repos))
		for _, repo := range report.Repos {
			statusByPath[repo.Path] = repo
		}

		entries := append([]registry.Entry(nil), reg.Entries...)
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].RepoID == entries[j].RepoID {
				return entries[i].Path < entries[j].Path
			}
			return entries[i].RepoID < entries[j].RepoID
		})

		runner := &gitx.GitRunner{}
		results := make([]repairUpstreamResult, 0, len(entries))
		registryMutated := false

		for _, entry := range entries {
			res := repairUpstreamResult{
				RepoID: entry.RepoID,
				Path:   entry.Path,
				OK:     true,
				Action: "unchanged",
			}

			if entry.Status == registry.StatusMissing {
				res.Action = "skip missing"
				results = append(results, res)
				continue
			}

			repo, found := statusByPath[entry.Path]
			if !found {
				res.OK = false
				res.Action = "failed"
				res.ErrorClass = "missing"
				res.Error = "status missing for registry path"
				results = append(results, res)
				continue
			}
			res.LocalBranch = repo.Head.Branch
			res.CurrentUpstream = strings.TrimSpace(repo.Tracking.Upstream)

			if repo.Error != "" {
				res.Action = "skip status error"
				res.ErrorClass = repo.ErrorClass
				res.Error = repo.Error
				results = append(results, res)
				continue
			}
			if repo.Head.Detached || strings.TrimSpace(repo.Head.Branch) == "" {
				res.Action = "skip detached"
				results = append(results, res)
				continue
			}

			remote := strings.TrimSpace(repo.PrimaryRemote)
			if remote == "" {
				res.Action = "skip no remote"
				results = append(results, res)
				continue
			}

			targetBranch := strings.TrimSpace(entry.Branch)
			if targetBranch == "" {
				targetBranch = strings.TrimSpace(repo.Head.Branch)
			}
			if targetBranch == "" {
				res.Action = "skip no branch"
				results = append(results, res)
				continue
			}
			targetUpstream := remote + "/" + targetBranch
			res.TargetUpstream = targetUpstream

			needsRepair := needsUpstreamRepair(repo, targetUpstream)
			if !repairUpstreamMatchesFilter(res.CurrentUpstream, targetUpstream, only) {
				res.Action = "filtered"
				results = append(results, res)
				continue
			}
			if !needsRepair {
				results = append(results, res)
				continue
			}

			if dryRun {
				res.Action = "would repair"
				results = append(results, res)
				continue
			}

			if _, err := runner.Run(cmd.Context(), entry.Path, "branch", "--set-upstream-to", targetUpstream, repo.Head.Branch); err != nil {
				res.OK = false
				res.Action = "failed"
				res.ErrorClass = gitx.ClassifyError(err)
				res.Error = err.Error()
				results = append(results, res)
				continue
			}

			res.Action = "repaired"
			results = append(results, res)

			entry.Branch = targetBranch
			entry.LastSeen = time.Now()
			entry.Status = registry.StatusPresent
			setRegistryEntryByRepoID(reg, entry)
			registryMutated = true
		}

		if registryMutated {
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

		switch strings.ToLower(format) {
		case "json":
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		case "table":
			writeRepairUpstreamTable(cmd, results, cwd, []string{cfgRoot}, noHeaders)
		default:
			return fmt.Errorf("unsupported format %q", format)
		}

		for _, res := range results {
			if !res.OK {
				raiseExitCode(2)
			}
		}
		return nil
	},
}

func needsUpstreamRepair(repo model.RepoStatus, targetUpstream string) bool {
	current := strings.TrimSpace(repo.Tracking.Upstream)
	target := strings.TrimSpace(targetUpstream)
	if target == "" {
		return false
	}
	if current != target {
		return true
	}
	return repo.Tracking.Status == model.TrackingNone
}

func repairUpstreamMatchesFilter(current, target, filter string) bool {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return true
	case "missing":
		return strings.TrimSpace(current) == ""
	case "mismatch":
		return strings.TrimSpace(current) != "" && strings.TrimSpace(current) != strings.TrimSpace(target)
	default:
		return true
	}
}

func writeRepairUpstreamTable(cmd *cobra.Command, results []repairUpstreamResult, cwd string, roots []string, noHeaders bool) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if !noHeaders {
		_, _ = fmt.Fprintln(w, "PATH\tACTION\tBRANCH\tCURRENT\tTARGET\tOK\tERROR_CLASS\tERROR\tREPO")
	}
	for _, res := range results {
		ok := "yes"
		if !res.OK {
			ok = "no"
		}
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			displayRepoPath(res.Path, cwd, roots),
			res.Action,
			res.LocalBranch,
			res.CurrentUpstream,
			res.TargetUpstream,
			ok,
			res.ErrorClass,
			res.Error,
			res.RepoID,
		)
	}
	_ = w.Flush()
}

func init() {
	repairUpstreamCmd.Flags().String("registry", "", "override registry file path")
	repairUpstreamCmd.Flags().Bool("dry-run", true, "preview upstream repairs without executing git changes")
	repairUpstreamCmd.Flags().String("only", "all", "filter: all, missing, mismatch")
	repairUpstreamCmd.Flags().StringP("format", "o", "table", "output format: table or json")
	repairUpstreamCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")
}
