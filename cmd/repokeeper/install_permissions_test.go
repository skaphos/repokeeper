// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/mcpinstall"
	"github.com/skaphos/repokeeper/internal/mcpserver"
)

// expectedAllowList is the prefixed permissions.allow set derived from the live
// server annotations — the single source of truth both the docs and the manual
// output must match.
func expectedAllowList() []string {
	names := mcpserver.ReadOnlyToolNames()
	allow := make([]string, 0, len(names))
	for _, n := range names {
		allow = append(allow, mcpinstall.ClaudePermissionToolName(n))
	}
	sort.Strings(allow)
	return allow
}

// mutationTools are the tools that must never appear in permissions.allow, so
// they keep prompting (ADR-0001).
var mutationTools = []string{
	"scan_workspace",
	"execute_sync",
	"set_labels",
	"add_repository",
	"remove_repository",
}

// TestInstallManualClaudeEmitsPermissionsBlock asserts `install --manual=claude`
// prints a permissions.allow block listing exactly the read-only tools and none
// of the mutation tools.
func TestInstallManualClaudeEmitsPermissionsBlock(t *testing.T) {
	withInstallEnv(t)
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "claude",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --manual=claude: %v", err)
	}
	out := stdout.String()

	if !strings.Contains(out, "\"permissions\"") || !strings.Contains(out, "\"allow\"") {
		t.Fatalf("expected a permissions.allow block, got: %q", out)
	}
	for _, want := range expectedAllowList() {
		if !strings.Contains(out, "\""+want+"\"") {
			t.Errorf("manual output missing read-only permission %q", want)
		}
	}
	for _, m := range mutationTools {
		perm := mcpinstall.ClaudePermissionToolName(m)
		if strings.Contains(out, "\""+perm+"\"") {
			t.Errorf("manual output must not allow-list mutation tool %q", perm)
		}
	}
}

// TestInstallManualSingleTargetExcludesClaudePermissions confirms the
// permissions block is Claude-specific: a non-claude target must not emit it.
func TestInstallManualSingleTargetExcludesClaudePermissions(t *testing.T) {
	withInstallEnv(t)
	stdout, _, err := runInstallWithFlags(t, map[string]string{
		"manual":  "codex",
		"command": "/fake/repokeeper",
	})
	if err != nil {
		t.Fatalf("install --manual=codex: %v", err)
	}
	if strings.Contains(stdout.String(), "\"permissions\"") {
		t.Errorf("codex manual output should not contain a Claude permissions block")
	}
}

// TestDocsPermissionsSnippetMatchesReadOnlyTools is the drift guard: the
// allow-list hardcoded in docs/mcp-setup.md must equal the read-only tool set
// currently registered. Adding/removing a tool without updating the docs fails.
func TestDocsPermissionsSnippetMatchesReadOnlyTools(t *testing.T) {
	const docPath = "../../docs/mcp-setup.md"
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	content := string(data)

	marker := "## Recommended Claude Code permissions"
	idx := strings.Index(content, marker)
	if idx < 0 {
		t.Fatalf("docs missing %q section", marker)
	}
	section := content[idx:]

	start := strings.Index(section, "```json")
	if start < 0 {
		t.Fatal("docs permissions section has no ```json block")
	}
	start += len("```json")
	end := strings.Index(section[start:], "```")
	if end < 0 {
		t.Fatal("docs permissions json block is not closed")
	}
	jsonText := section[start : start+end]

	var doc struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(jsonText), &doc); err != nil {
		t.Fatalf("docs permissions snippet is not valid JSON: %v\n%s", err, jsonText)
	}

	got := append([]string(nil), doc.Permissions.Allow...)
	sort.Strings(got)
	want := expectedAllowList()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("docs/mcp-setup.md permissions.allow drifted from registered read-only tools\n got: %v\nwant: %v", got, want)
	}
}
