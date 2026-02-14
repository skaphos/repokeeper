package repokeeper

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/registry"
)

func TestSelectRegistryEntryForDescribe(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "github.com/org/repo-a", Path: "/tmp/work/repo-a"},
		{RepoID: "github.com/org/repo-b", Path: "/tmp/root/repo-b"},
	}

	byID, err := selectRegistryEntryForDescribe(entries, "github.com/org/repo-a", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected id selector to match, got error: %v", err)
	}
	if byID.Path != "/tmp/work/repo-a" {
		t.Fatalf("unexpected id match: %#v", byID)
	}

	byCWD, err := selectRegistryEntryForDescribe(entries, "repo-a", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected cwd-relative selector to match, got error: %v", err)
	}
	if byCWD.RepoID != "github.com/org/repo-a" {
		t.Fatalf("unexpected cwd match: %#v", byCWD)
	}

	byRoot, err := selectRegistryEntryForDescribe(entries, "repo-b", "/tmp/work", []string{"/tmp/root"})
	if err != nil {
		t.Fatalf("expected root-relative selector to match, got error: %v", err)
	}
	if byRoot.RepoID != "github.com/org/repo-b" {
		t.Fatalf("unexpected root match: %#v", byRoot)
	}
}
