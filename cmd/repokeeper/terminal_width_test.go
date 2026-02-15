package repokeeper

import (
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/termstyle"
	"github.com/spf13/cobra"
)

func captureStatusTableOutputAtWidth(t *testing.T, width int) string {
	t.Helper()
	prevIsTerminalFD := isTerminalFD
	prevGetTerminalSize := getTerminalSize
	defer func() {
		isTerminalFD = prevIsTerminalFD
		getTerminalSize = prevGetTerminalSize
	}()
	isTerminalFD = func(int) bool { return true }
	getTerminalSize = func(int) (int, int, error) { return width, 24, nil }

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
				Path: "/tmp/repo",
				Head: model.Head{Branch: "main"},
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
	return string(data)
}

func captureSyncTableOutputAtWidth(t *testing.T, width int) string {
	t.Helper()
	prevIsTerminalFD := isTerminalFD
	prevGetTerminalSize := getTerminalSize
	defer func() {
		isTerminalFD = prevIsTerminalFD
		getTerminalSize = prevGetTerminalSize
	}()
	isTerminalFD = func(int) bool { return true }
	getTerminalSize = func(int) (int, int, error) { return width, 24, nil }

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe setup failed: %v", err)
	}
	defer reader.Close()

	cmd := &cobra.Command{}
	cmd.SetOut(writer)
	results := []engine.SyncResult{
		{
			RepoID: "github.com/org/repo",
			Path:   "/tmp/repo",
			OK:     false,
			Error:  "simulated failure",
			Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
		},
	}
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{Path: "/tmp/repo", Tracking: model.Tracking{Status: model.TrackingNone}},
		},
	}
	if err := writeSyncTable(cmd, results, report, "/tmp", nil, false, false, false); err != nil {
		t.Fatalf("writeSyncTable returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	return string(data)
}

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

func TestWriteStatusTableCompactsColumnsOnTinyTTY(t *testing.T) {
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
				Path: "/tmp/repo",
				Head: model.Head{Branch: "main"},
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
	if !strings.Contains(got, "PATH") || !strings.Contains(got, "TRACKING") {
		t.Fatalf("expected compact headers, got: %q", got)
	}
	if strings.Contains(got, "BRANCH") || strings.Contains(got, "DIRTY") {
		t.Fatalf("expected tiny mode to drop BRANCH/DIRTY, got: %q", got)
	}
}

func TestWriteSyncTableCompactsColumnsOnTinyTTY(t *testing.T) {
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
	results := []engine.SyncResult{
		{
			RepoID: "github.com/org/repo",
			Path:   "/tmp/repo",
			OK:     false,
			Error:  "simulated failure",
			Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
		},
	}
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{Path: "/tmp/repo", Tracking: model.Tracking{Status: model.TrackingNone}},
		},
	}
	if err := writeSyncTable(cmd, results, report, "/tmp", nil, false, false, false); err != nil {
		t.Fatalf("writeSyncTable returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "PATH") || !strings.Contains(got, "ACTION") || !strings.Contains(got, "OK") || !strings.Contains(got, "ERROR") {
		t.Fatalf("expected tiny sync headers, got: %q", got)
	}
	if strings.Contains(got, "REPO") || strings.Contains(got, "BRANCH") || strings.Contains(got, "TRACKING") {
		t.Fatalf("expected tiny mode to compact sync columns, got: %q", got)
	}
}

func TestWriteStatusTableTinyModeRetainsSemanticColor(t *testing.T) {
	prevIsTerminalFD := isTerminalFD
	prevGetTerminalSize := getTerminalSize
	prevColor := runtimeStateFor(rootCmd).colorOutputEnabled
	defer func() {
		isTerminalFD = prevIsTerminalFD
		getTerminalSize = prevGetTerminalSize
		runtimeStateFor(rootCmd).colorOutputEnabled = prevColor
	}()
	isTerminalFD = func(int) bool { return true }
	getTerminalSize = func(int) (int, int, error) { return 70, 24, nil }
	runtimeStateFor(rootCmd).colorOutputEnabled = true

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
				Path: "/tmp/repo",
				Tracking: model.Tracking{
					Status: model.TrackingGone,
				},
				Worktree: &model.Worktree{Dirty: true},
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
	if !strings.Contains(got, termstyle.Red) || !strings.Contains(got, termstyle.Reset) {
		t.Fatalf("expected semantic color output for tracking state, got: %q", got)
	}
	if strings.ContainsRune(got, '\xff') {
		t.Fatalf("expected no visible tabwriter escape marker in colorized output, got: %q", got)
	}
}

func TestStatusTableHeaderSnapshotsAcrossWidths(t *testing.T) {
	cases := []struct {
		width      int
		wantHeader string
	}{
		{width: 80, wantHeader: "PATH|BRANCH|TRACKING"},
		{width: 100, wantHeader: "PATH|BRANCH|DIRTY|TRACKING"},
		{width: 120, wantHeader: "PATH|BRANCH|DIRTY|TRACKING"},
		{width: 160, wantHeader: "PATH|BRANCH|DIRTY|TRACKING"},
	}

	for _, tc := range cases {
		t.Run("width_"+strconv.Itoa(tc.width), func(t *testing.T) {
			out := captureStatusTableOutputAtWidth(t, tc.width)
			header := strings.Split(strings.TrimSpace(out), "\n")[0]
			if strings.Join(strings.Fields(header), "|") != tc.wantHeader {
				t.Fatalf("unexpected header at width %d: got %q want %q", tc.width, header, tc.wantHeader)
			}
		})
	}
}

func TestSyncTableHeaderSnapshotsAcrossWidths(t *testing.T) {
	cases := []struct {
		width      int
		wantHeader string
	}{
		{width: 80, wantHeader: "PATH|ACTION|OK|ERROR|REPO"},
		{width: 100, wantHeader: "PATH|ACTION|BRANCH|DIRTY|TRACKING|OK|ERROR_CLASS|ERROR|REPO"},
		{width: 120, wantHeader: "PATH|ACTION|BRANCH|DIRTY|TRACKING|OK|ERROR_CLASS|ERROR|REPO"},
		{width: 160, wantHeader: "PATH|ACTION|BRANCH|DIRTY|TRACKING|OK|ERROR_CLASS|ERROR|REPO"},
	}

	for _, tc := range cases {
		t.Run("width_"+strconv.Itoa(tc.width), func(t *testing.T) {
			out := captureSyncTableOutputAtWidth(t, tc.width)
			header := strings.Split(strings.TrimSpace(out), "\n")[0]
			if strings.Join(strings.Fields(header), "|") != tc.wantHeader {
				t.Fatalf("unexpected header at width %d: got %q want %q", tc.width, header, tc.wantHeader)
			}
		})
	}
}
