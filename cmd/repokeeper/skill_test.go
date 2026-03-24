// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withSkillEnv(t *testing.T) (home string, configHome string) {
	t.Helper()
	home = t.TempDir()
	configHome = filepath.Join(home, ".config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	return home, configHome
}

func TestSkillInstallAutoInstallsToExistingRoots(t *testing.T) {
	home, _ := withSkillEnv(t)
	claudeRoot := filepath.Join(home, ".claude", "skills")
	agentsRoot := filepath.Join(home, ".agents", "skills")
	if err := os.MkdirAll(claudeRoot, 0o755); err != nil {
		t.Fatalf("mkdir claude root: %v", err)
	}
	if err := os.MkdirAll(agentsRoot, 0o755); err != nil {
		t.Fatalf("mkdir agents root: %v", err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	skillInstallCmd.SetOut(out)
	skillInstallCmd.SetErr(errOut)
	defer skillInstallCmd.SetOut(os.Stdout)
	defer skillInstallCmd.SetErr(os.Stderr)

	if err := skillInstallCmd.RunE(skillInstallCmd, nil); err != nil {
		t.Fatalf("skill install auto failed: %v", err)
	}
	for _, path := range []string{
		filepath.Join(claudeRoot, "repokeeper", "SKILL.md"),
		filepath.Join(agentsRoot, "repokeeper", "SKILL.md"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read installed skill %s: %v", path, err)
		}
		if !strings.Contains(string(data), "name: repokeeper") {
			t.Fatalf("expected skill frontmatter in %s, got %q", path, string(data))
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "skills", "repokeeper", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected auto install to skip non-existing opencode root, got err=%v", err)
	}
}

func TestSkillInstallExplicitTargets(t *testing.T) {
	home, configHome := withSkillEnv(t)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	skillInstallCmd.SetOut(out)
	skillInstallCmd.SetErr(errOut)
	defer skillInstallCmd.SetOut(os.Stdout)
	defer skillInstallCmd.SetErr(os.Stderr)

	if err := skillInstallCmd.RunE(skillInstallCmd, []string{"opencode"}); err != nil {
		t.Fatalf("skill install opencode failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configHome, "opencode", "skills", "repokeeper", "SKILL.md")); err != nil {
		t.Fatalf("expected opencode skill install, got %v", err)
	}
	if err := skillInstallCmd.RunE(skillInstallCmd, []string{"openai"}); err != nil {
		t.Fatalf("skill install openai failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "repokeeper", "SKILL.md")); err != nil {
		t.Fatalf("expected openai skill install, got %v", err)
	}
	if err := skillInstallCmd.RunE(skillInstallCmd, []string{"codex"}); err != nil {
		t.Fatalf("skill install codex failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "repokeeper", "SKILL.md")); err != nil {
		t.Fatalf("expected codex skill install, got %v", err)
	}
}

func TestSkillUninstallRemovesInstalledSkill(t *testing.T) {
	home, _ := withSkillEnv(t)
	claudeSkillDir := filepath.Join(home, ".claude", "skills", "repokeeper")
	agentsSkillDir := filepath.Join(home, ".agents", "skills", "repokeeper")
	for _, dir := range []string{claudeSkillDir, agentsSkillDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir skill dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: repokeeper\ndescription: test\n---\n"), 0o644); err != nil {
			t.Fatalf("write skill file %s: %v", dir, err)
		}
	}
	yesCleanup := withAssumeYes(t, true)
	defer yesCleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	skillUninstallCmd.SetOut(out)
	skillUninstallCmd.SetErr(errOut)
	defer skillUninstallCmd.SetOut(os.Stdout)
	defer skillUninstallCmd.SetErr(os.Stderr)

	if err := skillUninstallCmd.RunE(skillUninstallCmd, nil); err != nil {
		t.Fatalf("skill uninstall failed: %v", err)
	}
	for _, dir := range []string{claudeSkillDir, agentsSkillDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("expected skill dir removed %s, got err=%v", dir, err)
		}
	}
}
