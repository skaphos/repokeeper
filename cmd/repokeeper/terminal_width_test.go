package repokeeper

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/spf13/cobra"
)

func TestAdaptiveCellLimitForWidth(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		normal int
		narrow int
		tiny   int
		want   int
	}{
		{name: "normal width", width: 120, normal: 0, narrow: 48, tiny: 32, want: 0},
		{name: "narrow width", width: 95, normal: 0, narrow: 48, tiny: 32, want: 48},
		{name: "tiny width", width: 70, normal: 0, narrow: 48, tiny: 32, want: 32},
		{name: "missing narrow limit", width: 95, normal: 0, narrow: 0, tiny: 24, want: 0},
		{name: "missing tiny limit", width: 70, normal: 0, narrow: 48, tiny: 0, want: 48},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := adaptiveCellLimitForWidth(tc.width, tc.normal, tc.narrow, tc.tiny)
			if got != tc.want {
				t.Fatalf("adaptiveCellLimitForWidth() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWriteStatusTableTruncatesOnNarrowTTY(t *testing.T) {
	prevIsTerminalFD := isTerminalFD
	prevGetTerminalSize := getTerminalSize
	defer func() {
		isTerminalFD = prevIsTerminalFD
		getTerminalSize = prevGetTerminalSize
	}()
	isTerminalFD = func(int) bool { return true }
	getTerminalSize = func(int) (int, int, error) { return 70, 24, nil }

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe setup failed: %v", err)
	}
	defer reader.Close()

	cmd := &cobra.Command{}
	cmd.SetOut(writer)

	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				RepoID: "github.com/org/repo",
				Path:   "/tmp/workspace/very/long/path/that/should/be/truncated/in/narrow/terminals/repo",
				Head:   model.Head{Branch: "feature/really-long-branch-name-for-narrow-terminals"},
				Tracking: model.Tracking{
					Status: model.TrackingEqual,
				},
				Worktree: &model.Worktree{Dirty: false},
			},
		},
	}
	if err := writeStatusTable(cmd, report, "/tmp", nil, false, false); err != nil {
		t.Fatalf("writeStatusTable returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "...") {
		t.Fatalf("expected truncated cells for narrow tty, got: %q", got)
	}
	if strings.Contains(got, "feature/really-long-branch-name-for-narrow-terminals") {
		t.Fatalf("expected branch truncation for narrow tty, got: %q", got)
	}
}
