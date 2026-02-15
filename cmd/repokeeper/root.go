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
	// Global flags
	flagVerbose int
	flagQuiet   bool
	flagConfig  string
	flagNoColor bool
	flagYes     bool
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
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		// `NO_COLOR` is a standard opt-out and should behave like --no-color.
		if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			flagNoColor = true
		}
	},
}

func init() {
	rootCmd.PersistentFlags().CountVarP(&flagVerbose, "verbose", "v", "increase output verbosity (repeatable)")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "override config file path")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "accept mutating actions without interactive confirmation")
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
		fmt.Fprintln(os.Stderr, err)
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
	if flagQuiet {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...)
}

func debugf(cmd *cobra.Command, format string, args ...any) {
	if flagQuiet || flagVerbose <= 0 {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...)
}

func setColorOutputMode(cmd *cobra.Command, format string) {
	runtimeStateFor(cmd).colorOutputEnabled = shouldUseColorOutput(cmd, format)
}

func shouldUseColorOutput(cmd *cobra.Command, format string) bool {
	if flagNoColor || !isTabularFormat(format) {
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
