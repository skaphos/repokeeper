// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"reflect"
	"testing"

	"github.com/skaphos/repokeeper/internal/mcpserver"
)

// TestReadOnlyToolNames pins the read-only tool set derived from the live
// ReadOnlyHint annotations. If a tool is added, removed, or its annotation
// flipped, this fails — which is the signal to update the shipped
// permissions snippet (docs/mcp-setup.md and `install --manual`).
func TestReadOnlyToolNames(t *testing.T) {
	want := []string{
		"build_workspace_inventory",
		"get_authoritative_paths",
		"get_related_repositories",
		"get_repo_metadata",
		"get_repository_context",
		"get_workspace_config",
		"list_repositories",
		"plan_sync",
		"select_repositories",
	}
	got := mcpserver.ReadOnlyToolNames()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadOnlyToolNames() mismatch\n got: %v\nwant: %v", got, want)
	}
}

// TestReadOnlyToolNamesExcludesMutationTools guards the safety boundary: the
// mutation tools must never appear in the read-only set, or the shipped
// permissions.allow snippet would auto-approve them (the opposite of ADR-0001).
func TestReadOnlyToolNamesExcludesMutationTools(t *testing.T) {
	mutation := []string{
		"scan_workspace",
		"execute_sync",
		"set_labels",
		"add_repository",
		"remove_repository",
	}
	got := mcpserver.ReadOnlyToolNames()
	inSet := make(map[string]struct{}, len(got))
	for _, n := range got {
		inSet[n] = struct{}{}
	}
	for _, m := range mutation {
		if _, ok := inSet[m]; ok {
			t.Errorf("mutation tool %q must not be in the read-only set", m)
		}
	}
}
