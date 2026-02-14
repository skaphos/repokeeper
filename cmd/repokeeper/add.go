package repokeeper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <path> <git-repo-url>",
	Short: "Clone and register a repository",
	Args:  cobra.ExactArgs(2),
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

		registryOverride, _ := cmd.Flags().GetString("registry")
		var reg *registry.Registry
		if registryOverride != "" {
			reg, err = registry.Load(registryOverride)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if reg == nil {
				reg = &registry.Registry{}
			}
		} else {
			reg = cfg.Registry
			if reg == nil {
				reg = &registry.Registry{}
			}
		}

		branch, _ := cmd.Flags().GetString("branch")
		mirror, _ := cmd.Flags().GetBool("mirror")
		if mirror && strings.TrimSpace(branch) != "" {
			return fmt.Errorf("--branch and --mirror are mutually exclusive")
		}

		target := args[0]
		rawURL := args[1]
		targetAbs, err := filepath.Abs(filepath.Join(cwd, target))
		if err != nil {
			return err
		}
		targetAbs = filepath.Clean(targetAbs)

		if _, err := os.Stat(targetAbs); err == nil {
			return fmt.Errorf("target path already exists: %s", targetAbs)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return err
		}

		runner := &gitx.GitRunner{}
		cloneArgs := []string{"clone"}
		repoType := "checkout"
		if mirror {
			repoType = "mirror"
			cloneArgs = append(cloneArgs, "--mirror")
		} else if strings.TrimSpace(branch) != "" {
			cloneArgs = append(cloneArgs, "--branch", branch, "--single-branch")
		}
		cloneArgs = append(cloneArgs, rawURL, targetAbs)

		if _, err := runner.Run(cmd.Context(), "", cloneArgs...); err != nil {
			return fmt.Errorf("git %s: %w", strings.Join(cloneArgs, " "), err)
		}

		adapter := vcs.NewGitAdapter(runner)
		remotes, err := adapter.Remotes(cmd.Context(), targetAbs)
		if err != nil {
			return err
		}
		var remoteNames []string
		for _, r := range remotes {
			remoteNames = append(remoteNames, r.Name)
		}
		primary := adapter.PrimaryRemote(remoteNames)
		remoteURL := rawURL
		for _, r := range remotes {
			if r.Name == primary {
				remoteURL = r.URL
				break
			}
		}
		repoID := adapter.NormalizeURL(remoteURL)
		if repoID == "" {
			repoID = "local:" + filepath.ToSlash(targetAbs)
		}

		reg.Upsert(registry.Entry{
			RepoID:    repoID,
			Path:      targetAbs,
			RemoteURL: remoteURL,
			Type:      repoType,
			Branch:    strings.TrimSpace(branch),
			Status:    registry.StatusPresent,
			LastSeen:  time.Now(),
		})
		reg.UpdatedAt = time.Now()

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

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s) at %s\n", repoID, repoType, targetAbs)
		return nil
	},
}

func init() {
	addCmd.Flags().String("registry", "", "override registry file path")
	addCmd.Flags().String("branch", "", "clone and track a specific branch")
	addCmd.Flags().Bool("mirror", false, "create a full mirror clone (no working tree)")
	rootCmd.AddCommand(addCmd)
}
