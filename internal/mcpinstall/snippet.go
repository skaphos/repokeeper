// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Snippet returns a self-contained config-file fragment that registers
// repokeeper under the named runtime. Intended for `repokeeper install
// --manual`, which prints snippets instead of rewriting config. The
// rendered form matches what the adapter would write, so a user can
// paste the snippet verbatim into their runtime's config.
func Snippet(runtime string, e Entry) (string, error) {
	switch runtime {
	case "claude":
		return renderJSON(map[string]any{
			"mcpServers": map[string]any{
				repokeeperKey: claudeServer{Command: e.Command, Args: e.Args},
			},
		})
	case "codex":
		return renderTOML(map[string]any{
			"mcp_servers": map[string]any{
				repokeeperKey: codexServer{Command: e.Command, Args: e.Args},
			},
		})
	case "opencode":
		argv := append([]string{e.Command}, e.Args...)
		return renderJSON(map[string]any{
			"mcp": map[string]any{
				repokeeperKey: opencodeServer{Type: "local", Command: argv, Enabled: e.Enabled},
			},
		})
	case "grok":
		return renderTOML(map[string]any{
			"mcp_servers": map[string]any{
				repokeeperKey: grokServer{Command: e.Command, Args: e.Args, Enabled: e.Enabled},
			},
		})
	default:
		return "", fmt.Errorf("unknown runtime: %q", runtime)
	}
}

// ClaudePermissionToolName returns the identifier Claude Code matches in
// permissions.allow for a repokeeper MCP tool, i.e. mcp__<server>__<tool>.
func ClaudePermissionToolName(tool string) string {
	return "mcp__" + repokeeperKey + "__" + tool
}

// ClaudePermissionsSnippet renders a ~/.claude/settings.json fragment that
// allow-lists the given tools under permissions.allow. Callers pass the
// read-only tool set only; mutation tools are intentionally omitted so they
// keep prompting (ADR-0001's read-and-plan safety model). Entries are sorted
// for deterministic output.
func ClaudePermissionsSnippet(toolNames []string) (string, error) {
	allow := make([]string, 0, len(toolNames))
	for _, t := range toolNames {
		allow = append(allow, ClaudePermissionToolName(t))
	}
	sort.Strings(allow)
	return renderJSON(map[string]any{
		"permissions": map[string]any{
			"allow": allow,
		},
	})
}

func renderJSON(doc map[string]any) (string, error) {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), "\n") + "\n", nil
}

func renderTOML(doc map[string]any) (string, error) {
	raw, err := toml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), "\n") + "\n", nil
}
