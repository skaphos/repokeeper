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
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

var _ = Describe("Engine integration", func() {
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
			MachineID: "m1",
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: work, Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil))
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
			MachineID: "m1",
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: work, Status: registry.StatusPresent},
			},
		}
		eng := engine.New(&config.Config{Defaults: config.Defaults{TimeoutSeconds: 5, Concurrency: 1}}, reg, vcs.NewGitAdapter(nil))
		_, err := eng.Sync(context.Background(), engine.SyncOptions{Concurrency: 1, Timeout: 5})
		Expect(err).NotTo(HaveOccurred())

		out := strings.TrimSpace(runGit(work, "for-each-ref", "refs/remotes/origin/feature"))
		Expect(out).To(BeEmpty())
	})
})

func runGit(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
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
