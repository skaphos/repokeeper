// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exact child environment", func() {
	It("isolates process and Git state without mutating the host", func() {
		root := GinkgoT().TempDir()
		configPath := filepath.Join(root, "config", "config.yaml")
		before := append([]string(nil), os.Environ()...)
		environment, err := buildChildEnvironment(root, configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Environ()).To(Equal(before))

		values := environmentMap(environment)
		for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA", "TMPDIR", "TMP", "TEMP"} {
			Expect(values).To(HaveKey(key))
			Expect(filepath.IsAbs(values[key])).To(BeTrue())
			Expect(filepath.Rel(root, values[key])).NotTo(HavePrefix(".."))
		}
		Expect(values["REPOKEEPER_CONFIG"]).To(Equal(configPath))
		Expect(values["GIT_CONFIG_NOSYSTEM"]).To(Equal("1"))
		Expect(values["GIT_TERMINAL_PROMPT"]).To(Equal("0"))
		Expect(values["GIT_ALLOW_PROTOCOL"]).To(Equal("file"))
		Expect(values["GIT_CONFIG_VALUE_1"]).To(Equal("false"))
		for _, forbidden := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "SSH_AUTH_SOCK", "GITHUB_TOKEN", "GH_TOKEN", "AWS_ACCESS_KEY_ID"} {
			Expect(values).NotTo(HaveKey(forbidden))
		}
	})

	It("replaces differently-cased keys rather than duplicating them", func() {
		values := map[string]string{"Path": "old"}
		setEnvironmentValue(values, "PATH", "new")
		Expect(values).To(Equal(map[string]string{"PATH": "new"}))
	})

	It("emits no duplicate keys under case folding", func() {
		environment, err := buildChildEnvironment(GinkgoT().TempDir(), filepath.Join(GinkgoT().TempDir(), "config.yaml"))
		Expect(err).NotTo(HaveOccurred())
		seen := map[string]bool{}
		for _, item := range environment {
			key, _, _ := strings.Cut(item, "=")
			folded := strings.ToUpper(key)
			Expect(seen).NotTo(HaveKey(folded))
			seen[folded] = true
		}
	})
})
