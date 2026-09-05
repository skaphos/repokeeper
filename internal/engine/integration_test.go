// SPDX-License-Identifier: MIT
//go:build integration

package engine_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

var _ = Describe("Engine integration", func() {
	It("handles symlink discovery according to follow-symlinks setting", func() {
		base := GinkgoT().TempDir()
		realRoot := filepath.Join(base, "real-root")
		repoPath := filepath.Join(realRoot, "repo")
		linkedRoot := filepath.Join(base, "linked-root")

		Expect(os.MkdirAll(realRoot, 0o755)).To(Succeed())
		runGit("", "init", repoPath)
		err := os.Symlink(realRoot, linkedRoot)
		if err != nil {
			Skip("symlink not supported on this environment: " + err.Error())
		}

		cfg := &config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}
		reg := &registry.Registry{}
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil), nil, nil, nil)

		results, err := eng.Scan(context.Background(), engine.ScanOptions{
			Roots:          []string{realRoot, linkedRoot},
			FollowSymlinks: false,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(filepath.Clean(results[0].Path)).To(Equal(filepath.Clean(repoPath)))
	})

	It("reports bare repository status fields", func() {
		base := GinkgoT().TempDir()
		bare := filepath.Join(base, "bare.git")
		runGit("", "init", "--bare", bare)

		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, &registry.Registry{}, vcs.NewGitAdapter(nil), nil, nil, nil)
		status, err := eng.InspectRepo(context.Background(), bare)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Bare).To(BeTrue())
		Expect(status.Worktree).To(BeNil())
		Expect(status.Tracking.Status).To(Equal(model.TrackingNone))
	})

	It("reports missing registry entries in status output", func() {
		base := GinkgoT().TempDir()
		missing := filepath.Join(base, "missing-repo")

		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "missing-repo", Path: missing, Status: registry.StatusMissing},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
		report, err := eng.Status(context.Background(), engine.StatusOptions{Filter: engine.FilterAll, Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())
		Expect(report.Repos).To(HaveLen(1))
		Expect(report.Repos[0].Error).To(Equal("path missing"))
		Expect(report.Repos[0].ErrorClass).To(Equal("missing"))
	})

	It("fetch/prune does not change working tree files", func() {
		base := GinkgoT().TempDir()
		remote := filepath.Join(base, "remote.git")
		work := filepath.Join(base, "work")

		runGit("", "init", "--bare", remote)
		runGit("", "clone", remote, work)
		runGit(work, "config", "user.email", "test@example.com")
		runGit(work, "config", "user.name", "RepoKeeper Test")

		writeFile(filepath.Join(work, "file.txt"), "initial\n")
		runGit(work, "add", "file.txt")
		runGit(work, "commit", "-m", "init")
		runGit(work, "branch", "-M", "main")
		runGit(work, "push", "origin", "main")

		writeFile(filepath.Join(work, "file.txt"), "dirty\n")
		before := readFile(filepath.Join(work, "file.txt"))

		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: work, RemoteURL: remote, Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
		_, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())

		after := readFile(filepath.Join(work, "file.txt"))
		Expect(after).To(Equal(before))
		status := runGit(work, "status", "--porcelain=v1")
		Expect(status).To(ContainSubstring("file.txt"))
	})

	It("prunes stale remote-tracking branches", func() {
		base := GinkgoT().TempDir()
		remote := filepath.Join(base, "remote.git")
		work := filepath.Join(base, "work")
		other := filepath.Join(base, "other")

		runGit("", "init", "--bare", remote)
		runGit("", "clone", remote, work)
		runGit("", "clone", remote, other)
		runGit(other, "config", "user.email", "test@example.com")
		runGit(other, "config", "user.name", "RepoKeeper Test")

		writeFile(filepath.Join(other, "file.txt"), "base\n")
		runGit(other, "add", "file.txt")
		runGit(other, "commit", "-m", "base")
		runGit(other, "branch", "-M", "main")
		runGit(other, "push", "origin", "main")

		runGit(other, "checkout", "-b", "feature")
		writeFile(filepath.Join(other, "feature.txt"), "feature\n")
		runGit(other, "add", "feature.txt")
		runGit(other, "commit", "-m", "feature")
		runGit(other, "push", "origin", "feature")

		runGit(work, "fetch", "--all")
		Expect(strings.TrimSpace(runGit(work, "for-each-ref", "refs/remotes/origin/feature"))).To(ContainSubstring("origin/feature"))

		runGit(other, "push", "origin", "--delete", "feature")

		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: work, RemoteURL: remote, Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
		status, err := eng.InspectRepo(context.Background(), work)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.RemoteTrackingRefs.StaleCount).To(Equal(1))
		Expect(status.RemoteTrackingRefs.Stale).To(Equal([]string{"origin/feature"}))
		Expect(strings.TrimSpace(runGit(work, "for-each-ref", "refs/remotes/origin/feature"))).To(ContainSubstring("origin/feature"), "inspection must not prune local refs")

		_, err = eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())

		out := strings.TrimSpace(runGit(work, "for-each-ref", "refs/remotes/origin/feature"))
		Expect(out).To(BeEmpty())
		status, err = eng.InspectRepo(context.Background(), work)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.RemoteTrackingRefs.StaleCount).To(BeZero())
	})

	It("reports tracking gone after upstream branch is deleted", func() {
		base := GinkgoT().TempDir()
		remote := filepath.Join(base, "remote.git")
		work := filepath.Join(base, "work")
		other := filepath.Join(base, "other")

		runGit("", "init", "--bare", remote)
		runGit("", "clone", remote, work)
		runGit("", "clone", remote, other)

		runGit(work, "config", "user.email", "test@example.com")
		runGit(work, "config", "user.name", "RepoKeeper Test")
		writeFile(filepath.Join(work, "file.txt"), "base\n")
		runGit(work, "add", "file.txt")
		runGit(work, "commit", "-m", "base")
		runGit(work, "branch", "-M", "main")
		runGit(work, "push", "-u", "origin", "main")

		// Create a replacement default branch so deleting origin/main is allowed.
		runGit(other, "config", "user.email", "test@example.com")
		runGit(other, "config", "user.name", "RepoKeeper Test")
		runGit(other, "fetch", "origin", "main")
		runGit(other, "checkout", "-b", "keep", "origin/main")
		runGit(other, "push", "-u", "origin", "keep")
		runGit("", "--git-dir", remote, "symbolic-ref", "HEAD", "refs/heads/keep")
		runGit(other, "push", "origin", "--delete", "main")

		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: work, RemoteURL: remote, Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
		_, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())

		report, err := eng.Status(context.Background(), engine.StatusOptions{Filter: engine.FilterAll, Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())
		Expect(report.Repos).To(HaveLen(1))
		Expect(report.Repos[0].Tracking.Upstream).To(Equal("origin/main"))
		Expect(report.Repos[0].Tracking.Status).To(Equal(model.TrackingGone))
	})

	It("marks deleted repository paths as missing after re-scan", func() {
		base := GinkgoT().TempDir()
		repoPath := filepath.Join(base, "repo")
		runGit("", "init", repoPath)

		cfg := &config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}
		reg := &registry.Registry{}
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil), nil, nil, nil)

		_, err := eng.Scan(context.Background(), engine.ScanOptions{
			Roots:          []string{base},
			FollowSymlinks: false,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Status).To(Equal(registry.StatusPresent))

		Expect(os.RemoveAll(repoPath)).To(Succeed())
		_, err = eng.Scan(context.Background(), engine.ScanOptions{
			Roots:          []string{base},
			FollowSymlinks: false,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Status).To(Equal(registry.StatusMissing))
	})

	It("marks repositories as missing when git metadata is removed", func() {
		base := GinkgoT().TempDir()
		repoPath := filepath.Join(base, "repo")
		runGit("", "init", repoPath)

		cfg := &config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}
		reg := &registry.Registry{}
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil), nil, nil, nil)

		_, err := eng.Scan(context.Background(), engine.ScanOptions{
			Roots:          []string{base},
			FollowSymlinks: false,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Status).To(Equal(registry.StatusPresent))

		Expect(os.RemoveAll(filepath.Join(repoPath, ".git"))).To(Succeed())
		_, err = eng.Scan(context.Background(), engine.ScanOptions{
			Roots:          []string{base},
			FollowSymlinks: false,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Status).To(Equal(registry.StatusMissing))
	})

	It("preserves both checkout identities and local metadata when one disappears and returns", func() {
		base := GinkgoT().TempDir()
		root := filepath.Join(base, "workspace")
		primaryPath := filepath.Join(root, "a", "proj")
		secondaryPath := filepath.Join(root, "b", "proj")
		for _, path := range []string{primaryPath, secondaryPath} {
			runGit("", "init", path)
			runGit(path, "remote", "add", "origin", "git@github.com:example/proj.git")
		}
		cfg := &config.Config{}
		reg := &registry.Registry{}
		eng := engine.New(cfg, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
		opts := engine.ScanOptions{Roots: []string{root}}
		results, err := eng.Scan(context.Background(), opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(2))
		const repoID = "github.com/example/proj"
		primary := reg.FindEntry(repoID, primaryPath)
		secondary := reg.FindEntry(repoID, secondaryPath)
		Expect(primary).NotTo(BeNil())
		Expect(secondary).NotTo(BeNil())
		primary.Labels = map[string]string{"env": "prod", "role": "primary"}
		primary.Annotations = map[string]string{"owner": "operations"}
		secondary.Labels = map[string]string{"env": "dev"}
		secondary.Annotations = map[string]string{"owner": "development"}
		originalPrimary, originalSecondary := *primary, *secondary
		Expect(originalPrimary.CheckoutID).NotTo(Equal(originalSecondary.CheckoutID))

		// Reload between scans to model separate CLI invocations, including the
		// first scan after an unavailable volume returns.
		rescan := func() []model.RepoStatus {
			registryPath := filepath.Join(base, "registry.yaml")
			Expect(registry.Save(reg, registryPath)).To(Succeed())
			reg, err = registry.Load(registryPath)
			Expect(err).NotTo(HaveOccurred())
			eng = engine.New(cfg, reg, vcs.NewGitAdapter(nil), nil, nil, nil)
			statuses, scanErr := eng.Scan(context.Background(), opts)
			Expect(scanErr).NotTo(HaveOccurred())
			return statuses
		}
		assertCheckout := func(original registry.Entry, status registry.EntryStatus) {
			GinkgoHelper()
			entry := reg.FindEntry(repoID, original.Path)
			Expect(entry).NotTo(BeNil())
			Expect(entry.CheckoutID).To(Equal(original.CheckoutID))
			Expect(entry.Labels).To(Equal(original.Labels))
			Expect(entry.Annotations).To(Equal(original.Annotations))
			Expect(entry.Status).To(Equal(status))
		}
		away := filepath.Join(base, "away")
		Expect(os.Rename(primaryPath, away)).To(Succeed())
		for range 2 {
			results = rescan()
			Expect(results).To(HaveLen(1))
			Expect(results[0].CheckoutID).To(Equal(originalSecondary.CheckoutID))
			Expect(reg.Entries).To(HaveLen(2))
			assertCheckout(originalPrimary, registry.StatusMissing)
			assertCheckout(originalSecondary, registry.StatusPresent)
		}
		Expect(os.Rename(away, primaryPath)).To(Succeed())
		for range 2 {
			results = rescan()
			Expect(results).To(HaveLen(2))
			Expect(reg.Entries).To(HaveLen(2))
			assertCheckout(originalPrimary, registry.StatusPresent)
			assertCheckout(originalSecondary, registry.StatusPresent)
		}
	})
})

func runGit(dir string, args ...string) string {
	baseArgs := []string{"-c", "commit.gpgsign=false"}
	cmd := exec.Command("git", append(baseArgs, args...)...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		Fail("git command failed: " + stderr.String())
	}
	return stdout.String()
}

func writeFile(path, content string) {
	Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
