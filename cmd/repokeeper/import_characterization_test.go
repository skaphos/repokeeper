// SPDX-License-Identifier: MIT
package repokeeper

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

var _ = Describe("stringMapsEqual", func() {
	DescribeTable("characterization",
		func(a, b map[string]string, want bool) {
			Expect(stringMapsEqual(a, b)).To(Equal(want))
		},
		Entry("both nil is equal", nil, nil, true),
		Entry("both empty is equal", map[string]string{}, map[string]string{}, true),
		Entry("nil vs empty is equal", nil, map[string]string{}, true),
		Entry("empty vs nil is equal", map[string]string{}, nil, true),
		Entry("equal single-entry maps", map[string]string{"k": "v"}, map[string]string{"k": "v"}, true),
		Entry("equal multi-entry maps", map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1", "b": "2"}, true),
		Entry("a longer than b is not equal", map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1"}, false),
		Entry("b longer than a is not equal", map[string]string{"a": "1"}, map[string]string{"a": "1", "b": "2"}, false),
		Entry("same keys different values is not equal", map[string]string{"k": "v1"}, map[string]string{"k": "v2"}, false),
		Entry("different keys same length is not equal", map[string]string{"x": "v"}, map[string]string{"y": "v"}, false),
	)
})

var _ = Describe("mergeImportedRegistry", func() {
	Context("nil cfg guard", func() {
		It("is a no-op when cfg is nil and does not panic", func() {
			Expect(func() {
				mergeImportedRegistry(nil, importModeMerge, true, &registry.Registry{}, importConflictPolicyBundle)
			}).NotTo(Panic())
		})
	})

	Context("includeRegistry=false", func() {
		It("clears cfg.Registry when mode is replace", func() {
			cfg := &config.Config{Registry: &registry.Registry{}}
			mergeImportedRegistry(cfg, importModeReplace, false, nil, importConflictPolicyBundle)
			Expect(cfg.Registry).To(BeNil())
		})

		It("leaves cfg.Registry unchanged when mode is merge", func() {
			existing := &registry.Registry{}
			cfg := &config.Config{Registry: existing}
			mergeImportedRegistry(cfg, importModeMerge, false, &registry.Registry{}, importConflictPolicyBundle)
			Expect(cfg.Registry).To(BeIdenticalTo(existing))
		})
	})

	Context("mode=replace with includeRegistry=true", func() {
		It("replaces cfg.Registry with a clone of bundled", func() {
			bundled := &registry.Registry{
				Entries: []registry.Entry{
					{RepoID: "r1", Path: "/bundle/r1", Status: registry.StatusPresent},
				},
			}
			cfg := &config.Config{
				Registry: &registry.Registry{
					Entries: []registry.Entry{
						{RepoID: "old", Path: "/local/old"},
					},
				},
			}
			mergeImportedRegistry(cfg, importModeReplace, true, bundled, importConflictPolicyBundle)
			Expect(cfg.Registry).NotTo(BeNil())
			Expect(cfg.Registry.Entries).To(HaveLen(1))
			Expect(cfg.Registry.Entries[0].RepoID).To(Equal("r1"))
		})

		It("sets cfg.Registry to nil when bundled is nil", func() {
			cfg := &config.Config{Registry: &registry.Registry{}}
			mergeImportedRegistry(cfg, importModeReplace, true, nil, importConflictPolicyBundle)
			Expect(cfg.Registry).To(BeNil())
		})
	})

	Context("mode=merge with includeRegistry=true", func() {
		It("initializes cfg.Registry if nil before merging bundled entries", func() {
			bundled := &registry.Registry{
				Entries: []registry.Entry{
					{RepoID: "r1", Path: "/r1"},
				},
			}
			cfg := &config.Config{Registry: nil}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			Expect(cfg.Registry).NotTo(BeNil())
			Expect(cfg.Registry.Entries).To(HaveLen(1))
		})

		It("is a no-op when bundled is nil and leaves existing entries intact", func() {
			existing := &registry.Registry{
				Entries: []registry.Entry{{RepoID: "r1", Path: "/r1"}},
			}
			cfg := &config.Config{Registry: existing}
			mergeImportedRegistry(cfg, importModeMerge, true, nil, importConflictPolicyBundle)
			Expect(cfg.Registry.Entries).To(HaveLen(1))
		})

		It("appends new entries from bundled that do not exist locally", func() {
			cfg := &config.Config{
				Registry: &registry.Registry{
					Entries: []registry.Entry{
						{RepoID: "existing", Path: "/local/existing"},
					},
				},
			}
			bundled := &registry.Registry{
				Entries: []registry.Entry{
					{RepoID: "new", Path: "/bundle/new"},
				},
			}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			Expect(cfg.Registry.Entries).To(HaveLen(2))
			Expect(cfg.Registry.FindByRepoID("new")).NotTo(BeNil())
		})

		It("overwrites non-conflicting existing entry with bundled version", func() {
			local := registry.Entry{
				RepoID: "r1", Path: "/r1", RemoteURL: "git@gh/r1.git",
				Branch: "main", Type: "checkout", Status: registry.StatusPresent,
			}
			incoming := registry.Entry{
				RepoID: "r1", Path: "/r1", RemoteURL: "git@gh/r1.git",
				Branch: "main", Type: "checkout", Status: registry.StatusPresent,
				Labels: map[string]string{"env": "prod"},
			}
			cfg := &config.Config{
				Registry: &registry.Registry{Entries: []registry.Entry{local}},
			}
			bundled := &registry.Registry{Entries: []registry.Entry{incoming}}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			got := cfg.Registry.FindByRepoID("r1")
			Expect(got).NotTo(BeNil())
			Expect(got.Labels).To(HaveKeyWithValue("env", "prod"))
		})

		It("preserves labels and annotations from bundled on non-conflict update", func() {
			local := registry.Entry{
				RepoID: "r1", Path: "/r1", RemoteURL: "git@gh/r1.git",
				Branch: "main", Type: "checkout",
				Labels:      map[string]string{"team": "platform"},
				Annotations: map[string]string{"owner": "sre"},
			}
			incoming := registry.Entry{
				RepoID: "r1", Path: "/r1", RemoteURL: "git@gh/r1.git",
				Branch: "main", Type: "checkout",
				Labels:      map[string]string{"team": "platform"},
				Annotations: map[string]string{"owner": "sre"},
			}
			cfg := &config.Config{
				Registry: &registry.Registry{Entries: []registry.Entry{local}},
			}
			bundled := &registry.Registry{Entries: []registry.Entry{incoming}}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			got := cfg.Registry.FindByRepoID("r1")
			Expect(got).NotTo(BeNil())
			Expect(got.Labels).To(HaveKeyWithValue("team", "platform"))
			Expect(got.Annotations).To(HaveKeyWithValue("owner", "sre"))
		})

		It("sets UpdatedAt timestamp after merge", func() {
			cfg := &config.Config{Registry: &registry.Registry{}}
			bundled := &registry.Registry{
				Entries: []registry.Entry{{RepoID: "r1", Path: "/r1"}},
			}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			Expect(cfg.Registry.UpdatedAt.IsZero()).To(BeFalse())
		})

		It("handles duplicate repo_id entries in bundled by applying last write wins", func() {
			cfg := &config.Config{Registry: &registry.Registry{}}
			bundled := &registry.Registry{
				Entries: []registry.Entry{
					{RepoID: "r1", Path: "/bundle/r1-first"},
					{RepoID: "r1", Path: "/bundle/r1-second"},
				},
			}
			mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
			Expect(cfg.Registry.Entries).To(HaveLen(1))
			got := cfg.Registry.FindByRepoID("r1")
			Expect(got).NotTo(BeNil())
		})

		Context("conflict policy behavior", func() {
			var (
				localEntry    registry.Entry
				incomingEntry registry.Entry
			)

			BeforeEach(func() {
				localEntry = registry.Entry{
					RepoID: "r1", Path: "/local/r1", RemoteURL: "git@gh/r1.git",
					Branch: "main", Type: "checkout", Status: registry.StatusPresent,
				}
				incomingEntry = registry.Entry{
					RepoID: "r1", Path: "/bundle/r1", RemoteURL: "git@gh/r1.git",
					Branch: "main", Type: "checkout", Status: registry.StatusPresent,
				}
			})

			It("bundle policy overwrites local entry on path conflict", func() {
				cfg := &config.Config{
					Registry: &registry.Registry{Entries: []registry.Entry{localEntry}},
				}
				bundled := &registry.Registry{Entries: []registry.Entry{incomingEntry}}
				mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyBundle)
				got := cfg.Registry.FindByRepoID("r1")
				Expect(got).NotTo(BeNil())
				Expect(got.Path).To(Equal("/bundle/r1"))
			})

			It("skip policy keeps local entry on path conflict", func() {
				cfg := &config.Config{
					Registry: &registry.Registry{Entries: []registry.Entry{localEntry}},
				}
				bundled := &registry.Registry{Entries: []registry.Entry{incomingEntry}}
				mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicySkip)
				got := cfg.Registry.FindByRepoID("r1")
				Expect(got).NotTo(BeNil())
				Expect(got.Path).To(Equal("/local/r1"))
			})

			It("local policy keeps local entry on path conflict", func() {
				cfg := &config.Config{
					Registry: &registry.Registry{Entries: []registry.Entry{localEntry}},
				}
				bundled := &registry.Registry{Entries: []registry.Entry{incomingEntry}}
				mergeImportedRegistry(cfg, importModeMerge, true, bundled, importConflictPolicyLocal)
				got := cfg.Registry.FindByRepoID("r1")
				Expect(got).NotTo(BeNil())
				Expect(got.Path).To(Equal("/local/r1"))
			})
		})
	})
})
