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
			Entry("field selector behind", "", "tracking.status=behind", engine.FilterBehind),
			Entry("field selector ahead", "", "tracking.status=ahead", engine.FilterAhead),
			Entry("field selector equal", "", "tracking.status=equal", engine.FilterEqual),
		)

		It("rejects mixed --only and --field-selector", func() {
			_, err := selector.ResolveRepoFilter("dirty", "tracking.status=gone")
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("rejects unknown --only values instead of failing open to all repos",
			func(only string) {
				_, err := selector.ResolveRepoFilter(only, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported --only value"))
			},
			Entry("typo of errors", "errrors"),
			Entry("unrelated word", "bogus"),
			Entry("case-insensitive typo", "ERRRORS"),
			Entry("field-selector-shaped value used as --only", "tracking.status=gone"),
		)

		DescribeTable("accepts every documented --only value",
			func(only string, want engine.FilterKind) {
				got, err := selector.ResolveRepoFilter(only, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(want))
			},
			Entry("all", "all", engine.FilterAll),
			Entry("errors", "errors", engine.FilterErrors),
			Entry("dirty", "dirty", engine.FilterDirty),
			Entry("clean", "clean", engine.FilterClean),
			Entry("gone", "gone", engine.FilterGone),
			Entry("diverged", "diverged", engine.FilterDiverged),
			Entry("behind", "behind", engine.FilterBehind),
			Entry("ahead", "ahead", engine.FilterAhead),
			Entry("equal", "equal", engine.FilterEqual),
			Entry("remote-mismatch", "remote-mismatch", engine.FilterRemoteMismatch),
			Entry("missing", "missing", engine.FilterMissing),
			Entry("empty defaults to all", "", engine.FilterAll),
			Entry("uppercase is case-insensitive", "DIRTY", engine.FilterDirty),
		)

		It("rejects blank field-selector input", func() {
			_, err := selector.ResolveRepoFilter("", "   ")
			Expect(err).To(MatchError("--field-selector cannot be blank"))
		})

		It("rejects comma-only blank field-selector input before only validation", func() {
			_, err := selector.ResolveRepoFilter("dirty", ",")
			Expect(err).To(MatchError("--field-selector cannot be blank"))
		})

		It("rejects whitespace comma blank field-selector input before only validation", func() {
			_, err := selector.ResolveRepoFilter("dirty", " , ")
			Expect(err).To(MatchError("--field-selector cannot be blank"))
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
			Expect(err).To(MatchError("--field-selector cannot be blank"))
		})

		It("rejects comma-only blank selector", func() {
			_, err := selector.ParseFieldSelectorFilter(",")
			Expect(err).To(MatchError("--field-selector cannot be blank"))
		})

		It("rejects whitespace comma blank selector", func() {
			_, err := selector.ParseFieldSelectorFilter(" , ")
			Expect(err).To(MatchError("--field-selector cannot be blank"))
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
			_, err := selector.ParseFieldSelectorFilter("tracking.status=bogus")
			Expect(err).To(MatchError(ContainSubstring(`unsupported tracking.status value "bogus"`)))
		})

		It("rejects multi selector", func() {
			_, err := selector.ParseFieldSelectorFilter("tracking.status=gone,repo.error=true")
			Expect(err).To(HaveOccurred())
		})
	})
})
