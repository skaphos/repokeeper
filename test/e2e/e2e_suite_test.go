// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	moduleRoot     string
	repokeeperPath string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RepoKeeper end-to-end suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	if root, binary := os.Getenv("REPOKEEPER_E2E_MODULE_ROOT"), os.Getenv("REPOKEEPER_E2E_BINARY"); root != "" && binary != "" {
		return []byte(root + "\x00" + binary)
	}
	root, err := findModuleRoot()
	Expect(err).NotTo(HaveOccurred())

	buildRoot, err := os.MkdirTemp("", "repokeeper-e2e-build-")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(buildRoot)).To(Succeed()) })

	binary := filepath.Join(buildRoot, "repokeeper"+executableSuffix())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	Expect(ctx.Err()).NotTo(Equal(context.DeadlineExceeded), "RepoKeeper build exceeded 30 seconds\n%s", output)
	Expect(err).NotTo(HaveOccurred(), "building RepoKeeper failed\n%s", output)
	return []byte(root + "\x00" + binary)
}, func(payload []byte) {
	parts := splitNUL(string(payload))
	Expect(parts).To(HaveLen(2))
	moduleRoot, repokeeperPath = parts[0], parts[1]
})

func findModuleRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve e2e suite source path")
	}
	for dir := filepath.Dir(currentFile); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && info.Mode().IsRegular() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", currentFile)
		}
	}
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func splitNUL(value string) []string {
	for i := range value {
		if value[i] == 0 {
			return []string{value[:i], value[i+1:]}
		}
	}
	return []string{value}
}
