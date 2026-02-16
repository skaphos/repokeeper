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
