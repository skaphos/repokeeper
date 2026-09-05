// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"

	"github.com/skaphos/repokeeper/internal/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Compare wire data from two real processes, not two calls to the shared engine.
// Decoding planned into a bool treats an omitted false value like explicit false.
type syncPlanRecord struct {
	RepoID             string                        `json:"repo_id"`
	Path               string                        `json:"path"`
	Action             string                        `json:"action"`
	Outcome            string                        `json:"outcome"`
	OK                 bool                          `json:"ok"`
	Planned            bool                          `json:"planned"`
	Error              string                        `json:"error"`
	SkipReason         string                        `json:"skip_reason"`
	RemoteTrackingRefs model.RemoteTrackingRefStatus `json:"remote_tracking_refs"`
}

var _ = Describe("CLI and MCP sync plan parity", func() {
	DescribeTable("returns matching plans without modifying the workspace", func(filter string, updateLocal, noUpstream bool, expectedPaths []string) {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		clean := workspace.Repositories["clean metadata repository"]
		Expect(runGit(ctx, workspace, clean.CheckoutPath, "refresh tracking state for parity", "fetch", "origin")).To(Succeed())
		Expect(runGit(ctx, workspace, clean.CheckoutPath, "seed stale tracking ref for parity", "update-ref", "refs/remotes/origin/merged", clean.BaselineHEAD)).To(Succeed())
		if noUpstream {
			Expect(runGit(ctx, workspace, clean.CheckoutPath, "remove upstream for parity", "branch", "--unset-upstream")).To(Succeed())
		}
		before, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		session, err := startMCPSession(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(session.Close()).To(Succeed()) })
		result := invokeMCP(session, "plan_sync", map[string]any{"filter": filter, "update_local": updateLocal})
		Expect(requireToolSuccess(result)).To(Succeed())
		mcpPlan, err := decodeStructured[struct {
			Plan []syncPlanRecord `json:"plan"`
		}](result)
		Expect(err).NotTo(HaveOccurred())
		afterMCP, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(afterMCP).To(Equal(before))

		args := []string{"reconcile", "--dry-run", "-o", "json", "--only", filter}
		if updateLocal {
			args = append(args, "--update-local")
		}
		cli := runRepoKeeper(ctx, workspace, "compare CLI dry-run plan", args...)
		Expect(requireDomainExit(cli, 1)).To(Succeed(), cli.Diagnostics())
		var cliPlan []syncPlanRecord
		Expect(json.Unmarshal(cli.Stdout, &cliPlan)).To(Succeed(), cli.Diagnostics())
		paths := make([]string, 0, len(cliPlan))
		for _, entry := range cliPlan {
			paths = append(paths, semanticPath(workspace.WorkspaceRoot, entry.Path))
		}
		Expect(paths).To(ConsistOf(expectedPaths))
		Expect(mcpPlan.Plan).To(ConsistOf(cliPlan), "MCP plan must predict the same targets, outcomes, reasons and actions as CLI dry-run")
		// Pin the expected behavior too: agreement alone could hide a shared bug.
		for _, entry := range cliPlan {
			switch semanticPath(workspace.WorkspaceRoot, entry.Path) {
			case "missing-repo":
				Expect(entry.Outcome).To(Equal("skipped_missing"))
				Expect(entry.Error).NotTo(BeEmpty())
				Expect(entry.OK).To(BeFalse())
			case "dirty-repo":
				if updateLocal {
					Expect(entry.Outcome).To(Equal("skipped_local_update"))
					Expect(entry.SkipReason).To(Equal("dirty working tree"))
				} else {
					Expect(entry.Outcome).To(Equal("planned_fetch"))
				}
			case "clean repo":
				Expect(entry.RemoteTrackingRefs.StaleCount).To(Equal(1))
				Expect(entry.RemoteTrackingRefs.Stale).To(ConsistOf("origin/merged"))
				if noUpstream && updateLocal {
					Expect(entry.Outcome).To(Equal("skipped_local_update"))
					Expect(entry.SkipReason).To(Equal("branch is not tracking an upstream"))
				} else {
					Expect(entry.Outcome).To(Equal("planned_fetch"))
				}
			}
		}
		afterCLI, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(afterCLI).To(Equal(before))
	},
		Entry("fetch including a leading missing entry", "all", false, false, []string{"missing-repo", "clean repo", "dirty-repo"}),
		Entry("local updates and dirty skip reason", "all", true, false, []string{"missing-repo", "clean repo", "dirty-repo"}),
		Entry("missing upstream skip reason", "all", true, true, []string{"missing-repo", "clean repo", "dirty-repo"}),
		Entry("dirty filter", "dirty", true, false, []string{"missing-repo", "dirty-repo"}),
		Entry("clean filter", "clean", false, false, []string{"missing-repo", "clean repo"}),
		Entry("missing filter", "missing", false, false, []string{"missing-repo"}),
	)
})
