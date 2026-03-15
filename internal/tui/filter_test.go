// SPDX-License-Identifier: MIT
package tui

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

var testRepos = []model.RepoStatus{
	{RepoID: "acme/backend", Path: "/work/backend", Head: model.Head{Branch: "main"}, Tracking: model.Tracking{Status: model.TrackingEqual}},
	{RepoID: "acme/frontend", Path: "/work/frontend", Head: model.Head{Branch: "feat/ui"}, Tracking: model.Tracking{Status: model.TrackingAhead}},
	{RepoID: "acme/infra", Path: "/work/infra", Head: model.Head{Branch: "main"}, ErrorClass: "network"},
	{RepoID: "tools/cli", Path: "/work/cli", Head: model.Head{Branch: "develop"}, Labels: map[string]string{"team": "platform"}},
}

func TestFilterRowsEmptyQuery(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "")
	if len(got) != len(testRepos) {
		t.Fatalf("empty query: expected %d repos, got %d", len(testRepos), len(got))
	}
}

func TestFilterRowsByRepoID(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "backend")
	if len(got) != 1 || got[0].RepoID != "acme/backend" {
		t.Fatalf("expected [acme/backend], got %v", got)
	}
}

func TestFilterRowsByPath(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "/work/infra")
	if len(got) != 1 || got[0].RepoID != "acme/infra" {
		t.Fatalf("expected [acme/infra], got %v", got)
	}
}

func TestFilterRowsByBranch(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "feat/ui")
	if len(got) != 1 || got[0].RepoID != "acme/frontend" {
		t.Fatalf("expected [acme/frontend], got %v", got)
	}
}

func TestFilterRowsByTrackingStatus(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "ahead")
	if len(got) != 1 || got[0].RepoID != "acme/frontend" {
		t.Fatalf("expected [acme/frontend], got %v", got)
	}
}

func TestFilterRowsByErrorClass(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "network")
	if len(got) != 1 || got[0].RepoID != "acme/infra" {
		t.Fatalf("expected [acme/infra], got %v", got)
	}
}

func TestFilterRowsByLabelValue(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "platform")
	if len(got) != 1 || got[0].RepoID != "tools/cli" {
		t.Fatalf("expected [tools/cli], got %v", got)
	}
}

func TestFilterRowsCaseInsensitive(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "BACKEND")
	if len(got) != 1 || got[0].RepoID != "acme/backend" {
		t.Fatalf("expected case-insensitive match for BACKEND, got %v", got)
	}
}

func TestFilterRowsNoMatch(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "zzznomatch")
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %v", got)
	}
}

func TestFilterRowsMatchesMultiple(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "acme")
	if len(got) != 3 {
		t.Fatalf("expected 3 acme repos, got %d", len(got))
	}
}

func TestFilterRowsSharedBranch(t *testing.T) {
	t.Parallel()
	got := filterRows(testRepos, "main")
	if len(got) != 2 {
		t.Fatalf("expected 2 repos on main, got %d: %v", len(got), got)
	}
}
