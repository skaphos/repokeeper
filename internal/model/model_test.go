// SPDX-License-Identifier: MIT
package model_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/model"
)

var _ = Describe("Model JSON", func() {
	It("round-trips RepoStatus JSON", func() {
		now := time.Now().UTC()
		ahead := 2
		behind := 1
		status := model.RepoStatus{
			RepoID:        "github.com/org/repo",
			CheckoutID:    "checkout-main",
			Path:          "/tmp/repo",
			Labels:        map[string]string{"team": "platform"},
			Annotations:   map[string]string{"owner": "sre"},
			Bare:          false,
			Remotes:       []model.Remote{{Name: "origin", URL: "git@github.com:org/repo.git"}},
			PrimaryRemote: "origin",
			Head:          model.Head{Branch: "main", Detached: false},
			Worktree:      &model.Worktree{Dirty: true, Staged: 1, Unstaged: 2, Untracked: 0},
			Tracking:      model.Tracking{Upstream: "origin/main", Status: model.TrackingAhead, Ahead: &ahead, Behind: &behind},
			Submodules:    model.Submodules{HasSubmodules: false},
			LastSync:      &model.SyncResult{OK: true, At: now},
		}

		data, err := json.Marshal(status)
		Expect(err).NotTo(HaveOccurred())

		var decoded model.RepoStatus
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.RepoID).To(Equal(status.RepoID))
		Expect(decoded.CheckoutID).To(Equal(status.CheckoutID))
		Expect(decoded.Tracking.Status).To(Equal(model.TrackingAhead))
		Expect(decoded.Worktree).NotTo(BeNil())
		Expect(decoded.Labels).To(HaveKeyWithValue("team", "platform"))
	})

	It("round-trips StatusReport JSON", func() {
		report := model.StatusReport{
			GeneratedAt: time.Now().UTC(),
			Repos: []model.RepoStatus{
				{RepoID: "repo1", Path: "/tmp/repo1"},
			},
		}
		data, err := json.Marshal(report)
		Expect(err).NotTo(HaveOccurred())

		var decoded model.StatusReport
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Repos).To(HaveLen(1))
	})
})

var _ = Describe("Prune classification types", func() {
	DescribeTable("ParsePruneCategory",
		func(raw string, want model.PruneCategory, wantErr bool) {
			got, err := model.ParsePruneCategory(raw)
			if wantErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("keep", "keep", model.PruneKeep, false),
		Entry("safe_to_prune", "safe_to_prune", model.PruneSafeToPrune, false),
		Entry("probably_safe", "probably_safe", model.PruneProbablySafe, false),
		Entry("needs_review", "needs_review", model.PruneNeedsReview, false),
		Entry("case-insensitive and trimmed", "  KEEP ", model.PruneKeep, false),
		Entry("unknown", "bogus", model.PruneCategory(""), true),
		Entry("empty", "", model.PruneCategory(""), true),
	)

	It("returns hints for known reasons and empty for unknown", func() {
		Expect(model.HintForReason(model.ReasonMergedIntoBase)).NotTo(BeEmpty())
		Expect(model.HintForReason(model.ReasonPatchEquivalentToBase)).NotTo(BeEmpty())
		Expect(model.HintForReason(model.PruneReason("nonexistent"))).To(BeEmpty())
	})

	It("round-trips LocalBranchStatus JSON including tri-state signals", func() {
		merged := true
		last := time.Now().UTC()
		status := model.LocalBranchStatus{
			Branches: []model.LocalBranch{{
				Name:           "feature/x",
				UpstreamStatus: model.TrackingGone,
				MergedIntoBase: &merged,
				LastCommitAt:   &last,
				Category:       model.PruneSafeToPrune,
				Reasons:        []model.PruneReason{model.ReasonMergedIntoBase},
			}},
		}
		data, err := json.Marshal(status)
		Expect(err).NotTo(HaveOccurred())

		var decoded model.LocalBranchStatus
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Branches).To(HaveLen(1))
		Expect(decoded.Branches[0].Category).To(Equal(model.PruneSafeToPrune))
		Expect(decoded.Branches[0].MergedIntoBase).NotTo(BeNil())
		Expect(*decoded.Branches[0].MergedIntoBase).To(BeTrue())
		Expect(decoded.Branches[0].PatchEquivalentToBase).To(BeNil())
		Expect(decoded.Branches[0].Reasons).To(ConsistOf(model.ReasonMergedIntoBase))
	})
})
