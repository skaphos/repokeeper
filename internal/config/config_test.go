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

		Expect(config.Save(&cfg, path)).To(Succeed())
		loaded, err := config.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.RegistryPath).To(BeEmpty())
		Expect(loaded.Registry).To(BeNil())
		Expect(loaded.APIVersion).To(Equal(config.ConfigAPIVersion))
		Expect(loaded.Kind).To(Equal(config.ConfigKind))
		Expect(loaded.Defaults.RemoteName).To(Equal("origin"))
	})

	It("defaults missing gvk when loading legacy config", func() {
		dir := GinkgoT().TempDir()
		cfgPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(cfgPath, []byte("exclude: []\n"), 0o644)).To(Succeed())

		loaded, err := config.Load(cfgPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.APIVersion).To(Equal(config.LegacyConfigAPIVersion))
		Expect(loaded.Kind).To(Equal(config.ConfigKind))
	})

	It("migrates legacy gvk to v1beta1 on save", func() {
		dir := GinkgoT().TempDir()
		cfgPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(cfgPath, []byte("exclude: []\n"), 0o644)).To(Succeed())

		loaded, err := config.Load(cfgPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.APIVersion).To(Equal(config.LegacyConfigAPIVersion))

		Expect(config.Save(loaded, cfgPath)).To(Succeed())
		rewritten, err := config.Load(cfgPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(rewritten.APIVersion).To(Equal(config.ConfigAPIVersion))
		Expect(rewritten.Kind).To(Equal(config.ConfigKind))
	})

	It("rejects unsupported gvk", func() {
		dir := GinkgoT().TempDir()
		cfgPath := filepath.Join(dir, ".repokeeper.yaml")
		Expect(os.WriteFile(cfgPath, []byte("apiVersion: example.com/v1\nkind: RepoKeeperConfig\n"), 0o644)).To(Succeed())

		_, err := config.Load(cfgPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported config apiVersion"))
	})

	It("loads registry_path relative to config file directory", func() {
		dir := GinkgoT().TempDir()
		regPath := filepath.Join(dir, "state", "registry.yaml")
		Expect(os.MkdirAll(filepath.Dir(regPath), 0o755)).To(Succeed())
		Expect(os.WriteFile(regPath, []byte("updated_at: 2026-01-01T00:00:00Z\nrepos: []\n"), 0o644)).To(Succeed())

		cfgPath := filepath.Join(dir, "cfg", ".repokeeper.yaml")
		Expect(os.MkdirAll(filepath.Dir(cfgPath), 0o755)).To(Succeed())
		Expect(os.WriteFile(cfgPath, []byte("registry_path: ../state/registry.yaml\n"), 0o644)).To(Succeed())

		loaded, err := config.Load(cfgPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(loaded.Registry).NotTo(BeNil())
	})

	It("resolves relative registry_path against config path", func() {
		cfgPath := filepath.Join("/tmp", "workspace", ".repokeeper.yaml")
		resolved := config.ResolveRegistryPath(cfgPath, "./state/registry.yaml")
		Expect(resolved).To(Equal(filepath.Clean(filepath.Join("/tmp", "workspace", "state", "registry.yaml"))))
	})

	It("returns config root from config path", func() {
		cfgPath := filepath.Join("/tmp", "workspace", ".repokeeper.yaml")
		Expect(config.ConfigRoot(cfgPath)).To(Equal(filepath.Clean(filepath.Join("/tmp", "workspace"))))
	})

	It("uses config directory as effective root", func() {
		cfgPath := filepath.Join("/tmp", "workspace", ".repokeeper.yaml")
		Expect(config.EffectiveRoot(cfgPath, nil)).To(Equal(filepath.Clean(filepath.Join("/tmp", "workspace"))))
	})

	It("returns an RFC3339 timestamp for last updated", func() {
		ts := config.LastUpdated()
		_, err := time.Parse(time.RFC3339, ts)
		Expect(err).NotTo(HaveOccurred())
	})
})
