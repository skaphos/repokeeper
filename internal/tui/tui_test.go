package tui_test

import (
	"testing"

	"github.com/mfacenet/repokeeper/internal/tui"
)

func TestRun(t *testing.T) {
	if err := tui.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
