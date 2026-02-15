package repokeeper

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNOColorEnvSetsFlag(t *testing.T) {
	prev := flagNoColor
	flagNoColor = false
	defer func() { flagNoColor = prev }()

	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	if rootCmd.PersistentPreRun == nil {
		t.Fatal("expected persistent pre-run handler")
	}
	rootCmd.PersistentPreRun(rootCmd, nil)
	if !flagNoColor {
		t.Fatal("expected NO_COLOR to enable no-color mode")
	}
}

func TestRaiseExitCodeMonotonic(t *testing.T) {
	exitCode = 0
	raiseExitCode(1)
	raiseExitCode(0)
	raiseExitCode(2)
	raiseExitCode(1)
	if exitCode != 2 {
		t.Fatalf("expected highest exit code to win, got %d", exitCode)
	}
}

func TestShouldUseColorOutput(t *testing.T) {
	prevNoColor := flagNoColor
	prevTTY := isTerminalFD
	defer func() {
		flagNoColor = prevNoColor
		isTerminalFD = prevTTY
	}()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	flagNoColor = false
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

	flagNoColor = true
	if shouldUseColorOutput(cmd, "table") {
		t.Fatal("expected --no-color to disable color output")
	}
}
