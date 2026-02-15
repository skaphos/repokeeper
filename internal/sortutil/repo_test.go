package sortutil

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

func TestLessRepoIDPath(t *testing.T) {
	if !LessRepoIDPath("a", "/z", "b", "/a") {
		t.Fatal("expected repo id ordering to take precedence")
	}
	if !LessRepoIDPath("a", "/a", "a", "/b") {
		t.Fatal("expected path ordering when repo ids are equal")
	}
	if LessRepoIDPath("b", "/a", "a", "/z") {
		t.Fatal("did not expect reverse repo id ordering")
	}
}

func TestSortRepoStatuses(t *testing.T) {
	statuses := []model.RepoStatus{
		{RepoID: "b", Path: "/2"},
		{RepoID: "a", Path: "/9"},
		{RepoID: "a", Path: "/1"},
	}
	SortRepoStatuses(statuses)
	if statuses[0].RepoID != "a" || statuses[0].Path != "/1" {
		t.Fatalf("unexpected first item: %+v", statuses[0])
	}
	if statuses[1].RepoID != "a" || statuses[1].Path != "/9" {
		t.Fatalf("unexpected second item: %+v", statuses[1])
	}
	if statuses[2].RepoID != "b" || statuses[2].Path != "/2" {
		t.Fatalf("unexpected third item: %+v", statuses[2])
	}
}

func TestSortRegistryEntries(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "repo-b", Path: "/2"},
		{RepoID: "repo-a", Path: "/9"},
		{RepoID: "repo-a", Path: "/1"},
	}
	SortRegistryEntries(entries)
	if entries[0].RepoID != "repo-a" || entries[0].Path != "/1" {
		t.Fatalf("unexpected first item: %+v", entries[0])
	}
	if entries[1].RepoID != "repo-a" || entries[1].Path != "/9" {
		t.Fatalf("unexpected second item: %+v", entries[1])
	}
	if entries[2].RepoID != "repo-b" || entries[2].Path != "/2" {
		t.Fatalf("unexpected third item: %+v", entries[2])
	}
}
