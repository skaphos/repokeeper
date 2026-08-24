// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/test/e2e/internal/compatibility"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Compatibility command and evidence", func() {
	It("emits stable JSON-only routine and release matrices", func() {
		for _, scope := range []string{"routine", "release"} {
			stdout, stderr, err := runCompatibilityCommand("matrix", "--scope", scope)
			Expect(err).NotTo(HaveOccurred(), stderr)
			Expect(strings.TrimSpace(stderr)).To(BeEmpty())
			var result compatibility.MatrixResult
			Expect(json.Unmarshal(stdout, &result)).To(Succeed())
			Expect(result.Scope).To(Equal(scope))
			if scope == "routine" {
				Expect(result.Cells).To(HaveLen(4))
			} else {
				Expect(result.Cells).To(HaveLen(12))
			}
		}
	})

	It("binds a complete evidence set to one immutable tag and commit", func() {
		declaration, err := compatibility.Load(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		directory := GinkgoT().TempDir()
		writeCompleteEvidence(declaration, directory, "v1.2.3", "abc123")
		summary, err := compatibility.VerifyEvidence(declaration, directory, "v1.2.3", "abc123")
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.Complete).To(BeTrue())
	})

	DescribeTable("rejects incomplete release evidence", func(mutate func(compatibility.Declaration, string), field string) {
		declaration, err := compatibility.Load(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		directory := GinkgoT().TempDir()
		writeCompleteEvidence(declaration, directory, "v1.2.3", "abc123")
		mutate(declaration, directory)
		summary, err := compatibility.VerifyEvidence(declaration, directory, "v1.2.3", "abc123")
		Expect(err).To(HaveOccurred())
		Expect(summary.Complete).To(BeFalse())
		Expect(err.Error()).To(ContainSubstring(field))
	},
		Entry("missing", func(d compatibility.Declaration, dir string) {
			Expect(os.Remove(filepath.Join(dir, compatibility.CellKey(d.Cells[0])+".json"))).To(Succeed())
		}, "missing="),
		Entry("duplicate", func(d compatibility.Declaration, dir string) {
			data, err := os.ReadFile(filepath.Join(dir, compatibility.CellKey(d.Cells[0])+".json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(dir, "duplicate.json"), data, 0o600)).To(Succeed())
		}, "duplicate="),
		Entry("unexpected", func(_ compatibility.Declaration, dir string) {
			result := compatibility.CompatibilityResult{CellKey: "plan9-git-2.55", CandidateTag: "v1.2.3", Commit: "abc123", Status: "passed", InputDigests: map[string]string{}}
			_, err := compatibility.WriteEvidence(filepath.Join(dir, "unexpected.json"), result)
			Expect(err).NotTo(HaveOccurred())
		}, "unexpected="),
		Entry("failed", func(d compatibility.Declaration, dir string) {
			rewriteEvidence(d, d.Cells[0], dir, "failed", "git version "+d.Cells[0].GitPatch, "v1.2.3", "abc123")
		}, "failed="),
		Entry("skipped", func(d compatibility.Declaration, dir string) {
			rewriteEvidence(d, d.Cells[0], dir, "skipped", "git version "+d.Cells[0].GitPatch, "v1.2.3", "abc123")
		}, "skipped="),
		Entry("version mismatch", func(d compatibility.Declaration, dir string) {
			rewriteEvidence(d, d.Cells[0], dir, "passed", "git version 0.0.0", "v1.2.3", "abc123")
		}, "mismatched="),
		Entry("tag mismatch", func(d compatibility.Declaration, dir string) {
			rewriteEvidence(d, d.Cells[0], dir, "passed", "git version "+d.Cells[0].GitPatch, "v9.9.9", "abc123")
		}, "mismatched="),
		Entry("commit mismatch", func(d compatibility.Declaration, dir string) {
			rewriteEvidence(d, d.Cells[0], dir, "passed", "git version "+d.Cells[0].GitPatch, "v1.2.3", "different")
		}, "mismatched="),
	)
})

func runCompatibilityCommand(arguments ...string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := append([]string{"run", "-tags", "integration", "./test/e2e/cmd/compatibility"}, arguments...)
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = moduleRoot
	command.Env = append(os.Environ(), "GOCACHE=/tmp/repokeeper-go-build")
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return []byte(stdout.String()), stderr.String(), err
}

func writeCompleteEvidence(declaration compatibility.Declaration, directory, tag, commit string) {
	for _, cell := range declaration.Cells {
		rewriteEvidence(declaration, cell, directory, "passed", "git version "+cell.GitPatch, tag, commit)
	}
}
func rewriteEvidence(_ compatibility.Declaration, cell compatibility.Cell, directory, status, actual, tag, commit string) {
	digests := map[string]string{"provisioner": cell.Provisioner.SHA256}
	if cell.RootFS != nil {
		digests["rootfs"] = cell.RootFS.SHA256
		for _, prerequisite := range cell.RootFS.BuildPrerequisites {
			digests["package:"+prerequisite.Name] = prerequisite.Version
		}
	}
	_, err := compatibility.WriteEvidence(filepath.Join(directory, compatibility.CellKey(cell)+".json"), compatibility.CompatibilityResult{CellKey: compatibility.CellKey(cell), CandidateTag: tag, Commit: commit, Environment: cell.Environment, RunnerLabel: cell.RunnerLabel, GitMinor: cell.GitMinor, ExpectedGit: cell.GitPatch, ActualGit: actual, InputDigests: digests, Status: status})
	Expect(err).NotTo(HaveOccurred())
}
