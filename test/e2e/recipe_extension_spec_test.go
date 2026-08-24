// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recipe extensibility", func() {
	It("uses the shared lifecycle for a second declarative topology", func() {
		workspace, recipe, err := materializeExtension(context.Background(), GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		Expect(recipe.Repositories).To(HaveLen(1))
		result := runRepoKeeper(context.Background(), workspace, "extension scan", "scan", "--roots", workspace.WorkspaceRoot, "--format", "json")
		Expect(requireDomainExit(result, 0)).To(Succeed(), result.Diagnostics())
		response, err := decodeScanJSON(result)
		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(HaveLen(1))
		Expect(semanticPath(workspace.WorkspaceRoot, response[0].Path)).To(Equal("nested/service extension"))
	})

	It("reports field-qualified inconsistency before writes", func() {
		recipe, err := extensionRecipe()
		Expect(err).NotTo(HaveOccurred())
		recipe.Repositories[0].Upstream.Branch = "undeclared"
		parent := GinkgoT().TempDir()
		root := filepath.Join(parent, "not-created")
		_, err = materializeRecipe(context.Background(), recipe, root, SeedAllEntries)
		Expect(err).To(MatchError(ContainSubstring("upstream.branch")))
		_, statErr := os.Stat(root)
		Expect(os.IsNotExist(statErr)).To(BeTrue())
	})

	It("includes the full diagnostic contract for unexpected exits", func() {
		workspace, _, err := materializeExtension(context.Background(), GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		result := runRepoKeeper(context.Background(), workspace, "forced extension failure", "definitely-not-a-command")
		err = requireDomainExit(result, 0)
		Expect(err).To(HaveOccurred())
		for _, field := range []string{"operation: forced extension failure", "command:", "working directory:", "scenario root:", "config path:", "exit:", "stdout:", "stderr:"} {
			Expect(err.Error()).To(ContainSubstring(field))
		}
	})
})
