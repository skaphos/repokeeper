// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type normalizedCLIOutcome struct {
	Repositories []normalizedCLIRepository
	Registry     []string
}

type normalizedCLIRepository struct {
	RepoID        string
	Path          string
	Branch        string
	Tracking      string
	HEAD          string
	Dirty         bool
	ContentHashes []string
}

var _ = Describe("Real CLI workflows determinism", func() {
	It("produces five semantically identical fresh outcomes", func() {
		var baseline normalizedCLIOutcome
		for run := 0; run < 5; run++ {
			workspace, _, err := materializeCanonical(context.Background(), GinkgoT().TempDir(), SeedAllEntries)
			Expect(err).NotTo(HaveOccurred())
			scan := runRepoKeeper(context.Background(), workspace, fmt.Sprintf("determinism scan %d", run+1), "scan", "--roots", workspace.WorkspaceRoot, "--format", "json")
			Expect(requireDomainExit(scan, 1)).To(Succeed(), scan.Diagnostics())
			status := runRepoKeeper(context.Background(), workspace, fmt.Sprintf("determinism status %d", run+1), "get", "repos", "--format", "json")
			Expect(requireDomainExit(status, 2)).To(Succeed(), status.Diagnostics())
			response, err := decodeStatusJSON(status)
			Expect(err).NotTo(HaveOccurred())
			outcome := normalizedCLIOutcome{}
			heads := map[string]string{}
			semanticIDs := map[string]string{}
			contentHashes := map[string][]string{}
			for _, materialized := range workspace.Repositories {
				heads[materialized.CheckoutPath] = materialized.BaselineHEAD
				semanticIDs[materialized.CheckoutPath] = materialized.RepoID
				for path, hash := range materialized.DirtyHashes {
					contentHashes[materialized.CheckoutPath] = append(contentHashes[materialized.CheckoutPath], path+"="+hash)
				}
				sort.Strings(contentHashes[materialized.CheckoutPath])
			}
			for _, repository := range response.Repos {
				hashes := contentHashes[repository.Path]
				dirty := repository.Worktree != nil && repository.Worktree.Dirty
				repoID := repository.RepoID
				if semanticIDs[repository.Path] != "" {
					repoID = semanticIDs[repository.Path]
				}
				outcome.Repositories = append(outcome.Repositories, normalizedCLIRepository{RepoID: repoID, Path: semanticPath(workspace.WorkspaceRoot, repository.Path), Branch: repository.Head.Branch, Tracking: string(repository.Tracking.Status), HEAD: heads[repository.Path], Dirty: dirty, ContentHashes: hashes})
			}
			sort.Slice(outcome.Repositories, func(i, j int) bool { return outcome.Repositories[i].Path < outcome.Repositories[j].Path })
			_, reg, err := reloadWorkspaceState(workspace)
			Expect(err).NotTo(HaveOccurred())
			for _, entry := range reg.Entries {
				outcome.Registry = append(outcome.Registry, semanticPath(workspace.WorkspaceRoot, entry.Path)+"="+string(entry.Status))
			}
			sort.Strings(outcome.Registry)
			if run == 0 {
				baseline = outcome
			} else {
				Expect(outcome).To(Equal(baseline), "run %d differed", run+1)
			}
		}
	})
})
