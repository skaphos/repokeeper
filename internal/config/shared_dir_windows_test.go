// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// grantEveryone adds an access-allowed ACE for Everyone with the given rights
// and inheritance to dir's DACL.
func grantEveryone(t *testing.T, dir string, rights windows.ACCESS_MASK, inheritance uint32) {
	t.Helper()
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatalf("create Everyone SID: %v", err)
	}
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("read security info: %v", err)
	}
	current, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("read DACL: %v", err)
	}
	updated, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: rights,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       inheritance,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, current)
	if err != nil {
		t.Fatalf("build DACL: %v", err)
	}
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, updated, nil); err != nil {
		t.Fatalf("set DACL: %v", err)
	}
}

func TestIsSharedDirWindows(t *testing.T) {
	t.Run("private temp dir is not shared", func(t *testing.T) {
		if isSharedDir(t.TempDir()) {
			t.Fatal("expected a per-user temp dir to be private")
		}
	})
	t.Run("Everyone may add files", func(t *testing.T) {
		dir := t.TempDir()
		grantEveryone(t, dir, windows.FILE_WRITE_DATA, windows.NO_INHERITANCE)
		if !isSharedDir(dir) {
			t.Fatal("expected a dir where Everyone can add files to be shared")
		}
	})
	t.Run("Everyone may take ownership", func(t *testing.T) {
		dir := t.TempDir()
		grantEveryone(t, dir, windows.WRITE_OWNER, windows.NO_INHERITANCE)
		if !isSharedDir(dir) {
			t.Fatal("expected a dir whose owner Everyone can change to be shared")
		}
	})
	t.Run("read-only grant is not shared", func(t *testing.T) {
		dir := t.TempDir()
		grantEveryone(t, dir, windows.GENERIC_READ|windows.GENERIC_EXECUTE, windows.NO_INHERITANCE)
		if isSharedDir(dir) {
			t.Fatal("expected a read-only grant to keep the dir private")
		}
	})
	t.Run("inherit-only grant does not apply to the dir itself", func(t *testing.T) {
		dir := t.TempDir()
		grantEveryone(t, dir, windows.FILE_WRITE_DATA, windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT|windows.INHERIT_ONLY)
		if isSharedDir(dir) {
			t.Fatal("expected an inherit-only grant to leave the dir itself private")
		}
	})
}

// TestFindNearestConfigPathDoesNotWalkIntoSharedDirWindows is the Windows
// counterpart of TestFindNearestConfigPathDoesNotWalkIntoSharedDir.
func TestFindNearestConfigPathDoesNotWalkIntoSharedDirWindows(t *testing.T) {
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	planted := filepath.Join(shared, LocalConfigFilename)
	if err := os.WriteFile(planted, []byte("exclude: []\n"), 0o644); err != nil {
		t.Fatalf("write planted config: %v", err)
	}
	child := filepath.Join(shared, "victim")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir victim: %v", err)
	}
	grantEveryone(t, shared, windows.FILE_WRITE_DATA, windows.NO_INHERITANCE)

	got, err := findNearestConfigPath(child)
	if err != nil {
		t.Fatalf("findNearestConfigPath: %v", err)
	}
	if got != "" {
		t.Fatalf("expected walk to stop before adopting planted config, got %q", got)
	}
}
