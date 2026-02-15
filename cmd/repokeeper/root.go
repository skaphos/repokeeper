// Package repokeeper contains the Cobra command tree for the RepoKeeper CLI.
package repokeeper

import (
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
	// colorOutputEnabled is set per command execution based on output format and TTY detection.
	colorOutputEnabled bool
	// exitCode tracks the highest severity observed during a command run.
	exitCode int
	// isTerminalFD is overridable in tests.
	isTerminalFD = term.IsTerminal
	// exitFunc is overridable in tests.
	exitFunc = os.Exit
)

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
}

// Execute runs the root command.
func Execute() {
	exitFunc(ExecuteWithExitCode())
}

// ExecuteWithExitCode runs the root command and returns a shell-friendly exit code.
func ExecuteWithExitCode() int {
	exitCode = 0
	colorOutputEnabled = false
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	return exitCode
}

func raiseExitCode(code int) {
	// Keep the highest severity: 0 success, 1 warning, 2 error, 3 fatal.
	if code > exitCode {
		exitCode = code
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
	colorOutputEnabled = shouldUseColorOutput(cmd, format)
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
