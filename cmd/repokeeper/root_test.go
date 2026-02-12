package repokeeper

import (
	"os"
	"testing"
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
