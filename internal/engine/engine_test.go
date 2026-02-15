package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

type mockRunner struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	out string
	err error
}

func (m *mockRunner) Run(_ context.Context, dir string, args ...string) (string, error) {
	key := dir + ":" + joinArgs(args)
	if resp, ok := m.responses[key]; ok {
		return resp.out, resp.err
	}
	return "", errors.New("unexpected call")
}

func joinArgs(args []string) string {
	out := ""
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}

type blockingRunner struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingRunner) Run(ctx context.Context, _ string, _ ...string) (string, error) {
	select {
	case b.started <- struct{}{}:
	default:
	}
	select {
	case <-b.release:
		return "", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

var _ = Describe("Engine", func() {
	It("inspects repo status", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo:rev-parse --is-bare-repository":    {out: "false"},
			"/repo:remote":                            {out: "origin"},
			"/repo:remote get-url origin":             {out: "git@github.com:org/repo.git"},
			"/repo:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo:status --porcelain=v1":             {out: "M  file.go\n"},
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main||=",
			},
			"/repo:rev-list --left-right --count main...origin/main": {out: "0\t0"},
			"/repo:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
		}}
		eng := engine.New(&config.Config{}, &registry.Registry{}, vcs.NewGitAdapter(runner))
		status, err := eng.InspectRepo(context.Background(), "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(status.RepoID).To(Equal("github.com/org/repo"))
		Expect(status.Worktree).NotTo(BeNil())
		Expect(status.Worktree.Dirty).To(BeTrue())
	})

	It("syncs repositories with dry-run", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			DryRun:      true,
			Concurrency: 1,
			Timeout:     1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Error).To(Equal("dry-run"))
	})

	It("skips sync for repos without upstream remote", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			DryRun:      true,
			Concurrency: 1,
			Timeout:     1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].ErrorClass).To(Equal("skipped"))
		Expect(results[0].Error).To(Equal("skipped-no-upstream"))
	})

	It("prunes missing entries for sync filter", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "missing", Path: filepath.Join("C:", "missing"), Status: registry.StatusMissing, LastSeen: time.Now().Add(-48 * time.Hour)},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			Filter: engine.FilterMissing,
			DryRun: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].RepoID).To(Equal("missing"))
		Expect(results[0].OK).To(BeFalse())
	})

	It("checks out missing entries when enabled", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			":clone --branch main --single-branch git@github.com:org/missing.git /missing": {out: ""},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{
					RepoID:    "missing",
					Path:      "/missing",
					RemoteURL: "git@github.com:org/missing.git",
					Branch:    "main",
					Status:    registry.StatusMissing,
				},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(runner))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			Filter:          engine.FilterMissing,
			CheckoutMissing: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Action).To(ContainSubstring("git clone"))
	})

	It("filters dirty and clean during sync", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: ""},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: "M  file.go\n"},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main||=",
			},
			"/repo1:rev-list --left-right --count main...origin/main": {out: "0\t0"},
			"/repo1:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
			"/repo2:rev-parse --is-bare-repository":                   {out: "false"},
			"/repo2:remote":                                           {out: ""},
			"/repo2:symbolic-ref --quiet --short HEAD":                {out: "main"},
			"/repo2:status --porcelain=v1":                            {out: ""},
			"/repo2:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main||=",
			},
			"/repo2:rev-list --left-right --count main...origin/main": {out: "0\t0"},
			"/repo2:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
				{RepoID: "repo2", Path: "/repo2", RemoteURL: "git@github.com:org/repo2.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 2}}, reg, vcs.NewGitAdapter(runner))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			Filter:      engine.FilterDirty,
			DryRun:      true,
			Concurrency: 2,
			Timeout:     1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
	})

	It("respects concurrency by not exceeding it", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
				{RepoID: "repo2", Path: "/repo2", RemoteURL: "git@github.com:org/repo2.git", Status: registry.StatusPresent},
				{RepoID: "repo3", Path: "/repo3", RemoteURL: "git@github.com:org/repo3.git", Status: registry.StatusPresent},
			},
		}
		blocker := &blockingRunner{
			started: make(chan struct{}, 3),
			release: make(chan struct{}),
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 2, Concurrency: 1}}, reg, vcs.NewGitAdapter(blocker))

		done := make(chan []engine.SyncResult, 1)
		go func() {
			results, _ := eng.Sync(context.Background(), engine.SyncOptions{
				Concurrency: 1,
				Timeout:     2,
			})
			done <- results
		}()

		<-blocker.started
		select {
		case <-blocker.started:
			Fail("sync exceeded concurrency limit")
		case <-time.After(200 * time.Millisecond):
		}

		close(blocker.release)
		results := <-done
		Expect(results).To(HaveLen(3))
	})

	It("times out long-running git operations", func() {
		blocker := &blockingRunner{
			started: make(chan struct{}, 1),
			release: make(chan struct{}),
		}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(blocker))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{
			Concurrency: 1,
			Timeout:     1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeFalse())
	})

	It("produces accurate status json across many repos", func() {
		responses := map[string]mockResponse{}
		reg := &registry.Registry{}

		for i := 1; i <= 12; i++ {
			repoPath := fmt.Sprintf("/repo%d", i)
			repoID := fmt.Sprintf("repo%d", i)
			reg.Entries = append(reg.Entries, registry.Entry{
				RepoID:   repoID,
				Path:     repoPath,
				Status:   registry.StatusPresent,
				LastSeen: time.Now(),
			})
			responses[repoPath+":rev-parse --is-bare-repository"] = mockResponse{out: "false"}
			responses[repoPath+":remote"] = mockResponse{out: "origin"}
			responses[repoPath+":remote get-url origin"] = mockResponse{out: fmt.Sprintf("git@github.com:org/repo%d.git", i)}
			responses[repoPath+":symbolic-ref --quiet --short HEAD"] = mockResponse{out: "main"}
			responses[repoPath+":status --porcelain=v1"] = mockResponse{out: ""}
			responses[repoPath+":for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads"] = mockResponse{
				out: "main|origin/main||=",
			}
			responses[repoPath+":rev-list --left-right --count main...origin/main"] = mockResponse{out: "0\t0"}
			responses[repoPath+":config --file .gitmodules --get-regexp submodule"] = mockResponse{err: errors.New("none")}
		}

		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 4}}, reg, vcs.NewGitAdapter(&mockRunner{responses: responses}))
		report, err := eng.Status(context.Background(), engine.StatusOptions{Filter: engine.FilterAll, Concurrency: 4, Timeout: 1})
		Expect(err).NotTo(HaveOccurred())
		Expect(report.Repos).To(HaveLen(12))

		data, err := json.Marshal(report)
		Expect(err).NotTo(HaveOccurred())

		var decoded map[string]any
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		repos, ok := decoded["repos"].([]any)
		Expect(ok).To(BeTrue())
		Expect(repos).To(HaveLen(12))
	})

	It("surfaces inspect errors in status and supports --only errors", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: "origin"},
			"/repo1:remote get-url origin":             {out: "git@github.com:org/repo1.git"},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: ""},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main||=",
			},
			"/repo1:rev-list --left-right --count main...origin/main": {out: "0\t0"},
			"/repo1:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},

			"/repo2:rev-parse --is-bare-repository": {out: "false"},
			"/repo2:remote":                         {err: errors.New("permission denied")},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo2", Path: "/repo2", Status: registry.StatusPresent},
				{RepoID: "repo1", Path: "/repo1", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 2}}, reg, vcs.NewGitAdapter(runner))

		all, err := eng.Status(context.Background(), engine.StatusOptions{Filter: engine.FilterAll, Concurrency: 2, Timeout: 1})
		Expect(err).NotTo(HaveOccurred())
		Expect(all.Repos).To(HaveLen(2))
		Expect(all.Repos[0].RepoID).To(Equal("github.com/org/repo1"))
		Expect(all.Repos[1].RepoID).To(Equal("repo2"))
		Expect(all.Repos[1].Error).To(ContainSubstring("permission denied"))
		Expect(all.Repos[1].ErrorClass).To(Equal("auth"))

		onlyErrors, err := eng.Status(context.Background(), engine.StatusOptions{Filter: engine.FilterErrors, Concurrency: 2, Timeout: 1})
		Expect(err).NotTo(HaveOccurred())
		Expect(onlyErrors.Repos).To(HaveLen(1))
		Expect(onlyErrors.Repos[0].RepoID).To(Equal("repo2"))
	})

	It("classifies sync fetch failures", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		failing := &mockRunner{responses: map[string]mockResponse{
			"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {err: errors.New("could not resolve host")},
		}}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(failing))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeFalse())
		Expect(results[0].ErrorClass).To(Equal("network"))
	})

	It("runs local rebase update when enabled and repo is behind main", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: "origin"},
			"/repo1:remote get-url origin":             {out: "git@github.com:org/repo1.git"},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: ""},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main|[behind 1]|<",
			},
			"/repo1:rev-list --left-right --count main...origin/main":                       {out: "0\t1"},
			"/repo1:config --file .gitmodules --get-regexp submodule":                       {err: errors.New("none")},
			"/repo1:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {out: ""},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(runner))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1, UpdateLocal: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Action).To(Equal("git pull --rebase --no-recurse-submodules"))
	})

	It("skips local rebase update when branch has local commits", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: "origin"},
			"/repo1:remote get-url origin":             {out: "git@github.com:org/repo1.git"},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: ""},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main|[ahead 1]|>",
			},
			"/repo1:rev-list --left-right --count main...origin/main": {out: "1\t0"},
			"/repo1:config --file .gitmodules --get-regexp submodule": {err: errors.New("none")},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(runner))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1, UpdateLocal: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Error).To(ContainSubstring("skipped-local-update"))
	})

	It("stashes dirty changes before local rebase when enabled", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: "origin"},
			"/repo1:remote get-url origin":             {out: "git@github.com:org/repo1.git"},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: "M  file.go"},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main|[behind 1]|<",
			},
			"/repo1:rev-list --left-right --count main...origin/main":                       {out: "0\t1"},
			"/repo1:config --file .gitmodules --get-regexp submodule":                       {err: errors.New("none")},
			"/repo1:stash push -u -m repokeeper: pre-rebase stash":                          {out: "Saved working directory and index state"},
			"/repo1:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {out: ""},
			"/repo1:stash pop": {out: "Applied stash"},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(runner))
		results, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1, UpdateLocal: true, RebaseDirty: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Action).To(ContainSubstring("git stash push"))
		Expect(results[0].Action).To(ContainSubstring("git stash pop"))
	})

	It("skips diverged local rebase unless force is set", func() {
		runner := &mockRunner{responses: map[string]mockResponse{
			"/repo1:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {out: ""},
			"/repo1:rev-parse --is-bare-repository":    {out: "false"},
			"/repo1:remote":                            {out: "origin"},
			"/repo1:remote get-url origin":             {out: "git@github.com:org/repo1.git"},
			"/repo1:symbolic-ref --quiet --short HEAD": {out: "main"},
			"/repo1:status --porcelain=v1":             {out: ""},
			"/repo1:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				out: "main|origin/main|[ahead 1, behind 1]|<>",
			},
			"/repo1:rev-list --left-right --count main...origin/main":                       {out: "1\t1"},
			"/repo1:config --file .gitmodules --get-regexp submodule":                       {err: errors.New("none")},
			"/repo1:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {out: ""},
		}}
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/repo1", RemoteURL: "git@github.com:org/repo1.git", Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 1, Concurrency: 1}}, reg, vcs.NewGitAdapter(runner))

		results, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1, UpdateLocal: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].OK).To(BeTrue())
		Expect(results[0].Error).To(ContainSubstring("skipped-local-update: branch has diverged"))

		forced, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 1, UpdateLocal: true, Force: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(forced).To(HaveLen(1))
		Expect(forced[0].OK).To(BeTrue())
		Expect(forced[0].Action).To(Equal("git pull --rebase --no-recurse-submodules"))
	})
})
