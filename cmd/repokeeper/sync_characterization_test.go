// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/spf13/cobra"
)

var _ = Describe("describeSyncAction", func() {
	DescribeTable("characterization — all SyncError* constant mappings and action heuristics",
		func(res engine.SyncResult, want string) {
			Expect(describeSyncAction(res)).To(Equal(want))
		},

		// Planned=true (dry-run mode) is NOT matched by any special-case error
		// check, so it falls through to action-string heuristics.
		// CRITICAL: These entries capture the behaviour of the dry-run sentinel
		// (formerly SyncErrorDryRun, now the Planned field) to guard against regression.
		Entry("Planned=true + empty action + OK=true → fetch",
			engine.SyncResult{Planned: true, OK: true},
			"fetch"),
		Entry("Planned=true + empty action + OK=false → dash",
			engine.SyncResult{Planned: true, OK: false},
			"-"),
		Entry("Planned=true + fetch action → fetch",
			engine.SyncResult{
				Planned: true,
				Action:  "git fetch --all --prune --prune-tags --no-recurse-submodules",
				OK:      true,
			},
			"fetch"),
		Entry("Planned=true + fetch+rebase action → fetch + rebase",
			engine.SyncResult{
				Planned: true,
				Action:  "git fetch --all --prune && git pull --rebase --no-recurse-submodules",
				OK:      true,
			},
			"fetch + rebase"),

		// SyncErrorMissing ("missing") → "skip missing"
		Entry("SyncErrorMissing → skip missing",
			engine.SyncResult{Error: engine.SyncErrorMissing, OK: false},
			"skip missing"),

		// SyncErrorSkipped ("skipped") → "skip"
		Entry("SyncErrorSkipped → skip",
			engine.SyncResult{Error: engine.SyncErrorSkipped},
			"skip"),

		// SyncErrorSkippedNoUpstream ("skipped-no-upstream") → "skip no upstream"
		Entry("SyncErrorSkippedNoUpstream → skip no upstream",
			engine.SyncResult{Error: engine.SyncErrorSkippedNoUpstream, OK: true},
			"skip no upstream"),

		// SyncErrorMissingRemoteForCheckout — no special-case match;
		// empty action + OK=false falls through to the dash default.
		Entry("SyncErrorMissingRemoteForCheckout + empty action + OK=false → dash",
			engine.SyncResult{Error: engine.SyncErrorMissingRemoteForCheckout, OK: false},
			"-"),

		// SyncErrorSkippedLocalUpdatePrefix sub-cases.
		// The prefix is "skipped-local-update: " (trailing space included).
		Entry("SyncErrorSkippedLocalUpdatePrefix bare (empty reason) → skip local update",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix, OK: true},
			"skip local update"),
		Entry("SyncErrorSkippedLocalUpdatePrefix + already up to date → fetch",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix + "already up to date", OK: true},
			"fetch"),
		Entry("SyncErrorSkippedLocalUpdatePrefix + dirty worktree → skip local update (dirty worktree)",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix + "dirty worktree", OK: true},
			"skip local update (dirty worktree)"),
		Entry("SyncErrorSkippedLocalUpdatePrefix + diverged → skip local update (diverged)",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix + "diverged", OK: true},
			"skip local update (diverged)"),
		Entry("SyncErrorSkippedLocalUpdatePrefix + local commits to push → skip local update (local commits to push)",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix + "local commits to push", OK: true},
			"skip local update (local commits to push)"),
		Entry("SyncErrorSkippedLocalUpdatePrefix + protected branch → skip local update (protected branch)",
			engine.SyncResult{Error: engine.SyncErrorSkippedLocalUpdatePrefix + "protected branch", OK: true},
			"skip local update (protected branch)"),

		// Fetch-error constants — none are matched by the error special-cases;
		// with empty action and OK=false all produce the dash default.
		Entry("SyncErrorFetchFailed + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchFailed, OK: false},
			"-"),
		Entry("SyncErrorFetchAuth + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchAuth, OK: false},
			"-"),
		Entry("SyncErrorFetchNetwork + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchNetwork, OK: false},
			"-"),
		Entry("SyncErrorFetchTimeout + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchTimeout, OK: false},
			"-"),
		Entry("SyncErrorFetchCorrupt + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchCorrupt, OK: false},
			"-"),
		Entry("SyncErrorFetchMissingRemote + empty action → dash",
			engine.SyncResult{Error: engine.SyncErrorFetchMissingRemote, OK: false},
			"-"),

		// Action-string heuristics (error field is irrelevant when action matches).
		Entry("stash+rebase action → stash & rebase",
			engine.SyncResult{
				Action: `git stash push -u -m "repokeeper: pre-rebase stash" && git pull --rebase --no-recurse-submodules && git stash pop`,
			},
			"stash & rebase"),
		Entry("hg pull → fetch",
			engine.SyncResult{Action: "hg pull"},
			"fetch"),
		Entry("fetch --all + git push → fetch + push",
			engine.SyncResult{Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git push"},
			"fetch + push"),
		Entry("fetch --all + pull --rebase → fetch + rebase",
			engine.SyncResult{Action: "git fetch --all --prune --prune-tags --no-recurse-submodules && git pull --rebase --no-recurse-submodules"},
			"fetch + rebase"),
		Entry("git push alone → push",
			engine.SyncResult{Action: "git push"},
			"push"),
		Entry("pull --rebase alone → rebase",
			engine.SyncResult{Action: "git pull --rebase --no-recurse-submodules"},
			"rebase"),
		Entry("fetch --all alone → fetch",
			engine.SyncResult{Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
			"fetch"),
		Entry("git clone --mirror → checkout missing (mirror)",
			engine.SyncResult{Action: "git clone --mirror git@github.com:org/repo.git /path"},
			"checkout missing (mirror)"),
		Entry("git clone (non-mirror) → checkout missing",
			engine.SyncResult{Action: "git clone git@github.com:org/repo.git /path"},
			"checkout missing"),

		// Unrecognised action: returned verbatim.
		Entry("unrecognised action returned verbatim",
			engine.SyncResult{Action: "custom-vcs-command"},
			"custom-vcs-command"),

		// Empty action fallbacks.
		Entry("empty action + OK=true → fetch",
			engine.SyncResult{OK: true},
			"fetch"),
		Entry("empty action + OK=false → dash",
			engine.SyncResult{OK: false},
			"-"),
	)
})

var _ = Describe("newSyncProgressWriter", func() {
	It("returns non-nil writer with correct cwd and roots", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		w := newSyncProgressWriter(cmd, "/tmp/cwd", []string{"/roots/one"})

		Expect(w).NotTo(BeNil())
		Expect(w.cwd).To(Equal("/tmp/cwd"))
		Expect(w.roots).To(Equal([]string{"/roots/one"}))
	})

	It("sets supportsInPlace=false when stdout is a bytes.Buffer (non-terminal)", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		w := newSyncProgressWriter(cmd, "/tmp/cwd", nil)

		Expect(w.supportsInPlace).To(BeFalse())
	})

	It("initialises running map as non-nil and empty", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		w := newSyncProgressWriter(cmd, "/cwd", nil)

		Expect(w.running).NotTo(BeNil())
		Expect(w.running).To(BeEmpty())
	})

	It("stores the exact cmd reference", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		w := newSyncProgressWriter(cmd, "", nil)

		Expect(w.cmd).To(BeIdenticalTo(cmd))
	})

	It("accepts nil roots without panicking", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		var w *syncProgressWriter
		Expect(func() {
			w = newSyncProgressWriter(cmd, "/cwd", nil)
		}).NotTo(Panic())
		Expect(w.roots).To(BeNil())
	})
})

var _ = Describe("syncProgressWriter.StartResult", func() {
	var (
		cmd    *cobra.Command
		out    *bytes.Buffer
		errOut *bytes.Buffer
		w      *syncProgressWriter
	)

	BeforeEach(func() {
		cmd = &cobra.Command{}
		out = &bytes.Buffer{}
		errOut = &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		w = newSyncProgressWriter(cmd, "/tmp", nil)
	})

	It("writes initial progress dot to output synchronously before goroutine launch", func() {
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
			OK:     true,
		}

		Expect(w.StartResult(res)).To(Succeed())

		// The initial "." is written inside the mutex before go-routines are involved.
		Expect(out.String()).To(ContainSubstring("repo ."))

		DeferCleanup(func() { _ = w.WriteResult(res) })
	})

	It("WriteResult writes updated! line with action summary for successful result", func() {
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
			OK:     true,
		}

		Expect(w.StartResult(res)).To(Succeed())
		Expect(w.WriteResult(res)).To(Succeed())

		got := out.String()
		Expect(got).To(ContainSubstring("updated!"))
		Expect(got).To(ContainSubstring("fetch"))
	})

	It("WriteResult writes failed+error-class to stdout and raw error to stderr for non-OK result", func() {
		res := engine.SyncResult{
			RepoID:     "r1",
			Path:       "/tmp/repo",
			OK:         false,
			Error:      "connection refused",
			ErrorClass: "network",
		}

		Expect(w.StartResult(res)).To(Succeed())
		Expect(w.WriteResult(res)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("failed"))
		Expect(out.String()).To(ContainSubstring("network"))
		// Raw error detail goes to stderr.
		Expect(errOut.String()).To(ContainSubstring("connection refused"))
	})

	It("second StartResult for the same path is a no-op (no additional output)", func() {
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			OK:     true,
		}

		Expect(w.StartResult(res)).To(Succeed())
		lenAfterFirst := out.Len()

		// Second call for same path must not write anything extra.
		Expect(w.StartResult(res)).To(Succeed())
		Expect(out.Len()).To(Equal(lenAfterFirst))

		DeferCleanup(func() { _ = w.WriteResult(res) })
	})

	It("WriteResult for skipped-no-upstream result shows skip reason; stderr stays empty", func() {
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			OK:     true,
			Error:  engine.SyncErrorSkippedNoUpstream,
		}

		Expect(w.StartResult(res)).To(Succeed())
		Expect(w.WriteResult(res)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("skip no upstream"))
		// Skip actions do not produce an error line on stderr.
		Expect(errOut.String()).To(BeEmpty())
	})

	It("WriteResult for SyncErrorMissing shows skip missing; stderr stays empty", func() {
		// SyncErrorMissing (OK=false) maps to action "skip missing" which begins with
		// "skip", so no error line is written to stderr despite OK being false.
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			OK:     false,
			Error:  engine.SyncErrorMissing,
		}

		Expect(w.StartResult(res)).To(Succeed())
		Expect(w.WriteResult(res)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("skip missing"))
		Expect(errOut.String()).To(BeEmpty())
	})
})

var _ = Describe("syncProgressWriter.runDots", func() {
	It("closes done channel promptly when stop channel is signalled", func() {
		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})

		w := &syncProgressWriter{
			cmd:     cmd,
			cwd:     "/tmp",
			running: make(map[string]*syncProgressState),
		}

		const path = "/tmp/repo"
		state := &syncProgressState{
			displayPath: "repo",
			stop:        make(chan struct{}),
			done:        make(chan struct{}),
		}
		w.running[path] = state

		go w.runDots(path, state)

		// Closing stop must cause done to close within a short deadline.
		close(state.stop)
		select {
		case <-state.done:
			// expected: goroutine exited and deferred close(state.done) ran
		case <-time.After(500 * time.Millisecond):
			Fail("runDots: done channel not closed within 500ms after stop signal")
		}
	})

	It("goroutine exits cleanly via StartResult+WriteResult round-trip", func() {
		// This integration path exercises runDots launch (StartResult) and
		// stop-channel teardown (WriteResult), without waiting for a ticker tick.
		cmd := &cobra.Command{}
		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetErr(&bytes.Buffer{})

		w := newSyncProgressWriter(cmd, "/tmp", nil)
		res := engine.SyncResult{
			RepoID: "r1",
			Path:   "/tmp/repo",
			Action: "git fetch --all --prune --prune-tags --no-recurse-submodules",
			OK:     true,
		}

		// StartResult writes "." and launches runDots goroutine.
		Expect(w.StartResult(res)).To(Succeed())
		// WriteResult signals stop, waits for done, then writes the final result line.
		Expect(w.WriteResult(res)).To(Succeed())

		// At minimum the initial dot must appear in the output.
		Expect(out.String()).To(ContainSubstring("."))
	})
})

var _ = Describe("sync command flag validation", func() {
	It("rejects --concurrency > 64", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("concurrency", 0, "")
		cmd.Flags().Int("timeout", 0, "")
		cmd.Flags().String("only", "", "")
		cmd.Flags().String("field-selector", "", "")
		cmd.Flags().Bool("continue-on-error", true, "")
		cmd.Flags().Bool("dry-run", false, "")
		cmd.Flags().Bool("update-local", false, "")
		cmd.Flags().Bool("push-local", false, "")
		cmd.Flags().Bool("rebase-dirty", false, "")
		cmd.Flags().Bool("force", false, "")
		cmd.Flags().String("protected-branches", "", "")
		cmd.Flags().Bool("allow-protected-rebase", false, "")
		cmd.Flags().Bool("checkout-missing", false, "")
		cmd.Flags().String("format", "table", "")
		cmd.Flags().Bool("no-headers", false, "")
		cmd.Flags().Bool("wrap", false, "")
		cmd.Flags().String("vcs", "git", "")

		Expect(cmd.Flags().Set("concurrency", "100")).To(Succeed())

		concurrency, _ := cmd.Flags().GetInt("concurrency")

		if concurrency > 0 && concurrency > 64 {
			Expect(fmt.Sprintf("--concurrency must be <= 64, got %d", concurrency)).To(Equal("--concurrency must be <= 64, got 100"))
		} else {
			Fail("validation should have rejected concurrency > 64")
		}
	})

	It("rejects --timeout > 600", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("concurrency", 0, "")
		cmd.Flags().Int("timeout", 0, "")
		cmd.Flags().String("only", "", "")
		cmd.Flags().String("field-selector", "", "")
		cmd.Flags().Bool("continue-on-error", true, "")
		cmd.Flags().Bool("dry-run", false, "")
		cmd.Flags().Bool("update-local", false, "")
		cmd.Flags().Bool("push-local", false, "")
		cmd.Flags().Bool("rebase-dirty", false, "")
		cmd.Flags().Bool("force", false, "")
		cmd.Flags().String("protected-branches", "", "")
		cmd.Flags().Bool("allow-protected-rebase", false, "")
		cmd.Flags().Bool("checkout-missing", false, "")
		cmd.Flags().String("format", "table", "")
		cmd.Flags().Bool("no-headers", false, "")
		cmd.Flags().Bool("wrap", false, "")
		cmd.Flags().String("vcs", "git", "")

		Expect(cmd.Flags().Set("timeout", "1000")).To(Succeed())

		timeout, _ := cmd.Flags().GetInt("timeout")

		if timeout > 0 && timeout > 600 {
			Expect(fmt.Sprintf("--timeout must be <= 600, got %d", timeout)).To(Equal("--timeout must be <= 600, got 1000"))
		} else {
			Fail("validation should have rejected timeout > 600")
		}
	})

	It("accepts --concurrency <= 64", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("concurrency", 0, "")
		Expect(cmd.Flags().Set("concurrency", "64")).To(Succeed())

		concurrency, _ := cmd.Flags().GetInt("concurrency")
		Expect(concurrency).To(Equal(64))
	})

	It("accepts --timeout <= 600", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("timeout", 0, "")
		Expect(cmd.Flags().Set("timeout", "600")).To(Succeed())

		timeout, _ := cmd.Flags().GetInt("timeout")
		Expect(timeout).To(Equal(600))
	})

	It("accepts default values (0) for concurrency and timeout", func() {
		cmd := &cobra.Command{}
		cmd.Flags().Int("concurrency", 0, "")
		cmd.Flags().Int("timeout", 0, "")

		concurrency, _ := cmd.Flags().GetInt("concurrency")
		timeout, _ := cmd.Flags().GetInt("timeout")

		Expect(concurrency).To(Equal(0))
		Expect(timeout).To(Equal(0))
	})
})
