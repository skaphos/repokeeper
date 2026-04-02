// SPDX-License-Identifier: MIT
//go:build windows

package repokeeper

import (
	"path/filepath"
	"testing"

	"github.com/skaphos/repokeeper/internal/registry"
)

func TestInferRegistrySharedRootWindowsPaths(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r-present-a", Path: `C:\workspace\repos\team\repo-a`, Status: registry.StatusPresent},
			{RepoID: "r-present-b", Path: `C:\workspace\repos\team\repo-b`, Status: registry.StatusPresent},
			{RepoID: "r-missing", Path: `D:\legacy\repo-gone`, Status: registry.StatusMissing},
		},
	}

	got := inferRegistrySharedRoot(reg)
	want := filepath.Clean(`C:\workspace\repos\team`)
	if got != want {
		t.Fatalf("expected inferred root %q, got %q", want, got)
	}
}
