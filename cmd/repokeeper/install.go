// SPDX-License-Identifier: MIT
package repokeeper

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/skaphos/repokeeper/internal/mcpinstall"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Register RepoKeeper as an MCP server in your agent runtime",
	Long: "Writes a `repokeeper mcp` entry into the config file of each detected (or explicitly selected) runtime.\n\n" +
		"Auto-detects Claude Code, Codex, and OpenCode by default. Use --claude/--codex/--opencode to restrict the targets. " +
		"Use --scope project to write the current directory's project-scoped config instead of the user-scoped one. " +
		"Use --command to override the binary path written to config (defaults to the current executable).",
	RunE: runInstall,
}

func init() {
	installCmd.Flags().Bool("claude", false, "target Claude Code")
	installCmd.Flags().Bool("codex", false, "target Codex")
	installCmd.Flags().Bool("opencode", false, "target OpenCode")
	installCmd.Flags().String("scope", "user", "config scope (user|project)")
	installCmd.Flags().String("command", "", "binary path to write into config (default: os.Executable())")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, _ []string) error {
	claude, _ := cmd.Flags().GetBool("claude")
	codex, _ := cmd.Flags().GetBool("codex")
	opencode, _ := cmd.Flags().GetBool("opencode")
	scopeStr, _ := cmd.Flags().GetString("scope")
	override, _ := cmd.Flags().GetString("command")

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

	desired, err := desiredInstallEntry(override)
	if err != nil {
		return err
	}

	for _, r := range runtimes {
		path, err := r.ConfigPath(scope)
		if err != nil {
			if errors.Is(err, mcpinstall.ErrScopeUnsupported) {
				if explicit {
					return fmt.Errorf("%s does not support --scope %s", r.Name(), scope)
				}
				continue
			}
			return err
		}
		existing, present, err := r.ReadEntry(path)
		if err != nil {
			return err
		}
		switch {
		case !present:
			if err := r.WriteEntry(path, desired); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "registered %s at %s\n", r.Name(), path); err != nil {
				return err
			}
		case reflect.DeepEqual(existing, desired):
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "unchanged %s at %s\n", r.Name(), path); err != nil {
				return err
			}
		default:
			if err := r.WriteEntry(path, desired); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "updated %s at %s\n", r.Name(), path); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseInstallScope(s string) (mcpinstall.Scope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "user":
		return mcpinstall.ScopeUser, nil
	case "project":
		return mcpinstall.ScopeProject, nil
	default:
		return 0, fmt.Errorf("invalid --scope %q (want user or project)", s)
	}
}

func desiredInstallEntry(override string) (mcpinstall.Entry, error) {
	bin := strings.TrimSpace(override)
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return mcpinstall.Entry{}, fmt.Errorf("resolve binary path: %w", err)
		}
		bin = exe
	}
	return mcpinstall.Entry{Command: bin, Args: []string{"mcp"}}, nil
}
