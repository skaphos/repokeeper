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
	t.Setenv("USERPROFILE", home)
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

func TestRequestedSkillRootsTargets(t *testing.T) {
	home, configHome := withSkillEnv(t)
	all, err := requestedSkillRoots("all")
	if err != nil {
		t.Fatalf("requested all roots: %v", err)
	}
	wantAll := []string{
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(configHome, "opencode", "skills"),
	}
	if strings.Join(all, ",") != strings.Join(wantAll, ",") {
		t.Fatalf("expected all roots %v, got %v", wantAll, all)
	}

	openai, err := requestedSkillRoots("openai")
	if err != nil {
		t.Fatalf("requested openai roots: %v", err)
	}
	if len(openai) != 1 || openai[0] != filepath.Join(home, ".agents", "skills") {
		t.Fatalf("unexpected openai roots: %v", openai)
	}

	if _, err := requestedSkillRoots("unknown"); err == nil {
		t.Fatal("expected unsupported target to fail")
	}
}

func TestResolveSkillUninstallRootsFiltersMissingTargets(t *testing.T) {
	home, _ := withSkillEnv(t)
	claudeSkillDir := filepath.Join(home, ".claude", "skills", "repokeeper")
	if err := os.MkdirAll(claudeSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir claude skill dir: %v", err)
	}

	roots, err := resolveSkillUninstallRoots([]string{"all"})
	if err != nil {
		t.Fatalf("resolve uninstall roots: %v", err)
	}
	want := []string{filepath.Join(home, ".claude", "skills")}
	if strings.Join(roots, ",") != strings.Join(want, ",") {
		t.Fatalf("expected installed roots %v, got %v", want, roots)
	}
}

func TestResolveSkillInstallRootsWithoutExistingDirectoriesFails(t *testing.T) {
	withSkillEnv(t)

	if _, err := resolveSkillInstallRoots(nil); err == nil {
		t.Fatal("expected install root discovery to fail when no supported directories exist")
	}
}

func TestResolveSkillUninstallRootsWithoutArgsReturnsInstalledRoots(t *testing.T) {
	home, _ := withSkillEnv(t)
	agentsSkillDir := filepath.Join(home, ".agents", "skills", "repokeeper")
	if err := os.MkdirAll(agentsSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir agents skill dir: %v", err)
	}

	roots, err := resolveSkillUninstallRoots(nil)
	if err != nil {
		t.Fatalf("resolve uninstall roots without args: %v", err)
	}
	want := []string{filepath.Join(home, ".agents", "skills")}
	if strings.Join(roots, ",") != strings.Join(want, ",") {
		t.Fatalf("expected uninstall roots %v, got %v", want, roots)
	}
}

func TestUserConfigDirPrefersXDGConfigHome(t *testing.T) {
	_, configHome := withSkillEnv(t)
	got, err := userConfigDir()
	if err != nil {
		t.Fatalf("user config dir: %v", err)
	}
	if got != configHome {
		t.Fatalf("expected XDG config home %q, got %q", configHome, got)
	}
}

func TestDedupeSortedStringsAndDirExists(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}

	got := dedupeSortedStrings([]string{"b", "a", "b"})
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("expected deduped sorted strings, got %v", got)
	}
	if dedupeSortedStrings(nil) != nil {
		t.Fatal("expected nil input to stay nil")
	}

	exists, err := dirExists(filepath.Join(tmp, "dir"))
	if err != nil {
		t.Fatalf("dir exists for directory: %v", err)
	}
	if !exists {
		t.Fatal("expected existing directory")
	}
	exists, err = dirExists(filepath.Join(tmp, "missing"))
	if err != nil {
		t.Fatalf("dir exists for missing path: %v", err)
	}
	if exists {
		t.Fatal("expected missing directory to report false")
	}
}

func TestSkillInstalledAt(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(filepath.Join(root, "repokeeper"), 0o755); err != nil {
		t.Fatalf("mkdir installed skill: %v", err)
	}

	exists, err := skillInstalledAt(root)
	if err != nil {
		t.Fatalf("skill installed at existing root: %v", err)
	}
	if !exists {
		t.Fatal("expected installed skill to be detected")
	}

	exists, err = skillInstalledAt(filepath.Join(tmp, "missing-skills"))
	if err != nil {
		t.Fatalf("skill installed at missing root: %v", err)
	}
	if exists {
		t.Fatal("expected missing install root to report false")
	}
}
