// SPDX-License-Identifier: MIT
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
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/repometa"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <selector>",
	Short: "Show detailed status for one repository",
	Long:  repoSelectorUsage,
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribeRepo,
}

var describeRepoCmd = &cobra.Command{
	Use:   "repo <selector>",
	Short: "Show detailed status for one repository",
	Long:  repoSelectorUsage,
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
	cfgRoot := config.EffectiveRoot(cfgPath)
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

	entry, err := selectRegistryEntryForDescribe(reg.Entries, args[0], cwd, []string{cfgRoot})
	if err != nil {
		return err
	}

	repo := model.RepoStatus{
		RepoID:      entry.RepoID,
		CheckoutID:  entry.CheckoutID,
		Path:        entry.Path,
		Type:        entry.Type,
		Labels:      cloneMetadataMap(entry.Labels),
		Annotations: cloneMetadataMap(entry.Annotations),
		Tracking:    model.Tracking{Status: model.TrackingNone},
	}
	registry.SeedRepoMetadataStatus(entry, &repo)
	if entry.Status == registry.StatusMissing {
		repo.Error = "path missing"
		repo.ErrorClass = "missing"
	} else {
		classifier := vcs.NewGitErrorClassifier()
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil), classifier, vcs.NewGitURLNormalizer(), nil)
		status, err := eng.InspectRepo(cmd.Context(), entry.Path)
		if err != nil {
			repo.Error = err.Error()
			repo.ErrorClass = classifier.ClassifyError(err)
			repometa.Apply(&repo)
		} else {
			repo = *status
			if repo.RepoID == "" {
				repo.RepoID = entry.RepoID
			}
			if repo.CheckoutID == "" {
				repo.CheckoutID = entry.CheckoutID
			}
			if repo.Type == "" {
				repo.Type = entry.Type
			}
			repo.Labels = cloneMetadataMap(entry.Labels)
			repo.Annotations = cloneMetadataMap(entry.Annotations)
		}
	}

	if err := persistDescribeMetadataSnapshot(cfg, cfgPath, registryOverride, reg, entry, repo); err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	mode, err := parseOutputMode(format)
	if err != nil {
		return err
	}
	switch mode.kind {
	case outputKindJSON:
		data, err := json.MarshalIndent(repo, "", "  ")
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return err
		}
	case outputKindCustomColumns:
		if err := writeCustomColumnsOutput(cmd, repo, mode.expr, false); err != nil {
			return err
		}
	case outputKindTable:
		if err := writeStatusDetails(cmd, repo, cwd, []string{cfgRoot}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
	return nil
}

func persistDescribeMetadataSnapshot(
	cfg *config.Config,
	cfgPath string,
	registryOverride string,
	reg *registry.Registry,
	selected registry.Entry,
	repo model.RepoStatus,
) error {
	if reg == nil {
		return nil
	}
	idx := reg.FindEntryIndex(selected.RepoID, selected.Path)
	if idx < 0 {
		return nil
	}
	updated := reg.Entries[idx]
	registry.StoreRepoMetadataStatus(&updated, repo)
	reg.Entries[idx] = updated
	if registryOverride != "" {
		return registry.Save(reg, registryOverride)
	}
	if cfg == nil {
		return nil
	}
	cfg.Registry = reg
	return config.Save(cfg, cfgPath)
}

func init() {
	describeCmd.Flags().String("registry", "", "override registry file path")
	addFormatFlag(describeCmd, "output format: table or json")

	describeRepoCmd.Flags().String("registry", "", "override registry file path")
	addFormatFlag(describeRepoCmd, "output format: table or json")
	describeCmd.AddCommand(describeRepoCmd)

	rootCmd.AddCommand(describeCmd)
}

// selectRegistryEntryForDescribe is shared by all CLI commands that target one
// checkout. Identity collisions must fail before a mutation can select a repo.
func selectRegistryEntryForDescribe(entries []registry.Entry, selector, cwd string, roots []string) (registry.Entry, error) {
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return registry.Entry{}, fmt.Errorf("empty selector")
	}
	if isAbsoluteLikePath(sel, filepath.Clean(normalizePathLikeInput(sel))) {
		return uniqueCheckoutMatch(selectRegistryPaths(entries, []string{sel}), sel)
	}

	repoID, checkoutID, hasCheckoutID := splitRepoAndCheckoutSelector(sel)
	if hasCheckoutID {
		var matches []registry.Entry
		for _, entry := range entries {
			if entry.RepoID == repoID && describeCheckoutID(entry) == checkoutID {
				matches = append(matches, entry)
			}
		}
		if len(matches) > 0 {
			return uniqueCheckoutMatch(matches, sel)
		}
		// A relative directory can contain @ too; try path matching below when
		// this is not a known qualified identity.
	}

	var checkoutMatches, repoMatches []registry.Entry
	for _, entry := range entries {
		if describeCheckoutID(entry) == sel {
			checkoutMatches = append(checkoutMatches, entry)
		}
		if entry.RepoID == sel {
			repoMatches = append(repoMatches, entry)
		}
	}
	if len(checkoutMatches) > 0 {
		return uniqueCheckoutMatch(checkoutMatches, sel)
	}
	if len(repoMatches) > 0 {
		return uniqueCheckoutMatch(repoMatches, sel)
	}

	candidates := make([]string, 0, 1+len(roots))
	for _, base := range append([]string{cwd}, roots...) {
		candidate, ok := canonicalPathForMatch(filepath.Join(base, sel))
		if ok && pathWithinBase(candidate, base) {
			candidates = append(candidates, candidate)
		}
	}
	return uniqueCheckoutMatch(selectRegistryPaths(entries, candidates), sel)
}

func selectRegistryPaths(entries []registry.Entry, candidates []string) []registry.Entry {
	var matches []registry.Entry
	for _, entry := range entries {
		entryPath, ok := canonicalPathForMatch(entry.Path)
		if !ok {
			continue
		}
		for _, candidate := range candidates {
			candidatePath, ok := canonicalPathForMatch(candidate)
			if ok && samePathForMatch(entryPath, candidatePath) {
				matches = append(matches, entry)
				break
			}
		}
	}
	return matches
}

func uniqueCheckoutMatch(matches []registry.Entry, selector string) (registry.Entry, error) {
	switch len(matches) {
	case 0:
		return registry.Entry{}, fmt.Errorf("repo not found for selector %q", selector)
	case 1:
		return matches[0], nil
	default:
		candidates := make([]string, 0, len(matches))
		for _, entry := range matches {
			candidates = append(candidates, fmt.Sprintf("%q (absolute path %q)", entry.RepoID+"@"+describeCheckoutID(entry), entry.Path))
		}
		return registry.Entry{}, fmt.Errorf("selector %q is ambiguous (%d matches); use a qualified checkout ID or absolute path: %s", selector, len(matches), strings.Join(candidates, "; "))
	}
}

func splitRepoAndCheckoutSelector(selector string) (string, string, bool) {
	parts := strings.SplitN(selector, "@", 2)
	if len(parts) != 2 {
		return selector, "", false
	}
	repoID := strings.TrimSpace(parts[0])
	checkoutID := strings.TrimSpace(parts[1])
	if repoID == "" || checkoutID == "" {
		return selector, "", false
	}
	return repoID, checkoutID, true
}

func describeCheckoutID(entry registry.Entry) string {
	checkoutID := strings.TrimSpace(entry.CheckoutID)
	if checkoutID != "" {
		return checkoutID
	}
	trimmed := strings.TrimSpace(entry.Path)
	if trimmed == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(trimmed))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
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
	// Missing checkouts still need to be addressable, so retain lexical matching
	// when the filesystem cannot resolve the path.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), true
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
