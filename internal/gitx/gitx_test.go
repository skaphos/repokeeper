// SPDX-License-Identifier: MIT
package gitx_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
)

// writeFakeBin writes an executable POSIX shell script to a temp dir and
// returns its path. Used to test GitRunner.Run's process plumbing (env,
// stdout/stderr separation) without depending on real git's exact output.
func writeFakeBin(script string) string {
	if runtime.GOOS == "windows" {
		Skip("fake git script uses POSIX shell")
	}
	tmpDir, err := os.MkdirTemp("", "gitx-fakebin")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })
	binPath := filepath.Join(tmpDir, "fake-git")
	Expect(os.WriteFile(binPath, []byte(script), 0o755)).To(Succeed())
	return binPath
}

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

	It("does not merge stderr into stdout on a successful (exit-0) command", func() {
		// Regression test: an exit-0 warning on stderr (e.g. an unknown
		// gitconfig key) must not corrupt stdout, since callers like
		// IsRepo/IsBare/ParsePorcelainStatus parse stdout as
		// machine-readable output.
		script := "#!/usr/bin/env sh\n" +
			"echo 'true'\n" +
			"echo 'warning: unknown config key foo.bar' 1>&2\n" +
			"exit 0\n"
		fakeGit := writeFakeBin(script)

		fakeRunner := &gitx.GitRunner{GitBin: fakeGit}
		out, err := fakeRunner.Run(context.Background(), "", "rev-parse", "--is-inside-work-tree")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("true"))
		Expect(out).NotTo(ContainSubstring("warning"))
	})

	It("folds stderr text into the returned error without polluting stdout", func() {
		script := "#!/usr/bin/env sh\n" +
			"echo 'partial-stdout'\n" +
			"echo 'fatal: boom' 1>&2\n" +
			"exit 1\n"
		fakeGit := writeFakeBin(script)

		fakeRunner := &gitx.GitRunner{GitBin: fakeGit}
		out, err := fakeRunner.Run(context.Background(), "", "status")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fatal: boom"))
		Expect(out).NotTo(ContainSubstring("fatal: boom"))
	})

	It("forces the C locale regardless of the parent process's locale", func() {
		// Regression test: ClassifyError string-matches stderr text, so
		// git must always be run with a stable, untranslated locale.
		script := "#!/usr/bin/env sh\n" +
			"echo \"LC_ALL=$LC_ALL LANG=$LANG\"\n" +
			"exit 0\n"
		fakeGit := writeFakeBin(script)

		prevLCAll, hadLCAll := os.LookupEnv("LC_ALL")
		prevLang, hadLang := os.LookupEnv("LANG")
		Expect(os.Setenv("LC_ALL", "fr_FR.UTF-8")).To(Succeed())
		Expect(os.Setenv("LANG", "fr_FR.UTF-8")).To(Succeed())
		DeferCleanup(func() {
			if hadLCAll {
				_ = os.Setenv("LC_ALL", prevLCAll)
			} else {
				_ = os.Unsetenv("LC_ALL")
			}
			if hadLang {
				_ = os.Setenv("LANG", prevLang)
			} else {
				_ = os.Unsetenv("LANG")
			}
		})

		fakeRunner := &gitx.GitRunner{GitBin: fakeGit}
		out, err := fakeRunner.Run(context.Background(), "", "version")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("LC_ALL=C LANG=C"))
	})
})

var _ = Describe("IsRepo", func() {
	It("returns true for a valid repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Output: "true"},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("returns false on error", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Err: errors.New("not a repo")},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("returns false when output is not 'true'", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-inside-work-tree": {Output: "false"},
		}}
		ok, err := gitx.IsRepo(context.Background(), mock, "/repo", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("IsBare", func() {
	It("returns true for a bare repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-bare-repository": {Output: "true"},
		}}
		ok, err := gitx.IsBare(context.Background(), mock, "/repo", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("returns false for a non-bare repo", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:rev-parse --is-bare-repository": {Output: "false"},
		}}
		ok, err := gitx.IsBare(context.Background(), mock, "/repo", nil)
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
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo", nil)
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
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(remotes).To(BeNil())
	})

	It("skips remotes whose URL cannot be fetched", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote":                {Output: "origin\nbad"},
			"/repo:remote get-url origin": {Output: "https://github.com/org/repo.git"},
			"/repo:remote get-url bad":    {Err: errors.New("no such remote")},
		}}
		remotes, err := gitx.Remotes(context.Background(), mock, "/repo", nil)
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

var _ = Describe("StaleRemoteTrackingRefs", func() {
	It("returns sorted, deduplicated refs that a remote prune would remove", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote prune --dry-run -- origin": {
				Output: "Pruning origin\nURL: git@example.com:org/repo.git\n * [would prune] origin/zeta\n * [would prune] origin/alpha",
			},
			"/repo:remote prune --dry-run -- upstream": {
				Output: "Pruning upstream\n * [would prune] origin/alpha\n * [would prune] upstream/old",
			},
		}}

		refs, err := gitx.StaleRemoteTrackingRefs(context.Background(), mock, "/repo", []string{"origin", "upstream"})
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).To(Equal([]string{"origin/alpha", "origin/zeta", "upstream/old"}))
	})

	It("returns contextual errors without partial results", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:remote prune --dry-run -- origin": {Err: errors.New("network unavailable")},
		}}

		refs, err := gitx.StaleRemoteTrackingRefs(context.Background(), mock, "/repo", []string{"origin"})
		Expect(err).To(MatchError(ContainSubstring(`git remote prune --dry-run "origin"`)))
		Expect(refs).To(BeNil())
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

var _ = Describe("PullRebase", func() {
	It("runs pull --rebase with correct args", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {Output: ""},
		}}
		err := gitx.PullRebase(context.Background(), mock, "/repo")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error on pull failure", func() {
		mock := &MockRunner{Responses: map[string]MockResponse{
			"/repo:-c fetch.recurseSubmodules=false pull --rebase --no-recurse-submodules": {Err: errors.New("pull failed")},
		}}
		err := gitx.PullRebase(context.Background(), mock, "/repo")
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

		ok, err := gitx.IsRepo(ctx, runner, tmpDir, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())

		bare, err := gitx.IsBare(ctx, runner, tmpDir, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(bare).To(BeFalse())
	})

	It("detects a bare repo", func() {
		runner := &gitx.GitRunner{}
		ctx := context.Background()

		bareDir := filepath.Join(tmpDir, "bare.git")
		_, err := runner.Run(ctx, "", "init", "--bare", bareDir)
		Expect(err).NotTo(HaveOccurred())

		bare, err := gitx.IsBare(ctx, runner, bareDir, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(bare).To(BeTrue())
	})
})
