package repokeeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show detailed status for one repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribeRepo,
}

var describeRepoCmd = &cobra.Command{
	Use:   "repo <repo-id-or-path>",
	Short: "Show detailed status for one repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribeRepo,
}

func runDescribeRepo(cmd *cobra.Command, args []string) error {
	debugf(cmd, "starting describe")
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

	entry, err := selectRegistryEntryForDescribe(reg.Entries, args[0], cwd, []string{cfgRoot})
	if err != nil {
		return err
	}

	repo := model.RepoStatus{
		RepoID:   entry.RepoID,
		Path:     entry.Path,
		Type:     entry.Type,
		Tracking: model.Tracking{Status: model.TrackingNone},
	}
	if entry.Status == registry.StatusMissing {
		repo.Error = "path missing"
		repo.ErrorClass = "missing"
	} else {
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil))
		status, err := eng.InspectRepo(cmd.Context(), entry.Path)
		if err != nil {
			repo.Error = err.Error()
			repo.ErrorClass = gitx.ClassifyError(err)
		} else {
			repo = *status
			if repo.RepoID == "" {
				repo.RepoID = entry.RepoID
			}
			if repo.Type == "" {
				repo.Type = entry.Type
			}
		}
	}

	format, _ := cmd.Flags().GetString("format")
	switch strings.ToLower(format) {
	case "json":
		data, err := json.MarshalIndent(repo, "", "  ")
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return err
		}
	case "table":
		if err := writeStatusDetails(cmd, repo, cwd, []string{cfgRoot}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
	return nil
}

func init() {
	describeCmd.Flags().String("registry", "", "override registry file path")
	addFormatFlag(describeCmd, "output format: table or json")

	describeRepoCmd.Flags().String("registry", "", "override registry file path")
	addFormatFlag(describeRepoCmd, "output format: table or json")
	describeCmd.AddCommand(describeRepoCmd)

	rootCmd.AddCommand(describeCmd)
}

func selectRegistryEntryForDescribe(entries []registry.Entry, selector, cwd string, roots []string) (registry.Entry, error) {
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return registry.Entry{}, fmt.Errorf("empty selector")
	}

	for _, entry := range entries {
		if entry.RepoID == sel {
			return entry, nil
		}
	}

	candidates := make([]string, 0, 1+len(roots))
	if abs, err := filepath.Abs(filepath.Join(cwd, sel)); err == nil {
		if candidate, ok := canonicalPathForMatch(abs); ok && pathWithinBase(candidate, cwd) {
			candidates = append(candidates, candidate)
		}
	}
	for _, root := range roots {
		abs, err := filepath.Abs(filepath.Join(root, sel))
		if err != nil {
			continue
		}
		candidate, ok := canonicalPathForMatch(abs)
		if !ok || !pathWithinBase(candidate, root) {
			continue
		}
		candidates = append(candidates, candidate)
	}

	var matches []registry.Entry
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidatePath, ok := canonicalPathForMatch(candidate)
		if !ok {
			continue
		}
		for _, entry := range entries {
			entryPath, ok := canonicalPathForMatch(entry.Path)
			if !ok || !samePathForMatch(entryPath, candidatePath) {
				continue
			}
			key := entry.RepoID + "|" + entry.Path
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			matches = append(matches, entry)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return registry.Entry{}, fmt.Errorf("selector %q is ambiguous (%d matches)", sel, len(matches))
	}
	return registry.Entry{}, fmt.Errorf("repo not found for selector %q", sel)
}

func canonicalPathForMatch(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", false
	}
	return filepath.Clean(abs), true
}

func samePathForMatch(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func pathWithinBase(path, base string) bool {
	cleanPath, ok := canonicalPathForMatch(path)
	if !ok {
		return false
	}
	cleanBase, ok := canonicalPathForMatch(base)
	if !ok {
		return false
	}
	if samePathForMatch(cleanPath, cleanBase) {
		return true
	}
	rel, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
