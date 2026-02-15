// Package repokeeper contains the Cobra command tree for the RepoKeeper CLI.
package repokeeper

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// isTerminalFD is overridable in tests.
	isTerminalFD = term.IsTerminal
	// exitFunc is overridable in tests.
	exitFunc = os.Exit
)

type runtimeStateKey struct{}

type runtimeState struct {
	colorOutputEnabled bool
	exitCode           int
}

var rootCmd = &cobra.Command{
	Use:   "repokeeper",
	Short: "Cross-platform multi-repo hygiene tool",
	Long:  "RepoKeeper inventories git repos, reports drift and broken tracking, and performs safe sync actions (fetch/prune) without touching working trees or submodules.",
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		// `NO_COLOR` is a standard opt-out and should behave like --no-color.
		if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			_ = cmd.Flags().Set("no-color", "true")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().CountP("verbose", "v", "increase output verbosity (repeatable)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().String("config", "", "override config file path")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Bool("yes", false, "accept mutating actions without interactive confirmation")
}

// Execute runs the root command.
func Execute() {
	exitFunc(ExecuteWithExitCode())
}

// ExecuteWithExitCode runs the root command and returns a shell-friendly exit code.
func ExecuteWithExitCode() int {
	state := &runtimeState{}
	rootCmd.SetContext(context.WithValue(context.Background(), runtimeStateKey{}, state))
	if err := rootCmd.Execute(); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			return 3
		}
		return 3
	}
	return state.exitCode
}

func raiseExitCode(cmd *cobra.Command, code int) {
	// Keep the highest severity: 0 success, 1 warning, 2 error, 3 fatal.
	state := runtimeStateFor(cmd)
	if code > state.exitCode {
		state.exitCode = code
	}
}

func infof(cmd *cobra.Command, format string, args ...any) {
	if isQuiet(cmd) {
		return
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...); err != nil {
		if _, fallbackErr := fmt.Fprintf(os.Stderr, "repokeeper: output write failure (info): %v\n", err); fallbackErr != nil {
			return
		}
	}
}

func debugf(cmd *cobra.Command, format string, args ...any) {
	if isQuiet(cmd) || verbosity(cmd) <= 0 {
		return
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...); err != nil {
		if _, fallbackErr := fmt.Fprintf(os.Stderr, "repokeeper: output write failure (debug): %v\n", err); fallbackErr != nil {
			return
		}
	}
}

func setColorOutputMode(cmd *cobra.Command, format string) {
	runtimeStateFor(cmd).colorOutputEnabled = shouldUseColorOutput(cmd, format)
}

func shouldUseColorOutput(cmd *cobra.Command, format string) bool {
	if isNoColor(cmd) || !isTabularFormat(format) {
		return false
	}
	file, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return false
	}
	return isTerminalFD(int(file.Fd()))
}

func isTabularFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "table", "wide":
		return true
	default:
		return false
	}
}

func runtimeStateFor(cmd *cobra.Command) *runtimeState {
	root := cmd
	if root != nil {
		root = cmd.Root()
	}
	if root == nil {
		root = rootCmd
	}
	ctx := root.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if state, ok := ctx.Value(runtimeStateKey{}).(*runtimeState); ok && state != nil {
		return state
	}
	state := &runtimeState{}
	root.SetContext(context.WithValue(ctx, runtimeStateKey{}, state))
	return state
}

func isQuiet(cmd *cobra.Command) bool {
	return getBoolFlag(cmd, "quiet")
}

func verbosity(cmd *cobra.Command) int {
	return getCountFlag(cmd, "verbose")
}

func configOverride(cmd *cobra.Command) string {
	return strings.TrimSpace(getStringFlag(cmd, "config"))
}

func isNoColor(cmd *cobra.Command) bool {
	return getBoolFlag(cmd, "no-color")
}

func assumeYes(cmd *cobra.Command) bool {
	return getBoolFlag(cmd, "yes")
}

func getBoolFlag(cmd *cobra.Command, name string) bool {
	if cmd != nil {
		if cmd.Flags().Lookup(name) != nil {
			v, _ := cmd.Flags().GetBool(name)
			return v
		}
		if root := cmd.Root(); root != nil && root.PersistentFlags().Lookup(name) != nil {
			v, _ := root.PersistentFlags().GetBool(name)
			return v
		}
	}
	v, _ := rootCmd.PersistentFlags().GetBool(name)
	return v
}

func getCountFlag(cmd *cobra.Command, name string) int {
	if cmd != nil {
		if cmd.Flags().Lookup(name) != nil {
			v, _ := cmd.Flags().GetCount(name)
			return v
		}
		if root := cmd.Root(); root != nil && root.PersistentFlags().Lookup(name) != nil {
			v, _ := root.PersistentFlags().GetCount(name)
			return v
		}
	}
	v, _ := rootCmd.PersistentFlags().GetCount(name)
	return v
}

func getStringFlag(cmd *cobra.Command, name string) string {
	if cmd != nil {
		if cmd.Flags().Lookup(name) != nil {
			v, _ := cmd.Flags().GetString(name)
			return v
		}
		if root := cmd.Root(); root != nil && root.PersistentFlags().Lookup(name) != nil {
			v, _ := root.PersistentFlags().GetString(name)
			return v
		}
	}
	v, _ := rootCmd.PersistentFlags().GetString(name)
	return v
}
