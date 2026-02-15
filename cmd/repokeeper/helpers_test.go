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

	writeScanTable(cmd, []model.RepoStatus{{RepoID: "r1", Path: "/repo", Bare: true, PrimaryRemote: "origin"}}, false)
	if !strings.Contains(out.String(), "PRIMARY_REMOTE") {
		t.Fatal("expected scan header")
	}

	out.Reset()
	writeStatusTable(cmd, &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "r1", Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}}}}, "/tmp", nil, false, false)
	if !strings.Contains(out.String(), "TRACKING") {
		t.Fatal("expected status header")
	}

	out.Reset()
	writeSyncTable(
		cmd,
		[]engine.SyncResult{{RepoID: "r1", Path: "/repo", OK: false, ErrorClass: "network", Error: "x"}},
		&model.StatusReport{Repos: []model.RepoStatus{{Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}}}},
		"/tmp",
		nil,
		false,
		false,
		false,
	)
	if !strings.Contains(out.String(), "PATH") || !strings.Contains(out.String(), "TRACKING") || !strings.Contains(out.String(), "ERROR_CLASS") || !strings.Contains(out.String(), "REPO") {
		t.Fatal("expected sync header")
	}

	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)
	writeSyncPlan(cmd, []engine.SyncResult{{RepoID: "r1", Path: "/repo", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"}}, "/tmp", nil)
	if !strings.Contains(errOut.String(), "Planned sync operations:") {
		t.Fatal("expected sync plan heading")
	}
	if strings.Contains(errOut.String(), "git fetch --all --prune --prune-tags --no-recurse-submodules") {
		t.Fatal("expected summarized sync action in plan")
	}
}

func TestDescribeSyncAction(t *testing.T) {
	tests := []struct {
		name string
		in   engine.SyncResult
		want string
	}{
		{
			name: "fetch",
			in: engine.SyncResult{
				Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
			},
			want: "fetch",
		},
		{
			name: "fetch and rebase",
			in: engine.SyncResult{
				Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git pull --rebase --no-recurse-submodules",
			},
			want: "fetch + rebase",
		},
		{
			name: "fetch and push",
			in: engine.SyncResult{
				Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git push",
			},
			want: "fetch + push",
		},
		{
			name: "push",
			in: engine.SyncResult{
				Action: "git push",
			},
			want: "push",
		},
		{
			name: "skip no upstream",
			in: engine.SyncResult{
				OK:    true,
				Error: "skipped-no-upstream",
			},
			want: "skip no upstream",
		},
		{
			name: "stash and rebase",
			in: engine.SyncResult{
				Action: "git stash push -u -m \"repokeeper: pre-rebase stash\" && git pull --rebase --no-recurse-submodules && git stash pop",
			},
			want: "stash & rebase",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeSyncAction(tc.in); got != tc.want {
				t.Fatalf("describeSyncAction() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriteStatusTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeStatusTable(cmd, &model.StatusReport{
		Repos: []model.RepoStatus{
			{RepoID: "r1", Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}},
		},
	}, "/tmp", nil, true, false)

	if strings.Contains(out.String(), "PATH") {
		t.Fatalf("expected no table headers, got: %q", out.String())
	}
}

func TestWriteSyncTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeSyncTable(
		cmd,
		[]engine.SyncResult{{RepoID: "r1", Path: "/repo", OK: true, Outcome: "fetched"}},
		&model.StatusReport{Repos: []model.RepoStatus{{Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}}}},
		"/tmp",
		nil,
		false,
		true,
		false,
	)

	if strings.Contains(out.String(), "ACTION") {
		t.Fatalf("expected no sync table headers, got: %q", out.String())
	}
}

func TestWriteScanTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeScanTable(cmd, []model.RepoStatus{{RepoID: "r1", Path: "/repo"}}, true)
	if strings.Contains(out.String(), "PRIMARY_REMOTE") {
		t.Fatalf("expected no scan table headers, got: %q", out.String())
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
	writeStatusTable(cmd, report, "/tmp/work", nil, false, false)

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
	writeStatusTable(cmd, report, "/tmp", nil, false, false)

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
	writeStatusTable(cmd, report, "/tmp", nil, false, false)

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

func TestConfirmSyncExecution(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("yes\n"))
	cmd.SetErr(&bytes.Buffer{})
	ok, err := confirmSyncExecution(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected confirmation to be accepted")
	}

	cmd = &cobra.Command{}
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetErr(&bytes.Buffer{})
	ok, err = confirmSyncExecution(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected confirmation to be rejected")
	}
}

func TestSyncPlanNeedsConfirmation(t *testing.T) {
	fetchOnly := []engine.SyncResult{
		{RepoID: "r1", Path: "/repo", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "r2", Path: "/repo2", Error: "skipped-no-upstream"},
	}
	if syncPlanNeedsConfirmation(fetchOnly) {
		t.Fatal("expected fetch-only plan to skip confirmation")
	}

	withRebase := []engine.SyncResult{
		{RepoID: "r1", Path: "/repo", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git pull --rebase --no-recurse-submodules"},
	}
	if !syncPlanNeedsConfirmation(withRebase) {
		t.Fatal("expected rebase plan to require confirmation")
	}

	withClone := []engine.SyncResult{
		{RepoID: "r1", Path: "/missing", Action: "git clone git@github.com:org/repo.git /missing"},
	}
	if !syncPlanNeedsConfirmation(withClone) {
		t.Fatal("expected clone plan to require confirmation")
	}
}

func TestWriteStatusDetailsAndHelpers(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	ahead := 2
	behind := 1
	repo := model.RepoStatus{
		RepoID: "github.com/org/repo",
		Path:   "/tmp/work/repo",
		Head: model.Head{
			Branch:   "main",
			Detached: true,
		},
		Tracking: model.Tracking{
			Status:   model.TrackingEqual,
			Upstream: "origin/main",
			Ahead:    &ahead,
			Behind:   &behind,
		},
		Worktree:   &model.Worktree{Dirty: true},
		Error:      "boom",
		ErrorClass: "network",
	}
	writeStatusDetails(cmd, repo, "/tmp", nil)

	got := out.String()
	if !strings.Contains(got, "PATH: work/repo") {
		t.Fatalf("expected relative path in details, got: %q", got)
	}
	if !strings.Contains(got, "BRANCH: detached:main") {
		t.Fatalf("expected detached branch in details, got: %q", got)
	}
	if !strings.Contains(got, "TRACKING: up to date") {
		t.Fatalf("expected tracking label in details, got: %q", got)
	}
	if !strings.Contains(got, "ERROR_CLASS: network") || !strings.Contains(got, "ERROR: boom") {
		t.Fatalf("expected error details in output, got: %q", got)
	}

	if got := displayTrackingStatusNoColor(model.TrackingGone); got != string(model.TrackingGone) {
		t.Fatalf("expected raw non-equal tracking status, got %q", got)
	}
	if _, ok := relWithin("", "/tmp/path"); ok {
		t.Fatal("expected relWithin to fail with blank base")
	}
	if _, ok := relWithin("/tmp/base", ""); ok {
		t.Fatal("expected relWithin to fail with blank target")
	}
}

func TestColorizeGuardBranches(t *testing.T) {
	prevColor := colorOutputEnabled
	defer func() { colorOutputEnabled = prevColor }()

	colorOutputEnabled = true
	if got := colorize("", ansiGreen); got != "" {
		t.Fatalf("expected empty value passthrough, got %q", got)
	}
	if got := colorize("x", ""); got != "x" {
		t.Fatalf("expected empty color passthrough, got %q", got)
	}
}

func TestWriteSyncFailureSummary(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	writeSyncFailureSummary(cmd, []engine.SyncResult{
		{RepoID: "ok", Path: "/repos/ok", OK: true, Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "bad", Path: "/repos/bad", OK: false, ErrorClass: "network", Error: "timeout", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
	}, "/repos", nil)

	got := errOut.String()
	if !strings.Contains(got, "Failed sync operations:") {
		t.Fatalf("expected summary heading, got: %q", got)
	}
	if !strings.Contains(got, "bad") || strings.Contains(got, "ok") {
		t.Fatalf("expected only failing repo in summary, got: %q", got)
	}
}

func TestColorizeAndTrackingDisplayBranches(t *testing.T) {
	prevColor := colorOutputEnabled
	defer func() { colorOutputEnabled = prevColor }()

	colorOutputEnabled = false
	if got := colorize("up", ansiGreen); got != "up" {
		t.Fatalf("expected uncolored output when disabled, got %q", got)
	}

	colorOutputEnabled = true
	colored := colorize("up", ansiGreen)
	if !strings.Contains(colored, ansiGreen) || !strings.Contains(colored, ansiReset) {
		t.Fatalf("expected ansi-wrapped output, got %q", colored)
	}
	if got := displayTrackingStatus(model.TrackingDiverged); !strings.Contains(got, "diverged") {
		t.Fatalf("expected diverged tracking label, got %q", got)
	}
	if got := displayTrackingStatus(model.TrackingGone); !strings.Contains(got, "gone") {
		t.Fatalf("expected gone tracking label, got %q", got)
	}
	if got := displayTrackingStatus(model.TrackingNone); got != "none" {
		t.Fatalf("expected plain tracking status for none, got %q", got)
	}
}

func TestTruncateASCIIBranches(t *testing.T) {
	if got := truncateASCII("abcdef", 3); got != "abc" {
		t.Fatalf("expected hard truncate for short max, got %q", got)
	}
	if got := truncateASCII("abc", 10); got != "abc" {
		t.Fatalf("expected no truncate for short value, got %q", got)
	}
}

func TestStatusWarningsAndWideMirrorRow(t *testing.T) {
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				RepoID: "github.com/org/mirror",
				Path:   "/tmp/mirror",
				Type:   "mirror",
				Head:   model.Head{Branch: "main"},
				Tracking: model.Tracking{
					Status:   model.TrackingEqual,
					Upstream: "origin/main",
				},
			},
		},
	}
	reg := &registry.Registry{Entries: []registry.Entry{{Status: registry.StatusMoved}}}
	if !statusHasWarningsOrErrors(report, reg) {
		t.Fatal("expected moved registry status to trigger warnings")
	}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	writeStatusTable(cmd, report, "/tmp", nil, false, true)
	got := out.String()
	if !strings.Contains(got, "PRIMARY_REMOTE") || !strings.Contains(got, "UPSTREAM") {
		t.Fatalf("expected wide headers, got: %q", got)
	}
	if !strings.Contains(got, "mirror") {
		t.Fatalf("expected mirror row labeling, got: %q", got)
	}
}

func TestDescribeSyncActionAdditionalBranches(t *testing.T) {
	cases := []struct {
		name string
		in   engine.SyncResult
		want string
	}{
		{name: "skip generic", in: engine.SyncResult{Error: "skipped"}, want: "skip"},
		{name: "skip missing", in: engine.SyncResult{Error: "missing"}, want: "skip missing"},
		{name: "skip local update no reason", in: engine.SyncResult{Error: "skipped-local-update:"}, want: "skip local update"},
		{name: "empty action success defaults fetch", in: engine.SyncResult{OK: true}, want: "fetch"},
		{name: "empty action failure defaults dash", in: engine.SyncResult{OK: false}, want: "-"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeSyncAction(tc.in); got != tc.want {
				t.Fatalf("describeSyncAction() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestConfirmSyncExecutionEOF(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("y"))
	cmd.SetErr(&bytes.Buffer{})
	ok, err := confirmSyncExecution(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected EOF-terminated yes to be accepted")
	}
}

func TestWriteSyncTableWideBranches(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	ahead := 3
	behind := 2
	results := []engine.SyncResult{
		{RepoID: "r1", Path: "/tmp/repo-a", OK: true, Action: "git pull --rebase --no-recurse-submodules"},
		{RepoID: "r2", Path: "/tmp/repo-mirror", OK: true, Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "r3", Path: "/tmp/repo-missing-status", OK: false, ErrorClass: "network", Error: "timeout"},
	}
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{
				Path:          "/tmp/repo-a",
				Head:          model.Head{Branch: "main", Detached: true},
				Worktree:      &model.Worktree{Dirty: true},
				PrimaryRemote: "origin",
				Tracking: model.Tracking{
					Status:   model.TrackingBehind,
					Upstream: "origin/main",
					Ahead:    &ahead,
					Behind:   &behind,
				},
			},
			{
				Path: "/tmp/repo-mirror",
				Type: "mirror",
				Tracking: model.Tracking{
					Status: model.TrackingNone,
				},
			},
		},
	}

	writeSyncTable(cmd, results, report, "/tmp", nil, false, false, true)
	got := out.String()
	if !strings.Contains(got, "PRIMARY_REMOTE") || !strings.Contains(got, "AHEAD") || !strings.Contains(got, "BEHIND") {
		t.Fatalf("expected wide sync headers, got: %q", got)
	}
	if !strings.Contains(got, "detached:main") {
		t.Fatalf("expected detached branch rendering, got: %q", got)
	}
	if !strings.Contains(got, "repo-missing-status") {
		t.Fatalf("expected fallback path for missing status row, got: %q", got)
	}
	if !strings.Contains(got, "mirror") {
		t.Fatalf("expected mirror tracking label, got: %q", got)
	}
}
