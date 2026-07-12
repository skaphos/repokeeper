// SPDX-License-Identifier: MIT
package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/config"
)

var _ = Describe("BranchPolicy", func() {
	const gvk = "apiVersion: skaphos.io/repokeeper/v1beta1\nkind: RepoKeeperConfig\n"

	writeConfig := func(body string) string {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "config.yaml")
		Expect(os.WriteFile(p, []byte(gvk+body), 0o644)).To(Succeed())
		return p
	}

	It("DefaultConfig seeds a conservative branch policy", func() {
		cfg := config.DefaultConfig()
		Expect(cfg.BranchPolicy.ProtectedPatterns).To(ConsistOf("main", "master", "release/*"))
		Expect(cfg.BranchPolicy.RequireMerged).To(BeTrue())
		Expect(cfg.BranchPolicy.StaleDays).To(Equal(0))
		Expect(cfg.BranchPolicy.BaseBranch).To(BeEmpty())
	})

	It("omitted branch_policy loads seeded defaults", func() {
		cfg, err := config.Load(writeConfig(""))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BranchPolicy.RequireMerged).To(BeTrue())
		Expect(cfg.BranchPolicy.ProtectedPatterns).To(ConsistOf("main", "master", "release/*"))
	})

	It("explicit require_merged:false survives load (no zero-value backfill)", func() {
		cfg, err := config.Load(writeConfig("branch_policy:\n  require_merged: false\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BranchPolicy.RequireMerged).To(BeFalse())
	})

	It("explicit empty protected_patterns survives load", func() {
		cfg, err := config.Load(writeConfig("branch_policy:\n  protected_patterns: []\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BranchPolicy.ProtectedPatterns).To(BeEmpty())
	})

	It("a partial branch_policy keeps defaults for unmentioned fields", func() {
		cfg, err := config.Load(writeConfig("branch_policy:\n  stale_days: 45\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.BranchPolicy.StaleDays).To(Equal(45))
		Expect(cfg.BranchPolicy.RequireMerged).To(BeTrue())
		Expect(cfg.BranchPolicy.ProtectedPatterns).To(ConsistOf("main", "master", "release/*"))
	})

	DescribeTable("fails closed on invalid branch policy",
		func(policyBody string) {
			_, err := config.Load(writeConfig(policyBody))
			Expect(err).To(HaveOccurred())
		},
		Entry("negative stale_days", "branch_policy:\n  stale_days: -1\n"),
		Entry("malformed glob", "branch_policy:\n  protected_patterns: [\"release/[\"]\n"),
		Entry("over-broad star", "branch_policy:\n  protected_patterns: [\"*\"]\n"),
		Entry("glob base_branch", "branch_policy:\n  base_branch: \"release/*\"\n"),
	)

	It("round-trips a valid branch policy through Save then Load", func() {
		cfg := config.DefaultConfig()
		cfg.BranchPolicy.StaleDays = 90
		cfg.BranchPolicy.BaseBranch = "develop"
		cfg.BranchPolicy.RequireMerged = false
		p := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		Expect(config.Save(&cfg, p)).To(Succeed())

		loaded, err := config.Load(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.BranchPolicy.StaleDays).To(Equal(90))
		Expect(loaded.BranchPolicy.BaseBranch).To(Equal("develop"))
		Expect(loaded.BranchPolicy.RequireMerged).To(BeFalse())
	})

	It("rejects saving an invalid branch policy", func() {
		cfg := config.DefaultConfig()
		cfg.BranchPolicy.StaleDays = -5
		p := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		Expect(config.Save(&cfg, p)).NotTo(Succeed())
	})
})
