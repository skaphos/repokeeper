// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// repokeeperKey is the MCP server name RepoKeeper registers under in
// every runtime's config file. Kept as a package-level constant so all
// adapters use the same name.
const repokeeperKey = "repokeeper"

type claudeAdapter struct{}

func init() {
	register(&claudeAdapter{})
}

func (a *claudeAdapter) Name() string { return "claude" }

func (a *claudeAdapter) Detect() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func (a *claudeAdapter) ConfigPath(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude.json"), nil
	case ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".mcp.json"), nil
	default:
		return "", fmt.Errorf("unknown scope: %v", scope)
	}
}

// claudeServer is the typed JSON shape of an entry in the mcpServers
// object. We round-trip through this type on both read and write so
// the serialized form stays canonical.
type claudeServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

func (a *claudeAdapter) ReadEntry(path string) (Entry, bool, error) {
	doc, err := readJSONDoc(path)
	if err != nil {
		return Entry{}, false, err
	}
	var servers map[string]any
	if raw, ok := doc["mcpServers"]; ok {
		m, ok := raw.(map[string]any)
		if !ok {
			return Entry{}, false, fmt.Errorf("parse %q: mcpServers is not a JSON object (got %T)", path, raw)
		}
		servers = m
	}
	if servers == nil {
		return Entry{}, false, nil
	}
	raw, ok := servers[repokeeperKey]
	if !ok {
		return Entry{}, false, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return Entry{}, false, fmt.Errorf("parse %q: mcpServers.%s is not a JSON object (got %T)", path, repokeeperKey, raw)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcpServers.%s: %w", path, repokeeperKey, err)
	}
	var srv claudeServer
	if err := json.Unmarshal(b, &srv); err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcpServers.%s: %w", path, repokeeperKey, err)
	}
	return Entry(srv), true, nil
}

func (a *claudeAdapter) WriteEntry(path string, e Entry) error {
	doc, err := readJSONDoc(path)
	if err != nil {
		return err
	}
	var servers map[string]any
	if raw, ok := doc["mcpServers"]; ok {
		m, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("parse %q: mcpServers is not a JSON object (got %T)", path, raw)
		}
		servers = m
	}
	if servers == nil {
		servers = map[string]any{}
	}
	servers[repokeeperKey] = claudeServer(e)
	doc["mcpServers"] = servers
	return writeJSONDoc(path, doc, 0o644)
}

func (a *claudeAdapter) RemoveEntry(path string) (bool, error) {
	doc, err := readJSONDoc(path)
	if err != nil {
		return false, err
	}
	var servers map[string]any
	if raw, ok := doc["mcpServers"]; ok {
		m, ok := raw.(map[string]any)
		if !ok {
			return false, fmt.Errorf("parse %q: mcpServers is not a JSON object (got %T)", path, raw)
		}
		servers = m
	}
	if servers == nil {
		return false, nil
	}
	if _, ok := servers[repokeeperKey]; !ok {
		return false, nil
	}
	delete(servers, repokeeperKey)
	doc["mcpServers"] = servers
	return true, writeJSONDoc(path, doc, 0o644)
}

// readJSONDoc parses the file at path as a top-level JSON object. A
// non-existent file returns an empty doc (nil is normalized to an
// empty map). A malformed file returns a wrapped parse error with
// the file path.
//
// Shared between the Claude and OpenCode adapters — both use JSON
// top-level-object configs.
func readJSONDoc(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return doc, nil
}

// writeJSONDoc marshals doc with stable 2-space indent, appends a
// trailing newline, creates the parent directory if necessary, and
// writes atomically.
func writeJSONDoc(path string, doc map[string]any, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return WriteAtomic(path, raw, mode)
}
