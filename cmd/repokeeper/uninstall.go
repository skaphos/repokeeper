// SPDX-License-Identifier: MIT
package repokeeper

import (
	"errors"
	"fmt"

	"github.com/skaphos/repokeeper/internal/mcpinstall"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove RepoKeeper MCP server entries from runtime configs",
	Long: "Removes the `repokeeper` MCP entry from each detected (or explicitly selected) runtime config at the chosen scope.\n\n" +
		"Prompts once before deleting unless --yes is set. Silently no-ops when no entries are present to remove.",
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().Bool("claude", false, "target Claude Code")
	uninstallCmd.Flags().Bool("codex", false, "target Codex")
	uninstallCmd.Flags().Bool("opencode", false, "target OpenCode")
	uninstallCmd.Flags().String("scope", "user", "config scope (user|project)")
	rootCmd.AddCommand(uninstallCmd)
}

// uninstallTarget pairs a runtime adapter with the resolved config
// path its entry lives at.
type uninstallTarget struct {
	runtime mcpinstall.Runtime
	path    string
}

func runUninstall(cmd *cobra.Command, _ []string) error {
	claude, _ := cmd.Flags().GetBool("claude")
	codex, _ := cmd.Flags().GetBool("codex")
	opencode, _ := cmd.Flags().GetBool("opencode")
	scopeStr, _ := cmd.Flags().GetString("scope")

	scope, err := parseInstallScope(scopeStr)
	if err != nil {
		return err
	}

	sel := mcpinstall.SelectionFromFlags(claude, codex, opencode)
	explicit := len(sel.Explicit) > 0
	runtimes, err := sel.Resolve()
	if err != nil {
		return err
	}
	if len(runtimes) == 0 {
		return errors.New("no MCP-capable runtime detected; pass --claude, --codex, or --opencode explicitly")
	}

	targets, err := collectUninstallTargets(runtimes, scope, explicit)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	if !assumeYes(cmd) {
		prompt := fmt.Sprintf("Remove repokeeper MCP entry from %d config(s)? [y/N]: ", len(targets))
		confirmed, err := confirmWithPrompt(cmd, prompt)
		if err != nil {
			return err
		}
		if !confirmed {
			infof(cmd, "uninstall cancelled")
			return nil
		}
	}

	for _, t := range targets {
		removed, err := t.runtime.RemoveEntry(t.path)
		if err != nil {
			return err
		}
		if removed {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "removed %s from %s\n", t.runtime.Name(), t.path); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectUninstallTargets resolves config paths for each selected
// runtime, filters to those that actually have our entry to remove,
// and surfaces unsupported-scope errors only when the runtime was
// explicitly requested.
func collectUninstallTargets(runtimes []mcpinstall.Runtime, scope mcpinstall.Scope, explicit bool) ([]uninstallTarget, error) {
	var targets []uninstallTarget
	for _, r := range runtimes {
		path, err := r.ConfigPath(scope)
		if err != nil {
			if errors.Is(err, mcpinstall.ErrScopeUnsupported) {
				if explicit {
					return nil, fmt.Errorf("%s does not support --scope %s", r.Name(), scope)
				}
				continue
			}
			return nil, err
		}
		_, present, err := r.ReadEntry(path)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		targets = append(targets, uninstallTarget{runtime: r, path: path})
	}
	return targets, nil
}
