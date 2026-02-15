package repokeeper

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNOColorEnvSetsFlag(t *testing.T) {
	prev, _ := rootCmd.PersistentFlags().GetBool("no-color")
	_ = rootCmd.PersistentFlags().Set("no-color", "false")
	defer func() { _ = rootCmd.PersistentFlags().Set("no-color", boolToFlag(prev)) }()

	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	if rootCmd.PersistentPreRun == nil {
		t.Fatal("expected persistent pre-run handler")
	}
	rootCmd.PersistentPreRun(rootCmd, nil)
	got, _ := rootCmd.PersistentFlags().GetBool("no-color")
	if !got {
		t.Fatal("expected NO_COLOR to enable no-color mode")
	}
}

func TestRaiseExitCodeMonotonic(t *testing.T) {
	cmd := &cobra.Command{}
	state := runtimeStateFor(cmd)
	state.exitCode = 0
	raiseExitCode(cmd, 1)
	raiseExitCode(cmd, 0)
	raiseExitCode(cmd, 2)
	raiseExitCode(cmd, 1)
	if got := runtimeStateFor(cmd).exitCode; got != 2 {
		t.Fatalf("expected highest exit code to win, got %d", got)
	}
}

func TestShouldUseColorOutput(t *testing.T) {
	prevNoColor, _ := rootCmd.PersistentFlags().GetBool("no-color")
	prevTTY := isTerminalFD
	defer func() {
		_ = rootCmd.PersistentFlags().Set("no-color", boolToFlag(prevNoColor))
		isTerminalFD = prevTTY
	}()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	_ = rootCmd.PersistentFlags().Set("no-color", "false")
	isTerminalFD = func(_ int) bool { return true }
	if shouldUseColorOutput(cmd, "table") {
		t.Fatal("expected non-file output stream to disable color")
	}

	tmp, err := os.CreateTemp("", "repokeeper-color-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	cmd.SetOut(tmp)
	if !shouldUseColorOutput(cmd, "table") {
		t.Fatal("expected tty table output to enable color")
	}
	if shouldUseColorOutput(cmd, "json") {
		t.Fatal("expected non-table formats to disable color")
	}

	_ = rootCmd.PersistentFlags().Set("no-color", "true")
	if shouldUseColorOutput(cmd, "table") {
		t.Fatal("expected --no-color to disable color output")
	}
}

func TestSetColorOutputMode(t *testing.T) {
	prevNoColor, _ := rootCmd.PersistentFlags().GetBool("no-color")
	prevTTY := isTerminalFD
	cmd := &cobra.Command{}
	state := runtimeStateFor(cmd)
	prevColor := state.colorOutputEnabled
	defer func() {
		_ = rootCmd.PersistentFlags().Set("no-color", boolToFlag(prevNoColor))
		isTerminalFD = prevTTY
		runtimeStateFor(cmd).colorOutputEnabled = prevColor
	}()

	tmp, err := os.CreateTemp("", "repokeeper-color-mode-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	cmd.SetOut(tmp)

	_ = rootCmd.PersistentFlags().Set("no-color", "false")
	isTerminalFD = func(_ int) bool { return true }
	setColorOutputMode(cmd, "table")
	if !runtimeStateFor(cmd).colorOutputEnabled {
		t.Fatal("expected color output mode to be enabled")
	}

	setColorOutputMode(cmd, "json")
	if runtimeStateFor(cmd).colorOutputEnabled {
		t.Fatal("expected color output mode to be disabled for json")
	}
}

func TestExecuteWithExitCode(t *testing.T) {
	defer rootCmd.SetArgs(nil)

	rootCmd.SetArgs([]string{"version"})
	if code := ExecuteWithExitCode(); code != 0 {
		t.Fatalf("expected success exit code, got %d", code)
	}

	rootCmd.SetArgs([]string{"this-command-does-not-exist"})
	if code := ExecuteWithExitCode(); code != 3 {
		t.Fatalf("expected fatal exit code for command error, got %d", code)
	}
}

func TestExecuteUsesExitFunc(t *testing.T) {
	prevExit := exitFunc
	defer func() { exitFunc = prevExit }()
	defer rootCmd.SetArgs(nil)

	gotCode := -1
	exitFunc = func(code int) { gotCode = code }
	rootCmd.SetArgs([]string{"version"})

	Execute()
	if gotCode != 0 {
		t.Fatalf("expected Execute to pass success code to exit func, got %d", gotCode)
	}
}

func TestLogHelpersRespectQuietAndVerbose(t *testing.T) {
	prevQuiet, _ := rootCmd.PersistentFlags().GetBool("quiet")
	prevVerbose, _ := rootCmd.PersistentFlags().GetCount("verbose")
	defer func() {
		_ = rootCmd.PersistentFlags().Set("quiet", boolToFlag(prevQuiet))
		_ = rootCmd.PersistentFlags().Set("verbose", countToFlag(prevVerbose))
	}()

	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	_ = rootCmd.PersistentFlags().Set("quiet", "true")
	_ = rootCmd.PersistentFlags().Set("verbose", "1")
	infof(cmd, "hidden info")
	debugf(cmd, "hidden debug")
	if errOut.Len() != 0 {
		t.Fatalf("expected no output in quiet mode, got %q", errOut.String())
	}

	_ = rootCmd.PersistentFlags().Set("quiet", "false")
	_ = rootCmd.PersistentFlags().Set("verbose", "0")
	debugf(cmd, "still hidden debug")
	if errOut.Len() != 0 {
		t.Fatalf("expected debug to stay hidden without verbosity, got %q", errOut.String())
	}
}

func boolToFlag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func countToFlag(v int) string {
	return fmt.Sprintf("%d", v)
}
