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
		for _, name := range []string{"ci.yml", "release.yml", "release-please.yml"} {
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

	It("uses the executable four-cell routine matrix and a required race job", func() {
		data, err := os.ReadFile(filepath.Join(moduleRoot, ".github", "workflows", "ci.yml"))
		Expect(err).NotTo(HaveOccurred())
		text := string(data)
		for _, required := range []string{"matrix --scope routine", "fail-fast: false", "ubuntu-24.04", "macos-15", "windows-2025", "matrix.cell.environment == 'wsl'", "windows_to_wsl", "./test/e2e/cmd/compatibility", "REPOKEEPER_E2E_COMPATIBILITY_BINARY", "MSYS2_ARG_CONV_EXCL='*'", "test-integration-race", "-race", "needs: compatibility-matrix"} {
			Expect(text).To(ContainSubstring(required))
		}
		Expect(text).To(ContainSubstring("permissions:\n  contents: read"))
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
