// SPDX-License-Identifier: MIT
package selector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/selector"
)

var _ = Describe("Label Selector", func() {
	Describe("ParseLabelSelector", func() {
		It("returns nil for empty input", func() {
			reqs, err := selector.ParseLabelSelector("")
			Expect(err).NotTo(HaveOccurred())
			Expect(reqs).To(BeNil())
		})

		It("parses an existence check", func() {
			reqs, err := selector.ParseLabelSelector("team")
			Expect(err).NotTo(HaveOccurred())
			Expect(reqs).To(HaveLen(1))
			Expect(reqs[0].Key).To(Equal("team"))
			Expect(reqs[0].HasValue).To(BeFalse())
		})

		It("parses a key=value expression", func() {
			reqs, err := selector.ParseLabelSelector("team=platform")
			Expect(err).NotTo(HaveOccurred())
			Expect(reqs).To(HaveLen(1))
			Expect(reqs[0].Key).To(Equal("team"))
			Expect(reqs[0].HasValue).To(BeTrue())
			Expect(reqs[0].Value).To(Equal("platform"))
		})

		It("parses multiple comma-separated expressions", func() {
			reqs, err := selector.ParseLabelSelector("team=platform,env=prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(reqs).To(HaveLen(2))
		})

		It("rejects keys containing whitespace", func() {
			_, err := selector.ParseLabelSelector("bad key=1")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LabelsMatchSelector", func() {
		labels := map[string]string{
			"team": "platform",
			"env":  "prod",
		}

		It("matches when all requirements are met", func() {
			reqs, err := selector.ParseLabelSelector("team=platform,env")
			Expect(err).NotTo(HaveOccurred())
			Expect(selector.LabelsMatchSelector(labels, reqs)).To(BeTrue())
		})

		It("does not match when a value requirement differs", func() {
			reqs, err := selector.ParseLabelSelector("team=app")
			Expect(err).NotTo(HaveOccurred())
			Expect(selector.LabelsMatchSelector(labels, reqs)).To(BeFalse())
		})

		It("does not match when a required key is missing", func() {
			reqs, err := selector.ParseLabelSelector("missing-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(selector.LabelsMatchSelector(labels, reqs)).To(BeFalse())
		})

		It("matches everything when requirements are empty", func() {
			Expect(selector.LabelsMatchSelector(labels, nil)).To(BeTrue())
		})
	})
})
