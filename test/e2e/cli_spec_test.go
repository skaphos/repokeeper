// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/skaphos/repokeeper/internal/registry"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Real CLI workflows", Ordered, func() {
	It("scans and reports canonical state without modifying worktrees", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		before, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())

		scan := runRepoKeeper(ctx, workspace, "cli scan", "scan", "--roots", workspace.WorkspaceRoot, "--write-registry", "--format", "json")
		Expect(requireDomainExit(scan, 1)).To(Succeed(), scan.Diagnostics())
		scanResponse, err := decodeScanJSON(scan)
		Expect(err).NotTo(HaveOccurred())
		Expect(scanResponse).To(HaveLen(2))
		Expect(string(scan.Stderr)).To(ContainSubstring("scan completed: 2 repos"))

		status := runRepoKeeper(ctx, workspace, "cli status", "get", "repos", "--format", "json")
		Expect(requireDomainExit(status, 2)).To(Succeed(), status.Diagnostics())
		statusResponse, err := decodeStatusJSON(status)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusResponse.Repos).To(HaveLen(3))
		Expect(string(status.Stderr)).To(ContainSubstring("status completed: 3 repos"))

		_, reg, err := reloadWorkspaceState(workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(3))
		statuses := map[string]registry.EntryStatus{}
		for _, entry := range reg.Entries {
			statuses[semanticPath(workspace.WorkspaceRoot, entry.Path)] = entry.Status
		}
		Expect(statuses["clean repo"]).To(Equal(registry.StatusPresent))
		Expect(statuses["dirty-repo"]).To(Equal(registry.StatusPresent))
		Expect(statuses["missing-repo"]).To(Equal(registry.StatusMissing))

		after, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		for name, repositoryBefore := range before.Repositories {
			repositoryAfter := after.Repositories[name]
			Expect(repositoryAfter.HEAD).To(Equal(repositoryBefore.HEAD), name)
			Expect(repositoryAfter.Porcelain).To(Equal(repositoryBefore.Porcelain), name)
			Expect(repositoryAfter.Refs).To(Equal(repositoryBefore.Refs), name)
			Expect(repositoryAfter.Files).To(Equal(repositoryBefore.Files), name)
		}
		clean := workspace.Repositories["clean metadata repository"]
		Expect(before.Repositories[clean.Name].Porcelain).To(BeEmpty())
		dirty := workspace.Repositories["dirty repository"]
		data, err := os.ReadFile(filepath.Join(dirty.CheckoutPath, "tracked.txt"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("uncommitted bytes stay unchanged\n"))
		Expect(strings.TrimSpace(before.Repositories[dirty.Name].Porcelain)).NotTo(BeEmpty())
		Expect(assertContained(workspace.Root, workspace.ConfigPath, workspace.RegistryPath, clean.CheckoutPath, dirty.CheckoutPath, workspace.MissingEntries["missing repository"].Path)).To(Succeed())
	})
})
