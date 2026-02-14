package repokeeper

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a, ,b,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func TestHasRegistryWarnings(t *testing.T) {
	reg := &registry.Registry{Entries: []registry.Entry{{Status: registry.StatusPresent}}}
	if hasRegistryWarnings(reg) {
		t.Fatal("did not expect warnings")
	}
	reg.Entries[0].Status = registry.StatusMissing
	if !hasRegistryWarnings(reg) {
		t.Fatal("expected warnings")
	}
}

func TestStatusHasWarningsOrErrors(t *testing.T) {
	report := &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "r1"}}}
	reg := &registry.Registry{}
	if statusHasWarningsOrErrors(report, reg) {
		t.Fatal("did not expect warnings")
	}
	report.Repos[0].Error = "boom"
	if !statusHasWarningsOrErrors(report, reg) {
		t.Fatal("expected warnings")
	}
}

func TestWriters(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeScanTable(cmd, []model.RepoStatus{{RepoID: "r1", Path: "/repo", Bare: true, PrimaryRemote: "origin"}})
	if !strings.Contains(out.String(), "PRIMARY_REMOTE") {
		t.Fatal("expected scan header")
	}

	out.Reset()
	writeStatusTable(cmd, &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "r1", Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}}}}, "/tmp", nil)
	if !strings.Contains(out.String(), "TRACKING") {
		t.Fatal("expected status header")
	}

	out.Reset()
	writeSyncTable(cmd, []engine.SyncResult{{RepoID: "r1", OK: false, ErrorClass: "network", Error: "x"}}, false)
	if !strings.Contains(out.String(), "ERROR_CLASS") {
		t.Fatal("expected sync header")
	}
}

func TestLogHelpers(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	flagQuiet = false
	flagVerbose = 1
	infof(cmd, "hello %s", "info")
	debugf(cmd, "hello %s", "debug")
	if !strings.Contains(errOut.String(), "hello info") || !strings.Contains(errOut.String(), "hello debug") {
		t.Fatal("expected both info and debug logs")
	}
}

func TestWriteStatusTableUsesRelativePathAndLabel(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	prevNoColor := flagNoColor
	flagNoColor = true
	defer func() { flagNoColor = prevNoColor }()

	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				RepoID:   "github.com/org/repo",
				Path:     "/tmp/work/repos/repo-a",
				Tracking: model.Tracking{Status: model.TrackingEqual},
				Worktree: &model.Worktree{Dirty: false},
			},
		},
	}
	writeStatusTable(cmd, report, "/tmp/work", nil)

	got := out.String()
	if !strings.Contains(got, "repos/repo-a") {
		t.Fatalf("expected relative path in output, got: %q", got)
	}
	if !strings.Contains(got, "up to date") {
		t.Fatalf("expected 'up to date' label in output, got: %q", got)
	}
}

func TestFormatCellWrapControl(t *testing.T) {
	val := "abcdefghijklmnopqrstuvwxyz"
	if got := formatCell(val, false, 10); got != "abcdefg..." {
		t.Fatalf("expected truncated value, got %q", got)
	}
	if got := formatCell(val, true, 10); got != val {
		t.Fatalf("expected wrapped mode to keep full value, got %q", got)
	}
}

func TestWriteStatusTableDoesNotTruncatePathBranchOrTracking(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	prevNoColor := flagNoColor
	flagNoColor = true
	defer func() { flagNoColor = prevNoColor }()

	branch := "feature/really-long-branch-name-for-testing"
	path := "/tmp/workspace/very/long/path/that/should/not/be/truncated/repo"
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				RepoID: "github.com/org/some-really-long-repository-name",
				Path:   path,
				Head:   model.Head{Branch: branch},
				Tracking: model.Tracking{
					Status: model.TrackingEqual,
				},
				Worktree: &model.Worktree{Dirty: false},
			},
		},
	}
	writeStatusTable(cmd, report, "/tmp", nil)

	got := out.String()
	if !strings.Contains(got, "workspace/very/long/path/that/should/not/be/truncated/repo") {
		t.Fatalf("expected full path in output, got: %q", got)
	}
	if !strings.Contains(got, branch) {
		t.Fatalf("expected full branch in output, got: %q", got)
	}
	if !strings.Contains(got, "up to date") {
		t.Fatalf("expected full tracking label in output, got: %q", got)
	}
}

func TestWriteStatusTableStripsEscapeMarkers(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	prevNoColor := flagNoColor
	flagNoColor = false
	defer func() { flagNoColor = prevNoColor }()

	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				RepoID: "github.com/org/repo",
				Path:   "/tmp/repo",
				Head:   model.Head{Branch: "main"},
				Tracking: model.Tracking{
					Status: model.TrackingEqual,
				},
				Worktree: &model.Worktree{Dirty: false},
			},
		},
	}
	writeStatusTable(cmd, report, "/tmp", nil)

	got := out.String()
	if strings.ContainsRune(got, '\xff') {
		t.Fatalf("expected no visible tabwriter escape markers, got: %q", got)
	}
}

func TestDisplayRepoPathPrefersCWDThenRoot(t *testing.T) {
	if got := displayRepoPath("/tmp/work/app/repo", "/tmp/work", []string{"/tmp"}); got != "app/repo" {
		t.Fatalf("expected cwd-relative path, got %q", got)
	}
	if got := displayRepoPath("/tmp/root/repo", "/tmp/work", []string{"/tmp/root"}); got != "repo" {
		t.Fatalf("expected root-relative path, got %q", got)
	}
	if got := displayRepoPath("/opt/repo", "/tmp/work", []string{"/tmp/root"}); got != "/opt/repo" {
		t.Fatalf("expected absolute fallback path, got %q", got)
	}
}
