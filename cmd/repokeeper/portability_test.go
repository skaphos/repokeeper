package repokeeper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skaphos/repokeeper/internal/registry"
)

func TestIsDirectoryEmpty(t *testing.T) {
	dir := t.TempDir()
	empty, err := isDirectoryEmpty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !empty {
		t.Fatal("expected new temp dir to be empty")
	}

	f := filepath.Join(dir, "x")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	empty, err = isDirectoryEmpty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty {
		t.Fatal("expected dir with files to be non-empty")
	}
}

func TestImportTargetRelativePath(t *testing.T) {
	entry := registry.Entry{
		RepoID: "github.com/org/repo-a",
		Path:   "/source/root/team/repo-a",
	}

	if got := importTargetRelativePath(entry, []string{"/source/root"}); got != "team/repo-a" {
		t.Fatalf("expected root-relative target, got %q", got)
	}

	if got := importTargetRelativePath(entry, []string{"/other/root"}); got != "repo-a" {
		t.Fatalf("expected basename fallback, got %q", got)
	}

	entry = registry.Entry{RepoID: "github.com/org/repo-z", Path: ""}
	if got := importTargetRelativePath(entry, nil); got != "repo-z" {
		t.Fatalf("expected repo-id fallback, got %q", got)
	}
}
