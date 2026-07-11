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
	tests := []struct {
		name   string
		cwd    string
		target string
		want   string
	}{
		{name: "relative target joins cwd", cwd: "/work/root", target: "repos/repo-a", want: filepath.FromSlash("/work/root/repos/repo-a")},
		{name: "relative target with dot segments is cleaned", cwd: "/work/root", target: "./repos/../repos/repo-a", want: filepath.FromSlash("/work/root/repos/repo-a")},
		{
			// Regression test: an absolute target must be used as-is, not
			// re-rooted under cwd. filepath.Join("/work/root", "/abs/target")
			// would previously yield "/work/root/abs/target" instead of
			// "/abs/target".
			name:   "absolute target is preserved, not re-rooted under cwd",
			cwd:    "/work/root",
			target: "/abs/target",
			want:   filepath.FromSlash("/abs/target"),
		},
		{name: "absolute target is cleaned", cwd: "/work/root", target: "/abs//target/../target", want: filepath.FromSlash("/abs/target")},
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
	wrongTarget := filepath.Join(filepath.Dir(cfgPath), absTarget)
	if _, err := os.Stat(wrongTarget); !os.IsNotExist(err) {
		t.Fatalf("expected no repo re-rooted under cwd at %q, stat err=%v", wrongTarget, err)
	}
}
