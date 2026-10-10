// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAbsoluteTargetPath(t *testing.T) {
	// Root fixtures in a temp dir: "/abs/target" is not absolute on Windows.
	root := t.TempDir()
	cwd := filepath.Join(root, "work", "root")
	absTarget := filepath.Join(root, "abs", "target")
	sep := string(filepath.Separator)
	tests := []struct {
		name   string
		cwd    string
		target string
		want   string
	}{
		{name: "relative target joins cwd", cwd: cwd, target: "repos/repo-a", want: filepath.Join(cwd, "repos", "repo-a")},
		{name: "relative target with dot segments is cleaned", cwd: cwd, target: "./repos/../repos/repo-a", want: filepath.Join(cwd, "repos", "repo-a")},
		{
			// Regression test: an absolute target must be used as-is, not
			// re-rooted under cwd. filepath.Join(cwd, absTarget) would
			// previously nest the target under cwd.
			name:   "absolute target is preserved, not re-rooted under cwd",
			cwd:    cwd,
			target: absTarget,
			want:   absTarget,
		},
		{name: "absolute target is cleaned", cwd: cwd, target: root + sep + "abs" + sep + sep + "target" + sep + ".." + sep + "target", want: absTarget},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveAbsoluteTargetPath(tc.cwd, tc.target); got != tc.want {
				t.Fatalf("resolveAbsoluteTargetPath(%q, %q) = %q, want %q", tc.cwd, tc.target, got, tc.want)
			}
		})
	}
}

func TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	src := filepath.Join(t.TempDir(), "source")
	mustRunGit(t, filepath.Dir(src), "init", src)
	mustRunGit(t, src, "commit", "--allow-empty", "-m", "init")

	// The target is an absolute path that lives entirely outside cwd
	// (filepath.Dir(cfgPath), per withConfigAndCWD). Before the fix,
	// filepath.Join(cwd, target) re-rooted this under cwd instead of
	// cloning to the path the caller asked for.
	absTarget := filepath.Join(t.TempDir(), "elsewhere", "repo-a")

	addCmd.SetOut(&bytes.Buffer{})
	addCmd.SetContext(context.Background())
	defer addCmd.SetOut(os.Stdout)
	_ = addCmd.Flags().Set("registry", "")
	_ = addCmd.Flags().Set("branch", "")
	_ = addCmd.Flags().Set("mirror", "false")
	if err := addCmd.RunE(addCmd, []string{absTarget, src}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	if _, err := os.Stat(absTarget); err != nil {
		t.Fatalf("expected repo cloned at absolute target %q: %v", absTarget, err)
	}
	// On Windows the re-rooted path ("…\C:\…") is not even a valid path, so
	// Stat reports a syntax error rather than ErrNotExist; any error means the
	// repo was not created there.
	wrongTarget := filepath.Join(filepath.Dir(cfgPath), absTarget)
	if _, err := os.Stat(wrongTarget); err == nil {
		t.Fatalf("expected no repo re-rooted under cwd at %q, stat err=%v", wrongTarget, err)
	}
}
