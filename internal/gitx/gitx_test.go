package gitx_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mfacenet/repokeeper/internal/gitx"
	"github.com/mfacenet/repokeeper/internal/model"
)

var _ = Describe("GitRunner.Run", func() {
	var runner *gitx.GitRunner

	BeforeEach(func() {
		runner = &gitx.GitRunner{}
	})

	It("runs git version successfully", func() {
		out, err := runner.Run(context.Background(), "", "version")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("git version"))
	})

	It("errors for nonexistent directory", func() {
		_, err := runner.Run(context.Background(), "/nonexistent/path/xyz", "status")
		Expect(err).To(HaveOccurred())
	})

	It("respects context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := runner.Run(ctx, "", "version")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("IsRepo", func() {
	It("returns true for a valid repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Output: "true"},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("returns false on error", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Err: errors.New("not a repo")},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("returns false when output is not 'true'", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Output: "false"},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("IsBare", func() {
	It("returns true for a bare repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-bare-repository": {Output: "true"},
		}}
		ok, err := gitx.IsBare(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("returns false for a non-bare repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-bare-repository": {Output: "false"},
		}}
		ok, err := gitx.IsBare(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("Head", func() {
	It("returns branch name for attached HEAD", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:symbolic-ref --quiet --short HEAD": {Output: "main"},
		}}
		h, err := gitx.Head(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(h.Branch).To(Equal("main"))
		Expect(h.Detached).To(BeFalse())
	})

	It("returns commit hash for detached HEAD", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:symbolic-ref --quiet --short HEAD": {Err: errors.New("not symbolic")},
			"/repo:rev-parse --short HEAD":            {Output: "abc1234"},
		}}
		h, err := gitx.Head(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(h.Branch).To(Equal("abc1234"))
		Expect(h.Detached).To(BeTrue())
	})

	It("returns detached with empty branch when no commit", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:symbolic-ref --quiet --short HEAD": {Err: errors.New("not symbolic")},
			"/repo:rev-parse --short HEAD":            {Err: errors.New("no HEAD")},
		}}
		h, err := gitx.Head(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(h.Detached).To(BeTrue())
		Expect(h.Branch).To(Equal(""))
	})
})

var _ = Describe("Remotes", func() {
	It("returns all remotes with URLs", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote":                  {Output: "origin\nupstream"},
			"/repo:remote get-url origin":   {Output: "https://github.com/org/repo.git"},
			"/repo:remote get-url upstream": {Output: "https://github.com/other/repo.git"},
		}}
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(remotes).To(Equal([]model.Remote{
			{Name: "origin", URL: "https://github.com/org/repo.git"},
			{Name: "upstream", URL: "https://github.com/other/repo.git"},
		}))
	})

	It("returns nil for no remotes", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote": {Output: ""},
		}}
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(remotes).To(BeNil())
	})

	It("skips remotes whose URL cannot be fetched", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote":                {Output: "origin\nbad"},
			"/repo:remote get-url origin": {Output: "https://github.com/org/repo.git"},
			"/repo:remote get-url bad":    {Err: errors.New("no such remote")},
		}}
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(remotes).To(HaveLen(1))
		Expect(remotes[0].Name).To(Equal("origin"))
	})
})

var _ = Describe("WorktreeStatus", func() {
	It("returns parsed worktree status", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:status --porcelain=v1": {Output: "M  file.go\n?? new.go\n"},
		}}
		wt, err := gitx.WorktreeStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(wt.Staged).To(Equal(1))
		Expect(wt.Untracked).To(Equal(1))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("returns clean for empty status", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:status --porcelain=v1": {Output: ""},
		}}
		wt, err := gitx.WorktreeStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(wt.Dirty).To(BeFalse())
	})
})

var _ = Describe("TrackingStatus", func() {
	It("returns tracking ahead", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				Output: "main|origin/main|[ahead 2]|>",
			},
			"/repo:symbolic-ref --quiet --short HEAD":                {Output: "main"},
			"/repo:rev-list --left-right --count main...origin/main": {Output: "2\t0"},
		}}
		t, err := gitx.TrackingStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Status).To(Equal(model.TrackingAhead))
		Expect(*t.Ahead).To(Equal(2))
		Expect(*t.Behind).To(Equal(0))
	})

	It("returns gone for gone upstream", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				Output: "main|origin/main|[gone]|",
			},
			"/repo:symbolic-ref --quiet --short HEAD": {Output: "main"},
		}}
		t, err := gitx.TrackingStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Status).To(Equal(model.TrackingGone))
	})

	It("returns none when no upstream", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				Output: "feature|||",
			},
			"/repo:symbolic-ref --quiet --short HEAD": {Output: "feature"},
		}}
		t, err := gitx.TrackingStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Status).To(Equal(model.TrackingNone))
	})

	It("returns none on detached HEAD", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:for-each-ref --format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort) refs/heads": {
				Output: "main|origin/main||=",
			},
			"/repo:symbolic-ref --quiet --short HEAD": {Err: errors.New("detached")},
		}}
		t, err := gitx.TrackingStatus(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(t.Status).To(Equal(model.TrackingNone))
	})
})

var _ = Describe("HasSubmodules", func() {
	It("returns true when submodules exist", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:config --file .gitmodules --get-regexp submodule": {Output: "submodule.foo.path foo"},
		}}
		has, err := gitx.HasSubmodules(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(has).To(BeTrue())
	})

	It("returns false when no submodules", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:config --file .gitmodules --get-regexp submodule": {Err: errors.New("no .gitmodules")},
		}}
		has, err := gitx.HasSubmodules(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(has).To(BeFalse())
	})
})

var _ = Describe("Fetch", func() {
	It("runs fetch with correct args", func() {
		var calledArgs string
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {Output: ""},
		}}
		err := gitx.Fetch(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
		_ = calledArgs
	})

	It("returns error on fetch failure", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:-c fetch.recurseSubmodules=false fetch --all --prune --prune-tags --no-recurse-submodules": {Err: errors.New("fetch failed")},
		}}
		err := gitx.Fetch(context.Background(), mock, "/repo")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("GitRunner with real git", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "gitx-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("detects a real git repo", func() {
		runner := &gitx.GitRunner{}
		ctx := context.Background()

		// Init a repo
		_, err := runner.Run(ctx, tmpDir, "init")
		Expect(err).NotTo(HaveOccurred())

		ok, err := gitx.IsRepo(ctx, runner, tmpDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		bare, err := gitx.IsBare(ctx, runner, tmpDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(bare).To(BeFalse())
	})

	It("detects a bare repo", func() {
		runner := &gitx.GitRunner{}
		ctx := context.Background()

		bareDir := filepath.Join(tmpDir, "bare.git")
		_, err := runner.Run(ctx, "", "init", "--bare", bareDir)
		Expect(err).NotTo(HaveOccurred())

		bare, err := gitx.IsBare(ctx, runner, bareDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(bare).To(BeTrue())
	})
})
