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
