// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bounded process execution", func() {
	var root string
	var environment []string
	BeforeEach(func() {
		root = GinkgoT().TempDir()
		var err error
		environment, err = buildChildEnvironment(root, filepath.Join(root, "config", "config.yaml"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("captures normal and domain non-zero exits separately", func() {
		executable, args := portableExitCommand(0)
		result := runCommand(context.Background(), 5*time.Second, "normal exit", executable, args, root, environment, root, "config")
		Expect(expectExit(result, 0)).To(Succeed(), result.Diagnostics())

		executable, args = portableExitCommand(7)
		result = runCommand(context.Background(), 5*time.Second, "domain exit", executable, args, root, environment, root, "config")
		Expect(result.LaunchError).NotTo(HaveOccurred())
		Expect(result.ExitCode).To(Equal(7))
	})

	It("distinguishes launch failures", func() {
		result := runCommand(context.Background(), time.Second, "launch", filepath.Join(root, "absent"), nil, root, environment, root, "config")
		Expect(result.LaunchError).To(HaveOccurred())
		Expect(result.ExitCode).To(Equal(-1))
	})

	It("terminates timed-out process groups", func() {
		executable, args := portableSleepCommand()
		result := runCommand(context.Background(), 100*time.Millisecond, "timeout", executable, args, root, environment, root, "config")
		Expect(result.TimedOut).To(BeTrue(), result.Diagnostics())
		Expect(result.Duration).To(BeNumerically("<", 5*time.Second))
	})

	It("bounds output and renders the complete diagnostic contract", func() {
		buffer := newBoundedBuffer(4)
		_, err := buffer.Write([]byte("123456"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buffer.Bytes())).To(ContainSubstring("1234"))
		Expect(string(buffer.Bytes())).To(ContainSubstring("truncated"))

		result := ExecutionResult{Operation: "op", Executable: "exe", Arguments: []string{"a"}, WorkingDir: root, Root: root, ConfigPath: "config", ExitCode: 2, Stdout: []byte("out"), Stderr: []byte("err")}
		diagnostic := result.Diagnostics()
		for _, field := range []string{"operation: op", "command:", "working directory:", "scenario root:", "config path:", "exit: 2", "timed out:", "duration:", "stdout:", "out", "stderr:", "err"} {
			Expect(diagnostic).To(ContainSubstring(field))
		}
	})
})

func portableExitCommand(code int) (string, []string) {
	if runtime.GOOS == "windows" {
		return os.Getenv("ComSpec"), []string{"/d", "/s", "/c", "exit", strconv.Itoa(code)}
	}
	return "/bin/sh", []string{"-c", "exit " + strconv.Itoa(code)}
}

func portableSleepCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return os.Getenv("ComSpec"), []string{"/d", "/s", "/c", "ping -n 10 127.0.0.1 >NUL"}
	}
	return "/bin/sh", []string{"-c", "sleep 10"}
}
