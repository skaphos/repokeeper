// SPDX-License-Identifier: MIT
package gitx_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/gitx"
)

var _ = Describe("NormalizeURL", func() {
	DescribeTable("normalizes git remote URLs",
		func(input, expected string) {
			Expect(gitx.NormalizeURL(input)).To(Equal(expected))
		},
		Entry("SSH shorthand", "git@github.com:Org/Repo.git", "github.com/Org/Repo"),
		Entry("SSH shorthand without .git", "git@github.com:Org/Repo", "github.com/Org/Repo"),
		Entry("HTTPS with .git", "https://github.com/Org/Repo.git", "github.com/Org/Repo"),
		Entry("HTTPS without .git", "https://github.com/Org/Repo", "github.com/Org/Repo"),
		Entry("HTTPS with trailing slash", "https://github.com/Org/Repo/", "github.com/Org/Repo"),
		Entry("git:// protocol", "git://github.com/Org/Repo.git", "github.com/Org/Repo"),
		Entry("ssh:// protocol", "ssh://git@github.com/Org/Repo.git", "github.com/Org/Repo"),
		Entry("ssh:// with port", "ssh://git@github.com:22/Org/Repo.git", "github.com/Org/Repo"),
		Entry("host is lowercased", "git@GitHub.COM:Org/Repo.git", "github.com/Org/Repo"),
		Entry("path case preserved", "git@github.com:MyOrg/MyRepo.git", "github.com/MyOrg/MyRepo"),
		Entry("HTTP protocol", "http://github.com/Org/Repo.git", "github.com/Org/Repo"),
		Entry("HTTPS with credentials", "https://user:pass@github.com/Org/Repo.git", "github.com/Org/Repo"),
		Entry("empty string", "", ""),
		Entry("deeply nested path", "git@gitlab.com:group/sub/Repo.git", "gitlab.com/group/sub/Repo"),
	)
})

var _ = Describe("PrimaryRemote", func() {
	It("prefers origin", func() {
		Expect(gitx.PrimaryRemote([]string{"upstream", "origin", "fork"})).To(Equal("origin"))
	})

	It("falls back to first alphabetically", func() {
		Expect(gitx.PrimaryRemote([]string{"upstream", "fork"})).To(Equal("fork"))
	})

	It("returns empty for empty list", func() {
		Expect(gitx.PrimaryRemote([]string{})).To(Equal(""))
	})

	It("returns the single remote", func() {
		Expect(gitx.PrimaryRemote([]string{"myremote"})).To(Equal("myremote"))
	})
})
