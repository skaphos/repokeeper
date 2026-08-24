// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func canonicalRecipe() (WorkspaceRecipe, error) {
	return loadRecipe(filepath.Join(moduleRoot, "test", "e2e", "testdata", "recipes", "canonical.json"))
}

func materializeCanonical(ctx context.Context, root string, mode RegistrySeedMode) (*MaterializedWorkspace, WorkspaceRecipe, error) {
	recipe, err := canonicalRecipe()
	if err != nil {
		return nil, WorkspaceRecipe{}, err
	}
	workspace, err := materializeRecipe(ctx, recipe, root, mode)
	return workspace, recipe, err
}

var _ = Describe("Canonical recipe", func() {
	It("materializes CLI and missing-only MCP startup modes", func() {
		for _, mode := range []RegistrySeedMode{SeedAllEntries, SeedMissingOnly} {
			workspace, recipe, err := materializeCanonical(context.Background(), GinkgoT().TempDir(), mode)
			Expect(err).NotTo(HaveOccurred())
			Expect(workspace.Repositories).To(HaveLen(2))
			Expect(workspace.MissingEntries).To(HaveLen(1))
			_, reg, err := reloadWorkspaceState(workspace)
			Expect(err).NotTo(HaveOccurred())
			expected := len(recipe.MissingEntries)
			if mode == SeedAllEntries {
				expected += len(recipe.Repositories)
			}
			Expect(reg.Entries).To(HaveLen(expected))
		}
	})
})
