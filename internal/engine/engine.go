// Package engine orchestrates the core operations: scan, status, and sync.
// It coordinates between discovery, gitx, config, and registry packages.
package engine

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/discovery"
	"github.com/skaphos/repokeeper/internal/gitx"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// FilterKind represents the --only filter options.
type FilterKind string

const (
	FilterAll            FilterKind = "all"
	FilterErrors         FilterKind = "errors"
	FilterDirty          FilterKind = "dirty"
	FilterClean          FilterKind = "clean"
	FilterGone           FilterKind = "gone"
	FilterDiverged       FilterKind = "diverged"
	FilterRemoteMismatch FilterKind = "remote-mismatch"
	FilterMissing        FilterKind = "missing"
)

// Engine is the core orchestrator for RepoKeeper operations.
type Engine struct {
	Config   *config.Config
	Registry *registry.Registry
	Adapter  vcs.Adapter
}

// New creates a new Engine with the given configuration.
func New(cfg *config.Config, reg *registry.Registry, adapter vcs.Adapter) *Engine {
	return &Engine{
		Config:   cfg,
		Registry: reg,
		Adapter:  adapter,
	}
}

// ScanOptions configures a scan operation.
type ScanOptions struct {
	Roots          []string
	Exclude        []string
	FollowSymlinks bool
	WriteRegistry  bool
}

// Scan discovers repos and updates the registry.
func (e *Engine) Scan(ctx context.Context, opts ScanOptions) ([]model.RepoStatus, error) {
	if e.Adapter == nil {
		e.Adapter = vcs.NewGitAdapter(nil)
	}
	if e.Registry == nil {
		e.Registry = &registry.Registry{}
	}

	roots := opts.Roots
	if len(roots) == 0 {
		return nil, errors.New("no scan roots provided")
	}
	exclude := opts.Exclude
	if len(exclude) == 0 {
		exclude = e.Config.Exclude
	}

	if err := e.Registry.ValidatePaths(); err != nil {
		return nil, err
	}

	results, err := discovery.Scan(ctx, discovery.Options{
		Roots:          roots,
		Exclude:        exclude,
		FollowSymlinks: opts.FollowSymlinks,
		Adapter:        e.Adapter,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var statuses []model.RepoStatus
	for _, res := range results {
		repoID := res.RepoID
		if repoID == "" {
			repoID = "local:" + filepath.ToSlash(res.Path)
		}
		e.Registry.Upsert(registry.Entry{
			RepoID:    repoID,
			Path:      res.Path,
			RemoteURL: res.RemoteURL,
			LastSeen:  now,
			Status:    registry.StatusPresent,
		})

		statuses = append(statuses, model.RepoStatus{
			RepoID:        repoID,
			Path:          res.Path,
			Bare:          res.Bare,
			Remotes:       res.Remotes,
			PrimaryRemote: res.PrimaryRemote,
		})
	}
	sortRepoStatuses(statuses)
	e.Registry.UpdatedAt = now

	return statuses, nil
}

// StatusOptions configures a status operation.
type StatusOptions struct {
	Filter      FilterKind
	Concurrency int
	Timeout     int // seconds per repo
}

// Status inspects all registered repos and returns their status.
func (e *Engine) Status(ctx context.Context, opts StatusOptions) (*model.StatusReport, error) {
	if e.Adapter == nil {
		e.Adapter = vcs.NewGitAdapter(nil)
	}
	if e.Registry == nil {
		return nil, errors.New("registry not loaded")
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = e.Config.Defaults.Concurrency
		if concurrency <= 0 {
			concurrency = 4
		}
	}
	timeoutSeconds := opts.Timeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = e.Config.Defaults.TimeoutSeconds
	}

	type result struct {
		status model.RepoStatus
	}

	entries := e.Registry.Entries
	results := make([]model.RepoStatus, 0, len(entries))
	sem := make(chan struct{}, concurrency)
	out := make(chan result, len(entries))
	spawned := 0

	for _, entry := range entries {
		sem <- struct{}{}
		spawned++
		go func() {
			defer func() { <-sem }()
			if entry.Status == registry.StatusMissing {
				out <- result{status: model.RepoStatus{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Type:       entry.Type,
					Tracking:   model.Tracking{Status: model.TrackingNone},
					Error:      "path missing",
					ErrorClass: "missing",
				}}
				return
			}
			repoCtx := ctx
			if timeoutSeconds > 0 {
				var cancel context.CancelFunc
				repoCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
				defer cancel()
			}
			status, err := e.InspectRepo(repoCtx, entry.Path)
			if err != nil {
				// Preserve partial results: represent per-repo inspect failures in-band
				// instead of aborting the full status run.
				out <- result{status: model.RepoStatus{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Type:       entry.Type,
					Tracking:   model.Tracking{Status: model.TrackingNone},
					Error:      err.Error(),
					ErrorClass: gitx.ClassifyError(err),
				}}
				return
			}
			if status.RepoID == "" {
				status.RepoID = entry.RepoID
			}
			if entry.Type != "" {
				status.Type = entry.Type
			}
			out <- result{status: *status}
		}()
	}

	for i := 0; i < spawned; i++ {
		res := <-out
		if filterStatus(opts.Filter, res.status, e.Registry) {
			results = append(results, res.status)
		}
	}
	sortRepoStatuses(results)

	return &model.StatusReport{
		GeneratedAt: time.Now(),
		Repos:       results,
	}, nil
}

// SyncOptions configures a sync operation.
type SyncOptions struct {
	Filter               FilterKind
	Concurrency          int
	Timeout              int // seconds per repo
	ContinueOnError      bool
	DryRun               bool
	UpdateLocal          bool
	PushLocal            bool
	RebaseDirty          bool
	Force                bool
	ProtectedBranches    []string
	AllowProtectedRebase bool
	CheckoutMissing      bool
}

// SyncResult records the outcome for a single repo sync.
type SyncResult struct {
	RepoID     string
	Path       string
	Outcome    string
	OK         bool
	Error      string
	ErrorClass string
	Action     string
}

// ExecuteSyncPlan executes a previously computed dry-run sync plan.
// It avoids re-inspecting repo state so sync can analyze once and then apply.
func (e *Engine) ExecuteSyncPlan(ctx context.Context, plan []SyncResult, opts SyncOptions) ([]SyncResult, error) {
	if e.Adapter == nil {
		e.Adapter = vcs.NewGitAdapter(nil)
	}
	if e.Registry == nil {
		return nil, errors.New("registry not loaded")
	}

	results := make([]SyncResult, 0, len(plan))
	for _, item := range plan {
		// Non-dry-run execution only applies actions that were explicitly planned.
		if item.Error != "dry-run" {
			results = append(results, item)
			if !item.OK && !opts.ContinueOnError {
				break
			}
			continue
		}

		executed := item
		executed.Error = ""
		executed.ErrorClass = ""

		action := strings.ToLower(strings.TrimSpace(item.Action))
		if strings.Contains(action, "git clone") {
			entry := findRegistryEntryForSyncResult(e.Registry, item)
			if entry == nil {
				executed.OK = false
				executed.Outcome = "failed_invalid"
				executed.Error = "registry entry not found for planned clone"
				executed.ErrorClass = "invalid"
			} else if err := e.Adapter.Clone(ctx, strings.TrimSpace(entry.RemoteURL), entry.Path, strings.TrimSpace(entry.Branch), entry.Type == "mirror"); err != nil {
				executed.OK = false
				executed.Outcome = "failed_checkout_missing"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
			} else {
				executed.OK = true
				executed.Outcome = "checkout_missing"
				entry.Status = registry.StatusPresent
				entry.LastSeen = time.Now()
				e.Registry.Entries = replaceRegistryEntry(e.Registry.Entries, *entry)
			}
			results = append(results, executed)
			if !executed.OK && !opts.ContinueOnError {
				break
			}
			continue
		}

		var stashed bool
		if strings.Contains(action, "git fetch --all") {
			if err := e.Adapter.Fetch(ctx, item.Path); err != nil {
				executed.OK = false
				executed.Outcome = "failed_fetch"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
				results = append(results, executed)
				if !opts.ContinueOnError {
					break
				}
				continue
			}
			executed.Outcome = "fetched"
		}

		if strings.Contains(action, "stash push") {
			created, err := e.Adapter.StashPush(ctx, item.Path, "repokeeper: pre-rebase stash")
			if err != nil {
				executed.OK = false
				executed.Outcome = "failed_stash"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
				results = append(results, executed)
				if !opts.ContinueOnError {
					break
				}
				continue
			}
			stashed = created
		}

		if strings.Contains(action, "pull --rebase") {
			if err := e.Adapter.PullRebase(ctx, item.Path); err != nil {
				executed.OK = false
				executed.Outcome = "failed_rebase"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
				results = append(results, executed)
				if !opts.ContinueOnError {
					break
				}
				continue
			}
			executed.Outcome = outcomeForRebase(stashed)
		}

		if strings.Contains(action, "stash pop") && stashed {
			if err := e.Adapter.StashPop(ctx, item.Path); err != nil {
				executed.OK = false
				executed.Outcome = "failed_stash_pop"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
				results = append(results, executed)
				if !opts.ContinueOnError {
					break
				}
				continue
			}
		}

		if strings.Contains(action, "git push") {
			if err := e.Adapter.Push(ctx, item.Path); err != nil {
				executed.OK = false
				executed.Outcome = "failed_push"
				executed.Error = err.Error()
				executed.ErrorClass = gitx.ClassifyError(err)
				results = append(results, executed)
				if !opts.ContinueOnError {
					break
				}
				continue
			}
			executed.Outcome = "pushed"
		}

		executed.OK = true
		results = append(results, executed)
	}

	sortSyncResults(results)
	return results, nil
}

// Sync runs fetch/prune on repos matching the filter.
func (e *Engine) Sync(ctx context.Context, opts SyncOptions) ([]SyncResult, error) {
	if e.Adapter == nil {
		e.Adapter = vcs.NewGitAdapter(nil)
	}
	if e.Registry == nil {
		return nil, errors.New("registry not loaded")
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = e.Config.Defaults.Concurrency
		if concurrency <= 0 {
			concurrency = 4
		}
	}
	timeoutSeconds := opts.Timeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = e.Config.Defaults.TimeoutSeconds
	}

	type result struct {
		res SyncResult
		err error
	}

	entries := e.Registry.Entries
	mainBranch := "main"
	if e.Config != nil && strings.TrimSpace(e.Config.Defaults.MainBranch) != "" {
		mainBranch = strings.TrimSpace(e.Config.Defaults.MainBranch)
	}
	if !opts.ContinueOnError {
		// Preserve deterministic "stop on first failure" semantics by running one
		// entry at a time through the same Sync logic used for batch mode.
		results := make([]SyncResult, 0, len(entries))
		for _, entry := range entries {
			subReg := &registry.Registry{Entries: []registry.Entry{entry}}
			sub := &Engine{
				Config:   e.Config,
				Registry: subReg,
				Adapter:  e.Adapter,
			}
			subOpts := opts
			subOpts.ContinueOnError = true
			subOpts.Concurrency = 1
			subResults, err := sub.Sync(ctx, subOpts)
			if err != nil {
				return results, err
			}
			results = append(results, subResults...)
			for _, updated := range subReg.Entries {
				e.Registry.Entries = replaceRegistryEntry(e.Registry.Entries, updated)
			}
			if len(subResults) > 0 && !subResults[len(subResults)-1].OK {
				sortSyncResults(results)
				return results, nil
			}
		}
		sortSyncResults(results)
		return results, nil
	}

	sem := make(chan struct{}, concurrency)
	out := make(chan result, len(entries))
	spawned := 0
	results := make([]SyncResult, 0, len(entries))

	for _, entry := range entries {
		if opts.Filter == FilterMissing && entry.Status != registry.StatusMissing {
			continue
		}
		if entry.Status == registry.StatusMissing {
			if !opts.CheckoutMissing {
				results = append(results, SyncResult{RepoID: entry.RepoID, Path: entry.Path, Outcome: "skipped_missing", OK: false, Error: "missing"})
				continue
			}
			// Missing entries are recoverable only when we have enough material to
			// perform a fresh clone into the recorded path.
			remoteURL := strings.TrimSpace(entry.RemoteURL)
			if remoteURL == "" {
				results = append(results, SyncResult{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Outcome:    "failed_invalid",
					OK:         false,
					Error:      "missing remote_url for checkout",
					ErrorClass: "invalid",
				})
				continue
			}
			mirror := entry.Type == "mirror"
			branch := strings.TrimSpace(entry.Branch)
			action := "git clone"
			if mirror {
				action += " --mirror"
			} else if branch != "" {
				action += " --branch " + branch + " --single-branch"
			}
			action += " " + remoteURL + " " + entry.Path
			if opts.DryRun {
				// Dry-run reports the exact git action string that a live run would execute.
				results = append(results, SyncResult{
					RepoID:  entry.RepoID,
					Path:    entry.Path,
					Outcome: "planned_checkout_missing",
					OK:      true,
					Error:   "dry-run",
					Action:  action,
				})
				continue
			}
			if err := e.Adapter.Clone(ctx, remoteURL, entry.Path, branch, mirror); err != nil {
				results = append(results, SyncResult{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Outcome:    "failed_checkout_missing",
					OK:         false,
					Error:      err.Error(),
					ErrorClass: gitx.ClassifyError(err),
					Action:     action,
				})
				continue
			}
			entry.Status = registry.StatusPresent
			entry.LastSeen = time.Now()
			e.Registry.Entries = replaceRegistryEntry(e.Registry.Entries, entry)
			results = append(results, SyncResult{RepoID: entry.RepoID, Path: entry.Path, Outcome: "checkout_missing", OK: true, Action: action})
			continue
		}
		if opts.Filter == FilterGone && entry.Status != registry.StatusPresent {
			continue
		}
		if opts.Filter == FilterDirty || opts.Filter == FilterClean || opts.Filter == FilterGone || opts.Filter == FilterDiverged || opts.Filter == FilterRemoteMismatch {
			status, err := e.InspectRepo(ctx, entry.Path)
			if err != nil {
				results = append(results, SyncResult{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Outcome:    "failed_inspect",
					OK:         false,
					Error:      err.Error(),
					ErrorClass: gitx.ClassifyError(err),
				})
				continue
			}
			if opts.Filter == FilterDirty && (status.Worktree == nil || !status.Worktree.Dirty) {
				continue
			}
			if opts.Filter == FilterClean && status.Worktree != nil && status.Worktree.Dirty {
				continue
			}
			if opts.Filter == FilterGone && status.Tracking.Status != model.TrackingGone {
				continue
			}
			if opts.Filter == FilterDiverged && status.Tracking.Status != model.TrackingDiverged {
				continue
			}
			if opts.Filter == FilterRemoteMismatch && !hasRemoteMismatch(*status, entry) {
				continue
			}
		}
		if strings.TrimSpace(entry.RemoteURL) == "" {
			results = append(results, SyncResult{
				RepoID:     entry.RepoID,
				Path:       entry.Path,
				Outcome:    "skipped_no_upstream",
				OK:         true,
				ErrorClass: "skipped",
				Error:      "skipped-no-upstream",
			})
			continue
		}
		sem <- struct{}{}
		spawned++
		go func() {
			defer func() { <-sem }()
			repoCtx := ctx
			if timeoutSeconds > 0 {
				var cancel context.CancelFunc
				repoCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
				defer cancel()
			}
			if opts.DryRun {
				action := "git fetch --all --prune --prune-tags --no-recurse-submodules"
				if opts.UpdateLocal {
					// We still inspect during dry-run so skip reasons and planned actions
					// match live execution as closely as possible.
					status, err := e.InspectRepo(repoCtx, entry.Path)
					if err != nil {
						out <- result{res: SyncResult{
							RepoID:     entry.RepoID,
							Path:       entry.Path,
							Outcome:    "failed_inspect",
							OK:         false,
							Error:      err.Error(),
							ErrorClass: gitx.ClassifyError(err),
						}}
						return
					}
					if opts.PushLocal && status.Tracking.Status == model.TrackingAhead {
						action += " && git push"
						out <- result{res: SyncResult{
							RepoID:  entry.RepoID,
							Path:    entry.Path,
							Outcome: "planned_push",
							OK:      true,
							Error:   "dry-run",
							Action:  action,
						}}
						return
					}
					if reason := pullRebaseSkipReason(
						status,
						mainBranch,
						opts.RebaseDirty,
						opts.Force,
						opts.ProtectedBranches,
						opts.AllowProtectedRebase,
					); reason != "" {
						out <- result{res: SyncResult{
							RepoID:     entry.RepoID,
							Path:       entry.Path,
							Outcome:    "skipped_local_update",
							OK:         true,
							ErrorClass: "skipped",
							Error:      "skipped-local-update: " + reason,
							Action:     action,
						}}
						return
					}
					action += " && git pull --rebase --no-recurse-submodules"
				}
				out <- result{res: SyncResult{
					RepoID:  entry.RepoID,
					Path:    entry.Path,
					Outcome: "planned_fetch",
					OK:      true,
					Error:   "dry-run",
					Action:  action,
				}}
				return
			}
			if opts.Filter == FilterGone {
				status, err := e.InspectRepo(repoCtx, entry.Path)
				if err != nil {
					out <- result{res: SyncResult{
						RepoID:     entry.RepoID,
						Path:       entry.Path,
						Outcome:    "failed_inspect",
						OK:         false,
						Error:      err.Error(),
						ErrorClass: gitx.ClassifyError(err),
					}}
					return
				}
				if status.Tracking.Status != model.TrackingGone {
					out <- result{res: SyncResult{RepoID: entry.RepoID, Path: entry.Path, Outcome: "skipped", OK: true, Error: "skipped"}}
					return
				}
			}
			err := e.Adapter.Fetch(repoCtx, entry.Path)
			if err != nil {
				out <- result{res: SyncResult{
					RepoID:     entry.RepoID,
					Path:       entry.Path,
					Outcome:    "failed_fetch",
					OK:         false,
					Error:      err.Error(),
					ErrorClass: gitx.ClassifyError(err),
				}}
				return
			}

			if opts.UpdateLocal {
				status, err := e.InspectRepo(repoCtx, entry.Path)
				if err != nil {
					out <- result{res: SyncResult{
						RepoID:     entry.RepoID,
						Path:       entry.Path,
						Outcome:    "failed_inspect",
						OK:         false,
						Error:      err.Error(),
						ErrorClass: gitx.ClassifyError(err),
					}}
					return
				}
				if opts.PushLocal && status.Tracking.Status == model.TrackingAhead {
					if err := e.Adapter.Push(repoCtx, entry.Path); err != nil {
						out <- result{res: SyncResult{
							RepoID:     entry.RepoID,
							Path:       entry.Path,
							Outcome:    "failed_push",
							OK:         false,
							Error:      err.Error(),
							ErrorClass: gitx.ClassifyError(err),
							Action:     "git push",
						}}
						return
					}
					out <- result{res: SyncResult{
						RepoID:  entry.RepoID,
						Path:    entry.Path,
						Outcome: "pushed",
						OK:      true,
						Action:  "git push",
					}}
					return
				}
				if reason := pullRebaseSkipReason(
					status,
					mainBranch,
					opts.RebaseDirty,
					opts.Force,
					opts.ProtectedBranches,
					opts.AllowProtectedRebase,
				); reason != "" {
					out <- result{res: SyncResult{
						RepoID:     entry.RepoID,
						Path:       entry.Path,
						Outcome:    "skipped_local_update",
						OK:         true,
						ErrorClass: "skipped",
						Error:      "skipped-local-update: " + reason,
					}}
					return
				}
				action := "git pull --rebase --no-recurse-submodules"
				stashed := false
				if opts.RebaseDirty && status.Worktree != nil && status.Worktree.Dirty {
					// Stash only when needed so we do not create unnecessary stash entries.
					stashed, err = e.Adapter.StashPush(repoCtx, entry.Path, "repokeeper: pre-rebase stash")
					if err != nil {
						out <- result{res: SyncResult{
							RepoID:     entry.RepoID,
							Path:       entry.Path,
							Outcome:    "failed_stash",
							OK:         false,
							Error:      err.Error(),
							ErrorClass: gitx.ClassifyError(err),
							Action:     "git stash push -u -m \"repokeeper: pre-rebase stash\"",
						}}
						return
					}
					if stashed {
						action = "git stash push -u -m \"repokeeper: pre-rebase stash\" && " + action
					}
				}
				if err := e.Adapter.PullRebase(repoCtx, entry.Path); err != nil {
					out <- result{res: SyncResult{
						RepoID:     entry.RepoID,
						Path:       entry.Path,
						Outcome:    "failed_rebase",
						OK:         false,
						Error:      err.Error(),
						ErrorClass: gitx.ClassifyError(err),
						Action:     action,
					}}
					return
				}
				if stashed {
					if err := e.Adapter.StashPop(repoCtx, entry.Path); err != nil {
						out <- result{res: SyncResult{
							RepoID:     entry.RepoID,
							Path:       entry.Path,
							Outcome:    "failed_stash_pop",
							OK:         false,
							Error:      err.Error(),
							ErrorClass: gitx.ClassifyError(err),
							Action:     action + " && git stash pop",
						}}
						return
					}
					action += " && git stash pop"
				}
				out <- result{res: SyncResult{
					RepoID:  entry.RepoID,
					Path:    entry.Path,
					Outcome: outcomeForRebase(stashed),
					OK:      true,
					Action:  action,
				}}
				return
			}
			out <- result{res: SyncResult{RepoID: entry.RepoID, Path: entry.Path, Outcome: "fetched", OK: true}}
		}()
	}

	for i := 0; i < spawned; i++ {
		res := <-out
		if res.err != nil {
			return nil, res.err
		}
		results = append(results, res.res)
	}
	sortSyncResults(results)
	return results, nil
}

func pullRebaseSkipReason(
	status *model.RepoStatus,
	mainBranch string,
	rebaseDirty, force bool,
	protectedBranches []string,
	allowProtectedRebase bool,
) string {
	// This function is intentionally ordered from hard-safety checks to
	// state-based policy checks so callers get stable, actionable reasons.
	if status == nil {
		return "unknown status"
	}
	if status.Bare {
		return "bare repository"
	}
	if status.Head.Detached {
		return "detached HEAD"
	}
	if matchesProtectedBranch(status.Head.Branch, protectedBranches) && !allowProtectedRebase {
		return fmt.Sprintf("branch %q is protected", status.Head.Branch)
	}
	if status.Worktree == nil {
		return "dirty state unknown"
	}
	if status.Worktree.Dirty && !rebaseDirty {
		return "dirty working tree"
	}
	if status.Tracking.Status == model.TrackingGone {
		return "upstream no longer exists"
	}
	if status.Tracking.Upstream == "" || status.Tracking.Status == model.TrackingNone {
		return "branch is not tracking an upstream"
	}
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch == "" {
		mainBranch = "main"
	}
	if !strings.HasSuffix(status.Tracking.Upstream, "/"+mainBranch) {
		return fmt.Sprintf("upstream %q is not %s", status.Tracking.Upstream, mainBranch)
	}
	if status.Tracking.Status == model.TrackingAhead {
		return "branch has local commits to push"
	}
	if status.Tracking.Status == model.TrackingDiverged && !force {
		return "branch has diverged (use --force to rebase anyway)"
	}
	if status.Tracking.Status == model.TrackingEqual {
		return "already up to date"
	}
	return ""
}

func matchesProtectedBranch(branch string, patterns []string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		if ok, err := path.Match(p, branch); err == nil && ok {
			return true
		}
	}
	return false
}

func outcomeForRebase(stashed bool) string {
	if stashed {
		return "stashed_rebased"
	}
	return "rebased"
}

// InspectRepo gathers the full status for a single repository path.
func (e *Engine) InspectRepo(ctx context.Context, path string) (*model.RepoStatus, error) {
	if e.Adapter == nil {
		e.Adapter = vcs.NewGitAdapter(nil)
	}
	bare, _ := e.Adapter.IsBare(ctx, path)

	remotes, err := e.Adapter.Remotes(ctx, path)
	if err != nil {
		return nil, err
	}
	var remoteNames []string
	for _, r := range remotes {
		remoteNames = append(remoteNames, r.Name)
	}
	primary := e.Adapter.PrimaryRemote(remoteNames)
	var remoteURL string
	for _, r := range remotes {
		if r.Name == primary {
			remoteURL = r.URL
			break
		}
	}
	repoID := e.Adapter.NormalizeURL(remoteURL)
	if repoID == "" {
		repoID = "local:" + filepath.ToSlash(path)
	}

	head, err := e.Adapter.Head(ctx, path)
	if err != nil {
		return nil, err
	}
	var worktree *model.Worktree
	if !bare {
		worktree, err = e.Adapter.WorktreeStatus(ctx, path)
		if err != nil {
			return nil, err
		}
	}
	tracking := model.Tracking{Status: model.TrackingNone}
	if !bare {
		tracking, err = e.Adapter.TrackingStatus(ctx, path)
		if err != nil {
			return nil, err
		}
	}
	hasSubmodules, _ := e.Adapter.HasSubmodules(ctx, path)

	return &model.RepoStatus{
		RepoID:        repoID,
		Path:          path,
		Bare:          bare,
		Remotes:       remotes,
		PrimaryRemote: primary,
		Head:          head,
		Worktree:      worktree,
		Tracking:      tracking,
		Submodules:    model.Submodules{HasSubmodules: hasSubmodules},
	}, nil
}

func filterStatus(kind FilterKind, status model.RepoStatus, reg *registry.Registry) bool {
	switch kind {
	case FilterAll:
		return true
	case FilterMissing:
		if reg == nil {
			return false
		}
		entry := reg.FindByRepoID(status.RepoID)
		return entry != nil && entry.Status == registry.StatusMissing
	case FilterDirty:
		return status.Worktree != nil && status.Worktree.Dirty
	case FilterClean:
		return status.Worktree != nil && !status.Worktree.Dirty
	case FilterGone:
		return status.Tracking.Status == model.TrackingGone
	case FilterDiverged:
		return status.Tracking.Status == model.TrackingDiverged
	case FilterRemoteMismatch:
		if reg == nil {
			return false
		}
		entry := findRegistryEntryForStatus(reg, status)
		if entry == nil {
			return false
		}
		return hasRemoteMismatch(status, *entry)
	case FilterErrors:
		return status.Error != ""
	default:
		return true
	}
}

func findRegistryEntryForStatus(reg *registry.Registry, status model.RepoStatus) *registry.Entry {
	if reg == nil {
		return nil
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == status.RepoID && reg.Entries[i].Path == status.Path {
			return &reg.Entries[i]
		}
	}
	return reg.FindByRepoID(status.RepoID)
}

func hasRemoteMismatch(status model.RepoStatus, entry registry.Entry) bool {
	regRemote := strings.TrimSpace(entry.RemoteURL)
	if regRemote == "" {
		return false
	}
	normalizedRegistry := gitx.NormalizeURL(regRemote)
	if normalizedRegistry == "" {
		normalizedRegistry = regRemote
	}
	normalizedStatus := strings.TrimSpace(status.RepoID)
	if normalizedStatus == "" {
		return false
	}
	return normalizedRegistry != normalizedStatus
}

func sortRepoStatuses(statuses []model.RepoStatus) {
	// Group logically by repo identity first, then path for multiple checkouts.
	sort.SliceStable(statuses, func(i, j int) bool {
		if statuses[i].RepoID == statuses[j].RepoID {
			return statuses[i].Path < statuses[j].Path
		}
		return statuses[i].RepoID < statuses[j].RepoID
	})
}

func sortSyncResults(results []SyncResult) {
	// Sync is concurrent; explicit sort keeps output deterministic.
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].RepoID == results[j].RepoID {
			return results[i].Action < results[j].Action
		}
		return results[i].RepoID < results[j].RepoID
	})
}

func replaceRegistryEntry(entries []registry.Entry, updated registry.Entry) []registry.Entry {
	for i := range entries {
		if entries[i].RepoID == updated.RepoID && entries[i].Path == updated.Path {
			entries[i] = updated
			return entries
		}
	}
	return entries
}

func findRegistryEntryForSyncResult(reg *registry.Registry, item SyncResult) *registry.Entry {
	if reg == nil {
		return nil
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == item.RepoID && reg.Entries[i].Path == item.Path {
			return &reg.Entries[i]
		}
	}
	for i := range reg.Entries {
		if reg.Entries[i].RepoID == item.RepoID {
			return &reg.Entries[i]
		}
	}
	return nil
}
