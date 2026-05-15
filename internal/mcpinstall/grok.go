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

type grokAdapter struct{}

func init() {
	register(&grokAdapter{})
}

func (a *grokAdapter) Name() string { return "grok" }

// grokDir resolves Grok's user-scope config directory.
// It respects GROK_CONFIG_DIR if set; otherwise falls back to ~/.grok.
func grokDir() (string, error) {
	if v := os.Getenv("GROK_CONFIG_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".grok"), nil
}

func (a *grokAdapter) Detect() (bool, error) {
	if os.Getenv("GROK_CONFIG_DIR") != "" {
		return true, nil
	}
	dir, err := grokDir()
	if err != nil {
		return false, err
	}
	// Detect if the config file exists or the directory exists (user has Grok configured).
	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if _, err := os.Stat(dir); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func (a *grokAdapter) ConfigPath(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		dir, err := grokDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "config.toml"), nil
	case ScopeProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".grok", "config.toml"), nil
	default:
		return "", fmt.Errorf("unknown scope: %v", scope)
	}
}

// grokServer is the typed TOML shape of an mcp_servers entry for Grok.
// Grok requires an explicit "enabled" boolean (default true when omitted by Grok,
// but we always write true for explicit registration).
type grokServer struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args,omitempty"`
	Enabled bool              `toml:"enabled"`
	Env     map[string]string `toml:"env,omitempty"`
}

func (a *grokAdapter) ReadEntry(path string) (Entry, bool, error) {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return Entry{}, false, err
	}
	servers, err := grokServersMap(doc, path)
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
	b, err := toml.Marshal(m)
	if err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcp_servers.%s: %w", path, repokeeperKey, err)
	}
	var srv grokServer
	if err := toml.Unmarshal(b, &srv); err != nil {
		return Entry{}, false, fmt.Errorf("parse %q: mcp_servers.%s: %w", path, repokeeperKey, err)
	}
	return Entry{Command: srv.Command, Args: srv.Args, Enabled: srv.Enabled}, true, nil
}

func (a *grokAdapter) WriteEntry(path string, e Entry) error {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	servers, err := grokServersMap(doc, path)
	if err != nil {
		return err
	}
	if servers == nil {
		servers = map[string]any{}
	}
	servers[repokeeperKey] = grokServer{
		Command: e.Command,
		Args:    e.Args,
		Enabled: e.Enabled,
	}
	doc["mcp_servers"] = servers
	return writeTOMLDoc(path, doc, 0o644)
}

func (a *grokAdapter) RemoveEntry(path string) (bool, error) {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return false, err
	}
	servers, err := grokServersMap(doc, path)
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

// grokServersMap returns the mcp_servers table from doc, or nil if
// no mcp_servers key is present. Returns an error if the key exists
// but the value is not a table.
func grokServersMap(doc map[string]any, path string) (map[string]any, error) {
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
