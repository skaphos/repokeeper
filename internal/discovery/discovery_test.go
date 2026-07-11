// SPDX-License-Identifier: MIT
package discovery_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/discovery"
	"github.com/skaphos/repokeeper/internal/vcs"
)

var _ = Describe("Discovery", func() {
	It("matches exclude patterns", func() {
		Expect(discovery.MatchesExclude("C:/code/repo/.git", []string{"**/.git/**"})).To(BeTrue())
		Expect(discovery.MatchesExclude("C:/code/repo", []string{"**/node_modules/**"})).To(BeFalse())
	})

	It("scans for git repositories", func() {
		root := GinkgoT().TempDir()
		repo := filepath.Join(root, "repo1")
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())

		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:   []string{root},
			Adapter: vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Path).To(Equal(repo))
		Expect(results[0].Bare).To(BeFalse())
	})

	It("respects exclude patterns during scan", func() {
		root := GinkgoT().TempDir()
		repo := filepath.Join(root, "vendor", "repo2")
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())

		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:   []string{root},
			Exclude: []string{"**/vendor/**"},
			Adapter: vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})

	It("detects linked .git directories", func() {
		root := GinkgoT().TempDir()
		repo := filepath.Join(root, "repo3")
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())

		gitDir := filepath.Join(root, "repo3.gitdir")
		Expect(os.Rename(filepath.Join(repo, ".git"), gitDir)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: "+gitDir), 0o644)).To(Succeed())

		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:   []string{root},
			Adapter: vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Path).To(Equal(repo))
	})

	It("walks a symlinked root instead of yielding nothing", func() {
		base := GinkgoT().TempDir()
		realDir := filepath.Join(base, "real")
		repo := filepath.Join(realDir, "repo")
		link := filepath.Join(base, "link")

		Expect(os.MkdirAll(repo, 0o755)).To(Succeed())
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())
		if err := os.Symlink(realDir, link); err != nil {
			Skip("symlinks not supported on this platform: " + err.Error())
		}

		// Root itself is a symlink; WalkDir would otherwise lstat it, see a
		// non-directory entry, and never descend into it at all.
		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:   []string{link},
			Adapter: vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		// Following a symlinked root reports the resolved target path; resolve
		// the expectation too so a symlinked ancestor (macOS /var ->
		// /private/var) or path canonicalization does not spuriously fail.
		resolvedRepo, evalErr := filepath.EvalSymlinks(repo)
		Expect(evalErr).NotTo(HaveOccurred())
		Expect(filepath.Clean(results[0].Path)).To(Equal(filepath.Clean(resolvedRepo)))
	})

	It("does not descend into a symlinked subdirectory when follow-symlinks is disabled", func() {
		base := GinkgoT().TempDir()
		realDir := filepath.Join(base, "target")
		repo := filepath.Join(realDir, "repo")
		root := filepath.Join(base, "root")
		link := filepath.Join(root, "link")

		Expect(os.MkdirAll(repo, 0o755)).To(Succeed())
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())
		Expect(os.MkdirAll(root, 0o755)).To(Succeed())
		if err := os.Symlink(realDir, link); err != nil {
			Skip("symlinks not supported on this platform: " + err.Error())
		}

		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:          []string{root},
			FollowSymlinks: false,
			Adapter:        vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})

	It("follows a symlinked subdirectory and tolerates a symlink cycle when enabled", func() {
		base := GinkgoT().TempDir()
		realDir := filepath.Join(base, "target")
		repo := filepath.Join(realDir, "repo")
		root := filepath.Join(base, "root")
		link := filepath.Join(root, "link")
		cycle := filepath.Join(realDir, "cycle")

		Expect(os.MkdirAll(repo, 0o755)).To(Succeed())
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())
		Expect(os.MkdirAll(root, 0o755)).To(Succeed())
		if err := os.Symlink(realDir, link); err != nil {
			Skip("symlinks not supported on this platform: " + err.Error())
		}
		// A symlink back to the already-visited target directory: without a
		// visited-set guard this would recurse forever.
		Expect(os.Symlink(realDir, cycle)).To(Succeed())

		done := make(chan struct{})
		var results []discovery.Result
		var err error
		go func() {
			results, err = discovery.Scan(context.Background(), discovery.Options{
				Roots:          []string{root},
				FollowSymlinks: true,
				Adapter:        vcs.NewGitAdapter(nil),
			})
			close(done)
		}()

		Eventually(done, "5s").Should(BeClosed(), "scan did not terminate; likely stuck in a symlink cycle")
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		// Following the symlink reports the resolved target path; resolve the
		// expectation too so macOS /var -> /private/var (and Windows path
		// canonicalization) do not spuriously fail.
		resolvedRepo, evalErr := filepath.EvalSymlinks(repo)
		Expect(evalErr).NotTo(HaveOccurred())
		Expect(filepath.Clean(results[0].Path)).To(Equal(filepath.Clean(resolvedRepo)))
	})

	It("does not duplicate results when a followed symlink points inside the walked tree", func() {
		root := GinkgoT().TempDir()
		sub := filepath.Join(root, "sub")
		repo := filepath.Join(sub, "repo")
		link := filepath.Join(root, "link")

		Expect(os.MkdirAll(repo, 0o755)).To(Succeed())
		Expect(exec.Command("git", "init", repo).Run()).To(Succeed())
		if err := os.Symlink(sub, link); err != nil {
			Skip("symlinks not supported on this platform: " + err.Error())
		}

		// root/sub/repo is a real repo and root/link -> root/sub. With
		// FollowSymlinks the symlink resolves back into the tree WalkDir already
		// covers, so the repo must appear exactly once, not twice.
		results, err := discovery.Scan(context.Background(), discovery.Options{
			Roots:          []string{root},
			FollowSymlinks: true,
			Adapter:        vcs.NewGitAdapter(nil),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
	})

	It("does not duplicate results when roots overlap", func() {
		root := GinkgoT().TempDir()
		repoA := filepath.Join(root, "repoA")
		sub := filepath.Join(root, "sub")
		repoB := filepath.Join(sub, "repoB")
		Expect(exec.Command("git", "init", repoA).Run()).To(Succeed())
		Expect(exec.Command("git", "init", repoB).Run()).To(Succeed())

		for _, roots := range [][]string{
			{root, sub},
			{sub, root},
		} {
			results, err := discovery.Scan(context.Background(), discovery.Options{
				Roots:   roots,
				Adapter: vcs.NewGitAdapter(nil),
			})
			Expect(err).NotTo(HaveOccurred())
			var paths []string
			for _, r := range results {
				paths = append(paths, filepath.Clean(r.Path))
			}
			Expect(paths).To(ConsistOf(filepath.Clean(repoA), filepath.Clean(repoB)), "roots=%v", roots)
		}
	})
})
