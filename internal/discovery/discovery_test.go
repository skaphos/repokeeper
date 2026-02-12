package discovery_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mfacenet/repokeeper/internal/discovery"
	"github.com/mfacenet/repokeeper/internal/vcs"
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
})
