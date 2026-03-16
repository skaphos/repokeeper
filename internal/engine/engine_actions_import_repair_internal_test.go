// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func TestPlanImportClonesGuardsAndErrors(t *testing.T) {
	t.Run("nil registry returns empty plan", func(t *testing.T) {
		eng := &Engine{cfg: &config.Config{}, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}
		plan, err := eng.PlanImportClones(nil, ImportCloneOptions{CWD: t.TempDir()})
		if err != nil {
			t.Fatalf("plan import clones: %v", err)
		}
		if len(plan.Clones) != 0 || len(plan.Skipped) != 0 {
			t.Fatalf("expected empty plan, got %+v", plan)
		}
	})

	t.Run("empty entries returns empty plan", func(t *testing.T) {
		eng := &Engine{cfg: &config.Config{}, registry: &registry.Registry{}, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}
		plan, err := eng.PlanImportClones(nil, ImportCloneOptions{CWD: t.TempDir()})
		if err != nil {
			t.Fatalf("plan import clones: %v", err)
		}
		if len(plan.Clones) != 0 || len(plan.Skipped) != 0 {
			t.Fatalf("expected empty plan, got %+v", plan)
		}
	})

	t.Run("rejects targets outside cwd", func(t *testing.T) {
		eng := &Engine{registry: &registry.Registry{}, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}
		_, err := eng.PlanImportClones([]registry.Entry{{
			RepoID:    "repo",
			Path:      "ignored",
			RemoteURL: "git@github.com:org/repo.git",
			Branch:    "main",
		}}, ImportCloneOptions{
			CWD: t.TempDir(),
			ResolveTargetRelativePath: func(_ registry.Entry, _ string) string {
				return "../escape"
			},
		})
		if err == nil || !strings.Contains(err.Error(), "outside current directory") {
			t.Fatalf("expected outside cwd error, got %v", err)
		}
	})

	t.Run("rejects duplicate target paths", func(t *testing.T) {
		eng := &Engine{registry: &registry.Registry{}, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}
		entries := []registry.Entry{
			{RepoID: "a", Path: "a", RemoteURL: "git@github.com:org/a.git", Branch: "main"},
			{RepoID: "b", Path: "b", RemoteURL: "git@github.com:org/b.git", Branch: "main"},
		}
		_, err := eng.PlanImportClones(entries, ImportCloneOptions{
			CWD: t.TempDir(),
			ResolveTargetRelativePath: func(_ registry.Entry, _ string) string {
				return "same/path"
			},
		})
		if err == nil || !strings.Contains(err.Error(), "multiple repos resolve to same target path") {
			t.Fatalf("expected duplicate target error, got %v", err)
		}
	})

	t.Run("rejects existing targets unless dangerous delete", func(t *testing.T) {
		cwd := t.TempDir()
		target := filepath.Join(cwd, "repo")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatalf("mkdir target: %v", err)
		}
		eng := &Engine{registry: &registry.Registry{}, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}
		_, err := eng.PlanImportClones([]registry.Entry{{
			RepoID:    "repo",
			Path:      "repo",
			RemoteURL: "git@github.com:org/repo.git",
			Branch:    "main",
		}}, ImportCloneOptions{CWD: cwd})
		if err == nil || !strings.Contains(err.Error(), "import target conflicts detected") {
			t.Fatalf("expected conflict error, got %v", err)
		}
	})
}

func TestPlanImportClonesSkipsAndSuccess(t *testing.T) {
	cwd := t.TempDir()
	ignoredPath := filepath.Join(cwd, "ignored/repo")
	eng := &Engine{
		cfg:      &config.Config{IgnoredPaths: []string{ignoredPath}},
		registry: &registry.Registry{},
		adapter:  &planAdapter{},
	}

	entries := []registry.Entry{
		{RepoID: "ok", Path: "ok/repo", RemoteURL: "git@github.com:org/ok.git", Branch: "main"},
		{RepoID: "missing-remote", Path: "missing/remote", Branch: "main"},
		{RepoID: "missing-branch", Path: "missing/branch", RemoteURL: "git@github.com:org/branch.git"},
		{RepoID: "ignored", Path: "ignored/repo", RemoteURL: "git@github.com:org/ignored.git", Branch: "main"},
		{RepoID: "bundle-missing", Path: "gone/repo", RemoteURL: "git@github.com:org/gone.git", Branch: "main", Status: registry.StatusMissing},
	}

	plan, err := eng.PlanImportClones(entries, ImportCloneOptions{CWD: cwd})
	if err != nil {
		t.Fatalf("plan import clones: %v", err)
	}
	if plan.CWD != filepath.Clean(cwd) {
		t.Fatalf("unexpected cwd in plan: %q", plan.CWD)
	}
	if len(plan.Clones) != 1 || plan.Clones[0].Entry.RepoID != "ok" {
		t.Fatalf("expected one clone target for ok repo, got %+v", plan.Clones)
	}
	if len(plan.Skipped) != 4 {
		t.Fatalf("expected 4 skipped entries, got %+v", plan.Skipped)
	}

	reasons := map[string]string{}
	for _, skip := range plan.Skipped {
		reasons[skip.Entry.RepoID] = skip.Reason
	}
	if reasons["missing-remote"] != "no remote URL configured" {
		t.Fatalf("unexpected missing-remote reason: %q", reasons["missing-remote"])
	}
	if reasons["missing-branch"] != "no upstream branch configured" {
		t.Fatalf("unexpected missing-branch reason: %q", reasons["missing-branch"])
	}
	if reasons["ignored"] != "path is ignored by local config" {
		t.Fatalf("unexpected ignored reason: %q", reasons["ignored"])
	}
	if reasons["bundle-missing"] != "marked missing in bundle" {
		t.Fatalf("unexpected bundle-missing reason: %q", reasons["bundle-missing"])
	}
}

func TestExecuteImportClonesSuccessFailureAndSkips(t *testing.T) {
	t.Run("successful clone with dangerous delete updates registry and callbacks", func(t *testing.T) {
		cwd := t.TempDir()
		targetPath := filepath.Join(cwd, "repos", "repo")
		if err := os.MkdirAll(targetPath, 0o755); err != nil {
			t.Fatalf("mkdir target: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetPath, "old.txt"), []byte("old"), 0o644); err != nil {
			t.Fatalf("write old file: %v", err)
		}

		adapter := &planAdapter{cloneErrByDir: map[string]error{}}
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "old", Status: registry.StatusMissing}}}
		eng := &Engine{registry: reg, adapter: adapter, classifier: vcs.NewGitErrorClassifier()}

		plan := ImportClonePlan{
			DangerouslyDeleteExisting: true,
			Clones: []ImportCloneTarget{{
				Path: targetPath,
				Entry: registry.Entry{
					RepoID:    "repo",
					RemoteURL: "git@github.com:org/repo.git",
					Branch:    "main",
				},
			}},
		}

		started := 0
		completed := 0
		failures, err := eng.ExecuteImportClones(context.Background(), plan, ImportCloneCallbacks{
			OnStart: func(SyncResult) { started++ },
			OnComplete: func(SyncResult) {
				completed++
			},
		})
		if err != nil {
			t.Fatalf("execute import clones: %v", err)
		}
		if len(failures) != 0 {
			t.Fatalf("expected no failures, got %+v", failures)
		}
		if started != 1 || completed != 1 {
			t.Fatalf("expected one callback pair, got start=%d complete=%d", started, completed)
		}
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Fatalf("expected old target removed before clone, stat err=%v", err)
		}
		entry := reg.FindByRepoID("repo")
		if entry == nil || entry.Path != targetPath || entry.Status != registry.StatusPresent {
			t.Fatalf("expected registry entry updated to present clone target, got %+v", entry)
		}
	})

	t.Run("clone error is classified and stored", func(t *testing.T) {
		targetPath := filepath.Join(t.TempDir(), "repo")
		adapter := &planAdapter{cloneErrByDir: map[string]error{targetPath: errors.New("could not resolve host")}}
		eng := &Engine{
			registry:   &registry.Registry{},
			adapter:    adapter,
			classifier: vcs.NewGitErrorClassifier(),
		}

		plan := ImportClonePlan{Clones: []ImportCloneTarget{{
			Path: targetPath,
			Entry: registry.Entry{
				RepoID:    "repo",
				RemoteURL: "git@github.com:org/repo.git",
				Branch:    "main",
			},
		}}}

		completions := 0
		failures, err := eng.ExecuteImportClones(context.Background(), plan, ImportCloneCallbacks{
			OnComplete: func(SyncResult) { completions++ },
		})
		if err != nil {
			t.Fatalf("execute import clones: %v", err)
		}
		if completions != 1 {
			t.Fatalf("expected one completion callback, got %d", completions)
		}
		if len(failures) != 1 {
			t.Fatalf("expected one failure, got %+v", failures)
		}
		if failures[0].ErrorClass != "network" || failures[0].Error != "import-clone-network" {
			t.Fatalf("unexpected failure classification: %+v", failures[0])
		}
		entry := eng.registry.FindByRepoID("repo")
		if entry == nil || entry.Status != registry.StatusMissing || entry.Path != targetPath {
			t.Fatalf("expected missing registry entry on clone failure, got %+v", entry)
		}
	})

	t.Run("skipped entries update or remove registry", func(t *testing.T) {
		reg := &registry.Registry{Entries: []registry.Entry{
			{RepoID: "ignored", Path: "/old/ignored", Status: registry.StatusPresent},
			{RepoID: "incomplete", Path: "/old/incomplete", Status: registry.StatusPresent},
		}}
		eng := &Engine{registry: reg, adapter: &planAdapter{}, classifier: vcs.NewGitErrorClassifier()}

		plan := ImportClonePlan{Skipped: []ImportCloneSkip{
			{
				Path:   "/new/ignored",
				Entry:  registry.Entry{RepoID: "ignored", Status: registry.StatusPresent},
				Reason: "path is ignored by local config",
			},
			{
				Path:   "/new/incomplete",
				Entry:  registry.Entry{RepoID: "incomplete"},
				Reason: "no remote URL configured",
			},
		}}

		failures, err := eng.ExecuteImportClones(context.Background(), plan, ImportCloneCallbacks{})
		if err != nil {
			t.Fatalf("execute import clones: %v", err)
		}
		if len(failures) != 0 {
			t.Fatalf("expected no failures for skipped-only plan, got %+v", failures)
		}
		if reg.FindByRepoID("ignored") != nil {
			t.Fatalf("expected ignored repo removed from registry: %+v", reg.Entries)
		}
		entry := reg.FindByRepoID("incomplete")
		if entry == nil || entry.Path != "/new/incomplete" || entry.Status != registry.StatusMissing {
			t.Fatalf("expected incomplete repo marked missing at new path, got %+v", entry)
		}
	})
}

func TestImportCloneHelperFunctions(t *testing.T) {
	t.Run("import clone failure messages", func(t *testing.T) {
		cases := map[string]string{
			"auth":           "import-clone-auth",
			"network":        "import-clone-network",
			"timeout":        "import-clone-timeout",
			"corrupt":        "import-clone-corrupt",
			"missing_remote": "import-clone-missing-remote",
			"other":          "import-clone-failed",
		}
		for class, want := range cases {
			if got := importCloneFailureMessage(class); got != want {
				t.Fatalf("importCloneFailureMessage(%q)=%q want %q", class, got, want)
			}
		}
	})

	t.Run("find import clone conflicts", func(t *testing.T) {
		existingPath := filepath.Join(t.TempDir(), "exists")
		if err := os.MkdirAll(existingPath, 0o755); err != nil {
			t.Fatalf("mkdir existing: %v", err)
		}
		targets := map[string]ImportCloneTarget{
			"a": {Path: existingPath, Entry: registry.Entry{RepoID: "a"}},
			"b": {Path: filepath.Join(t.TempDir(), "missing"), Entry: registry.Entry{RepoID: "b"}},
		}
		conflicts := findImportCloneConflicts(targets, map[string]ImportCloneSkip{})
		if len(conflicts) != 1 || conflicts[0].entry.RepoID != "a" {
			t.Fatalf("unexpected conflicts: %+v", conflicts)
		}

		skipped := findImportCloneConflicts(targets, map[string]ImportCloneSkip{"a": {Reason: "skip"}})
		if len(skipped) != 0 {
			t.Fatalf("expected skipped target not to conflict, got %+v", skipped)
		}
	})

	t.Run("ignored import path set", func(t *testing.T) {
		if got := ignoredImportPathSet(nil); len(got) != 0 {
			t.Fatalf("expected empty map for nil config, got %+v", got)
		}
		if got := ignoredImportPathSet(&config.Config{}); len(got) != 0 {
			t.Fatalf("expected empty map for empty ignored paths, got %+v", got)
		}
		root := t.TempDir()
		ignored := filepath.Join(root, "foo", "bar")
		got := ignoredImportPathSet(&config.Config{IgnoredPaths: []string{ignored}})
		if !got[pathCleanCanonical(ignored)] {
			t.Fatalf("expected canonical ignored path in set, got %+v", got)
		}
	})

	t.Run("set and remove import registry entries by repo id", func(t *testing.T) {
		eng := &Engine{registry: &registry.Registry{Entries: []registry.Entry{{RepoID: "a", Path: "/old"}}}}
		eng.setImportRegistryEntryByRepoID(registry.Entry{RepoID: "a", Path: "/new", Status: registry.StatusPresent})
		if got := eng.registry.FindByRepoID("a"); got == nil || got.Path != "/new" {
			t.Fatalf("expected existing repo updated, got %+v", got)
		}
		eng.setImportRegistryEntryByRepoID(registry.Entry{RepoID: "b", Path: "/b", Status: registry.StatusPresent})
		if got := eng.registry.FindByRepoID("b"); got == nil {
			t.Fatalf("expected new repo inserted, entries=%+v", eng.registry.Entries)
		}

		eng.removeImportRegistryEntryByRepoID("a")
		if eng.registry.FindByRepoID("a") != nil {
			t.Fatalf("expected repo a removed, entries=%+v", eng.registry.Entries)
		}

		var nilRegEngine Engine
		nilRegEngine.removeImportRegistryEntryByRepoID("missing")
	})
}

func TestRepairUpstreamScenarios(t *testing.T) {
	t.Run("nil registry returns error", func(t *testing.T) {
		eng := &Engine{cfg: &config.Config{}, adapter: vcs.NewGitAdapter(&testRunner{}), classifier: vcs.NewGitErrorClassifier()}
		_, err := eng.RepairUpstream(context.Background(), "repo", filepath.Join(t.TempDir(), "config.yaml"))
		if err == nil || !strings.Contains(err.Error(), "registry not available") {
			t.Fatalf("expected nil registry error, got %v", err)
		}
	})

	t.Run("repo not found returns error", func(t *testing.T) {
		eng := &Engine{cfg: &config.Config{}, registry: &registry.Registry{}, adapter: vcs.NewGitAdapter(&testRunner{}), classifier: vcs.NewGitErrorClassifier()}
		_, err := eng.RepairUpstream(context.Background(), "repo", filepath.Join(t.TempDir(), "config.yaml"))
		if err == nil || !strings.Contains(err.Error(), "not found in registry") {
			t.Fatalf("expected repo not found error, got %v", err)
		}
	})

	t.Run("missing entry status is skipped", func(t *testing.T) {
		eng := &Engine{
			cfg:      &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusMissing}}},
			adapter:  vcs.NewGitAdapter(&testRunner{}),
		}
		res, err := eng.RepairUpstream(context.Background(), "repo", filepath.Join(t.TempDir(), "config.yaml"))
		if err != nil {
			t.Fatalf("repair upstream: %v", err)
		}
		if !res.OK || res.Action != "skip missing" {
			t.Fatalf("expected skip missing result, got %+v", res)
		}
	})

	t.Run("detached head is skipped", func(t *testing.T) {
		runner := &testRunner{responses: map[string]testResponse{
			"/repo:rev-parse --is-bare-repository":    {out: "false"},
			"/repo:remote":                            {out: "origin"},
			"/repo:remote get-url origin":             {out: "git@github.com:org/repo.git"},
			"/repo:symbolic-ref --quiet --short HEAD": {err: errors.New("detached")},
			"/repo:rev-parse --short HEAD":            {out: "abc1234"},
			"/repo:status --porcelain=v1":             {out: ""},
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main||=",
			},
			"/repo:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
		}}
		eng := &Engine{
			cfg:      &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}}},
			adapter:  vcs.NewGitAdapter(runner),
		}
		res, err := eng.RepairUpstream(context.Background(), "repo", filepath.Join(t.TempDir(), "config.yaml"))
		if err != nil {
			t.Fatalf("repair upstream: %v", err)
		}
		if !res.OK || res.Action != "skip detached" {
			t.Fatalf("expected detached skip result, got %+v", res)
		}
	})

	t.Run("no remote is skipped", func(t *testing.T) {
		runner := &testRunner{responses: map[string]testResponse{
			"/repo:rev-parse --is-bare-repository":    {out: "false"},
			"/repo:remote":                            {out: ""},
			"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo:status --porcelain=v1":             {out: ""},
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|||",
			},
			"/repo:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
		}}
		eng := &Engine{
			cfg:      &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}}},
			adapter:  vcs.NewGitAdapter(runner),
		}
		res, err := eng.RepairUpstream(context.Background(), "repo", filepath.Join(t.TempDir(), "config.yaml"))
		if err != nil {
			t.Fatalf("repair upstream: %v", err)
		}
		if !res.OK || res.Action != "skip no remote" {
			t.Fatalf("expected no-remote skip result, got %+v", res)
		}
	})

	t.Run("successful repair sets upstream and saves config", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		runner := &testRunner{responses: map[string]testResponse{
			"/repo:rev-parse --is-bare-repository":    {out: "false"},
			"/repo:remote":                            {out: "origin"},
			"/repo:remote get-url origin":             {out: "git@github.com:org/repo.git"},
			"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo:status --porcelain=v1":             {out: ""},
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/dev|[behind 1]|<",
			},
			"/repo:rev-list --left-right --count main...origin/dev":  {out: "0\t1"},
			"/repo:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
			"/repo:branch --set-upstream-to origin/main main":        {out: ""},
		}}

		entry := registry.Entry{RepoID: "repo", Path: "/repo", Branch: "main", Status: registry.StatusPresent}
		eng := &Engine{
			cfg:      &config.Config{Defaults: config.Defaults{MainBranch: "trunk"}},
			registry: &registry.Registry{Entries: []registry.Entry{entry}},
			adapter:  vcs.NewGitAdapter(runner),
		}

		res, err := eng.RepairUpstream(context.Background(), "repo", cfgPath)
		if err != nil {
			t.Fatalf("repair upstream: %v", err)
		}
		if !res.OK || res.Action != "repaired" {
			t.Fatalf("expected successful repair, got %+v", res)
		}
		if res.TargetUpstream != "origin/main" {
			t.Fatalf("unexpected target upstream: %+v", res)
		}
		updated := eng.registry.FindByRepoID("repo")
		if updated == nil || updated.Status != registry.StatusPresent || updated.Branch != "main" {
			t.Fatalf("expected registry updated after repair, got %+v", updated)
		}
		if _, err := os.Stat(cfgPath); err != nil {
			t.Fatalf("expected config file saved, stat err=%v", err)
		}
	})
}

func TestRepairHelpers(t *testing.T) {
	t.Run("repairResolveTargetBranch precedence", func(t *testing.T) {
		if got := repairResolveTargetBranch(
			registry.Entry{Branch: " release "},
			model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/main"}, Head: model.Head{Branch: "head"}},
			&config.Config{Defaults: config.Defaults{MainBranch: "cfg-main"}},
		); got != "release" {
			t.Fatalf("expected entry branch to win, got %q", got)
		}

		if got := repairResolveTargetBranch(
			registry.Entry{},
			model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/dev"}, Head: model.Head{Branch: "head"}},
			&config.Config{Defaults: config.Defaults{MainBranch: "cfg-main"}},
		); got != "dev" {
			t.Fatalf("expected upstream-derived branch, got %q", got)
		}

		if got := repairResolveTargetBranch(
			registry.Entry{},
			model.RepoStatus{Head: model.Head{Branch: "head"}},
			&config.Config{Defaults: config.Defaults{MainBranch: "cfg-main"}},
		); got != "cfg-main" {
			t.Fatalf("expected config default branch, got %q", got)
		}

		if got := repairResolveTargetBranch(
			registry.Entry{},
			model.RepoStatus{Head: model.Head{Branch: "head-fallback"}},
			nil,
		); got != "head-fallback" {
			t.Fatalf("expected head branch fallback, got %q", got)
		}
	})

	t.Run("repairNeedsUpstream branches", func(t *testing.T) {
		if repairNeedsUpstream(model.RepoStatus{}, " ") {
			t.Fatal("blank target should not need upstream")
		}
		if repairNeedsUpstream(model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/main", Status: model.TrackingBehind}}, "origin/main") {
			t.Fatal("matching upstream with tracking should not need repair")
		}
		if !repairNeedsUpstream(model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/dev", Status: model.TrackingBehind}}, "origin/main") {
			t.Fatal("different upstream should need repair")
		}
		if !repairNeedsUpstream(model.RepoStatus{Tracking: model.Tracking{Upstream: "origin/main", Status: model.TrackingNone}}, "origin/main") {
			t.Fatal("tracking none should require upstream re-set")
		}
	})
}

func TestActionsResetDeleteCloneAndRegister(t *testing.T) {
	t.Run("ResetRepo guards and success", func(t *testing.T) {
		eng := &Engine{cfg: &config.Config{}, adapter: vcs.NewGitAdapter(&testRunner{})}
		if err := eng.ResetRepo(context.Background(), "repo", ""); err == nil || !strings.Contains(err.Error(), "registry not available") {
			t.Fatalf("expected nil registry error, got %v", err)
		}

		eng.registry = &registry.Registry{}
		if err := eng.ResetRepo(context.Background(), "repo", ""); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected repo not found error, got %v", err)
		}

		eng.registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusMissing}}}
		if err := eng.ResetRepo(context.Background(), "repo", ""); err == nil || !strings.Contains(err.Error(), "path is missing") {
			t.Fatalf("expected missing path error, got %v", err)
		}

		runner := &testRunner{responses: map[string]testResponse{
			"/repo:reset --hard HEAD": {out: ""},
			"/repo:clean -f -d":       {out: ""},
		}}
		eng = &Engine{
			cfg:      &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}}},
			adapter:  vcs.NewGitAdapter(runner),
		}
		if err := eng.ResetRepo(context.Background(), "repo", ""); err != nil {
			t.Fatalf("expected successful reset, got %v", err)
		}
	})

	t.Run("DeleteRepo guards and file deletion modes", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		eng := &Engine{cfg: &config.Config{}, adapter: vcs.NewGitAdapter(&testRunner{})}
		if err := eng.DeleteRepo(context.Background(), "repo", cfgPath, false); err == nil || !strings.Contains(err.Error(), "registry not available") {
			t.Fatalf("expected nil registry error, got %v", err)
		}

		eng.registry = &registry.Registry{}
		if err := eng.DeleteRepo(context.Background(), "repo", cfgPath, false); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected repo not found error, got %v", err)
		}

		keepPath := filepath.Join(t.TempDir(), "keep")
		if err := os.MkdirAll(keepPath, 0o755); err != nil {
			t.Fatalf("mkdir keep path: %v", err)
		}
		eng = &Engine{
			cfg: &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "keep", Path: keepPath, Status: registry.StatusPresent},
			}},
			adapter: vcs.NewGitAdapter(&testRunner{}),
		}
		if err := eng.DeleteRepo(context.Background(), "keep", cfgPath, false); err != nil {
			t.Fatalf("delete repo registry-only: %v", err)
		}
		if eng.registry.FindByRepoID("keep") != nil {
			t.Fatalf("expected repo removed from registry, entries=%+v", eng.registry.Entries)
		}
		if _, err := os.Stat(keepPath); err != nil {
			t.Fatalf("expected files kept on disk, stat err=%v", err)
		}

		deletePath := filepath.Join(t.TempDir(), "delete")
		if err := os.MkdirAll(deletePath, 0o755); err != nil {
			t.Fatalf("mkdir delete path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(deletePath, "file.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write delete file: %v", err)
		}
		eng = &Engine{
			cfg: &config.Config{},
			registry: &registry.Registry{Entries: []registry.Entry{
				{RepoID: "gone", Path: deletePath, Status: registry.StatusPresent},
			}},
			adapter: vcs.NewGitAdapter(&testRunner{}),
		}
		if err := eng.DeleteRepo(context.Background(), "gone", cfgPath, true); err != nil {
			t.Fatalf("delete repo with files: %v", err)
		}
		if _, err := os.Stat(deletePath); !os.IsNotExist(err) {
			t.Fatalf("expected files deleted, stat err=%v", err)
		}
	})

	t.Run("CloneAndRegister success mirror and local fallback repo id", func(t *testing.T) {
		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, "config.yaml")

		targetCheckout := filepath.Join(tmp, "checkout")
		targetMirror := filepath.Join(tmp, "mirror.git")
		targetLocal := filepath.Join(tmp, "local")

		runner := &testRunner{responses: map[string]testResponse{
			":" + strings.Join([]string{"clone", "git@github.com:org/repo.git", targetCheckout}, " "):             {out: ""},
			":" + strings.Join([]string{"clone", "--mirror", "git@github.com:org/mirror.git", targetMirror}, " "): {out: ""},
			":" + strings.Join([]string{"clone", "/", targetLocal}, " "):                                          {out: ""},
		}}

		eng := &Engine{
			cfg:        &config.Config{},
			registry:   &registry.Registry{},
			adapter:    vcs.NewGitAdapter(runner),
			classifier: vcs.NewGitErrorClassifier(),
		}

		if err := eng.CloneAndRegister(context.Background(), "git@github.com:org/repo.git", targetCheckout, cfgPath, false); err != nil {
			t.Fatalf("clone and register checkout: %v", err)
		}
		checkoutEntry := eng.registry.FindByRepoID("github.com/org/repo")
		if checkoutEntry == nil || checkoutEntry.Type != "checkout" || checkoutEntry.Path != targetCheckout {
			t.Fatalf("unexpected checkout entry: %+v", checkoutEntry)
		}

		if err := eng.CloneAndRegister(context.Background(), "git@github.com:org/mirror.git", targetMirror, cfgPath, true); err != nil {
			t.Fatalf("clone and register mirror: %v", err)
		}
		mirrorEntry := eng.registry.FindByRepoID("github.com/org/mirror")
		if mirrorEntry == nil || mirrorEntry.Type != "mirror" || mirrorEntry.Path != targetMirror {
			t.Fatalf("unexpected mirror entry: %+v", mirrorEntry)
		}

		if err := eng.CloneAndRegister(context.Background(), "/", targetLocal, cfgPath, false); err != nil {
			t.Fatalf("clone and register local fallback: %v", err)
		}
		localID := "local:" + filepath.ToSlash(targetLocal)
		localEntry := eng.registry.FindByRepoID(localID)
		if localEntry == nil || localEntry.Path != targetLocal {
			t.Fatalf("expected local fallback repo id entry, got %+v", localEntry)
		}
	})
}

func TestRemoteMismatchWrapperFunctions(t *testing.T) {
	t.Run("ParseRemoteMismatchReconcileMode", func(t *testing.T) {
		for _, raw := range []string{"", "none", "registry", "git"} {
			if _, err := ParseRemoteMismatchReconcileMode(raw); err != nil {
				t.Fatalf("expected mode %q to parse, got %v", raw, err)
			}
		}
		if _, err := ParseRemoteMismatchReconcileMode("bogus"); err == nil {
			t.Fatal("expected invalid reconcile mode error")
		}
	})

	t.Run("BuildRemoteMismatchPlans empty and mismatches", func(t *testing.T) {
		eng := &Engine{registry: &registry.Registry{}, adapter: vcs.NewGitAdapter(nil)}
		if plans := eng.BuildRemoteMismatchPlans(nil, RemoteMismatchReconcileRegistry); len(plans) != 0 {
			t.Fatalf("expected no plans for empty repos, got %+v", plans)
		}

		eng.registry.Entries = []registry.Entry{
			{RepoID: "github.com/org/mismatch", Path: "/repos/mismatch", RemoteURL: "git@github.com:other/mismatch.git"},
			{RepoID: "github.com/org/match", Path: "/repos/match", RemoteURL: "git@github.com:org/match.git"},
		}
		repos := []model.RepoStatus{
			{
				RepoID:        "github.com/org/mismatch",
				Path:          "/repos/mismatch",
				PrimaryRemote: "origin",
				Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/mismatch.git"}},
			},
			{
				RepoID:        "github.com/org/match",
				Path:          "/repos/match",
				PrimaryRemote: "origin",
				Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/match.git"}},
			},
		}
		plans := eng.BuildRemoteMismatchPlans(repos, RemoteMismatchReconcileRegistry)
		if len(plans) != 1 {
			t.Fatalf("expected one mismatch plan, got %+v", plans)
		}
		if plans[0].RepoID != "github.com/org/mismatch" || plans[0].Action == "" {
			t.Fatalf("unexpected mismatch plan: %+v", plans[0])
		}
	})

	t.Run("ApplyRemoteMismatchPlans updates registry", func(t *testing.T) {
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", RemoteURL: "git@github.com:old/repo.git"}}}
		eng := &Engine{registry: reg, adapter: vcs.NewGitAdapter(nil)}
		plans := []RemoteMismatchPlan{{
			RepoID:        "repo",
			Path:          "/repo",
			RepoRemoteURL: "git@github.com:new/repo.git",
			RegistryURL:   "git@github.com:old/repo.git",
			EntryIndex:    0,
		}}
		if err := eng.ApplyRemoteMismatchPlans(context.Background(), plans, RemoteMismatchReconcileRegistry); err != nil {
			t.Fatalf("apply remote mismatch plans: %v", err)
		}
		if got := reg.Entries[0].RemoteURL; got != "git@github.com:new/repo.git" {
			t.Fatalf("expected registry remote URL updated, got %q", got)
		}
	})
}

func pathCleanCanonical(path string) string {
	return filepath.Clean(path)
}
