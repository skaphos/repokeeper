// SPDX-License-Identifier: MIT
package vcs

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHgAdapterSyncCapabilityMetadata(t *testing.T) {
	adapter := NewHgAdapter()

	supported, reason, err := adapter.SupportsLocalUpdate(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("SupportsLocalUpdate returned error: %v", err)
	}
	if supported {
		t.Fatal("expected local updates to be unsupported for hg")
	}
	if reason == "" {
		t.Fatal("expected non-empty skip reason")
	}

	action, err := adapter.FetchAction(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("FetchAction returned error: %v", err)
	}
	if action != "hg pull" {
		t.Fatalf("unexpected fetch action: %q", action)
	}
}

func TestHgAdapterEndToEndWithFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake hg script uses POSIX shell")
	}

	tmp := t.TempDir()
	hgPath := filepath.Join(tmp, "hg")
	script := `#!/usr/bin/env sh
cmd="$1"
case "$cmd" in
  root) echo "/tmp/repo"; exit 0 ;;
  paths)
    if [ "$2" = "default" ]; then
      echo "ssh://example/repo.hg"
      exit 0
    fi
    ;;
  branch) echo "default"; exit 0 ;;
  status) echo "M changed.txt"; exit 0 ;;
  pull) exit 0 ;;
  push) exit 0 ;;
  clone) exit 0 ;;
esac
exit 1
`
	if err := os.WriteFile(hgPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake hg: %v", err)
	}

	prevPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tmp+":"+prevPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer func() { _ = os.Setenv("PATH", prevPath) }()

	adapter := NewHgAdapter()
	repoDir := tmp

	if ok, err := adapter.IsRepo(context.Background(), repoDir); err != nil || !ok {
		t.Fatalf("IsRepo unexpected result: ok=%v err=%v", ok, err)
	}
	if remotes, err := adapter.Remotes(context.Background(), repoDir); err != nil || len(remotes) != 1 {
		t.Fatalf("Remotes unexpected result: remotes=%+v err=%v", remotes, err)
	}
	head, err := adapter.Head(context.Background(), repoDir)
	if err != nil || head.Branch != "default" {
		t.Fatalf("Head unexpected result: head=%+v err=%v", head, err)
	}
	worktree, err := adapter.WorktreeStatus(context.Background(), repoDir)
	if err != nil || worktree == nil || !worktree.Dirty {
		t.Fatalf("WorktreeStatus unexpected result: wt=%+v err=%v", worktree, err)
	}
	if err := adapter.Fetch(context.Background(), repoDir); err != nil {
		t.Fatalf("Fetch unexpected error: %v", err)
	}
	if err := adapter.Push(context.Background(), repoDir); err != nil {
		t.Fatalf("Push unexpected error: %v", err)
	}
	if err := adapter.Clone(context.Background(), "ssh://example/repo.hg", "/tmp/clone", "default", false); err != nil {
		t.Fatalf("Clone unexpected error: %v", err)
	}

	if got := adapter.NormalizeURL("SSH://EXAMPLE/REPO.HG "); got != "ssh://example/repo" {
		t.Fatalf("NormalizeURL unexpected value: %q", got)
	}
	if got := adapter.PrimaryRemote([]string{"other", "default"}); got != "default" {
		t.Fatalf("PrimaryRemote unexpected value: %q", got)
	}
}

func TestHgAdapterUnsupportedOperations(t *testing.T) {
	adapter := NewHgAdapter()
	if err := adapter.PullRebase(context.Background(), "/repo"); err == nil {
		t.Fatal("expected PullRebase to be unsupported")
	}
	if err := adapter.SetUpstream(context.Background(), "/repo", "default/main", "main"); err == nil {
		t.Fatal("expected SetUpstream to be unsupported")
	}
	if err := adapter.SetRemoteURL(context.Background(), "/repo", "default", "ssh://example/repo"); err == nil {
		t.Fatal("expected SetRemoteURL to be unsupported")
	}
	if _, err := adapter.StashPush(context.Background(), "/repo", "message"); err == nil {
		t.Fatal("expected StashPush to be unsupported")
	}
	if err := adapter.StashPop(context.Background(), "/repo"); err == nil {
		t.Fatal("expected StashPop to be unsupported")
	}
	if err := adapter.Clone(context.Background(), "ssh://example/repo", "/tmp/repo", "default", true); err == nil {
		t.Fatal("expected mirror clone to be unsupported")
	}

	if tracking, err := adapter.TrackingStatus(context.Background(), "/repo"); err != nil || tracking.Status != "none" {
		t.Fatalf("TrackingStatus unexpected result: tracking=%+v err=%v", tracking, err)
	}
	if hasSubmodules, err := adapter.HasSubmodules(context.Background(), "/repo"); err != nil || hasSubmodules {
		t.Fatalf("HasSubmodules unexpected result: has=%v err=%v", hasSubmodules, err)
	}
}

func TestHgAdapterIsRepoGracefullyHandlesCommandError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake hg script uses POSIX shell")
	}
	tmp := t.TempDir()
	hgPath := filepath.Join(tmp, "hg")
	if err := os.WriteFile(hgPath, []byte("#!/usr/bin/env sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake hg: %v", err)
	}
	prevPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", strings.Join([]string{tmp, prevPath}, ":")); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer func() { _ = os.Setenv("PATH", prevPath) }()

	ok, err := NewHgAdapter().IsRepo(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("expected IsRepo to swallow hg root error, got %v", err)
	}
	if ok {
		t.Fatal("expected IsRepo false when hg root fails")
	}
}
