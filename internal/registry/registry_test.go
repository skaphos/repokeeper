// SPDX-License-Identifier: MIT
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

	It("preserves type and branch when not provided on upsert", func() {
		reg := &registry.Registry{}
		reg.Upsert(registry.Entry{
			RepoID: "repo1",
			Path:   "/a",
			Type:   "mirror",
			Branch: "main",
			Status: registry.StatusPresent,
		})
		reg.Upsert(registry.Entry{
			RepoID: "repo1",
			Path:   "/a",
			Status: registry.StatusPresent,
		})
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Type).To(Equal("mirror"))
		Expect(reg.Entries[0].Branch).To(Equal("main"))
	})

	It("preserves labels and annotations when not provided on upsert", func() {
		reg := &registry.Registry{}
		reg.Upsert(registry.Entry{
			RepoID:      "repo1",
			Path:        "/a",
			Labels:      map[string]string{"team": "platform"},
			Annotations: map[string]string{"owner": "sre"},
			Status:      registry.StatusPresent,
		})
		reg.Upsert(registry.Entry{
			RepoID: "repo1",
			Path:   "/a",
			Status: registry.StatusPresent,
		})
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].Labels).To(HaveKeyWithValue("team", "platform"))
		Expect(reg.Entries[0].Annotations).To(HaveKeyWithValue("owner", "sre"))
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

	It("returns nil when repo ID not found", func() {
		reg := &registry.Registry{Entries: []registry.Entry{{RepoID: "repo1"}}}
		entry := reg.FindByRepoID("nonexistent")
		Expect(entry).To(BeNil())
	})

	It("finds entry by exact path and repo ID match", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
				{RepoID: "repo1", Path: "/path/b"},
			},
		}
		entry := reg.FindEntry("repo1", "/path/b")
		Expect(entry).NotTo(BeNil())
		Expect(entry.Path).To(Equal("/path/b"))
	})

	It("falls back to repoID-only match when path doesn't match", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
			},
		}
		entry := reg.FindEntry("repo1", "/path/nonexistent")
		Expect(entry).NotTo(BeNil())
		Expect(entry.Path).To(Equal("/path/a"))
	})

	It("returns nil when entry not found by FindEntry", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
			},
		}
		entry := reg.FindEntry("nonexistent", "/path/a")
		Expect(entry).To(BeNil())
	})

	It("returns correct index for exact path and repo ID match", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
				{RepoID: "repo1", Path: "/path/b"},
				{RepoID: "repo2", Path: "/path/c"},
			},
		}
		idx := reg.FindEntryIndex("repo1", "/path/b")
		Expect(idx).To(Equal(1))
	})

	It("falls back to repoID-only match when path doesn't match in FindEntryIndex", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
				{RepoID: "repo2", Path: "/path/b"},
			},
		}
		idx := reg.FindEntryIndex("repo1", "/path/nonexistent")
		Expect(idx).To(Equal(0))
	})

	It("returns -1 when entry not found by FindEntryIndex", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "repo1", Path: "/path/a"},
			},
		}
		idx := reg.FindEntryIndex("nonexistent", "/path/a")
		Expect(idx).To(Equal(-1))
	})

	It("returns error when saving nil registry", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "registry.yaml")
		err := registry.Save(nil, path)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("registry is nil"))
	})

	It("returns 0 when pruning with zero duration", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "old", Status: registry.StatusMissing, LastSeen: time.Now().Add(-48 * time.Hour)},
			},
		}
		pruned := reg.PruneStale(0)
		Expect(pruned).To(Equal(0))
		Expect(reg.Entries).To(HaveLen(1))
	})

	It("returns 0 when pruning with negative duration", func() {
		reg := &registry.Registry{
			Entries: []registry.Entry{
				{RepoID: "old", Status: registry.StatusMissing, LastSeen: time.Now().Add(-48 * time.Hour)},
			},
		}
		pruned := reg.PruneStale(-1 * time.Hour)
		Expect(pruned).To(Equal(0))
		Expect(reg.Entries).To(HaveLen(1))
	})
})
