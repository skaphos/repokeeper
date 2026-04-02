// SPDX-License-Identifier: MIT
//go:build windows

package repokeeper

import (
	"path/filepath"
	"testing"

	"github.com/skaphos/repokeeper/internal/registry"
)

func TestSelectRegistryEntryForDescribeAbsolutePathWindows(t *testing.T) {
	entries := []registry.Entry{
		{RepoID: "github.com/org/repo-a", Path: `C:\workspace\repo-a`},
		{RepoID: "github.com/org/repo-b", Path: `C:\root\repo-b`},
	}

	entry, err := selectRegistryEntryForDescribe(entries, `C:\root\repo-b`, `C:\workspace`, []string{`C:\root`})
	if err != nil {
		t.Fatalf("expected absolute path selector to match, got error: %v", err)
	}
	if entry.RepoID != "github.com/org/repo-b" {
		t.Fatalf("unexpected absolute-path match: %#v", entry)
	}
}

func TestCanonicalPathForMatchTreatsSlashRootAsAbsoluteOnWindows(t *testing.T) {
	got, ok := canonicalPathForMatch(`/tmp/root/repo-b`)
	if !ok {
		t.Fatal("expected slash-root path to canonicalize on windows")
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected canonicalized path to be absolute, got %q", got)
	}
}
