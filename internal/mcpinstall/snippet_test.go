// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestSnippetUnknownRuntime(t *testing.T) {
	t.Parallel()
	if _, err := Snippet("nope", Entry{Command: "/bin/x"}); err == nil {
		t.Fatal("expected unknown-runtime error")
	}
}

func TestSnippetClaudeIsValidJSON(t *testing.T) {
	t.Parallel()
	got, err := Snippet("claude", Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, got)
	}
	servers, ok := doc["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing: %v", doc)
	}
	entry := servers["repokeeper"].(map[string]any)
	if entry["command"] != "/bin/repokeeper" {
		t.Fatalf("command: %v", entry["command"])
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("snippet should end with newline: %q", got)
	}
}

func TestSnippetCodexIsValidTOML(t *testing.T) {
	t.Parallel()
	got, err := Snippet("codex", Entry{Command: "/bin/repokeeper", Args: []string{"mcp"}})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := toml.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatalf("not valid TOML: %v\n%s", err, got)
	}
	if !strings.Contains(got, "[mcp_servers.repokeeper]") {
		t.Fatalf("codex snippet missing table header: %s", got)
	}
}

func TestSnippetOpenCodeUsesArgvArray(t *testing.T) {
	t.Parallel()
	got, err := Snippet("opencode", Entry{Command: "/bin/repokeeper", Args: []string{"mcp", "-v"}})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	servers := doc["mcp"].(map[string]any)
	entry := servers["repokeeper"].(map[string]any)
	if entry["type"] != "local" {
		t.Fatalf("type: %v", entry["type"])
	}
	cmd, ok := entry["command"].([]any)
	if !ok || len(cmd) != 3 || cmd[0] != "/bin/repokeeper" || cmd[1] != "mcp" || cmd[2] != "-v" {
		t.Fatalf("expected argv array [/bin/repokeeper mcp -v], got %v", entry["command"])
	}
}
