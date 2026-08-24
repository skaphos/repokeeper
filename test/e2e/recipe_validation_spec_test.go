// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recipe preflight", func() {
	valid := func() WorkspaceRecipe {
		return WorkspaceRecipe{SchemaVersion: 1, Name: "valid", Repositories: []RepositoryRecipe{{Name: "repo", RemotePath: "repo.git", CheckoutPath: "repo", Files: map[string]string{"README.md": "ok\n"}, CurrentBranch: "main", Upstream: UpstreamRecipe{Remote: "origin", Branch: "main"}}}, MissingEntries: []MissingEntryRecipe{{Name: "missing", RepoID: "missing", CheckoutID: "missing", Path: "missing", RemoteURL: "file:///missing", Status: "missing"}}}
	}

	DescribeTable("rejects invalid portable recipes",
		func(mutate func(*WorkspaceRecipe), field string) {
			recipe := valid()
			mutate(&recipe)
			err := validateRecipe(recipe)
			Expect(err).To(MatchError(ContainSubstring(field)))
		},
		Entry("absolute checkout", func(recipe *WorkspaceRecipe) {
			recipe.Repositories[0].CheckoutPath = filepath.Join(string(filepath.Separator), "escape")
		}, "checkout_path"),
		Entry("traversal", func(recipe *WorkspaceRecipe) { recipe.Repositories[0].CheckoutPath = "../escape" }, "checkout_path"),
		Entry("backslash", func(recipe *WorkspaceRecipe) { recipe.Repositories[0].CheckoutPath = `repo\\nested` }, "checkout_path"),
		Entry("drive path", func(recipe *WorkspaceRecipe) { recipe.Repositories[0].CheckoutPath = `C:/escape` }, "checkout_path"),
		Entry("UNC path", func(recipe *WorkspaceRecipe) { recipe.Repositories[0].CheckoutPath = `//server/share` }, "checkout_path"),
		Entry("duplicate names", func(recipe *WorkspaceRecipe) { recipe.MissingEntries[0].Name = "repo" }, "duplicate name"),
		Entry("path collision", func(recipe *WorkspaceRecipe) { recipe.MissingEntries[0].Path = "repo" }, "collides"),
		Entry("unknown branch", func(recipe *WorkspaceRecipe) { recipe.Repositories[0].CurrentBranch = "absent" }, "current_branch"),
		Entry("unknown relationship", func(recipe *WorkspaceRecipe) {
			recipe.Repositories[0].Metadata = &MetadataRecipe{Related: []RelationshipRecipe{{Repository: "absent"}}}
		}, "related_repositories"),
	)

	It("strictly decodes unknown and trailing fields", func() {
		root := GinkgoT().TempDir()
		path := filepath.Join(root, "recipe.json")
		Expect(os.WriteFile(path, []byte(`{"schema_version":1,"name":"bad","repositories":[],"missing_entries":[],"config":{},"surprise":true}`), 0o600)).To(Succeed())
		_, err := loadRecipe(path)
		Expect(err).To(MatchError(ContainSubstring("unknown field")))
		Expect(os.WriteFile(path, []byte(`{"schema_version":1,"name":"bad","repositories":[],"missing_entries":[],"config":{}} {}`), 0o600)).To(Succeed())
		_, err = loadRecipe(path)
		Expect(err).To(MatchError(ContainSubstring("trailing JSON")))
	})

	It("fails before materialization writes anything", func() {
		recipe := valid()
		recipe.Repositories[0].CheckoutPath = "../escape"
		parent := GinkgoT().TempDir()
		root := filepath.Join(parent, "scenario")
		_, err := materializeRecipe(context.Background(), recipe, root, SeedAllEntries)
		Expect(err).To(HaveOccurred())
		_, statErr := os.Stat(root)
		Expect(os.IsNotExist(statErr)).To(BeTrue())
		entries, readErr := os.ReadDir(parent)
		Expect(readErr).NotTo(HaveOccurred())
		Expect(entries).To(BeEmpty())
	})

	It("round-trips valid JSON into typed recipes", func() {
		data, err := json.Marshal(valid())
		Expect(err).NotTo(HaveOccurred())
		path := filepath.Join(GinkgoT().TempDir(), "valid.json")
		Expect(os.WriteFile(path, data, 0o600)).To(Succeed())
		loaded, err := loadRecipe(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.TrimSpace(loaded.Name)).To(Equal("valid"))
	})
})
