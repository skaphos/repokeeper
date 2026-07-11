// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	return Entry{
		Command: srv.Command,
		Args:    srv.Args,
		Enabled: true,
	}, true, nil
}

func (a *codexAdapter) WriteEntry(path string, e Entry) error {
	// Guard before any rewrite so a commented config is never clobbered.
	if err := refuseIfTOMLComments(path); err != nil {
		return err
	}
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
	servers[repokeeperKey] = codexServer{Command: e.Command, Args: e.Args}
	doc["mcp_servers"] = servers
	return writeTOMLDoc(path, doc, newConfigFileMode)
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
	// An entry exists and will be rewritten out; guard comments only now,
	// since a no-op remove (handled above) never rewrites the file.
	if err := refuseIfTOMLComments(path); err != nil {
		return false, err
	}
	delete(servers, repokeeperKey)
	doc["mcp_servers"] = servers
	return true, writeTOMLDoc(path, doc, newConfigFileMode)
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

// refuseIfTOMLComments returns an error if the TOML file at path contains
// comments.
//
// Design note (data-loss avoidance): these adapters edit config by reading the
// file into a bare map[string]any, mutating it, and re-marshaling. go-toml/v2
// has no comment trivia on that map, so every round-trip erases ALL comments
// in the user's hand-maintained Codex/Grok config.toml. A lossless in-place
// TOML edit that preserves comments is not feasible with the current library
// within this change's scope, so we take the conservative fail-safe path:
// detect any comment and refuse to rewrite the file, directing the user to
// edit it manually. Non-existent/empty files (and files with no comments) are
// written normally. tomlHasComments biases toward detection, so at worst we
// refuse a safe write — we never silently destroy comments.
func refuseIfTOMLComments(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if tomlHasComments(raw) {
		return fmt.Errorf("refusing to rewrite %q: it contains comments this installer cannot preserve; add or remove the [mcp_servers.%s] entry manually", path, repokeeperKey)
	}
	return nil
}

// tomlHasComments reports whether raw contains a TOML comment (an unquoted
// '#'). It skips over basic ("..."), literal ('...'), and multi-line
// ("""...""" / '''...''') strings so a '#' inside a string value is not
// mistaken for a comment. Malformed/unterminated strings cause the remainder
// to be treated as string content; such input is rejected earlier by
// readTOMLDoc's parse, so it never reaches a write.
func tomlHasComments(raw []byte) bool {
	s := string(raw)
	for i := 0; i < len(s); {
		switch {
		case s[i] == '#':
			return true
		case s[i] == '"':
			if strings.HasPrefix(s[i:], `"""`) {
				i = skipTOMLDelim(s, i+3, `"""`)
			} else {
				i = skipTOMLSingleLine(s, i+1, '"', true)
			}
		case s[i] == '\'':
			if strings.HasPrefix(s[i:], `'''`) {
				i = skipTOMLDelim(s, i+3, `'''`)
			} else {
				i = skipTOMLSingleLine(s, i+1, '\'', false)
			}
		default:
			i++
		}
	}
	return false
}

// skipTOMLSingleLine advances past a single-line string body starting at i
// (just after the opening quote) and returns the index after the closing
// quote. A single-line string never spans a newline; if one is reached the
// string is treated as ended there. When escapes is true (basic strings), a
// backslash escapes the next byte.
func skipTOMLSingleLine(s string, i int, quote byte, escapes bool) int {
	for i < len(s) {
		switch {
		case escapes && s[i] == '\\':
			i += 2
		case s[i] == '\n':
			return i
		case s[i] == quote:
			return i + 1
		default:
			i++
		}
	}
	return i
}

// skipTOMLDelim returns the index just past the next occurrence of delim at or
// after i, or len(s) if delim does not occur again (unterminated multi-line
// string).
func skipTOMLDelim(s string, i int, delim string) int {
	if idx := strings.Index(s[i:], delim); idx >= 0 {
		return i + idx + len(delim)
	}
	return len(s)
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
