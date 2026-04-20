// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteAtomicCreatesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	if err := WriteAtomic(path, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi\n" {
		t.Fatalf("got %q want %q", got, "hi\n")
	}
	info, _ := os.Stat(path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
		t.Fatalf("got mode %v want 0644", info.Mode().Perm())
	}
}

func TestWriteAtomicOverwritesExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Fatalf("got %q want %q", got, "new")
	}
}

func TestWriteAtomicPreservesExistingMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode semantics differ on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "preserve.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Caller requests 0o644, but the pre-existing mode 0o600 must be kept.
	if err := WriteAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("got mode %v want 0600 (preserved)", info.Mode().Perm())
	}
}

func TestWriteAtomicFollowsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink requires privileges on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(link, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The symlink must still exist (not replaced by a regular file),
	// and the target file must have the new contents.
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced by regular file")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Fatalf("got %q want %q", got, "new")
	}
}

func TestWriteAtomicNoFileAtRequestedMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode semantics differ on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.json")
	if err := WriteAtomic(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("got mode %v want 0600 (fresh file)", info.Mode().Perm())
	}
}
