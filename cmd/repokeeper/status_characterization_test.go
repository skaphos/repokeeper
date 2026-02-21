// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/skaphos/repokeeper/internal/model"
)

func statusIntPtr(n int) *int { return &n }

var _ = Describe("metadataMapString", func() {
	DescribeTable("characterization",
		func(input map[string]string, want string) {
			Expect(metadataMapString(input)).To(Equal(want))
		},
		Entry("nil map returns dash", nil, "-"),
		Entry("empty map returns dash", map[string]string{}, "-"),
		Entry("single entry formats as key=value", map[string]string{"key": "val"}, "key=val"),
		Entry("two entries are sorted alphabetically", map[string]string{"b": "2", "a": "1"}, "a=1,b=2"),
		Entry("three entries are sorted alphabetically",
			map[string]string{"z": "last", "a": "first", "m": "mid"}, "a=first,m=mid,z=last"),
		Entry("special chars in value are preserved",
			map[string]string{"url": "https://example.com?q=1&x=2"}, "url=https://example.com?q=1&x=2"),
		Entry("dots in key name are preserved",
			map[string]string{"key.with.dots": "v"}, "key.with.dots=v"),
	)
})

var _ = Describe("divergedReasonAndAction", func() {
	DescribeTable("characterization",
		func(repo model.RepoStatus, wantReason, wantAction string) {
			reason, action := divergedReasonAndAction(repo)
			Expect(reason).To(Equal(wantReason))
			Expect(action).To(Equal(wantAction))
		},
		Entry("ahead tracking returns empty strings",
			model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingAhead}},
			"", ""),
		Entry("equal tracking returns empty strings",
			model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingEqual}},
			"", ""),
		Entry("gone tracking returns empty strings",
			model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingGone}},
			"", ""),
		Entry("none tracking returns empty strings",
			model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingNone}},
			"", ""),
		Entry("diverged with nil worktree and nil ahead/behind returns generic message",
			model.RepoStatus{
				Tracking: model.Tracking{Status: model.TrackingDiverged},
			},
			"local and upstream histories diverged",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
		Entry("diverged with clean worktree and nil ahead/behind returns generic message",
			model.RepoStatus{
				Tracking: model.Tracking{Status: model.TrackingDiverged},
				Worktree: &model.Worktree{Dirty: false},
			},
			"local and upstream histories diverged",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
		Entry("diverged with dirty worktree returns uncommitted-changes reason",
			model.RepoStatus{
				Tracking: model.Tracking{Status: model.TrackingDiverged},
				Worktree: &model.Worktree{Dirty: true},
			},
			"local and upstream histories diverged with uncommitted changes",
			"commit or stash changes, then resolve with manual rebase/merge"),
		Entry("diverged with dirty worktree ignores ahead/behind counts",
			model.RepoStatus{
				Tracking: model.Tracking{
					Status: model.TrackingDiverged,
					Ahead:  statusIntPtr(3),
					Behind: statusIntPtr(1),
				},
				Worktree: &model.Worktree{Dirty: true},
			},
			"local and upstream histories diverged with uncommitted changes",
			"commit or stash changes, then resolve with manual rebase/merge"),
		Entry("diverged with clean worktree and both ahead/behind counts returns count-specific message",
			model.RepoStatus{
				Tracking: model.Tracking{
					Status: model.TrackingDiverged,
					Ahead:  statusIntPtr(2),
					Behind: statusIntPtr(1),
				},
				Worktree: &model.Worktree{Dirty: false},
			},
			"branch is 2 ahead and 1 behind upstream",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
		Entry("diverged with nil worktree and both ahead/behind counts returns count-specific message",
			model.RepoStatus{
				Tracking: model.Tracking{
					Status: model.TrackingDiverged,
					Ahead:  statusIntPtr(5),
					Behind: statusIntPtr(3),
				},
			},
			"branch is 5 ahead and 3 behind upstream",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
		Entry("diverged with only behind set returns generic message",
			model.RepoStatus{
				Tracking: model.Tracking{
					Status: model.TrackingDiverged,
					Behind: statusIntPtr(2),
				},
			},
			"local and upstream histories diverged",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
		Entry("diverged with only ahead set returns generic message",
			model.RepoStatus{
				Tracking: model.Tracking{
					Status: model.TrackingDiverged,
					Ahead:  statusIntPtr(4),
				},
			},
			"local and upstream histories diverged",
			"resolve manually, or run reconcile with --update-local --force if acceptable"),
	)
})

var _ = Describe("writeStatusDetails", func() {
	var (
		out *bytes.Buffer
		cmd *cobra.Command
	)

	BeforeEach(func() {
		out = &bytes.Buffer{}
		cmd = &cobra.Command{}
		cmd.SetOut(out)
	})

	Context("minimal repo snapshot — zero-value fields", func() {
		It("writes all mandatory lines and omits TYPE, ERROR_CLASS, ERROR", func() {
			repo := model.RepoStatus{
				Path: "/repos/testrepo",
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			want := "" +
				"PATH: testrepo\n" +
				"PATH_ABS: /repos/testrepo\n" +
				"REPO: \n" +
				"BARE: false\n" +
				"BRANCH: \n" +
				"DIRTY: -\n" +
				"TRACKING: \n" +
				"UPSTREAM: \n" +
				"LABELS: -\n" +
				"ANNOTATIONS: -\n" +
				"AHEAD: -\n" +
				"BEHIND: -\n"
			Expect(out.String()).To(Equal(want))
		})
	})

	Context("full repo snapshot — all fields populated", func() {
		It("matches complete expected output exactly", func() {
			repo := model.RepoStatus{
				RepoID: "github.com/org/repo",
				Path:   "/repos/testrepo",
				Head:   model.Head{Branch: "main", Detached: true},
				Tracking: model.Tracking{
					Status:   model.TrackingEqual,
					Upstream: "origin/main",
					Ahead:    statusIntPtr(2),
					Behind:   statusIntPtr(1),
				},
				Worktree:   &model.Worktree{Dirty: true},
				Error:      "boom",
				ErrorClass: "network",
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			want := "" +
				"PATH: testrepo\n" +
				"PATH_ABS: /repos/testrepo\n" +
				"REPO: github.com/org/repo\n" +
				"BARE: false\n" +
				"BRANCH: detached:main\n" +
				"DIRTY: yes\n" +
				"TRACKING: up to date\n" +
				"UPSTREAM: origin/main\n" +
				"LABELS: -\n" +
				"ANNOTATIONS: -\n" +
				"AHEAD: 2\n" +
				"BEHIND: 1\n" +
				"ERROR_CLASS: network\n" +
				"ERROR: boom\n"
			Expect(out.String()).To(Equal(want))
		})
	})

	Context("mirror type repo", func() {
		It("renders TYPE line, BRANCH as dash, and TRACKING as mirror", func() {
			repo := model.RepoStatus{
				RepoID: "github.com/org/mirror",
				Path:   "/repos/mirror",
				Type:   "mirror",
				Head:   model.Head{Branch: "main"},
				Tracking: model.Tracking{
					Status:   model.TrackingEqual,
					Upstream: "origin/main",
				},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("TYPE: mirror\n"))
			Expect(got).To(ContainSubstring("BRANCH: -\n"))
			Expect(got).To(ContainSubstring("TRACKING: mirror\n"))
		})
	})

	Context("repo with detached HEAD", func() {
		It("prefixes BRANCH value with detached:", func() {
			repo := model.RepoStatus{
				Path:     "/repos/testrepo",
				Head:     model.Head{Branch: "abc1234", Detached: true},
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			Expect(out.String()).To(ContainSubstring("BRANCH: detached:abc1234\n"))
		})
	})

	Context("repo with dirty worktree", func() {
		It("renders DIRTY as yes", func() {
			repo := model.RepoStatus{
				Path:     "/repos/testrepo",
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: true},
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			Expect(out.String()).To(ContainSubstring("DIRTY: yes\n"))
		})
	})

	Context("repo with clean worktree", func() {
		It("renders DIRTY as no", func() {
			repo := model.RepoStatus{
				Path:     "/repos/testrepo",
				Head:     model.Head{Branch: "main"},
				Worktree: &model.Worktree{Dirty: false},
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			Expect(out.String()).To(ContainSubstring("DIRTY: no\n"))
		})
	})

	Context("repo with nil worktree (bare)", func() {
		It("renders DIRTY as dash and BARE as true", func() {
			repo := model.RepoStatus{
				Path:     "/repos/bare",
				Bare:     true,
				Head:     model.Head{Branch: "main"},
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("BARE: true\n"))
			Expect(got).To(ContainSubstring("DIRTY: -\n"))
		})
	})

	Context("repo with TrackingEqual", func() {
		It("renders TRACKING as up to date", func() {
			repo := model.RepoStatus{
				Path: "/repos/testrepo",
				Tracking: model.Tracking{
					Status:   model.TrackingEqual,
					Upstream: "origin/main",
				},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("TRACKING: up to date\n"))
			Expect(got).To(ContainSubstring("UPSTREAM: origin/main\n"))
		})
	})

	Context("repo with TrackingGone", func() {
		It("renders TRACKING as gone", func() {
			repo := model.RepoStatus{
				Path:     "/repos/testrepo",
				Tracking: model.Tracking{Status: model.TrackingGone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			Expect(out.String()).To(ContainSubstring("TRACKING: gone\n"))
		})
	})

	Context("repo with labels and annotations", func() {
		It("writes sorted label and annotation key=value pairs", func() {
			repo := model.RepoStatus{
				Path:        "/repos/testrepo",
				Head:        model.Head{Branch: "main"},
				Tracking:    model.Tracking{Status: model.TrackingNone},
				Labels:      map[string]string{"env": "prod", "team": "platform"},
				Annotations: map[string]string{"owner": "sre"},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("LABELS: env=prod,team=platform\n"))
			Expect(got).To(ContainSubstring("ANNOTATIONS: owner=sre\n"))
		})
	})

	Context("repo with ahead and behind counts", func() {
		It("renders numeric AHEAD and BEHIND values", func() {
			repo := model.RepoStatus{
				Path: "/repos/testrepo",
				Tracking: model.Tracking{
					Status:   model.TrackingDiverged,
					Upstream: "origin/main",
					Ahead:    statusIntPtr(3),
					Behind:   statusIntPtr(7),
				},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("AHEAD: 3\n"))
			Expect(got).To(ContainSubstring("BEHIND: 7\n"))
		})
	})

	Context("repo with error class and error message set", func() {
		It("writes ERROR_CLASS and ERROR lines", func() {
			repo := model.RepoStatus{
				Path:       "/repos/testrepo",
				Head:       model.Head{Branch: "main"},
				Tracking:   model.Tracking{Status: model.TrackingNone},
				Error:      "authentication failed",
				ErrorClass: "auth",
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("ERROR_CLASS: auth\n"))
			Expect(got).To(ContainSubstring("ERROR: authentication failed\n"))
		})
	})

	Context("repo with no error fields", func() {
		It("omits ERROR_CLASS and ERROR lines entirely", func() {
			repo := model.RepoStatus{
				Path:     "/repos/testrepo",
				Head:     model.Head{Branch: "main"},
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).NotTo(ContainSubstring("ERROR_CLASS:"))
			Expect(got).NotTo(ContainSubstring("ERROR:"))
		})
	})

	Context("repo path outside cwd falls back to absolute", func() {
		It("writes the absolute path when no relative path can be computed", func() {
			repo := model.RepoStatus{
				Path:     "/opt/external/repo",
				Tracking: model.Tracking{Status: model.TrackingNone},
			}
			Expect(writeStatusDetails(cmd, repo, "/repos", nil)).To(Succeed())
			got := out.String()
			Expect(got).To(ContainSubstring("PATH: /opt/external/repo\n"))
			Expect(got).To(ContainSubstring("PATH_ABS: /opt/external/repo\n"))
		})
	})
})
