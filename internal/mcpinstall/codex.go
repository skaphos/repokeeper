// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type codexAdapter struct{}

func init() {
	register(&codexAdapter{})
}

func (a *codexAdapter) Name() string { return "codex" }

func (a *codexAdapter) Detect() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(home, ".codex")); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func (a *codexAdapter) ConfigPath(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".codex", "config.toml"), nil
	case ScopeProject:
		return "", ErrScopeUnsupported
	default:
		return "", fmt.Errorf("unknown scope: %v", scope)
	}
}

// codexServer is the typed TOML shape of an mcp_servers entry. Tags
// are explicit even though they match Go field names, so TOML
// round-trips stay stable across go-toml/v2 version bumps.
type codexServer struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args,omitempty"`
}

func (a *codexAdapter) ReadEntry(path string) (Entry, bool, error) {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return Entry{}, false, err
	}
	servers, err := codexServersMap(doc, path)
	if err != nil {
		return Entry{}, false, err
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
		return Entry{}, false, fmt.Errorf("parse %q: mcp_servers.%s is not a TOML table (got %T)", path, repokeeperKey, raw)
	}
	// Round-trip through TOML to normalize into the typed struct.
	b, err := toml.Marshal(m)
	if err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcp_servers.%s: %w", path, repokeeperKey, err)
	}
	var srv codexServer
	if err := toml.Unmarshal(b, &srv); err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcp_servers.%s: %w", path, repokeeperKey, err)
	}
	return Entry(srv), true, nil
}

func (a *codexAdapter) WriteEntry(path string, e Entry) error {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	servers, err := codexServersMap(doc, path)
	if err != nil {
		return err
	}
	if servers == nil {
		servers = map[string]any{}
	}
	// Use a typed struct so TOML serializes the table in a canonical form.
	servers[repokeeperKey] = codexServer(e)
	doc["mcp_servers"] = servers
	return writeTOMLDoc(path, doc, 0o644)
}

func (a *codexAdapter) RemoveEntry(path string) (bool, error) {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return false, err
	}
	servers, err := codexServersMap(doc, path)
	if err != nil {
		return false, err
	}
	if servers == nil {
		return false, nil
	}
	if _, ok := servers[repokeeperKey]; !ok {
		return false, nil
	}
	delete(servers, repokeeperKey)
	doc["mcp_servers"] = servers
	return true, writeTOMLDoc(path, doc, 0o644)
}

// codexServersMap returns the mcp_servers table from doc, or nil if
// no mcp_servers key is present. Returns an error if the key exists
// but the value is not a table (TOML sub-table -> map[string]any).
func codexServersMap(doc map[string]any, path string) (map[string]any, error) {
	raw, ok := doc["mcp_servers"]
	if !ok {
		return nil, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parse %q: mcp_servers is not a TOML table (got %T)", path, raw)
	}
	return m, nil
}

// readTOMLDoc parses the file at path as a top-level TOML document.
// Non-existent or empty files return an empty doc. Malformed TOML
// returns a wrapped parse error with the file path.
func readTOMLDoc(path string) (map[string]any, error) {
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
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return doc, nil
}

// writeTOMLDoc marshals doc as TOML and writes atomically, creating
// the parent directory if necessary.
func writeTOMLDoc(path string, doc map[string]any, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := toml.Marshal(doc)
	if err != nil {
		return err
	}
	return WriteAtomic(path, raw, mode)
}
