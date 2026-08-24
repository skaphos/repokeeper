// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Real MCP stdio", func() {
	It("negotiates the real process and exercises every live registered tool", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedMissingOnly)
		Expect(err).NotTo(HaveOccurred())
		baseline, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		cases, err := orderedMCPToolCases()
		Expect(err).NotTo(HaveOccurred())

		session, err := startMCPSession(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(session.Close()).To(Succeed()) })
		callContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		tools, err := session.listTools(callContext)
		cancel()
		Expect(err).NotTo(HaveOccurred())
		Expect(compareLiveToolCoverage(tools, cases)).To(Succeed())
		Expect(tools).To(HaveLen(14))

		scanCase, err := findMCPToolCase(cases, "scan_workspace")
		Expect(err).NotTo(HaveOccurred())
		invokeMCPCase(session, workspace, scanCase)
		_, reg, err := reloadWorkspaceState(workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(reg.Entries).To(HaveLen(3))

		for _, name := range []string{"list_repositories", "get_repository_context", "get_workspace_config", "build_workspace_inventory", "select_repositories", "get_repo_metadata", "get_authoritative_paths", "get_related_repositories", "plan_sync"} {
			toolCase, caseErr := findMCPToolCase(cases, name)
			Expect(caseErr).NotTo(HaveOccurred())
			invokeMCPCase(session, workspace, toolCase)
		}

		executeCase, err := findMCPToolCase(cases, "execute_sync")
		Expect(err).NotTo(HaveOccurred())
		beforeRefusal, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		refusalArguments := executeCase.Arguments(workspace)
		refusalArguments["confirm"] = false
		refusal := invokeMCP(session, executeCase.Name, refusalArguments)
		Expect(refusal.IsError).To(BeTrue())
		Expect(fmt.Sprint(refusal.Content)).To(ContainSubstring("safety gate"))
		afterRefusal, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(afterRefusal).To(Equal(beforeRefusal))
		executeResult := invokeMCPCase(session, workspace, executeCase)
		results, err := requireListField(executeResult, "results")
		Expect(err).NotTo(HaveOccurred())
		Expect(results).NotTo(BeEmpty())
		cleanRepository := workspace.Repositories["clean metadata repository"]
		remoteHEAD, err := gitOutput(ctx, workspace, cleanRepository.RemotePath, "read advanced remote HEAD", "rev-parse", "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())
		fetchedHEAD, err := gitOutput(ctx, workspace, cleanRepository.CheckoutPath, "read fetched remote-tracking HEAD", "rev-parse", "refs/remotes/origin/main")
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.TrimSpace(fetchedHEAD)).To(Equal(strings.TrimSpace(remoteHEAD)))

		setLabelsCase, err := findMCPToolCase(cases, "set_labels")
		Expect(err).NotTo(HaveOccurred())
		invokeMCPCase(session, workspace, setLabelsCase)
		_, reg, err = reloadWorkspaceState(workspace)
		Expect(err).NotTo(HaveOccurred())
		labelFound := false
		for _, entry := range reg.Entries {
			if entry.Path == workspace.Repositories["clean metadata repository"].CheckoutPath {
				labelFound = entry.Labels["e2e"] == "true"
			}
		}
		Expect(labelFound).To(BeTrue())

		addCase, err := findMCPToolCase(cases, "add_repository")
		Expect(err).NotTo(HaveOccurred())
		invokeMCPCase(session, workspace, addCase)
		addedPath := addedRepositoryPath(workspace)
		Expect(assertContained(workspace.Root, addedPath)).To(Succeed())
		info, err := os.Stat(addedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
		addedHEAD, err := gitOutput(ctx, workspace, addedPath, "snapshot added repository HEAD", "rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())

		removeCase, err := findMCPToolCase(cases, "remove_repository")
		Expect(err).NotTo(HaveOccurred())
		beforeRemoveRefusal, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		refuseRemove := removeCase.Arguments(workspace)
		refuseRemove["confirm"] = false
		refusal = invokeMCP(session, removeCase.Name, refuseRemove)
		Expect(refusal.IsError).To(BeTrue())
		Expect(fmt.Sprint(refusal.Content)).To(ContainSubstring("safety gate"))
		_, err = os.Stat(addedPath)
		Expect(err).NotTo(HaveOccurred())
		addedHEADAfterRefusal, err := gitOutput(ctx, workspace, addedPath, "verify added repository refusal HEAD", "rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		Expect(addedHEADAfterRefusal).To(Equal(addedHEAD))
		afterRemoveRefusal, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(afterRemoveRefusal).To(Equal(beforeRemoveRefusal))
		invokeMCPCase(session, workspace, removeCase)
		_, err = os.Stat(addedPath)
		Expect(os.IsNotExist(err)).To(BeTrue())

		Expect(session.Close()).To(Succeed())
		Expect(validateJSONRPCFrames(session.stdout.Bytes())).To(Succeed())
		Expect(strings.TrimSpace(string(session.stderr.Bytes()))).To(BeEmpty())

		after, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		for name, repositoryBefore := range baseline.Repositories {
			repositoryAfter := after.Repositories[name]
			Expect(repositoryAfter.HEAD).To(Equal(repositoryBefore.HEAD), name)
			Expect(repositoryAfter.Porcelain).To(Equal(repositoryBefore.Porcelain), name)
			Expect(repositoryAfter.Files).To(Equal(repositoryBefore.Files), name)
		}
		_, reg, err = reloadWorkspaceState(workspace)
		Expect(err).NotTo(HaveOccurred())
		addedStillRegistered := false
		for _, entry := range reg.Entries {
			Expect(assertContained(workspace.Root, entry.Path)).To(Succeed(), entry.Path)
			if entry.Path == addedPath {
				addedStillRegistered = true
			}
		}
		Expect(addedStillRegistered).To(BeFalse())
	})
})

func invokeMCPCase(session *mcpSession, workspace *MaterializedWorkspace, toolCase MCPToolCase) *mcp.CallToolResult {
	result := invokeMCP(session, toolCase.Name, toolCase.Arguments(workspace))
	Expect(toolCase.Assert(result)).To(Succeed(), "tool %s result: %#v", toolCase.Name, result)
	return result
}

func invokeMCP(session *mcpSession, name string, arguments map[string]any) *mcp.CallToolResult {
	ctx, cancel := context.WithTimeout(session.ctx, 10*time.Second)
	defer cancel()
	result, err := session.callTool(ctx, name, arguments)
	Expect(err).NotTo(HaveOccurred(), "call MCP tool %s", name)
	return result
}
