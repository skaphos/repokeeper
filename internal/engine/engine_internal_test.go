package engine

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

type testRunner struct {
	responses map[string]testResponse
}

type testResponse struct {
	out string
	err error
}

func (r *testRunner) Run(_ context.Context, dir string, args ...string) (string, error) {
	key := dir + ":" + strings.Join(args, " ")
	if resp, ok := r.responses[key]; ok {
		return resp.out, resp.err
	}
	return "", errors.New("unexpected call: " + key)
}

func TestScanUpdatesRegistry(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v %s", err, string(out))
	}

	cfg := &config.Config{Exclude: []string{}}
	reg := &registry.Registry{}
	eng := New(cfg, reg, vcs.NewGitAdapter(nil))
	statuses, err := eng.Scan(context.Background(), ScanOptions{Roots: []string{root}})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("unexpected registry state: %+v", reg)
	}
}

func TestFilterAndSortHelpers(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{{RepoID: "r1", Status: registry.StatusMissing}},
	}
	if !filterStatus(FilterMissing, model.RepoStatus{RepoID: "r1"}, reg) {
		t.Fatal("expected missing filter match")
	}
	if !filterStatus(FilterErrors, model.RepoStatus{Error: "boom"}, reg) {
		t.Fatal("expected errors filter match")
	}
	if !filterStatus(FilterDiverged, model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingDiverged}}, reg) {
		t.Fatal("expected diverged filter match")
	}
	reg = &registry.Registry{
		Entries: []registry.Entry{{
			RepoID:    "github.com/org/repo",
			Path:      "/repo",
			RemoteURL: "git@github.com:other/repo.git",
		}},
	}
	if !filterStatus(
		FilterRemoteMismatch,
		model.RepoStatus{RepoID: "github.com/org/repo", Path: "/repo"},
		reg,
	) {
		t.Fatal("expected remote mismatch filter match")
	}
	if !matchesProtectedBranch("main", []string{"main", "release/*"}) {
		t.Fatal("expected protected branch match")
	}
	if matchesProtectedBranch("feature/one", []string{"main", "release/*"}) {
		t.Fatal("did not expect feature branch to match protected patterns")
	}

	repos := []model.RepoStatus{{RepoID: "b", Path: "/2"}, {RepoID: "a", Path: "/1"}}
	sortRepoStatuses(repos)
	if repos[0].RepoID != "a" {
		t.Fatalf("expected sorted repos, got %#v", repos)
	}

	results := []SyncResult{{RepoID: "b"}, {RepoID: "a"}}
	sortSyncResults(results)
	if results[0].RepoID != "a" {
		t.Fatalf("expected sorted sync results, got %#v", results)
	}
}

func TestSyncRuntime(t *testing.T) {
	eng := New(&config.Config{Defaults: config.Defaults{
		Concurrency:    3,
		TimeoutSeconds: 9,
		MainBranch:     "trunk",
	}}, &registry.Registry{}, vcs.NewGitAdapter(nil))

	concurrency, timeout, main := eng.syncRuntime(SyncOptions{})
	if concurrency != 3 || timeout != 9 || main != "trunk" {
		t.Fatalf("unexpected defaults: %d %d %q", concurrency, timeout, main)
	}

	concurrency, timeout, main = eng.syncRuntime(SyncOptions{Concurrency: 2, Timeout: 4})
	if concurrency != 2 || timeout != 4 || main != "trunk" {
		t.Fatalf("unexpected override values: %d %d %q", concurrency, timeout, main)
	}

	eng.Config.Defaults.MainBranch = ""
	_, _, main = eng.syncRuntime(SyncOptions{})
	if main != "main" {
		t.Fatalf("expected default main branch fallback, got %q", main)
	}
}

func TestPrepareSyncEntryBranches(t *testing.T) {
	eng := New(&config.Config{Defaults: config.Defaults{MainBranch: "main"}}, &registry.Registry{}, vcs.NewGitAdapter(nil))

	present := registry.Entry{
		RepoID:    "repo",
		Path:      "/repo",
		RemoteURL: "git@github.com:org/repo.git",
		Status:    registry.StatusPresent,
	}

	queue, immediate := eng.prepareSyncEntry(context.Background(), present, SyncOptions{})
	if !queue || immediate != nil {
		t.Fatalf("expected queued present repo, got queue=%v immediate=%+v", queue, immediate)
	}

	queue, immediate = eng.prepareSyncEntry(context.Background(), present, SyncOptions{Filter: FilterMissing})
	if queue || immediate != nil {
		t.Fatalf("expected missing-filter skip, got queue=%v immediate=%+v", queue, immediate)
	}

	missing := present
	missing.Status = registry.StatusMissing
	queue, immediate = eng.prepareSyncEntry(context.Background(), missing, SyncOptions{CheckoutMissing: false})
	if queue || immediate == nil || immediate.Error != SyncErrorMissing {
		t.Fatalf("expected missing immediate result, got queue=%v immediate=%+v", queue, immediate)
	}

	noUpstream := present
	noUpstream.RemoteURL = ""
	queue, immediate = eng.prepareSyncEntry(context.Background(), noUpstream, SyncOptions{})
	if queue || immediate == nil || immediate.Error != SyncErrorSkippedNoUpstream || !immediate.OK {
		t.Fatalf("expected skipped_no_upstream immediate result, got queue=%v immediate=%+v", queue, immediate)
	}
}

func TestSyncEntryMatchesInspectFilterFailure(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:rev-parse --is-bare-repository": {out: "false"},
		"/repo:remote":                         {err: errors.New("permission denied")},
	}}
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(runner))
	entry := registry.Entry{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}

	match, failure := eng.syncEntryMatchesInspectFilter(context.Background(), entry, SyncOptions{Filter: FilterDirty})
	if match || failure == nil {
		t.Fatalf("expected inspect failure, got match=%v failure=%+v", match, failure)
	}
	if failure.Outcome != "failed_inspect" || failure.ErrorClass != "auth" {
		t.Fatalf("unexpected inspect failure mapping: %+v", failure)
	}
}

func TestHandleMissingSyncEntry(t *testing.T) {
	entry := registry.Entry{
		RepoID:    "repo",
		Path:      "/missing",
		RemoteURL: "git@github.com:org/repo.git",
		Branch:    "main",
		Status:    registry.StatusMissing,
	}
	reg := &registry.Registry{Entries: []registry.Entry{entry}}
	runner := &testRunner{responses: map[string]testResponse{
		":clone --branch main --single-branch git@github.com:org/repo.git /missing": {out: ""},
	}}
	eng := New(&config.Config{}, reg, vcs.NewGitAdapter(runner))

	planned := eng.handleMissingSyncEntry(context.Background(), entry, SyncOptions{CheckoutMissing: true, DryRun: true})
	if planned.Outcome != "planned_checkout_missing" || planned.Error != SyncErrorDryRun || !strings.Contains(planned.Action, "git clone") {
		t.Fatalf("unexpected planned missing result: %+v", planned)
	}

	applied := eng.handleMissingSyncEntry(context.Background(), entry, SyncOptions{CheckoutMissing: true})
	if !applied.OK || applied.Outcome != "checkout_missing" {
		t.Fatalf("unexpected applied missing result: %+v", applied)
	}
	if reg.Entries[0].Status != registry.StatusPresent {
		t.Fatalf("expected missing entry promoted to present, got %s", reg.Entries[0].Status)
	}
}

func TestRunSyncDryRunAndApplyHelpers(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:rev-parse --is-bare-repository":    {out: "false"},
		"/repo:remote":                            {out: "origin"},
		"/repo:remote get-url origin":             {out: "git@github.com:org/repo.git"},
		"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
		"/repo:status --porcelain=v1":             {out: ""},
		"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
			out: "main|origin/main|[ahead 1]|>",
		},
		"/repo:rev-list --left-right --count main...origin/main":                               {out: "1\t0"},
		"/repo:config --file .gitmodules --get-regexp submodule":                               {err: errors.New("none")},
		"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
	}}
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(runner))
	entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

	dry := eng.runSyncDryRun(context.Background(), entry, SyncOptions{UpdateLocal: true, PushLocal: true}, "main")
	if dry.Outcome != "planned_push" || dry.Error != SyncErrorDryRun || !strings.Contains(dry.Action, "git push") {
		t.Fatalf("unexpected dry-run push plan: %+v", dry)
	}

	applied := eng.runSyncApply(context.Background(), entry, SyncOptions{UpdateLocal: false}, "main")
	if !applied.OK || applied.Outcome != "fetched" {
		t.Fatalf("unexpected apply fetch result: %+v", applied)
	}
}

func TestRunSyncRebaseApply(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:stash push -u -m repokeeper: pre-rebase stash":                                  {out: "Saved working directory and index state"},
		"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules":         {out: ""},
		"/repo:stash pop":                                                                       {out: "Applied stash"},
	}}
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(runner))
	entry := registry.Entry{RepoID: "repo", Path: "/repo"}
	status := &model.RepoStatus{Worktree: &model.Worktree{Dirty: true}}

	res := eng.runSyncRebaseApply(context.Background(), entry, status, true)
	if !res.OK || res.Outcome != "stashed_rebased" {
		t.Fatalf("unexpected rebase apply result: %+v", res)
	}
	if !strings.Contains(res.Action, "git stash push") || !strings.Contains(res.Action, "git stash pop") {
		t.Fatalf("expected stash actions in result: %+v", res)
	}
}

func TestInspectFailureResult(t *testing.T) {
	entry := registry.Entry{RepoID: "repo", Path: "/repo"}
	res := inspectFailureResult(entry, errors.New("permission denied"))
	if res.RepoID != "repo" || res.Path != "/repo" || res.Outcome != "failed_inspect" || res.ErrorClass != "auth" {
		t.Fatalf("unexpected inspect failure result: %+v", res)
	}
}
