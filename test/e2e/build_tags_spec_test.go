// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E build-tag source policy", func() {
	It("requires integration on every E2E Go source and compound platform constraints", func() {
		e2eRoot := filepath.Join(moduleRoot, "test", "e2e")
		err := filepath.WalkDir(e2eRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			header := string(data)
			if len(header) > 512 {
				header = header[:512]
			}
			Expect(header).To(ContainSubstring("//go:build integration"), path)
			base := filepath.Base(path)
			switch {
			case strings.Contains(base, "_windows"):
				Expect(header).To(ContainSubstring("//go:build integration && windows"), path)
			case strings.Contains(base, "_unix"):
				Expect(header).To(ContainSubstring("//go:build integration && !windows"), path)
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("keeps the E2E package out of the ordinary package graph", func() {
		root := GinkgoT().TempDir()
		environment, err := buildChildEnvironment(root, filepath.Join(root, "config.yaml"))
		Expect(err).NotTo(HaveOccurred())
		for _, key := range []string{"GOMODCACHE", "GOCACHE"} {
			output, commandErr := exec.Command("go", "env", key).Output()
			Expect(commandErr).NotTo(HaveOccurred())
			environment = append(environment, key+"="+strings.TrimSpace(string(output)))
		}
		result := runCommand(context.Background(), 30*time.Second, "ordinary go list", "go", []string{"list", "-e", "./..."}, moduleRoot, environment, root, filepath.Join(root, "config.yaml"))
		Expect(expectExit(result, 0)).To(Succeed(), result.Diagnostics())
		Expect(string(result.Stdout)).NotTo(ContainSubstring("github.com/skaphos/repokeeper/test/e2e"))
	})
})
