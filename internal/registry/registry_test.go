// SPDX-License-Identifier: MIT
package registry_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/model"
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

	It("saves and loads repo metadata snapshot cache fields", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "registry.yaml")
		reg := &registry.Registry{Entries: []registry.Entry{{
			RepoID:                  "repo1",
			Path:                    filepath.Join(dir, "repo1"),
			Status:                  registry.StatusPresent,
			RepoMetadataFile:        filepath.Join(dir, "repo1", ".repokeeper-repo.yaml"),
			RepoMetadataFingerprint: "file:/repo:1:2",
			RepoMetadataError:       "",
			RepoMetadata:            &model.RepoMetadata{Name: "Repo One", Labels: map[string]string{"team": "platform"}},
		}}}
		Expect(registry.Save(reg, path)).To(Succeed())
		loaded, err := registry.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Entries).To(HaveLen(1))
		Expect(loaded.Entries[0].RepoMetadataFile).To(Equal(reg.Entries[0].RepoMetadataFile))
		Expect(loaded.Entries[0].RepoMetadataFingerprint).To(Equal("file:/repo:1:2"))
		Expect(loaded.Entries[0].RepoMetadata).NotTo(BeNil())
		Expect(loaded.Entries[0].RepoMetadata.Name).To(Equal("Repo One"))
		Expect(loaded.Entries[0].RepoMetadata.Labels).To(HaveKeyWithValue("team", "platform"))
	})

	It("upserts entries for the same checkout identity", func() {
		reg := &registry.Registry{}
		reg.Upsert(registry.Entry{RepoID: "repo1", Path: "/worktrees/primary", Status: registry.StatusPresent})
		reg.Upsert(registry.Entry{RepoID: "repo1", Path: "/alternate/primary", Status: registry.StatusPresent})
		Expect(reg.Entries).To(HaveLen(1))
		Expect(reg.Entries[0].CheckoutID).To(Equal("primary"))
		Expect(reg.Entries[0].Path).To(Equal("/alternate/primary"))
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

func TestFindEntriesByRepoID(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{}

	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/primary", Status: registry.StatusPresent})
	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/another-root/primary", Status: registry.StatusPresent})

	g.Expect(reg.Entries).To(HaveLen(1), "same checkout identity should update the existing entry")
	g.Expect(reg.Entries[0].CheckoutID).To(Equal("primary"), "default checkout identity should be derived from path basename")
	g.Expect(reg.Entries[0].Path).To(Equal("/another-root/primary"), "same checkout updates keep the latest path")
	g.Expect(reg.FindByRepoID("github.com/acme/repo")).NotTo(BeNil())
	g.Expect(reg.FindByRepoID("github.com/acme/repo").Path).To(Equal("/another-root/primary"), "repo_id lookup should still return the updated entry")
	g.Expect(reg.FindEntry("github.com/acme/repo", "/worktrees/primary")).NotTo(BeNil())
	g.Expect(reg.FindEntry("github.com/acme/repo", "/worktrees/primary").Path).To(Equal("/another-root/primary"), "path miss should continue to fall back to repo_id when only one checkout exists")
}

func TestUpsertAllowsDuplicateRepoIDWithDistinctCheckoutID(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{}

	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/primary", Status: registry.StatusPresent})
	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/secondary", Status: registry.StatusPresent})

	g.Expect(reg.Entries).To(HaveLen(2), "future checkout-aware upsert should keep both entries when checkout_id differs even if repo_id matches")
	g.Expect(reg.Entries).To(ContainElements(
		SatisfyAll(
			HaveField("Path", "/worktrees/primary"),
			HaveField("CheckoutID", "primary"),
		),
		SatisfyAll(
			HaveField("Path", "/worktrees/secondary"),
			HaveField("CheckoutID", "secondary"),
		),
	), "distinct checkout_id values should prevent repo_id-only collapse")
}

func TestUpsertCollapsesDuplicatePathAcrossRepoIDs(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{}

	reg.Upsert(registry.Entry{
		RepoID:      "github.com/acme/old-repo",
		Path:        "/worktrees/shared",
		Status:      registry.StatusPresent,
		Labels:      map[string]string{"team": "platform"},
		Annotations: map[string]string{"owner": "sre"},
		RepoMetadata: &model.RepoMetadata{
			Name:   "Shared Repo",
			Labels: map[string]string{"scope": "shared"},
		},
	})
	reg.Upsert(registry.Entry{
		RepoID:    "github.com/acme/new-repo",
		Path:      "/worktrees/shared",
		RemoteURL: "git@github.com:acme/new-repo.git",
		Status:    registry.StatusPresent,
	})

	g.Expect(reg.Entries).To(HaveLen(1))
	g.Expect(reg.Entries[0].RepoID).To(Equal("github.com/acme/new-repo"))
	g.Expect(reg.Entries[0].Path).To(Equal("/worktrees/shared"))
	g.Expect(reg.Entries[0].Labels).To(HaveKeyWithValue("team", "platform"))
	g.Expect(reg.Entries[0].Annotations).To(HaveKeyWithValue("owner", "sre"))
	g.Expect(reg.Entries[0].RepoMetadata).NotTo(BeNil())
	g.Expect(reg.Entries[0].RepoMetadata.Name).To(Equal("Shared Repo"))
}

func TestUpsertRemovesLegacyDuplicatePathEntries(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/acme/one", Path: "/worktrees/shared", Status: registry.StatusPresent},
			{RepoID: "github.com/acme/two", Path: "/worktrees/shared", Status: registry.StatusPresent, Labels: map[string]string{"team": "platform"}},
		},
	}

	reg.Upsert(registry.Entry{RepoID: "github.com/acme/canonical", Path: "/worktrees/shared", Status: registry.StatusPresent})

	g.Expect(reg.Entries).To(HaveLen(1))
	g.Expect(reg.Entries[0].RepoID).To(Equal("github.com/acme/canonical"))
	g.Expect(reg.Entries[0].Labels).To(HaveKeyWithValue("team", "platform"))
}

func TestLegacyEntryBackfillsCheckoutID(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/acme/repo", Path: "/worktrees/legacy-one", Status: registry.StatusPresent},
			{RepoID: "github.com/acme/repo", Path: "/worktrees/legacy-two", Status: registry.StatusPresent},
		},
	}

	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/legacy-two", Status: registry.StatusPresent})

	g.Expect(reg.Entries).To(HaveLen(2), "legacy entries without checkout_id should be backfilled during migration so existing duplicate checkouts remain distinct")
	g.Expect(reg.FindEntry("github.com/acme/repo", "/worktrees/legacy-one")).NotTo(BeNil())
	g.Expect(reg.FindEntry("github.com/acme/repo", "/worktrees/legacy-one").Path).To(Equal("/worktrees/legacy-one"), "after checkout_id backfill, path-specific lookup should not fall back across legacy entries")
}

func TestFindEntriesByRepoIDReturnsAllCheckoutMatches(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{}

	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/primary", Status: registry.StatusPresent})
	reg.Upsert(registry.Entry{RepoID: "github.com/acme/repo", Path: "/worktrees/secondary", Status: registry.StatusPresent})
	reg.Upsert(registry.Entry{RepoID: "github.com/acme/other", Path: "/worktrees/other", Status: registry.StatusPresent})

	entries := reg.FindEntriesByRepoID("github.com/acme/repo")
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries).To(ContainElements(
		SatisfyAll(
			HaveField("Path", "/worktrees/primary"),
			HaveField("CheckoutID", "primary"),
		),
		SatisfyAll(
			HaveField("Path", "/worktrees/secondary"),
			HaveField("CheckoutID", "secondary"),
		),
	))
	g.Expect(reg.FindEntriesByRepoID("github.com/acme/missing")).To(BeNil())
}

func TestFindByRepoIDAndCheckoutIDBackfillsLegacyEntries(t *testing.T) {
	g := NewWithT(t)
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/acme/repo", Path: "/worktrees/primary", Status: registry.StatusPresent},
			{RepoID: "github.com/acme/repo", Path: "/worktrees/secondary", Status: registry.StatusPresent},
		},
	}

	primary := reg.FindByRepoIDAndCheckoutID("github.com/acme/repo", "primary")
	g.Expect(primary).NotTo(BeNil())
	g.Expect(primary.Path).To(Equal("/worktrees/primary"))
	g.Expect(primary.CheckoutID).To(Equal("primary"))

	secondary := reg.FindByRepoIDAndCheckoutID("github.com/acme/repo", "secondary")
	g.Expect(secondary).NotTo(BeNil())
	g.Expect(secondary.Path).To(Equal("/worktrees/secondary"))
	g.Expect(secondary.CheckoutID).To(Equal("secondary"))

	g.Expect(reg.FindByRepoIDAndCheckoutID("github.com/acme/repo", "missing")).To(BeNil())
}
