// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/skaphos/repokeeper/internal/mcpinstall"
	"github.com/spf13/cobra"
)

var installListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show whether repokeeper is registered with each runtime",
	Long: "Iterates every runtime adapter (not just detected ones) and reports its registration state at the chosen scope.\n\n" +
		"A runtime is 'registered' when its config already points at the current executable, 'registered (stale)' when the " +
		"config points at a different binary (e.g. after `brew upgrade` with a Cellar path), and 'not registered' when " +
		"no entry is present. Codex reports 'unsupported' under --scope project.",
	RunE: runInstallList,
}

func init() {
	installListCmd.Flags().String("scope", "user", "config scope (user|project)")
	installListCmd.Flags().Bool("json", false, "emit JSON instead of a table")
	installCmd.AddCommand(installListCmd)
}

// listRow is the per-runtime record surfaced by `install list`. The
// JSON tags double as the stable `--json` output schema.
type listRow struct {
	Name    string `json:"name"`
	Scope   string `json:"scope"`
	Path    string `json:"path"`
	State   string `json:"state"`
	Command string `json:"command,omitempty"`
}

func runInstallList(cmd *cobra.Command, _ []string) error {
	scopeStr, _ := cmd.Flags().GetString("scope")
	asJSON, _ := cmd.Flags().GetBool("json")

	scope, err := parseInstallScope(scopeStr)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	rows := make([]listRow, 0, len(mcpinstall.All()))
	for _, r := range mcpinstall.All() {
		row := listRow{Name: r.Name(), Scope: scope.String()}
		path, err := r.ConfigPath(scope)
		if err != nil {
			if errors.Is(err, mcpinstall.ErrScopeUnsupported) {
				row.State = "unsupported"
				rows = append(rows, row)
				continue
			}
			return err
		}
		row.Path = path
		entry, present, err := r.ReadEntry(path)
		if err != nil {
			return err
		}
		switch {
		case !present:
			row.State = "not registered"
		case entry.Command != exe:
			row.State = "registered (stale)"
			row.Command = entry.Command
		default:
			row.State = "registered"
			row.Command = entry.Command
		}
		rows = append(rows, row)
	}

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"scope":    scope.String(),
			"runtimes": rows,
		})
	}
	return writeInstallListTable(cmd.OutOrStdout(), rows)
}

func writeInstallListTable(w io.Writer, rows []listRow) error {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		cmdStr := r.Command
		if cmdStr == "" {
			cmdStr = "-"
		}
		path := r.Path
		if path == "" {
			path = "-"
		}
		out = append(out, []string{r.Name, r.Scope, path, r.State, cmdStr})
	}
	return cliio.WriteTable(w, false, false, []string{"NAME", "SCOPE", "PATH", "STATE", "COMMAND"}, out)
}
