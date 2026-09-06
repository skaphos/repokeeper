// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/skaphos/repokeeper/v2/internal/contract"
	"github.com/skaphos/repokeeper/v2/internal/mcpserver"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Adapter JSON on the wire", func() {
	It("emits envelopes for every read CLI surface without changing workspace state", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		clean := workspace.Repositories["clean metadata repository"].CheckoutPath
		before, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		cases := []struct {
			args []string
			key  string
		}{
			{[]string{"get"}, "repos"}, {[]string{"get", "repos"}, "repos"}, {[]string{"get", "repo"}, "repos"},
			{[]string{"describe", clean}, "repo"}, {[]string{"describe", "repo", clean}, "repo"},
			{[]string{"scan", "--roots", workspace.WorkspaceRoot, "--write-registry=false"}, "repos"},
			{[]string{"reconcile", "--dry-run"}, "results"},
			{[]string{"reconcile", "repos", "--dry-run"}, "results"},
			{[]string{"reconcile", "repo", "--dry-run"}, "results"},
			{[]string{"repair", "upstream", "--dry-run"}, "results"},
			{[]string{"label", clean}, "labels"}, {[]string{"version"}, "version"},
		}
		for _, tc := range cases {
			By(strings.Join(tc.args, " "))
			result := runRepoKeeper(ctx, workspace, "read adapter surface", append(tc.args, "-o", "json")...)
			Expect(result.LaunchError).NotTo(HaveOccurred())
			Expect(result.TimedOut).To(BeFalse(), result.Diagnostics())
			envelope := wireEnvelope(result.Stdout)
			Expect(envelope).To(HaveKey(tc.key), result.Diagnostics())
			Expect(string(envelope[tc.key])).NotTo(Equal("null"))
			after, snapshotErr := captureWorkspaceSnapshot(ctx, workspace)
			Expect(snapshotErr).NotTo(HaveOccurred())
			Expect(after).To(Equal(before), strings.Join(tc.args, " "))
		}
	})

	It("keeps fatal errors separate from completed reports with repository failures", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		fatal := runRepoKeeper(ctx, workspace, "invalid selector", "describe", "absent-checkout", "-o", "json")
		Expect(fatal.ExitCode).NotTo(BeZero())
		Expect(fatal.Stdout).To(BeEmpty(), fatal.Diagnostics())
		Expect(fatal.Stderr).NotTo(BeEmpty())
		report := runRepoKeeper(ctx, workspace, "missing repository report", "get", "-o", "json")
		Expect(report.ExitCode).NotTo(BeZero())
		var repos []map[string]any
		Expect(json.Unmarshal(wireEnvelope(report.Stdout)["repos"], &repos)).To(Succeed())
		Expect(repos).To(ContainElement(HaveKey("error")))
	})

	It("keeps every read MCP tool stable and preserves structured/text parity", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		session, err := startMCPSession(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(session.Close()).To(Succeed()) })
		before, err := captureWorkspaceSnapshot(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		cases, err := orderedMCPToolCases()
		Expect(err).NotTo(HaveOccurred())
		for _, name := range mcpserver.ReadOnlyToolNames() {
			By(name)
			tc, caseErr := findMCPToolCase(cases, name)
			Expect(caseErr).NotTo(HaveOccurred())
			result := invokeMCPCase(session, workspace, tc)
			structured, marshalErr := json.Marshal(result.StructuredContent)
			Expect(marshalErr).NotTo(HaveOccurred())
			Expect(result.Content).To(HaveLen(1))
			text, ok := result.Content[0].(mcp.TextContent)
			Expect(ok).To(BeTrue())
			wireEnvelope([]byte(text.Text))
			wireEnvelope(structured)
			Expect(text.Text).To(MatchJSON(structured))
			after, snapshotErr := captureWorkspaceSnapshot(ctx, workspace)
			Expect(snapshotErr).NotTo(HaveOccurred())
			Expect(after).To(Equal(before), name)
		}
		failure := invokeMCP(session, "get_repository_context", map[string]any{"repo": "absent-checkout"})
		Expect(failure.IsError).To(BeTrue())
		Expect(failure.StructuredContent).To(BeNil())
		_, reg, err := reloadWorkspaceState(workspace)
		Expect(err).NotTo(HaveOccurred())
		clean := workspace.Repositories["clean metadata repository"].CheckoutPath
		var repoID string
		for _, entry := range reg.Entries {
			if entry.Path == clean {
				repoID = entry.RepoID
			}
		}
		Expect(repoID).NotTo(BeEmpty())
		for _, uri := range []string{"repokeeper://config", "repokeeper://registry", "repokeeper://repo/" + repoID, "repokeeper://repo/" + repoID + "/metadata"} {
			req := mcp.ReadResourceRequest{}
			req.Params.URI = uri
			resource, readErr := session.client.ReadResource(ctx, req)
			Expect(readErr).NotTo(HaveOccurred(), uri)
			Expect(resource.Contents).To(HaveLen(1))
			text, ok := resource.Contents[0].(mcp.TextResourceContents)
			Expect(ok).To(BeTrue())
			wireEnvelope([]byte(text.Text))
			after, snapshotErr := captureWorkspaceSnapshot(ctx, workspace)
			Expect(snapshotErr).NotTo(HaveOccurred())
			Expect(after).To(Equal(before), uri)
		}
	})

	It("preserves shared scan identities and complete execution records across CLI and MCP", func() {
		ctx := context.Background()
		workspace, _, err := materializeCanonical(ctx, GinkgoT().TempDir(), SeedAllEntries)
		Expect(err).NotTo(HaveOccurred())
		session, err := startMCPSession(ctx, workspace)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(session.Close()).To(Succeed()) })
		scan := runRepoKeeper(ctx, workspace, "CLI scan parity", "scan", "--roots", workspace.WorkspaceRoot, "--write-registry=false", "-o", "json")
		var cliRepos []map[string]any
		Expect(json.Unmarshal(wireEnvelope(scan.Stdout)["repos"], &cliRepos)).To(Succeed())
		mcpScan := invokeMCP(session, "scan_workspace", map[string]any{"roots": []string{workspace.WorkspaceRoot}})
		Expect(requireToolSuccess(mcpScan)).To(Succeed())
		scanned, err := decodeStructured[struct {
			Scan struct {
				Repos []map[string]any `json:"repos"`
			} `json:"scan"`
		}](mcpScan)
		Expect(err).NotTo(HaveOccurred())
		// MCP scan reports registry outcomes; CLI scan carries full health.
		// Their shared identity fields must still have identical values/types.
		identities := func(records []map[string]any) []map[string]any {
			out := make([]map[string]any, 0, len(records))
			for _, record := range records {
				out = append(out, map[string]any{"repo_id": record["repo_id"], "path": record["path"]})
			}
			return out
		}
		Expect(cliRepos).NotTo(BeEmpty())
		Expect(identities(scanned.Scan.Repos)).To(ConsistOf(identities(cliRepos)))
		cli := runRepoKeeper(ctx, workspace, "CLI execute parity", "reconcile", "--only", "clean", "-o", "json")
		var executed []map[string]any
		Expect(json.Unmarshal(wireEnvelope(cli.Stdout)["results"], &executed)).To(Succeed())
		result := invokeMCP(session, "execute_sync", map[string]any{"filter": "clean", "confirm": true})
		Expect(requireToolSuccess(result)).To(Succeed())
		mcpExecuted, err := decodeStructured[struct {
			Results []map[string]any `json:"results"`
		}](result)
		Expect(err).NotTo(HaveOccurred())
		Expect(executed).NotTo(BeEmpty())
		Expect(mcpExecuted.Results).To(ConsistOf(executed))
	})
})

func wireEnvelope(data []byte) map[string]json.RawMessage {
	var envelope map[string]json.RawMessage
	Expect(json.Unmarshal(data, &envelope)).To(Succeed(), string(data))
	Expect(envelope).To(HaveKey("apiVersion"))
	Expect(string(envelope["apiVersion"])).To(Equal(`"` + contract.APIVersion + `"`))
	return envelope
}
