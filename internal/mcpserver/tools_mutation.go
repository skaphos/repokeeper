// SPDX-License-Identifier: MIT
package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/sortutil"
)

// --- scan_workspace ---

type scanResponse struct {
	Discovered int             `json:"discovered"`
	New        int             `json:"new"`
	Missing    int             `json:"missing"`
	Pruned     int             `json:"pruned"`
	Repos      []scanRepoEntry `json:"repos"`
}

type scanRepoEntry struct {
	RepoID string `json:"repo_id"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

func (s *MCPServer) handleScanWorkspace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rootsRaw, err := optionalStringSliceArg(req, "roots")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pruneStale := req.GetBool("prune_stale", false)

	cfg := s.engine.Config()
	if cfg == nil {
		return mcp.NewToolResultError("config not loaded"), nil
	}

	scanRoots := rootsRaw
	if len(scanRoots) == 0 {
		scanRoots = []string{config.EffectiveRoot(s.cfgPath)}
	}

	reg := s.engine.Registry()
	prevCount := 0
	if reg != nil {
		prevCount = len(reg.Entries)
	}

	statuses, err := s.engine.Scan(ctx, engine.ScanOptions{
		Roots:   scanRoots,
		Exclude: cfg.Exclude,
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reg = s.engine.Registry()
	pruned := 0
	if pruneStale && reg != nil {
		staleDays := cfg.RegistryStaleDays
		if staleDays <= 0 {
			staleDays = 30
		}
		pruned = reg.PruneStale(time.Duration(staleDays) * 24 * time.Hour)
	}

	if reg != nil {
		sortutil.SortRegistryEntries(reg.Entries)
	}

	if err := s.saveConfig(); err != nil {
		return mcp.NewToolResultError("scan succeeded but failed to save: " + err.Error()), nil
	}

	newCount := 0
	if reg != nil && len(reg.Entries) > prevCount {
		newCount = len(reg.Entries) - prevCount
	}

	missing := 0
	repos := make([]scanRepoEntry, 0, len(statuses))
	for _, st := range statuses {
		status := "present"
		if st.Error != "" {
			status = "error"
			missing++
		}
		repos = append(repos, scanRepoEntry{
			RepoID: st.RepoID,
			Path:   st.Path,
			Status: status,
		})
	}

	resp := scanResponse{
		Discovered: len(statuses),
		New:        newCount,
		Missing:    missing,
		Pruned:     pruned,
		Repos:      repos,
	}
	return mcp.NewToolResultJSON(resp)
}

// --- plan_sync ---

type syncPlanEntry struct {
	RepoID     string `json:"repo_id"`
	Path       string `json:"path"`
	Action     string `json:"action"`
	Outcome    string `json:"outcome"`
	Planned    bool   `json:"planned"`
	SkipReason string `json:"skip_reason,omitempty"`
}

func (s *MCPServer) handlePlanSync(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := parseSyncOptions(req)
	opts.DryRun = true // plan_sync is always dry-run

	results, err := s.engine.Sync(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	entries := make([]syncPlanEntry, 0, len(results))
	for _, r := range results {
		entries = append(entries, syncPlanEntry{
			RepoID:     r.RepoID,
			Path:       r.Path,
			Action:     r.Action,
			Outcome:    string(r.Outcome),
			Planned:    true,
			SkipReason: r.SkipReason,
		})
	}

	return newStructuredListResult("plan", entries)
}

// --- execute_sync ---

type syncResultEntry struct {
	RepoID     string `json:"repo_id"`
	Path       string `json:"path"`
	Action     string `json:"action"`
	Outcome    string `json:"outcome"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
}

func (s *MCPServer) handleExecuteSync(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	confirm, err := requireStrictBoolArg(req, "confirm")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !confirm {
		return mcp.NewToolResultError("safety gate: execute_sync requires confirm=true"), nil
	}

	opts := parseSyncOptions(req)
	opts.DryRun = false
	opts.ContinueOnError = true

	// First get the plan, then execute it.
	planOpts := opts
	planOpts.DryRun = true
	plan, err := s.engine.Sync(ctx, planOpts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results, err := s.engine.ExecuteSyncPlanWithCallbacks(ctx, plan, opts, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	entries := make([]syncResultEntry, 0, len(results))
	for _, r := range results {
		entries = append(entries, syncResultEntry{
			RepoID:     r.RepoID,
			Path:       r.Path,
			Action:     r.Action,
			Outcome:    string(r.Outcome),
			OK:         r.OK,
			Error:      r.Error,
			SkipReason: r.SkipReason,
		})
	}

	return newStructuredListResult("results", entries)
}

// --- set_labels ---

type setLabelsResponse struct {
	RepoID string            `json:"repo_id"`
	Labels map[string]string `json:"labels"`
}

func (s *MCPServer) handleSetLabels(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoArg, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: repo"), nil
	}

	reg := s.engine.Registry()
	entry, err := resolveRepo(reg, repoArg)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	setRaw, err := optionalStringMapArg(req, "set")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	removeRaw, err := optionalStringSliceArg(req, "remove")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if entry.Labels == nil {
		entry.Labels = make(map[string]string)
	}

	for k, v := range setRaw {
		entry.Labels[k] = v
	}

	for _, key := range removeRaw {
		delete(entry.Labels, strings.TrimSpace(key))
	}

	if len(entry.Labels) == 0 {
		entry.Labels = nil
	}

	entry.LastSeen = time.Now()
	reg.UpdatedAt = time.Now()

	if err := s.saveConfig(); err != nil {
		return mcp.NewToolResultError("labels updated but failed to save: " + err.Error()), nil
	}

	return mcp.NewToolResultJSON(setLabelsResponse{
		RepoID: entry.RepoID,
		Labels: entry.Labels,
	})
}

// --- add_repository ---

type addRepoResponse struct {
	RepoID string `json:"repo_id"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

func (s *MCPServer) handleAddRepository(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: url"), nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: path"), nil
	}
	mirror := req.GetBool("mirror", false)

	if err := s.engine.CloneAndRegister(ctx, url, path, s.cfgPath, mirror); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Find the newly registered entry.
	reg := s.engine.Registry()
	repoID := "unknown"
	if reg != nil {
		for _, e := range reg.Entries {
			if e.Path == path {
				repoID = e.RepoID
				break
			}
		}
	}

	return mcp.NewToolResultJSON(addRepoResponse{
		RepoID: repoID,
		Path:   path,
		Status: "cloned",
	})
}

// --- remove_repository ---

type removeRepoResponse struct {
	RepoID  string `json:"repo_id"`
	Removed bool   `json:"removed"`
}

func (s *MCPServer) handleRemoveRepository(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoArg, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: repo"), nil
	}
	deleteFiles := req.GetBool("delete_files", false)

	if err := s.engine.DeleteRepo(context.Background(), repoArg, s.cfgPath, deleteFiles); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultJSON(removeRepoResponse{
		RepoID:  repoArg,
		Removed: true,
	})
}

// --- shared helpers ---

func parseSyncOptions(req mcp.CallToolRequest) engine.SyncOptions {
	filterRaw := strings.ToLower(strings.TrimSpace(req.GetString("filter", "all")))
	return engine.SyncOptions{
		Filter:      engine.FilterKind(filterRaw),
		UpdateLocal: req.GetBool("update_local", false),
		PushLocal:   req.GetBool("push_local", false),
		Force:       req.GetBool("force", false),
	}
}

func optionalStringSliceArg(req mcp.CallToolRequest, name string) ([]string, error) {
	raw, ok := req.GetArguments()[name]
	if !ok || raw == nil {
		return nil, nil
	}

	switch items := raw.(type) {
	case []string:
		return append([]string(nil), items...), nil
	case []any:
		values := make([]string, 0, len(items))
		for i, item := range items {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("argument %q item %d must be a string", name, i)
			}
			values = append(values, value)
		}

		return values, nil
	default:
		return nil, fmt.Errorf("argument %q must be an array of strings", name)
	}
}

func optionalStringMapArg(req mcp.CallToolRequest, name string) (map[string]string, error) {
	raw, ok := req.GetArguments()[name]
	if !ok || raw == nil {
		return nil, nil
	}

	switch items := raw.(type) {
	case map[string]string:
		values := make(map[string]string, len(items))
		for key, value := range items {
			values[key] = value
		}
		return values, nil
	case map[string]any:
		values := make(map[string]string, len(items))
		for key, item := range items {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("argument %q key %q must have a string value", name, key)
			}
			values[key] = value
		}
		return values, nil
	default:
		return nil, fmt.Errorf("argument %q must be an object with string values", name)
	}
}

func requireStrictBoolArg(req mcp.CallToolRequest, name string) (bool, error) {
	raw, ok := req.GetArguments()[name]
	if !ok {
		return false, fmt.Errorf("required argument %q not found", name)
	}

	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("argument %q must be a boolean", name)
	}

	return value, nil
}

// saveConfig persists the current config (including embedded registry) to disk.
func (s *MCPServer) saveConfig() error {
	cfg := s.engine.Config()
	if cfg == nil {
		return nil
	}
	cfg.Registry = s.engine.Registry()
	return config.Save(cfg, s.cfgPath)
}
