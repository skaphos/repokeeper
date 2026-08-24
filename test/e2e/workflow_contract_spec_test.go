// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var actionReferencePattern = regexp.MustCompile(`uses:\s+[^\s@]+@([0-9a-f]{40})(?:\s|$)`)

var _ = Describe("Compatibility workflow contracts", func() {
	It("pins runners, actions, permissions, and timeouts in every workflow", func() {
		for _, name := range []string{"ci.yml", "qualification.yml", "release.yml", "release-please.yml"} {
			path := filepath.Join(moduleRoot, ".github", "workflows", name)
			data, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			text := string(data)
			Expect(text).NotTo(MatchRegexp(`(?:ubuntu|macos|windows)-latest`), name)
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, "uses:") {
					Expect(actionReferencePattern.MatchString(strings.TrimSpace(line))).To(BeTrue(), "%s has unpinned action: %s", name, line)
				}
			}
			var document struct {
				Jobs map[string]struct {
					TimeoutMinutes *int `yaml:"timeout-minutes"`
				} `yaml:"jobs"`
			}
			Expect(yaml.Unmarshal(data, &document)).To(Succeed())
			for job, body := range document.Jobs {
				Expect(body.TimeoutMinutes).NotTo(BeNil(), "%s job %s has no timeout", name, job)
				Expect(*body.TimeoutMinutes).To(BeNumerically("<=", 30), "%s job %s", name, job)
			}
		}
	})

	It("keeps the compatibility matrix out of routine pull request CI", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "ci.yml"))
		Expect(err).NotTo(HaveOccurred())
		text := string(data)
		// The matrix provisions Git on four runner images per pull request, so
		// routine CI must not carry it. Qualification runs on the release
		// pull request instead, before a version is cut.
		for _, forbidden := range []string{"cmd/compatibility", "--scope routine", "--scope release", "wsl.exe"} {
			Expect(text).NotTo(ContainSubstring(forbidden), "ci.yml must not run the compatibility matrix")
		}
		for _, required := range []string{"test-integration-race", "-race"} {
			Expect(text).To(ContainSubstring(required))
		}
		Expect(text).To(ContainSubstring("permissions:\n  contents: read"))
	})

	It("gates the release pull request on the executable twelve-cell matrix", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "qualification.yml"))
		Expect(err).NotTo(HaveOccurred())
		text := string(data)
		// Runner labels come from the declaration rather than being written
		// here, and Validate pins each cell to an exact non-latest image.
		for _, required := range []string{"matrix --scope release", "fail-fast: false", "runs-on: ${{ matrix.cell.runner_label }}", "matrix.cell.environment == 'wsl'", "windows_to_wsl", "./test/e2e/cmd/compatibility", "REPOKEEPER_E2E_COMPATIBILITY_BINARY", "MSYS2_ARG_CONV_EXCL='*'", "needs: compatibility-matrix"} {
			Expect(text).To(ContainSubstring(required))
		}
		// Both jobs carry the guard, so a matrix expansion never runs for an
		// ordinary contributor pull request. The guard must check the head
		// repository too: a branch name alone is attacker-controlled, so a
		// fork could otherwise spend twelve runners by naming its branch after
		// the release-please one.
		guard := "if: github.event.pull_request.head.repo.full_name == github.repository && github.head_ref == 'release-please--branches--main'"
		Expect(strings.Count(text, guard)).To(Equal(2))
		Expect(text).NotTo(MatchRegexp(`if: github\.head_ref ==[^\n]*\n`), "the branch guard must also pin the head repository")
		Expect(text).To(ContainSubstring("permissions:\n  contents: read"))
		Expect(text).NotTo(ContainSubstring("secrets."))
		// A single always-reporting job carries a stable check name, so the
		// gate can be a required status check even though the per-cell jobs
		// are skipped on ordinary pull requests.
		for _, required := range []string{"name: Release Qualification", "needs: compatibility", "if: always()", `RESULT: ${{ needs.compatibility.result }}`} {
			Expect(text).To(ContainSubstring(required))
		}
		// The gate reports on every pull request, so it repeats the same
		// head-repository check rather than trusting the branch name.
		for _, required := range []string{`HEAD_REPO: ${{ github.event.pull_request.head.repo.full_name }}`, `THIS_REPO: ${{ github.repository }}`, `[ "$HEAD_REPO" != "$THIS_REPO" ]`} {
			Expect(text).To(ContainSubstring(required))
		}
	})

	It("gates every publisher on exact twelve-cell evidence", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "release.yml"))
		Expect(err).NotTo(HaveOccurred())
		text := string(data)
		for _, required := range []string{"matrix --scope release", "fail-fast: false", "Compatibility Evidence Gate", "verify-evidence", "needs: compatibility-evidence", "matrix.cell.environment == 'wsl'", "windows_to_wsl", "./test/e2e/cmd/compatibility", "REPOKEEPER_E2E_COMPATIBILITY_BINARY", "MSYS2_ARG_CONV_EXCL='*'", "wsl.exe --unregister", "contents: read", "contents: write", "id-token: write", "packages: write"} {
			Expect(text).To(ContainSubstring(required))
		}
		releaseJob := strings.Index(text, "  release:\n")
		Expect(releaseJob).To(BeNumerically(">", 0), "release workflow must contain the publisher job boundary")
		qualification := text[:releaseJob]
		Expect(qualification).NotTo(ContainSubstring("secrets."))
		Expect(qualification).NotTo(ContainSubstring("contents: write"))
	})

	It("keeps tags as candidates without release-please publishing a GitHub release", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "release-please.yml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("skip-github-release: true"))
		release, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "release.yml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(release)).To(ContainSubstring(`- "v*"`))
	})

	It("keeps integration uncached, bounded, and excluded from the default tier", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, "Taskfile.yml"))
		Expect(err).NotTo(HaveOccurred())
		text := string(data)
		for _, required := range []string{"test-integration:", "-count=1", "-timeout 10m", "./internal/engine ./test/e2e/...", "test-integration-race:", "-race", "test-default-excludes-e2e:"} {
			Expect(text).To(ContainSubstring(required))
		}
	})
})
