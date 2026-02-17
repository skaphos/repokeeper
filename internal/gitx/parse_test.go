// SPDX-License-Identifier: MIT
package gitx_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/gitx"
)

var _ = Describe("ParsePorcelainStatus", func() {
	It("returns clean worktree for empty output", func() {
		wt := gitx.ParsePorcelainStatus("")
		Expect(wt.Dirty).To(BeFalse())
		Expect(wt.Staged).To(Equal(0))
		Expect(wt.Unstaged).To(Equal(0))
		Expect(wt.Untracked).To(Equal(0))
	})

	It("counts staged files", func() {
		output := "M  file1.go\nA  file2.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(2))
		Expect(wt.Unstaged).To(Equal(0))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("counts unstaged files", func() {
		output := " M file1.go\n D file2.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Unstaged).To(Equal(2))
		Expect(wt.Staged).To(Equal(0))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("counts untracked files", func() {
		output := "?? new_file.go\n?? other.txt\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Untracked).To(Equal(2))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("handles mixed status", func() {
		output := "M  staged.go\n M unstaged.go\n?? untracked.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(1))
		Expect(wt.Unstaged).To(Equal(1))
		Expect(wt.Untracked).To(Equal(1))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("handles both staged and unstaged on same file", func() {
		output := "MM both.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(1))
		Expect(wt.Unstaged).To(Equal(1))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("handles renamed files", func() {
		output := "R  old.go -> new.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(1))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("handles deleted files", func() {
		output := "D  deleted.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(1))
		Expect(wt.Dirty).To(BeTrue())
	})

	It("handles added files", func() {
		output := "A  added.go\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Staged).To(Equal(1))
	})

	It("skips blank lines", func() {
		output := "\n\n"
		wt := gitx.ParsePorcelainStatus(output)
		Expect(wt.Dirty).To(BeFalse())
	})
})

var _ = Describe("ParseForEachRef", func() {
	It("parses ahead entry", func() {
		output := "main|origin/main|[ahead 2]|>"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Branch).To(Equal("main"))
		Expect(entries[0].Upstream).To(Equal("origin/main"))
		Expect(entries[0].Track).To(Equal("[ahead 2]"))
		Expect(entries[0].TrackShort).To(Equal(">"))
	})

	It("parses behind entry", func() {
		output := "main|origin/main|[behind 1]|<"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Track).To(Equal("[behind 1]"))
		Expect(entries[0].TrackShort).To(Equal("<"))
	})

	It("parses diverged entry", func() {
		output := "main|origin/main|[ahead 2, behind 1]|<>"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].TrackShort).To(Equal("<>"))
	})

	It("parses equal entry", func() {
		output := "main|origin/main||="
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Track).To(Equal(""))
		Expect(entries[0].TrackShort).To(Equal("="))
	})

	It("parses gone entry", func() {
		output := "main|origin/main|[gone]|"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Track).To(Equal("[gone]"))
	})

	It("parses no upstream", func() {
		output := "feature|||"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Branch).To(Equal("feature"))
		Expect(entries[0].Upstream).To(Equal(""))
	})

	It("parses multiple lines", func() {
		output := "main|origin/main||=\nfeature|origin/feature|[ahead 1]|>"
		entries := gitx.ParseForEachRef(output)
		Expect(entries).To(HaveLen(2))
	})

	It("returns empty for empty output", func() {
		entries := gitx.ParseForEachRef("")
		Expect(entries).To(BeEmpty())
	})
})

var _ = Describe("ParseRevListCount", func() {
	It("parses normal counts", func() {
		ahead, behind := gitx.ParseRevListCount("2\t3")
		Expect(ahead).To(Equal(2))
		Expect(behind).To(Equal(3))
	})

	It("parses zeros", func() {
		ahead, behind := gitx.ParseRevListCount("0\t0")
		Expect(ahead).To(Equal(0))
		Expect(behind).To(Equal(0))
	})

	It("handles empty string", func() {
		ahead, behind := gitx.ParseRevListCount("")
		Expect(ahead).To(Equal(0))
		Expect(behind).To(Equal(0))
	})

	It("handles whitespace", func() {
		ahead, behind := gitx.ParseRevListCount("5\t10\n")
		Expect(ahead).To(Equal(5))
		Expect(behind).To(Equal(10))
	})
})

var _ = Describe("ClassifyGitError", func() {
	It("detects auth errors", func() {
		Expect(gitx.ClassifyGitError("fatal: Authentication failed")).To(Equal(gitx.ErrAuth))
	})

	It("detects permission denied", func() {
		Expect(gitx.ClassifyGitError("Permission denied (publickey)")).To(Equal(gitx.ErrAuth))
	})

	It("detects network errors", func() {
		Expect(gitx.ClassifyGitError("fatal: unable to access: Could not resolve host")).To(Equal(gitx.ErrNetwork))
	})

	It("detects connection refused", func() {
		Expect(gitx.ClassifyGitError("fatal: unable to connect: Connection refused")).To(Equal(gitx.ErrNetwork))
	})

	It("detects no remote", func() {
		Expect(gitx.ClassifyGitError("fatal: No remote repository specified")).To(Equal(gitx.ErrNoRemote))
	})

	It("detects no such remote", func() {
		Expect(gitx.ClassifyGitError("fatal: No such remote 'origin'")).To(Equal(gitx.ErrNoRemote))
	})

	It("detects corrupt repo", func() {
		Expect(gitx.ClassifyGitError("error: object file is empty")).To(Equal(gitx.ErrCorrupt))
	})

	It("detects not a repo", func() {
		Expect(gitx.ClassifyGitError("fatal: not a git repository")).To(Equal(gitx.ErrNotARepo))
	})

	It("detects timeout", func() {
		Expect(gitx.ClassifyGitError("context deadline exceeded")).To(Equal(gitx.ErrTimeout))
	})

	It("returns unknown for unrecognized", func() {
		Expect(gitx.ClassifyGitError("some random error")).To(Equal(gitx.ErrUnknown))
	})
})
