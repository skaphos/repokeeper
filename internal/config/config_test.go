package config_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/config"
)

var _ = Describe("Config", func() {
	It("resolves config path from override directory", func() {
		path, err := config.ConfigPath(filepath.Join("C:", "tmp", "repokeeper"))
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(HaveSuffix(filepath.Join("repokeeper", "config.yaml")))
	})

	It("resolves config path from override file", func() {
		path, err := config.ConfigPath(filepath.Join("C:", "tmp", "config.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(HaveSuffix(filepath.Join("tmp", "config.yaml")))
	})

	It("resolves config path from env", func() {
		Expect(os.Setenv("REPOKEEPER_CONFIG", filepath.Join("C:", "cfg", "config.yaml"))).To(Succeed())
		defer func() { _ = os.Unsetenv("REPOKEEPER_CONFIG") }()
		path, err := config.ConfigPath("")
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(HaveSuffix(filepath.Join("cfg", "config.yaml")))
	})

	It("resolves init path to local dotfile by default", func() {
		dir := GinkgoT().TempDir()
		path, err := config.InitConfigPath("", dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(dir, ".repokeeper.yaml")))
	})

	It("prefers local dotfile for runtime config resolution", func() {
		dir := GinkgoT().TempDir()
		localPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(localPath, []byte("roots: [\".\"]\n"), 0o644)).To(Succeed())

		path, err := config.ResolveConfigPath("", dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(localPath))
	})

	It("resolves runtime config from nearest parent dotfile", func() {
		dir := GinkgoT().TempDir()
		parentPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(parentPath, []byte("roots: [\".\"]\n"), 0o644)).To(Succeed())

		nested := filepath.Join(dir, "a", "b", "c")
		Expect(os.MkdirAll(nested, 0o755)).To(Succeed())

		path, err := config.ResolveConfigPath("", nested)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(parentPath))
	})

	It("prefers nearer dotfile over farther parent", func() {
		dir := GinkgoT().TempDir()
		parentPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(parentPath, []byte("roots: [\".\"]\n"), 0o644)).To(Succeed())

		childDir := filepath.Join(dir, "a", "b")
		Expect(os.MkdirAll(childDir, 0o755)).To(Succeed())
		childPath := filepath.Join(childDir, ".repokeeper.yaml")
		Expect(os.WriteFile(childPath, []byte("roots: [\".\"]\n"), 0o644)).To(Succeed())

		nested := filepath.Join(childDir, "c")
		Expect(os.MkdirAll(nested, 0o755)).To(Succeed())

		path, err := config.ResolveConfigPath("", nested)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(childPath))
	})

	It("falls back to global runtime config when local dotfile is absent", func() {
		dir := GinkgoT().TempDir()
		path, err := config.ResolveConfigPath("", dir)
		Expect(err).NotTo(HaveOccurred())

		globalPath, err := config.ConfigPath("")
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(globalPath))
	})

	It("saves and loads config with defaults", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "config.yaml")
		cfg := config.DefaultConfig()
		cfg.Roots = []string{filepath.Join(dir, "repos")}

		Expect(config.Save(&cfg, path)).To(Succeed())
		loaded, err := config.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.RegistryPath).To(BeEmpty())
		Expect(loaded.Registry).To(BeNil())
		Expect(loaded.Defaults.RemoteName).To(Equal("origin"))
	})

	It("returns an RFC3339 timestamp for last updated", func() {
		ts := config.LastUpdated()
		_, err := time.Parse(time.RFC3339, ts)
		Expect(err).NotTo(HaveOccurred())
	})
})
