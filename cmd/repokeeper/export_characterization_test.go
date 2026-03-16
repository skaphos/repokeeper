// SPDX-License-Identifier: MIT
package repokeeper

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("exportEntryPath", func() {
	DescribeTable("characterization",
		func(path, root, want string) {
			Expect(exportEntryPath(path, root)).To(Equal(want))
		},
		Entry("absolute path within root returns relative", "/source/root/team/repo", "/source/root", "team/repo"),
		Entry("already-relative path returned unchanged", "team/repo", "/source/root", "team/repo"),
		Entry("absolute path outside root returns cleaned absolute", "/opt/repo", "/source/root", filepath.FromSlash("/opt/repo")),
		Entry("path with spaces within root returns space-preserved relative", "/source/root/my repo", "/source/root", "my repo"),
		Entry("windows backslash separators normalized to forward slash", `team\repo`, "/source/root", "team/repo"),
		Entry("empty path and empty root fall back to filepath.Clean dot", "", "", "."),
		Entry("nested path within root returns multi-segment relative", "/source/root/a/b/c", "/source/root", "a/b/c"),
		Entry("relative path with empty root goes through cleanRelativePath", "sub/repo", "", "sub/repo"),
	)
})
