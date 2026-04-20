// SPDX-License-Identifier: MIT
package mcpinstall

import (
	"errors"
	"sort"
)

// Scope selects user-scoped or project-scoped config files.
type Scope int

const (
	ScopeUser Scope = iota
	ScopeProject
)

func (s Scope) String() string {
	switch s {
	case ScopeUser:
		return "user"
	case ScopeProject:
		return "project"
	default:
		return "unknown"
	}
}

// Entry is the runtime-agnostic MCP server entry we write into each
// agent's config file. Runtime adapters translate this to their native
// shape (JSON object vs TOML table, command+args vs argv-array, etc).
type Entry struct {
	Command string
	Args    []string
}

// ErrScopeUnsupported is returned by adapters that do not support a
// given Scope (e.g. Codex has no project scope).
var ErrScopeUnsupported = errors.New("scope not supported by this runtime")

// Runtime is the adapter contract. Implementations live alongside this
// file (claude.go, codex.go, opencode.go).
type Runtime interface {
	Name() string
	Detect() (bool, error)
	ConfigPath(scope Scope) (string, error)
	ReadEntry(path string) (entry Entry, present bool, err error)
	WriteEntry(path string, entry Entry) error
	RemoveEntry(path string) (removed bool, err error)
}

// registered holds all adapter instances. Populated via init() in each
// adapter file. Tests may replace this slice via the withFakes helper
// in runtime_test.go.
var registered []Runtime

func register(r Runtime) {
	registered = append(registered, r)
}

// All returns a copy of the registered adapters sorted by name for
// deterministic ordering in CLI output.
func All() []Runtime {
	out := make([]Runtime, len(registered))
	copy(out, registered)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// ByName looks up a single adapter by canonical name.
func ByName(name string) (Runtime, bool) {
	for _, r := range registered {
		if r.Name() == name {
			return r, true
		}
	}
	return nil, false
}

// Selection expresses which runtimes a CLI invocation targets. If
// Explicit is empty, the caller should use auto-detection; otherwise
// only the named runtimes are in scope.
type Selection struct {
	Explicit []string
}

// SelectionFromFlags builds a Selection from individual runtime bool
// flags. Order is deterministic: claude, codex, opencode.
func SelectionFromFlags(claude, codex, opencode bool) Selection {
	s := Selection{}
	if claude {
		s.Explicit = append(s.Explicit, "claude")
	}
	if codex {
		s.Explicit = append(s.Explicit, "codex")
	}
	if opencode {
		s.Explicit = append(s.Explicit, "opencode")
	}
	return s
}

// Resolve returns the adapters targeted by this Selection. If the
// Selection is empty (no Explicit runtimes), auto-detection via
// Detect() filters the registered adapters. Otherwise, the explicit
// list is honored even for runtimes Detect() would return false for.
func (s Selection) Resolve() ([]Runtime, error) {
	if len(s.Explicit) == 0 {
		present := []Runtime{}
		for _, r := range All() {
			ok, err := r.Detect()
			if err != nil {
				return nil, err
			}
			if ok {
				present = append(present, r)
			}
		}
		return present, nil
	}
	out := make([]Runtime, 0, len(s.Explicit))
	for _, name := range s.Explicit {
		r, ok := ByName(name)
		if !ok {
			return nil, errors.New("unknown runtime: " + name)
		}
		out = append(out, r)
	}
	return out, nil
}
