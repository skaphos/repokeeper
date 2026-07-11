// SPDX-License-Identifier: MIT
package gitx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/gitx"
)

func TestPushWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		"/repo:push": {Output: ""},
	}}
	if err := gitx.Push(context.Background(), mock, "/repo"); err != nil {
		t.Fatalf("expected push success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:push": {Err: errors.New("push failed")},
	}}
	if err := gitx.Push(context.Background(), mock, "/repo"); err == nil {
		t.Fatal("expected push failure")
	}
}

func TestStashPushWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash push -u -m test": {Output: "Saved working directory and index state"},
	}}
	stashed, err := gitx.StashPush(context.Background(), mock, "/repo", "test")
	if err != nil {
		t.Fatalf("unexpected stash push error: %v", err)
	}
	if !stashed {
		t.Fatal("expected stash to be created")
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash push -u -m test": {Output: "No local changes to save"},
	}}
	stashed, err = gitx.StashPush(context.Background(), mock, "/repo", "test")
	if err != nil {
		t.Fatalf("unexpected stash push error: %v", err)
	}
	if stashed {
		t.Fatal("expected no stash when no local changes")
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash push -u": {Output: "Saved working directory and index state"},
	}}
	stashed, err = gitx.StashPush(context.Background(), mock, "/repo", "")
	if err != nil || !stashed {
		t.Fatalf("expected stash push without message to succeed: stashed=%v err=%v", stashed, err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash push -u": {Err: errors.New("stash failed")},
	}}
	if _, err := gitx.StashPush(context.Background(), mock, "/repo", ""); err == nil {
		t.Fatal("expected stash push error")
	}
}

func TestSetUpstreamWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		"/repo:branch --set-upstream-to origin/main main": {Output: ""},
	}}
	if err := gitx.SetUpstream(context.Background(), mock, "/repo", "origin/main", "main"); err != nil {
		t.Fatalf("expected setupstream success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:branch --set-upstream-to origin/main main": {Err: errors.New("set-upstream failed")},
	}}
	if err := gitx.SetUpstream(context.Background(), mock, "/repo", "origin/main", "main"); err == nil {
		t.Fatal("expected setupstream failure")
	}
}

func TestSetUpstreamRejectsFlagLikeArgs(t *testing.T) {
	// A malicious/malformed upstream or branch beginning with "-" must be
	// rejected before it ever reaches git, where it would otherwise be
	// parsed as a flag instead of a literal ref.
	mock := &MockRunner{Responses: map[string]MockResponse{}}

	if err := gitx.SetUpstream(context.Background(), mock, "/repo", "--upload-pack=evil", "main"); err == nil {
		t.Fatal("expected error for flag-like upstream")
	}
	if err := gitx.SetUpstream(context.Background(), mock, "/repo", "origin/main", "-x"); err == nil {
		t.Fatal("expected error for flag-like branch")
	}
}

func TestSetRemoteURLWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		"/repo:remote set-url origin git@github.com:org/repo.git": {Output: ""},
	}}
	if err := gitx.SetRemoteURL(context.Background(), mock, "/repo", "origin", "git@github.com:org/repo.git"); err != nil {
		t.Fatalf("expected set remote url success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:remote set-url origin git@github.com:org/repo.git": {Err: errors.New("set-url failed")},
	}}
	if err := gitx.SetRemoteURL(context.Background(), mock, "/repo", "origin", "git@github.com:org/repo.git"); err == nil {
		t.Fatal("expected set remote url failure")
	}
}

func TestSetRemoteURLRejectsFlagLikeArgs(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{}}

	if err := gitx.SetRemoteURL(context.Background(), mock, "/repo", "-o", "git@github.com:org/repo.git"); err == nil {
		t.Fatal("expected error for flag-like remote name")
	}
	if err := gitx.SetRemoteURL(context.Background(), mock, "/repo", "origin", "--upload-pack=evil"); err == nil {
		t.Fatal("expected error for flag-like remote URL")
	}
}

func TestStashPopWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash pop": {Output: ""},
	}}
	if err := gitx.StashPop(context.Background(), mock, "/repo"); err != nil {
		t.Fatalf("expected stash pop success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		"/repo:stash pop": {Err: errors.New("conflict")},
	}}
	if err := gitx.StashPop(context.Background(), mock, "/repo"); err == nil {
		t.Fatal("expected stash pop failure")
	}
}

func TestCloneWrapper(t *testing.T) {
	mock := &MockRunner{Responses: map[string]MockResponse{
		":clone --mirror git@github.com:org/repo.git /target": {Output: ""},
	}}
	if err := gitx.Clone(context.Background(), mock, "git@github.com:org/repo.git", "/target", "main", true); err != nil {
		t.Fatalf("expected mirror clone success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		":clone --branch main --single-branch git@github.com:org/repo.git /target": {Output: ""},
	}}
	if err := gitx.Clone(context.Background(), mock, "git@github.com:org/repo.git", "/target", "main", false); err != nil {
		t.Fatalf("expected branch clone success, got %v", err)
	}

	mock = &MockRunner{Responses: map[string]MockResponse{
		":clone git@github.com:org/repo.git /target": {Err: errors.New("clone failed")},
	}}
	if err := gitx.Clone(context.Background(), mock, "git@github.com:org/repo.git", "/target", "", false); err == nil {
		t.Fatal("expected clone error")
	}
}

func TestCloneRejectsFlagLikeArgs(t *testing.T) {
	// A remote URL, target path, or branch beginning with "-" must be
	// rejected before it reaches git, where it would otherwise be parsed
	// as a flag (e.g. "--upload-pack=...") instead of a literal argument.
	mock := &MockRunner{Responses: map[string]MockResponse{}}

	if err := gitx.Clone(context.Background(), mock, "--upload-pack=evil", "/target", "", false); err == nil {
		t.Fatal("expected error for flag-like remote URL")
	}
	if err := gitx.Clone(context.Background(), mock, "git@github.com:org/repo.git", "-x", "", false); err == nil {
		t.Fatal("expected error for flag-like target path")
	}
	if err := gitx.Clone(context.Background(), mock, "git@github.com:org/repo.git", "/target", "-x", false); err == nil {
		t.Fatal("expected error for flag-like branch")
	}
}

func TestCloneDoesNotTrimURLOrPath(t *testing.T) {
	// Leading/trailing whitespace is legal in local paths, so the remote URL and
	// target path must reach git verbatim; trimming would clone into the wrong
	// directory. The flag guard only rejects values whose first byte is '-'.
	spacedURL := " https://example.com/repo.git"
	spacedPath := " /tmp/my repo "
	mock := &MockRunner{Responses: map[string]MockResponse{}}
	// The clone "fails" with an unexpected-call error because we do not register
	// a response; we only care that the args reached the runner untouched.
	_ = gitx.Clone(context.Background(), mock, spacedURL, spacedPath, "", false)

	if len(mock.LastArgs) < 3 {
		t.Fatalf("expected clone to reach the runner, got args %v", mock.LastArgs)
	}
	gotURL := mock.LastArgs[len(mock.LastArgs)-2]
	gotPath := mock.LastArgs[len(mock.LastArgs)-1]
	if gotURL != spacedURL {
		t.Fatalf("remote URL passed to git = %q, want verbatim %q", gotURL, spacedURL)
	}
	if gotPath != spacedPath {
		t.Fatalf("target path passed to git = %q, want verbatim %q", gotPath, spacedPath)
	}
}
