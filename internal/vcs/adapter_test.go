package vcs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/vcs"
)

type runnerStub struct {
	responses map[string]struct {
		out string
		err error
	}
}

func (r *runnerStub) Run(_ context.Context, dir string, args ...string) (string, error) {
	key := dir + ":"
	for i, a := range args {
		if i > 0 {
			key += " "
		}
		key += a
	}
	if resp, ok := r.responses[key]; ok {
		return resp.out, resp.err
	}
	return "", errors.New("unexpected")
}

func TestGitAdapterMethods(t *testing.T) {
	r := &runnerStub{responses: map[string]struct {
		out string
		err error
	}{
		"/repo:rev-parse --is-inside-work-tree":   {out: "true"},
		"/repo:rev-parse --is-bare-repository":    {out: "false"},
		"/repo:remote":                            {out: "origin"},
		"/repo:remote get-url origin":             {out: "git@github.com:Org/Repo.git"},
		"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
		"/repo:status --porcelain=v1":             {out: "M  file.go"},
		"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {out: "main|origin/main||="},
		"/repo:rev-list --left-right --count main...origin/main":                                                             {out: "0\t0"},
		"/repo:config --file .gitmodules --get-regexp submodule":                                                             {out: "submodule.foo.path foo"},
		"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules":                    {out: ""},
		"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules":                                       {out: ""},
		"/repo:push": {out: ""},
		"/repo:branch --set-upstream-to origin/main main":         {out: ""},
		"/repo:remote set-url origin git@github.com:org/repo.git": {out: ""},
		"/repo:stash push -u -m repokeeper: pre-rebase stash":     {out: "Saved working directory and index state"},
		"/repo:stash pop": {out: ""},
		":clone --branch main --single-branch git@github.com:Org/Repo.git /tmp/repo": {out: ""},
	}}
	a := vcs.NewGitAdapter(r)
	if a.Name() != "git" {
		t.Fatalf("unexpected adapter name: %s", a.Name())
	}
	if ok, _ := a.IsRepo(context.Background(), "/repo"); !ok {
		t.Fatal("expected IsRepo true")
	}
	if bare, _ := a.IsBare(context.Background(), "/repo"); bare {
		t.Fatal("expected non-bare")
	}
	if remotes, err := a.Remotes(context.Background(), "/repo"); err != nil || len(remotes) != 1 {
		t.Fatalf("unexpected remotes: %v %#v", err, remotes)
	}
	if head, err := a.Head(context.Background(), "/repo"); err != nil || head.Branch != "main" {
		t.Fatalf("unexpected head: %v %#v", err, head)
	}
	if wt, err := a.WorktreeStatus(context.Background(), "/repo"); err != nil || !wt.Dirty {
		t.Fatalf("unexpected worktree: %v %#v", err, wt)
	}
	if tr, err := a.TrackingStatus(context.Background(), "/repo"); err != nil || tr.Status == "" {
		t.Fatalf("unexpected tracking: %v %#v", err, tr)
	}
	if has, err := a.HasSubmodules(context.Background(), "/repo"); err != nil || !has {
		t.Fatalf("unexpected submodules: %v %v", err, has)
	}
	if err := a.Fetch(context.Background(), "/repo"); err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if err := a.PullRebase(context.Background(), "/repo"); err != nil {
		t.Fatalf("unexpected pull rebase error: %v", err)
	}
	if err := a.Push(context.Background(), "/repo"); err != nil {
		t.Fatalf("unexpected push error: %v", err)
	}
	if err := a.SetUpstream(context.Background(), "/repo", "origin/main", "main"); err != nil {
		t.Fatalf("unexpected set upstream error: %v", err)
	}
	if err := a.SetRemoteURL(context.Background(), "/repo", "origin", "git@github.com:org/repo.git"); err != nil {
		t.Fatalf("unexpected set remote url error: %v", err)
	}
	if stashed, err := a.StashPush(context.Background(), "/repo", "repokeeper: pre-rebase stash"); err != nil || !stashed {
		t.Fatalf("unexpected stash push result: stashed=%v err=%v", stashed, err)
	}
	if err := a.StashPop(context.Background(), "/repo"); err != nil {
		t.Fatalf("unexpected stash pop error: %v", err)
	}
	if err := a.Clone(context.Background(), "git@github.com:Org/Repo.git", "/tmp/repo", "main", false); err != nil {
		t.Fatalf("unexpected clone error: %v", err)
	}
	if got := a.NormalizeURL("git@github.com:Org/Repo.git"); got == "" {
		t.Fatal("expected normalized url")
	}
	if got := a.PrimaryRemote([]string{"upstream", "origin"}); got != "origin" {
		t.Fatalf("unexpected primary remote: %s", got)
	}
}

func TestNewGitAdapterDefaultsRunnerAndCloneErrors(t *testing.T) {
	a := vcs.NewGitAdapter(nil)
	if a == nil {
		t.Fatal("expected adapter")
	}

	r := &runnerStub{responses: map[string]struct {
		out string
		err error
	}{
		":clone git@github.com:org/repo.git /tmp/repo": {err: errors.New("clone failed")},
	}}
	a = vcs.NewGitAdapter(r)
	if err := a.Clone(context.Background(), "git@github.com:org/repo.git", "/tmp/repo", "", false); err == nil {
		t.Fatal("expected clone error")
	}
}
