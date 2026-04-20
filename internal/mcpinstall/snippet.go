// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"fmt"
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
				repokeeperKey: claudeServer(e),
			},
		})
	case "codex":
		return renderTOML(map[string]any{
			"mcp_servers": map[string]any{
				repokeeperKey: codexServer(e),
			},
		})
	case "opencode":
		argv := append([]string{e.Command}, e.Args...)
		return renderJSON(map[string]any{
			"mcp": map[string]any{
				repokeeperKey: opencodeServer{Type: "local", Command: argv, Enabled: true},
			},
		})
	default:
		return "", fmt.Errorf("unknown runtime: %q", runtime)
	}
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
