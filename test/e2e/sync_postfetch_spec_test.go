// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// reconcile plans from local state, then executes the plan. Upstream commits
// that only the plan's fetch discovers must still be applied (#358).
var _ = Describe("reconcile --update-local with upstream changes", func() {
	It("applies commits that the plan's own fetch discovers", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		clean := workspace.Repositories["clean metadata repository"]
		// Start exactly up to date with upstream.
		Expect(runGit(ctx, workspace, clean.CheckoutPath, "refresh tracking state", "fetch", "origin")).To(Succeed())
		Expect(runGit(ctx, workspace, clean.CheckoutPath, "catch up with upstream", "merge", "--ff-only", "@{u}")).To(Succeed())
		localHEAD := headOf(ctx, workspace, clean.CheckoutPath)

		// Publish a new upstream commit from a second clone. The checkout's
		// remote-tracking ref is now stale, so pre-fetch state reads "up to date".
		other := filepath.Join(GinkgoT().TempDir(), "other")
		Expect(runGit(ctx, workspace, filepath.Dir(other), "clone upstream", "clone", clean.RemotePath, other)).To(Succeed())
		Expect(runGit(ctx, workspace, other, "commit upstream change", "commit", "--allow-empty", "-m", "upstream change")).To(Succeed())
		Expect(runGit(ctx, workspace, other, "push upstream change", "push", "origin", "HEAD")).To(Succeed())
		upstreamHEAD := headOf(ctx, workspace, other)
		Expect(upstreamHEAD).NotTo(Equal(localHEAD))

		cli := runRepoKeeper(ctx, workspace, "reconcile with local update", "reconcile", "--update-local", "--only", "clean", "--yes", "-o", "json")
		Expect(requireDomainExit(cli, 0, 1)).To(Succeed(), cli.Diagnostics())
		var envelope struct {
			Results []syncPlanRecord `json:"results"`
		}
		Expect(json.Unmarshal(cli.Stdout, &envelope)).To(Succeed(), cli.Diagnostics())
		var cleanResult *syncPlanRecord
		for i := range envelope.Results {
			if semanticPath(workspace.WorkspaceRoot, envelope.Results[i].Path) == "clean repo" {
				cleanResult = &envelope.Results[i]
			}
		}
		Expect(cleanResult).NotTo(BeNil(), cli.Diagnostics())
		Expect(cleanResult.Outcome).To(Equal("rebased"), cli.Diagnostics())
		Expect(cleanResult.SkipReason).To(BeEmpty(), cli.Diagnostics())
		Expect(headOf(ctx, workspace, clean.CheckoutPath)).To(Equal(upstreamHEAD), "local branch must include the upstream commit")
	})
})

func headOf(ctx context.Context, workspace *MaterializedWorkspace, dir string) string {
	GinkgoHelper()
	head, err := gitOutput(ctx, workspace, dir, "read HEAD", "rev-parse", "HEAD")
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(head)
}
