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
	}}, &registry.Registry{}, vcs.NewGitAdapter(nil))

	concurrency, timeout := eng.syncRuntime(SyncOptions{})
	if concurrency != 3 || timeout != 9 {
		t.Fatalf("unexpected defaults: %d %d", concurrency, timeout)
	}

	concurrency, timeout = eng.syncRuntime(SyncOptions{Concurrency: 2, Timeout: 4})
	if concurrency != 2 || timeout != 4 {
		t.Fatalf("unexpected override values: %d %d", concurrency, timeout)
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
		"/repo:rev-list --left-right --count main...origin/main":                                          {out: "1\t0"},
		"/repo:config --file .gitmodules --get-regexp submodule":                                          {err: errors.New("none")},
		"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
	}}
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(runner))
	entry := registry.Entry{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:org/repo.git", Status: registry.StatusPresent}

	dry := eng.runSyncDryRun(context.Background(), entry, SyncOptions{UpdateLocal: true, PushLocal: true})
	if dry.Outcome != "planned_push" || dry.Error != SyncErrorDryRun || !strings.Contains(dry.Action, "git push") {
		t.Fatalf("unexpected dry-run push plan: %+v", dry)
	}

	applied := eng.runSyncApply(context.Background(), entry, SyncOptions{UpdateLocal: false})
	if !applied.OK || applied.Outcome != "fetched" {
		t.Fatalf("unexpected apply fetch result: %+v", applied)
	}
}

func TestRunSyncRebaseApply(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:stash push -u -m repokeeper: pre-rebase stash":                          {out: "Saved working directory and index state"},
		"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {out: ""},
		"/repo:stash pop": {out: "Applied stash"},
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

func TestEngineGuardErrors(t *testing.T) {
	eng := New(&config.Config{}, nil, vcs.NewGitAdapter(nil))
	if _, err := eng.Scan(context.Background(), ScanOptions{}); err == nil {
		t.Fatal("expected scan no roots error")
	}

	eng = New(&config.Config{}, nil, vcs.NewGitAdapter(nil))
	if _, err := eng.Status(context.Background(), StatusOptions{}); err == nil {
		t.Fatal("expected status registry not loaded error")
	}
	if _, err := eng.ExecuteSyncPlan(context.Background(), nil, SyncOptions{}); err == nil {
		t.Fatal("expected execute sync plan registry not loaded error")
	}
}

func TestExecuteSyncPlanAppliesActions(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
		"/repo:stash push -u -m repokeeper: pre-rebase stash":                                             {out: "Saved working directory and index state"},
		"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules":                    {out: ""},
		"/repo:stash pop": {out: "Applied stash"},
		"/repo:push":      {out: ""},
	}}
	eng := New(&config.Config{}, &registry.Registry{Entries: []registry.Entry{
		{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent, RemoteURL: "git@github.com:org/repo.git"},
	}}, vcs.NewGitAdapter(runner))

	plan := []SyncResult{{
		RepoID:  "repo",
		Path:    "/repo",
		OK:      true,
		Error:   SyncErrorDryRun,
		Outcome: "planned_fetch",
		Action:  "git fetch --all --prune --prune-tags --no-recurse-submodules && git stash push -u -m \"repokeeper: pre-rebase stash\" && git pull --rebase --no-recurse-submodules && git stash pop && git push",
	}}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: true})
	if err != nil {
		t.Fatalf("execute sync plan failed: %v", err)
	}
	if len(results) != 1 || !results[0].OK || results[0].Outcome != "pushed" {
		t.Fatalf("unexpected execute result: %+v", results)
	}
}

func TestExecuteSyncPlanStopsOnFailure(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {err: errors.New("network timeout")},
		"/repo2:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
	}}
	eng := New(&config.Config{}, &registry.Registry{Entries: []registry.Entry{
		{RepoID: "repo1", Path: "/repo1", Status: registry.StatusPresent, RemoteURL: "git@github.com:org/repo1.git"},
		{RepoID: "repo2", Path: "/repo2", Status: registry.StatusPresent, RemoteURL: "git@github.com:org/repo2.git"},
	}}, vcs.NewGitAdapter(runner))

	plan := []SyncResult{
		{RepoID: "repo1", Path: "/repo1", OK: true, Error: SyncErrorDryRun, Outcome: "planned_fetch", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
		{RepoID: "repo2", Path: "/repo2", OK: true, Error: SyncErrorDryRun, Outcome: "planned_fetch", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
	}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: false})
	if err != nil {
		t.Fatalf("execute sync plan failed: %v", err)
	}
	if len(results) != 1 || results[0].OK {
		t.Fatalf("expected stop on first failure, got %+v", results)
	}
	if results[0].ErrorClass != "timeout" || results[0].Error != SyncErrorFetchTimeout {
		t.Fatalf("expected normalized fetch failure, got class=%q error=%q", results[0].ErrorClass, results[0].Error)
	}
}

func TestExecuteSyncPlanStopsOnNonDryRunFailure(t *testing.T) {
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(nil))
	plan := []SyncResult{
		{RepoID: "repo1", Path: "/repo1", OK: false, Error: "boom", Outcome: "failed_fetch"},
		{RepoID: "repo2", Path: "/repo2", OK: true, Error: SyncErrorDryRun, Outcome: "planned_fetch", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
	}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: false})
	if err != nil {
		t.Fatalf("execute sync plan failed: %v", err)
	}
	if len(results) != 1 || results[0].RepoID != "repo1" {
		t.Fatalf("expected stop on first non-dry-run failure, got %+v", results)
	}
}

func TestExecuteSyncPlanCloneAction(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		":clone --branch main --single-branch git@github.com:org/missing.git /missing": {out: ""},
	}}
	reg := &registry.Registry{Entries: []registry.Entry{
		{
			RepoID:    "missing",
			Path:      "/missing",
			RemoteURL: "git@github.com:org/missing.git",
			Branch:    "main",
			Status:    registry.StatusMissing,
		},
	}}
	eng := New(&config.Config{}, reg, vcs.NewGitAdapter(runner))

	plan := []SyncResult{{
		RepoID:  "missing",
		Path:    "/missing",
		OK:      true,
		Error:   SyncErrorDryRun,
		Outcome: "planned_checkout_missing",
		Action:  "git clone --branch main --single-branch git@github.com:org/missing.git /missing",
	}}
	results, err := eng.ExecuteSyncPlan(context.Background(), plan, SyncOptions{ContinueOnError: true})
	if err != nil {
		t.Fatalf("execute clone plan failed: %v", err)
	}
	if len(results) != 1 || !results[0].OK || results[0].Outcome != "checkout_missing" {
		t.Fatalf("unexpected clone execute result: %+v", results)
	}
	if reg.Entries[0].Status != registry.StatusPresent {
		t.Fatalf("expected cloned entry status present, got %s", reg.Entries[0].Status)
	}
}

func TestExecuteSyncPlanWithCallbackInvokesPerResult(t *testing.T) {
	runner := &testRunner{responses: map[string]testResponse{
		"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
	}}
	eng := New(&config.Config{}, &registry.Registry{Entries: []registry.Entry{
		{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent, RemoteURL: "git@github.com:org/repo.git"},
	}}, vcs.NewGitAdapter(runner))

	plan := []SyncResult{{
		RepoID:  "repo",
		Path:    "/repo",
		OK:      true,
		Error:   SyncErrorDryRun,
		Outcome: SyncOutcomePlannedFetch,
		Action:  "git fetch --all --prune --prune-tags --no-recurse-submodules",
	}}

	seen := 0
	results, err := eng.ExecuteSyncPlanWithCallback(context.Background(), plan, SyncOptions{ContinueOnError: true}, func(res SyncResult) {
		seen++
		if res.Path != "/repo" {
			t.Fatalf("unexpected callback result path: %q", res.Path)
		}
	})
	if err != nil {
		t.Fatalf("execute sync plan with callback failed: %v", err)
	}
	if seen != 1 {
		t.Fatalf("expected callback once, got %d", seen)
	}
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestFilterAndLookupEdgeBranches(t *testing.T) {
	if filterStatus(FilterMissing, model.RepoStatus{RepoID: "r1"}, nil) {
		t.Fatal("expected missing filter false without registry")
	}
	if filterStatus(FilterRemoteMismatch, model.RepoStatus{RepoID: "r1"}, nil) {
		t.Fatal("expected remote mismatch false without registry")
	}
	if !filterStatus(FilterKind("unknown"), model.RepoStatus{}, nil) {
		t.Fatal("expected unknown filter default true")
	}

	if findRegistryEntryForStatus(nil, model.RepoStatus{}) != nil {
		t.Fatal("expected nil status lookup for nil registry")
	}

	if hasRemoteMismatch(model.RepoStatus{RepoID: "github.com/org/repo"}, registry.Entry{}) {
		t.Fatal("expected no mismatch when registry remote is empty")
	}
	if hasRemoteMismatch(model.RepoStatus{}, registry.Entry{RemoteURL: "not-a-normalizable-url"}) {
		t.Fatal("expected no mismatch when status repo id is empty")
	}

	if findRegistryEntryForSyncResult(nil, SyncResult{}) != nil {
		t.Fatal("expected nil sync lookup for nil registry")
	}
	reg := &registry.Registry{Entries: []registry.Entry{
		{RepoID: "repo", Path: "/repo-a"},
	}}
	match := findRegistryEntryForSyncResult(reg, SyncResult{RepoID: "repo", Path: "/repo-b"})
	if match == nil || match.Path != "/repo-a" {
		t.Fatalf("expected fallback match by repo id, got %+v", match)
	}

	entries := []registry.Entry{{RepoID: "a", Path: "/a"}}
	updated := replaceRegistryEntry(entries, registry.Entry{RepoID: "b", Path: "/b"})
	if len(updated) != 1 || updated[0].RepoID != "a" {
		t.Fatalf("expected unchanged entries for missing replacement target, got %+v", updated)
	}
}

func TestRunSyncHelperEdgeBranches(t *testing.T) {
	entry := registry.Entry{
		RepoID:    "repo",
		Path:      "/repo",
		RemoteURL: "git@github.com:org/repo.git",
		Status:    registry.StatusPresent,
	}

	inspectFailRunner := &testRunner{responses: map[string]testResponse{
		"/repo:rev-parse --is-bare-repository": {out: "false"},
		"/repo:remote":                         {err: errors.New("permission denied")},
	}}
	eng := New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(inspectFailRunner))
	dry := eng.runSyncDryRun(context.Background(), entry, SyncOptions{UpdateLocal: true})
	if dry.Outcome != "failed_inspect" || dry.ErrorClass != "auth" {
		t.Fatalf("expected inspect failure dry-run result, got %+v", dry)
	}

	filterGoneRunner := &testRunner{responses: map[string]testResponse{
		"/repo:rev-parse --is-bare-repository":    {out: "false"},
		"/repo:remote":                            {out: "origin"},
		"/repo:remote get-url origin":             {out: "git@github.com:org/repo.git"},
		"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
		"/repo:status --porcelain=v1":             {out: ""},
		"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
			out: "main|origin/main||=",
		},
		"/repo:rev-list --left-right --count main...origin/main": {out: "0\t0"},
		"/repo:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
	}}
	eng = New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(filterGoneRunner))
	gone := eng.runSyncApply(context.Background(), entry, SyncOptions{Filter: FilterGone})
	if !gone.OK || gone.Outcome != "skipped" || gone.Error != SyncErrorSkipped {
		t.Fatalf("expected filter-gone skip result, got %+v", gone)
	}
}

type unsupportedLocalUpdateAdapter struct {
	*planAdapter
}

func (u *unsupportedLocalUpdateAdapter) SupportsLocalUpdate(context.Context, string) (bool, string, error) {
	return false, "local update unsupported for vcs hg", nil
}

func (u *unsupportedLocalUpdateAdapter) FetchAction(context.Context, string) (string, error) {
	return "hg pull", nil
}

func TestSyncSkipsUnsupportedLocalUpdateByAdapterCapability(t *testing.T) {
	adapter := &unsupportedLocalUpdateAdapter{planAdapter: &planAdapter{}}
	eng := &Engine{
		cfg:      &config.Config{},
		registry: &registry.Registry{},
		adapter:  adapter,
	}
	entry := registry.Entry{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}

	dry := eng.runSyncDryRun(context.Background(), entry, SyncOptions{UpdateLocal: true})
	if dry.Outcome != SyncOutcomeSkippedLocalUpdate || !dry.OK {
		t.Fatalf("unexpected dry-run result: %+v", dry)
	}
	if dry.Error != SyncErrorSkippedLocalUpdatePrefix+"local update unsupported for vcs hg" {
		t.Fatalf("unexpected dry-run skip reason: %q", dry.Error)
	}
	if dry.Action != "hg pull" {
		t.Fatalf("unexpected dry-run action: %q", dry.Action)
	}

	applied := eng.runSyncApply(context.Background(), entry, SyncOptions{UpdateLocal: true})
	if applied.Outcome != SyncOutcomeSkippedLocalUpdate || !applied.OK {
		t.Fatalf("unexpected apply result: %+v", applied)
	}
	if applied.Error != SyncErrorSkippedLocalUpdatePrefix+"local update unsupported for vcs hg" {
		t.Fatalf("unexpected apply skip reason: %q", applied.Error)
	}
	if len(adapter.calls) != 1 || adapter.calls[0] != "fetch:/repo" {
		t.Fatalf("expected fetch-only apply call sequence, got %+v", adapter.calls)
	}
}
