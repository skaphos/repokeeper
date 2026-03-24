// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/repometa"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var indexCmd = &cobra.Command{
	Use:   "index <repo-id-or-path>",
	Short: "Interactively propose repo-local metadata for a tracked repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		if cfg.Registry == nil {
			return fmt.Errorf("registry not found in %q (run repokeeper scan first)", cfgPath)
		}

		entry, err := selectRegistryEntryForDescribe(cfg.Registry.Entries, args[0], cwd, []string{cfgRoot})
		if err != nil {
			return err
		}
		writeFile, _ := cmd.Flags().GetBool("write")
		force, _ := cmd.Flags().GetBool("force")
		yes := assumeYes(cmd)

		existingPath, existing, existingErr := repometa.Load(entry.Path)
		switch {
		case existingErr == nil:
		case errors.Is(existingErr, repometa.ErrNotFound):
			existingPath = filepath.Join(entry.Path, repometa.PreferredFilename)
		case force:
			existingPath = fallbackMetadataPath(entry.Path, existingPath)
		default:
			return fmt.Errorf("load existing repo metadata: %w (use --force to replace)", existingErr)
		}
		if writeFile && existingErr == nil && !force {
			return fmt.Errorf("repo metadata already exists at %s (use --force to overwrite)", existingPath)
		}

		proposal, err := buildIndexProposal(entry, existing, yes, cmd.InOrStdin(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		proposal.APIVersion = repometa.APIVersion
		proposal.Kind = repometa.Kind
		preview, err := yaml.Marshal(proposal)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "# Repo metadata preview\n# Target: %s\n%s", existingPath, string(preview)); err != nil {
			return err
		}
		if !writeFile {
			infof(cmd, "preview only; rerun with --write to save repo metadata")
			return nil
		}
		if !yes {
			confirmed, err := confirmWithPrompt(cmd, fmt.Sprintf("Write repo metadata to %s? [y/N]: ", existingPath))
			if err != nil {
				return err
			}
			if !confirmed {
				infof(cmd, "index cancelled")
				return nil
			}
		}
		writtenPath, err := repometa.Save(entry.Path, proposal, force)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote repo metadata for %s to %s\n", entry.RepoID, writtenPath); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	indexCmd.Flags().Bool("write", false, "write repo metadata to the repository root after preview")
	indexCmd.Flags().Bool("force", false, "overwrite or replace an existing repo metadata file")
	rootCmd.AddCommand(indexCmd)
}

func buildIndexProposal(entry registry.Entry, existing *model.RepoMetadata, yes bool, in io.Reader, out io.Writer) (*model.RepoMetadata, error) {
	defaults := guessRepoMetadataDefaults(entry, existing)
	if yes {
		return defaults, nil
	}
	q := newIndexQuestioner(in, out)
	name, err := q.askString("Repository name", defaults.Name)
	if err != nil {
		return nil, err
	}
	repoID, err := q.askString("Repo ID assertion", defaults.RepoID)
	if err != nil {
		return nil, err
	}
	labels, err := q.askAssignments("Labels (key=value,key=value)", defaults.Labels)
	if err != nil {
		return nil, err
	}
	entrypoints, err := q.askAssignments("Entrypoints (key=path,key=path)", defaults.Entrypoints)
	if err != nil {
		return nil, err
	}
	authoritative, err := q.askList("Authoritative paths (comma-separated)", defaults.Paths.Authoritative)
	if err != nil {
		return nil, err
	}
	lowValue, err := q.askList("Low-value paths (comma-separated)", defaults.Paths.LowValue)
	if err != nil {
		return nil, err
	}
	provides, err := q.askList("Provides (comma-separated)", defaults.Provides)
	if err != nil {
		return nil, err
	}
	related, err := q.askRelatedRepos("Related repos (repo_id[:relationship],...)", defaults.RelatedRepos)
	if err != nil {
		return nil, err
	}
	proposal := &model.RepoMetadata{
		Name:        strings.TrimSpace(name),
		RepoID:      strings.TrimSpace(repoID),
		Labels:      normalizeMetadataMap(labels),
		Entrypoints: normalizeMetadataMap(entrypoints),
		Paths: model.RepoMetadataPaths{
			Authoritative: authoritative,
			LowValue:      lowValue,
		},
		Provides:     provides,
		RelatedRepos: related,
	}
	if proposal.RepoID != "" && proposal.RepoID != entry.RepoID {
		return nil, fmt.Errorf("repo metadata repo_id %q must match tracked repo_id %q", proposal.RepoID, entry.RepoID)
	}
	return proposal, nil
}

type indexQuestioner struct {
	reader *bufio.Reader
	out    io.Writer
}

func newIndexQuestioner(in io.Reader, out io.Writer) *indexQuestioner {
	return &indexQuestioner{reader: bufio.NewReader(in), out: out}
}

func (q *indexQuestioner) askString(label, defaultValue string) (string, error) {
	response, err := q.readLine(label, defaultValue)
	if err != nil {
		return "", err
	}
	if response == "" {
		return defaultValue, nil
	}
	return response, nil
}

func (q *indexQuestioner) askAssignments(label string, defaults map[string]string) (map[string]string, error) {
	response, err := q.readLine(label, formatAssignmentDefaults(defaults))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response) == "" {
		return cloneMetadataMap(defaults), nil
	}
	assignments, err := parseIndexAssignments(response)
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

func (q *indexQuestioner) askList(label string, defaults []string) ([]string, error) {
	response, err := q.readLine(label, strings.Join(defaults, ","))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response) == "" {
		return append([]string(nil), defaults...), nil
	}
	return parseIndexList(response), nil
}

func (q *indexQuestioner) askRelatedRepos(label string, defaults []model.RepoMetadataRelatedRepo) ([]model.RepoMetadataRelatedRepo, error) {
	response, err := q.readLine(label, formatRelatedRepoDefaults(defaults))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response) == "" {
		return append([]model.RepoMetadataRelatedRepo(nil), defaults...), nil
	}
	return parseRelatedRepos(response)
}

func (q *indexQuestioner) readLine(label, defaultValue string) (string, error) {
	prompt := label + ": "
	if strings.TrimSpace(defaultValue) != "" {
		prompt = fmt.Sprintf("%s [%s]: ", label, defaultValue)
	}
	if _, err := fmt.Fprint(q.out, prompt); err != nil {
		return "", err
	}
	line, err := q.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func guessRepoMetadataDefaults(entry registry.Entry, existing *model.RepoMetadata) *model.RepoMetadata {
	if existing != nil {
		copy := *existing
		copy.Labels = cloneMetadataMap(existing.Labels)
		copy.Entrypoints = cloneMetadataMap(existing.Entrypoints)
		copy.Paths.Authoritative = append([]string(nil), existing.Paths.Authoritative...)
		copy.Paths.LowValue = append([]string(nil), existing.Paths.LowValue...)
		copy.Provides = append([]string(nil), existing.Provides...)
		copy.RelatedRepos = append([]model.RepoMetadataRelatedRepo(nil), existing.RelatedRepos...)
		return &copy
	}
	metadata := &model.RepoMetadata{RepoID: entry.RepoID}
	if base := filepath.Base(entry.Path); base != "" && base != "." {
		metadata.Name = humanizeRepoName(base)
	}
	metadata.Entrypoints = make(map[string]string)
	if readme := detectReadmeEntrypoint(entry.Path); readme != "" {
		metadata.Entrypoints["readme"] = readme
	}
	metadata.Paths.Authoritative = detectAuthoritativePaths(entry.Path)
	metadata.Paths.LowValue = detectLowValuePaths(entry.Path)
	return metadata
}

func fallbackMetadataPath(repoRoot, current string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return filepath.Join(repoRoot, repometa.PreferredFilename)
}

func detectReadmeEntrypoint(repoRoot string) string {
	for _, candidate := range []string{"README.md", "README.rst", "README.txt", "README"} {
		if _, err := os.Stat(filepath.Join(repoRoot, candidate)); err == nil {
			return candidate
		}
	}
	return ""
}

func detectAuthoritativePaths(repoRoot string) []string {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil
	}
	preferred := []string{"docs", "src", "cmd", "internal", "pkg", "app", "lib", "templates", "scripts", "examples"}
	available := make(map[string]bool, len(entries))
	for _, entry := range entries {
		available[entry.Name()] = true
	}
	out := make([]string, 0, len(preferred))
	for _, name := range preferred {
		if available[name] {
			out = append(out, name+"/")
		}
	}
	return out
}

func detectLowValuePaths(repoRoot string) []string {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil
	}
	preferred := []string{"generated", "dist", "build", "archive", ".github", "vendor", "node_modules"}
	available := make(map[string]bool, len(entries))
	for _, entry := range entries {
		available[entry.Name()] = true
	}
	out := make([]string, 0, len(preferred))
	for _, name := range preferred {
		if available[name] {
			out = append(out, name+"/")
		}
	}
	return out
}

func humanizeRepoName(name string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ")
	parts := strings.Fields(replacer.Replace(name))
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func formatAssignmentDefaults(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func parseIndexAssignments(raw string) (map[string]string, error) {
	parts := splitCSV(raw)
	if len(parts) == 0 {
		return nil, nil
	}
	return parseMetadataAssignments(parts, "repo metadata")
}

func parseIndexList(raw string) []string {
	return splitCSV(raw)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func formatRelatedRepoDefaults(values []model.RepoMetadataRelatedRepo) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Relationship) == "" {
			parts = append(parts, value.RepoID)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s", value.RepoID, value.Relationship))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func parseRelatedRepos(raw string) ([]model.RepoMetadataRelatedRepo, error) {
	parts := splitCSV(raw)
	out := make([]model.RepoMetadataRelatedRepo, 0, len(parts))
	for _, part := range parts {
		repoID, relationship, _ := strings.Cut(part, ":")
		repoID = strings.TrimSpace(repoID)
		relationship = strings.TrimSpace(relationship)
		if repoID == "" {
			return nil, fmt.Errorf("related repos entries require repo_id")
		}
		out = append(out, model.RepoMetadataRelatedRepo{RepoID: repoID, Relationship: relationship})
	}
	return out, nil
}
