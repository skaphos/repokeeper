package registry_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/registry"
)

var _ = Describe("Registry", func() {
	It("saves and loads registry", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "registry.yaml")
		reg := &registry.Registry{
			UpdatedAt: time.Now(),
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: filepath.Join(dir, "repo1"), LastSeen: time.Now(), Status: registry.StatusPresent},
			},
		}
		Expect(registry.Save(reg, path)).To(Succeed())
		loaded, err := registry.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Entries).To(HaveLen(1))
	})

	It("upserts entries by repo ID", func() {
		reg := &registry.Registry{}
		reg.Upsert(registry.Entry{RepoID: "repo1", Path: "/a", Status: registry.StatusPresent})
		reg.Upsert(registry.Entry{RepoID: "repo1", Path: "/b", Status: registry.StatusPresent})
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Path).To(Equal("/b"))
	})

	It("validates paths and marks missing", func() {
		dir := GinkgoT().TempDir()
		existing := filepath.Join(dir, "exists")
		Expect(os.MkdirAll(existing, 0o755)).To(Succeed())
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: existing},
				{RepoID: "repo2", Path: filepath.Join(dir, "missing")},
			},
		}
		Expect(reg.ValidatePaths()).To(Succeed())
		Expect(reg.Entries[0].Status).To(Equal(registry.StatusPresent))
		Expect(reg.Entries[1].Status).To(Equal(registry.StatusMissing))
	})

	It("prunes stale missing entries", func() {
		old := time.Now().Add(-48 * time.Hour)
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "old", Status: registry.StatusMissing, LastSeen: old},
				{RepoID: "new", Status: registry.StatusMissing, LastSeen: time.Now()},
			},
		}
		pruned := reg.PruneStale(24 * time.Hour)
		Expect(pruned).To(Equal(1))
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].RepoID).To(Equal("new"))
	})

	It("finds entries by repo ID", func() {
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "repo1"}}}
		entry := reg.FindByRepoID("repo1")
		Expect(entry).NotTo(BeNil())
		Expect(entry.RepoID).To(Equal("repo1"))
	})
})
