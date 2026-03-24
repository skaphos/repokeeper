// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/skillbundle"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Install or remove bundled RepoKeeper agent skills",
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [opencode|claude|openai|codex|all]",
	Short: "Install the bundled RepoKeeper skill into supported user-scope directories",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roots, err := resolveSkillInstallRoots(args)
		if err != nil {
			return err
		}
		written := make([]string, 0, len(roots))
		for _, root := range roots {
			targetDir := filepath.Join(root, skillbundle.RepoKeeperSkillName)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return err
			}
			targetFile := filepath.Join(targetDir, "SKILL.md")
			if err := os.WriteFile(targetFile, []byte(skillbundle.RepoKeeperSkill()), 0o644); err != nil {
				return err
			}
			written = append(written, targetFile)
		}
		for _, path := range written {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "installed repokeeper skill at %s\n", path); err != nil {
				return err
			}
		}
		return nil
	},
}

var skillUninstallCmd = &cobra.Command{
	Use:   "uninstall [opencode|claude|openai|codex|all]",
	Short: "Remove the bundled RepoKeeper skill from supported user-scope directories",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roots, err := resolveSkillUninstallRoots(args)
		if err != nil {
			return err
		}
		if len(roots) == 0 {
			infof(cmd, "no installed repokeeper skill directories found")
			return nil
		}
		if !assumeYes(cmd) {
			confirmed, err := confirmWithPrompt(cmd, fmt.Sprintf("Remove repokeeper skill from %d location(s)? [y/N]: ", len(roots)))
			if err != nil {
				return err
			}
			if !confirmed {
				infof(cmd, "skill uninstall cancelled")
				return nil
			}
		}
		for _, root := range roots {
			targetDir := filepath.Join(root, skillbundle.RepoKeeperSkillName)
			if err := os.RemoveAll(targetDir); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "removed repokeeper skill from %s\n", targetDir); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillUninstallCmd)
	rootCmd.AddCommand(skillCmd)
}

func resolveSkillInstallRoots(args []string) ([]string, error) {
	if len(args) == 0 {
		roots, err := existingSkillRoots()
		if err != nil {
			return nil, err
		}
		if len(roots) == 0 {
			return nil, fmt.Errorf("no supported user-scope skill directories found; pass a target such as claude, opencode, openai, codex, or all")
		}
		return roots, nil
	}
	return requestedSkillRoots(args[0])
}

func resolveSkillUninstallRoots(args []string) ([]string, error) {
	if len(args) == 0 {
		return installedSkillRoots()
	}
	roots, err := requestedSkillRoots(args[0])
	if err != nil {
		return nil, err
	}
	installed := make([]string, 0, len(roots))
	for _, root := range roots {
		if exists, err := skillInstalledAt(root); err != nil {
			return nil, err
		} else if exists {
			installed = append(installed, root)
		}
	}
	return installed, nil
}

func installedSkillRoots() ([]string, error) {
	known, err := allSupportedSkillRoots()
	if err != nil {
		return nil, err
	}
	installed := make([]string, 0, len(known))
	for _, root := range known {
		if exists, err := skillInstalledAt(root); err != nil {
			return nil, err
		} else if exists {
			installed = append(installed, root)
		}
	}
	return installed, nil
}

func existingSkillRoots() ([]string, error) {
	known, err := allSupportedSkillRoots()
	if err != nil {
		return nil, err
	}
	existing := make([]string, 0, len(known))
	for _, root := range known {
		if ok, err := dirExists(root); err != nil {
			return nil, err
		} else if ok {
			existing = append(existing, root)
		}
	}
	return existing, nil
}

func requestedSkillRoots(target string) ([]string, error) {
	key := strings.ToLower(strings.TrimSpace(target))
	roots, err := supportedSkillRootsByRuntime()
	if err != nil {
		return nil, err
	}
	switch key {
	case "all":
		return dedupeSortedStrings([]string{roots["claude"], roots["agents"], roots["opencode"]}), nil
	case "claude":
		return []string{roots["claude"]}, nil
	case "openai", "codex":
		return []string{roots["agents"]}, nil
	case "opencode":
		return []string{roots["opencode"]}, nil
	default:
		return nil, fmt.Errorf("unsupported skill target %q", target)
	}
}

func allSupportedSkillRoots() ([]string, error) {
	rootsByRuntime, err := supportedSkillRootsByRuntime()
	if err != nil {
		return nil, err
	}
	return dedupeSortedStrings([]string{rootsByRuntime["claude"], rootsByRuntime["agents"], rootsByRuntime["opencode"]}), nil
}

func supportedSkillRootsByRuntime() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir, err := userConfigDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"claude":   filepath.Join(home, ".claude", "skills"),
		"agents":   filepath.Join(home, ".agents", "skills"),
		"opencode": filepath.Join(configDir, "opencode", "skills"),
	}, nil
}

func userConfigDir() (string, error) {
	if path := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); path != "" {
		return path, nil
	}
	return os.UserConfigDir()
}

func skillInstalledAt(root string) (bool, error) {
	return dirExists(filepath.Join(root, skillbundle.RepoKeeperSkillName))
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
