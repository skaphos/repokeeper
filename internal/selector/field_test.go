// SPDX-License-Identifier: MIT
package selector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/selector"
)

var _ = Describe("Field Selector", func() {
	Describe("ResolveRepoFilter", func() {
		DescribeTable("valid combinations",
			func(only, fieldSelector string, want engine.FilterKind) {
				got, err := selector.ResolveRepoFilter(only, fieldSelector)
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(want))
			},
			Entry("only all default", "all", "", engine.FilterAll),
			Entry("only dirty", "dirty", "", engine.FilterDirty),
			Entry("field selector diverged", "all", "tracking.status=diverged", engine.FilterDiverged),
			Entry("field selector missing", "", "repo.missing=true", engine.FilterMissing),
			Entry("field selector dirty false", "", "worktree.dirty=false", engine.FilterClean),
			Entry("field selector error", "", "repo.error=true", engine.FilterErrors),
			Entry("field selector remote mismatch", "", "remote.mismatch=true", engine.FilterRemoteMismatch),
			Entry("field selector gone", "", "tracking.status=gone", engine.FilterGone),
		)

		It("rejects mixed --only and --field-selector", func() {
			_, err := selector.ResolveRepoFilter("dirty", "tracking.status=gone")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseFieldSelectorFilter", func() {
		It("parses tracking.status=all", func() {
			got, err := selector.ParseFieldSelectorFilter("tracking.status=all")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(engine.FilterAll))
		})

		It("rejects blank selector", func() {
			_, err := selector.ParseFieldSelectorFilter("  ")
			Expect(err).To(HaveOccurred())
		})

		It("rejects missing equals sign", func() {
			_, err := selector.ParseFieldSelectorFilter("tracking.status")
			Expect(err).To(HaveOccurred())
		})

		It("rejects unsupported key", func() {
			_, err := selector.ParseFieldSelectorFilter("repo.name=foo")
			Expect(err).To(HaveOccurred())
		})

		It("rejects unsupported value", func() {
			_, err := selector.ParseFieldSelectorFilter("tracking.status=equal")
			Expect(err).To(HaveOccurred())
		})

		It("rejects multi selector", func() {
			_, err := selector.ParseFieldSelectorFilter("tracking.status=gone,repo.error=true")
			Expect(err).To(HaveOccurred())
		})
	})
})
