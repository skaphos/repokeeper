// SPDX-License-Identifier: MIT
package vcs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/strutil"
)

// ParseAdapterSelection parses --vcs selections.
func ParseAdapterSelection(raw string) ([]string, error) {
	values := strutil.SplitCSV(raw)
	if len(values) == 0 {
		return []string{"git"}, nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := strings.ToLower(strings.TrimSpace(value))
		switch name {
		case "git", "hg":
		default:
			return nil, fmt.Errorf("unsupported vcs %q (supported: git,hg)", value)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return []string{"git"}, nil
	}
	return out, nil
}

// NewAdapterForSelection creates an adapter for --vcs selection.
func NewAdapterForSelection(raw string) (Adapter, error) {
	selected, err := ParseAdapterSelection(raw)
	if err != nil {
		return nil, err
	}
	adapters := make([]Adapter, 0, len(selected))
	for _, name := range selected {
		switch name {
		case "git":
			adapters = append(adapters, NewGitAdapter(nil))
		case "hg":
			adapters = append(adapters, NewHgAdapter())
		}
	}
	if len(adapters) == 1 {
		return adapters[0], nil
	}
	return &MultiAdapter{
		adapters: adapters,
		byPath:   map[string]Adapter{},
	}, nil
}

// MultiAdapter delegates per-path operations to the first matching backend.
// This enables --vcs=git,hg scans/status in mixed roots.
type MultiAdapter struct {
	adapters []Adapter
	byPath   map[string]Adapter
	mu       sync.Mutex
}

func (m *MultiAdapter) Name() string { return "multi" }

func (m *MultiAdapter) IsRepo(ctx context.Context, dir string) (bool, error) {
	adapter, ok := m.detectAdapter(ctx, dir)
	if !ok {
		return false, nil
	}
	m.cache(dir, adapter)
	return true, nil
}

func (m *MultiAdapter) IsBare(ctx context.Context, dir string) (bool, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return false, err
	}
	return adapter.IsBare(ctx, dir)
}

func (m *MultiAdapter) Remotes(ctx context.Context, dir string) ([]model.Remote, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return nil, err
	}
	return adapter.Remotes(ctx, dir)
}

func (m *MultiAdapter) Head(ctx context.Context, dir string) (model.Head, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return model.Head{}, err
	}
	return adapter.Head(ctx, dir)
}

func (m *MultiAdapter) WorktreeStatus(ctx context.Context, dir string) (*model.Worktree, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return nil, err
	}
	return adapter.WorktreeStatus(ctx, dir)
}

func (m *MultiAdapter) TrackingStatus(ctx context.Context, dir string) (model.Tracking, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return model.Tracking{}, err
	}
	return adapter.TrackingStatus(ctx, dir)
}

func (m *MultiAdapter) HasSubmodules(ctx context.Context, dir string) (bool, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return false, err
	}
	return adapter.HasSubmodules(ctx, dir)
}

func (m *MultiAdapter) Fetch(ctx context.Context, dir string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.Fetch(ctx, dir)
}

func (m *MultiAdapter) PullRebase(ctx context.Context, dir string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.PullRebase(ctx, dir)
}

func (m *MultiAdapter) Push(ctx context.Context, dir string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.Push(ctx, dir)
}

func (m *MultiAdapter) SetUpstream(ctx context.Context, dir, upstream, branch string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.SetUpstream(ctx, dir, upstream, branch)
}

func (m *MultiAdapter) SetRemoteURL(ctx context.Context, dir, remote, remoteURL string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.SetRemoteURL(ctx, dir, remote, remoteURL)
}

func (m *MultiAdapter) StashPush(ctx context.Context, dir, message string) (bool, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return false, err
	}
	return adapter.StashPush(ctx, dir, message)
}

func (m *MultiAdapter) StashPop(ctx context.Context, dir string) error {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return err
	}
	return adapter.StashPop(ctx, dir)
}

func (m *MultiAdapter) Clone(ctx context.Context, remoteURL, targetPath, branch string, mirror bool) error {
	// Clone target backend defaults to the first selected adapter.
	return m.adapters[0].Clone(ctx, remoteURL, targetPath, branch, mirror)
}

func (m *MultiAdapter) NormalizeURL(rawURL string) string {
	return NewGitAdapter(nil).NormalizeURL(rawURL)
}

func (m *MultiAdapter) PrimaryRemote(remoteNames []string) string {
	return NewGitAdapter(nil).PrimaryRemote(remoteNames)
}

// SupportsLocalUpdate reports local-update capability for the detected backend.
func (m *MultiAdapter) SupportsLocalUpdate(ctx context.Context, dir string) (bool, string, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return false, "", err
	}
	capable, ok := adapter.(interface {
		SupportsLocalUpdate(context.Context, string) (bool, string, error)
	})
	if !ok {
		return true, "", nil
	}
	return capable.SupportsLocalUpdate(ctx, dir)
}

// FetchAction returns the fetch action for the detected backend.
func (m *MultiAdapter) FetchAction(ctx context.Context, dir string) (string, error) {
	adapter, err := m.adapterForPath(ctx, dir)
	if err != nil {
		return "", err
	}
	provider, ok := adapter.(interface {
		FetchAction(context.Context, string) (string, error)
	})
	if !ok {
		return "git fetch --all --prune --prune-tags --no-recurse-submodules", nil
	}
	return provider.FetchAction(ctx, dir)
}

func (m *MultiAdapter) adapterForPath(ctx context.Context, dir string) (Adapter, error) {
	m.mu.Lock()
	if adapter, ok := m.byPath[dir]; ok {
		m.mu.Unlock()
		return adapter, nil
	}
	m.mu.Unlock()

	adapter, ok := m.detectAdapter(ctx, dir)
	if !ok {
		return nil, fmt.Errorf("no selected vcs adapter matched repo at %q", dir)
	}
	m.cache(dir, adapter)
	return adapter, nil
}

func (m *MultiAdapter) detectAdapter(ctx context.Context, dir string) (Adapter, bool) {
	for _, adapter := range m.adapters {
		ok, err := adapter.IsRepo(ctx, dir)
		if err != nil || !ok {
			continue
		}
		return adapter, true
	}
	return nil, false
}

func (m *MultiAdapter) cache(dir string, adapter Adapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byPath[dir] = adapter
}
