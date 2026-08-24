// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"errors"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
		_, err := build.Default.ImportDir(filepath.Join(moduleRoot, "test", "e2e"), 0)
		var noGoError *build.NoGoError
		Expect(errors.As(err, &noGoError)).To(BeTrue(), "ordinary build context unexpectedly selected E2E Go sources: %v", err)
	})
})
