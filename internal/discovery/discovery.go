// SPDX-License-Identifier: MIT
// Package discovery walks configured root directories to find git repositories.
package discovery

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// Result represents a discovered git repository.
type Result struct {
	Path          string // absolute path to the repo root
	RepoID        string // normalized remote URL
	RemoteURL     string // raw remote URL of the primary remote
	PrimaryRemote string // primary remote name
	Remotes       []model.Remote
	Bare          bool // true if bare repo
}

// Options configures the discovery scan.
type Options struct {
	Roots          []string
	Exclude        []string // glob patterns to skip
	FollowSymlinks bool
	Adapter        vcs.Adapter
}

// Scan walks all roots and returns discovered repos.
// It skips directories matching exclude patterns and does not recurse
// into .git directories or matched exclusions.
func Scan(ctx context.Context, opts Options) ([]Result, error) {
	if opts.Adapter == nil {
		opts.Adapter = vcs.NewGitAdapter(nil)
	}

	warnInvalidExcludePatterns(opts.Exclude)

	var absRoots []string
	for _, root := range opts.Roots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		absRoots = append(absRoots, absRoot)
	}

	// Sort lexically so a parent directory is always processed before any
	// descendant, then skip roots already covered by a previously accepted
	// (shorter) root. This prevents overlapping roots, e.g. ["/A", "/A/sub"],
	// from being walked twice and producing duplicate results.
	sort.Strings(absRoots)

	visited := make(map[string]struct{})
	skipDirs := make(map[string]struct{})
	var acceptedRoots []string
	var results []Result

	for _, absRoot := range absRoots {
		if rootCovered(absRoot, acceptedRoots) {
			continue
		}
		acceptedRoots = append(acceptedRoots, absRoot)
		if err := walkRoot(ctx, absRoot, opts, visited, skipDirs, &results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// rootCovered reports whether path is equal to, or nested under, any of the
// already-accepted root directories.
func rootCovered(path string, accepted []string) bool {
	sep := string(filepath.Separator)
	for _, root := range accepted {
		if path == root {
			return true
		}
		// A filesystem-root root (e.g. "/" on Unix or "C:\" on Windows) already
		// ends in a separator; appending another would form "//" / "C:\\" and
		// break the prefix check, so only add a separator when one is absent.
		prefix := root
		if !strings.HasSuffix(prefix, sep) {
			prefix += sep
		}
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// warnInvalidExcludePatterns logs a warning for every --exclude pattern that
// fails to parse, so a typoed pattern does not silently disable the
// exclusion it was meant to apply. Patterns are validated once up front,
// rather than on every MatchesExclude call made during the walk, to avoid
// repeating the same warning for every directory visited.
func warnInvalidExcludePatterns(patterns []string) {
	for _, pattern := range patterns {
		if !doublestar.ValidatePattern(filepath.ToSlash(pattern)) {
			slog.Warn("discovery: invalid exclude pattern, it will be ignored", "pattern", pattern)
		}
	}
}

// MatchesExclude checks whether a path matches any of the given exclude
// glob patterns. Patterns that fail to parse are skipped rather than
// treated as a match; call warnInvalidExcludePatterns (or ValidatePattern)
// ahead of time to surface malformed patterns.
func MatchesExclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	slashPath := filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		match, err := doublestar.Match(pattern, slashPath)
		if err != nil {
			continue
		}
		if match {
			return true
		}
	}
	return false
}

func walkRoot(ctx context.Context, root string, opts Options, visited map[string]struct{}, skipDirs map[string]struct{}, results *[]Result) error {
	realRoot := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		realRoot = resolved
	}
	// Track resolved roots so the same tree is not walked twice via different
	// symlink paths, and so following symlinks below cannot recurse forever
	// on a cycle.
	if _, ok := visited[realRoot]; ok {
		return nil
	}
	visited[realRoot] = struct{}{}

	// Walk the root as the caller supplied it so discovered paths keep the
	// caller's path form; a symlinked ancestor (e.g. macOS /var -> /private/var)
	// must not rewrite every returned path. Only when the root's final component
	// is itself a symlink do we walk the resolved target, because WalkDir lstats
	// the root and would otherwise treat a symlinked directory as a
	// non-directory and never descend into it.
	walkTarget := root
	if fi, err := os.Lstat(root); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		walkTarget = realRoot
	}
	return filepath.WalkDir(walkTarget, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
				// Transient per-directory failures (permission denied, or a
				// directory removed mid-scan) should not abort the whole
				// scan; skip just that subtree and keep going.
				slog.Warn("discovery: skipping path after walk error", "path", path, "error", err)
				return fs.SkipDir
			}
			return err
		}

		if d.Type()&os.ModeSymlink != 0 {
			// filepath.WalkDir never follows symlinks on its own: a symlinked
			// directory is reported with Type()==ModeSymlink and
			// IsDir()==false, so it must be detected and, when enabled,
			// resolved and recursed into explicitly here.
			if !opts.FollowSymlinks {
				return nil
			}
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			info, err := os.Stat(target)
			if err != nil || !info.IsDir() {
				return nil
			}
			// If the symlink resolves back inside the tree WalkDir is already
			// walking, descending it as a new root would visit that subtree
			// twice and duplicate results. The visited set only tracks roots,
			// not every directory WalkDir descends into, so guard explicitly.
			if target == realRoot || strings.HasPrefix(target, realRoot+string(filepath.Separator)) {
				return nil
			}
			// Recurse into the symlink target as its own root; the visited
			// set above guards against symlink cycles. Returning nil (not
			// SkipDir) is required here because a symlink is a non-directory
			// entry from WalkDir's point of view, and SkipDir on a
			// non-directory entry skips the remaining siblings too.
			return walkRoot(ctx, target, opts, visited, skipDirs, results)
		}

		if !d.IsDir() {
			return nil
		}

		if _, ok := skipDirs[path]; ok {
			return fs.SkipDir
		}
		if d.Name() == ".git" {
			// Never recurse through git internals during root discovery.
			return fs.SkipDir
		}
		if MatchesExclude(path, opts.Exclude) {
			return fs.SkipDir
		}

		isRepoRoot, bare, gitdir, err := detectRepo(ctx, opts.Adapter, path)
		if err != nil {
			return err
		}
		if isRepoRoot {
			if gitdir != "" {
				// Linked worktrees can share a gitdir outside the repo path; mark it
				// skipped so we do not treat it as an independent repository later.
				skipDirs[gitdir] = struct{}{}
			}
			result, err := buildResult(ctx, opts.Adapter, path, bare)
			if err != nil {
				return err
			}
			*results = append(*results, result)
			return fs.SkipDir
		}

		return nil
	})
}

func detectRepo(ctx context.Context, adapter vcs.Adapter, dir string) (bool, bool, string, error) {
	gitPath := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitPath); err == nil {
		if info.Mode().IsRegular() {
			// A file-based .git indicates a linked worktree with an external gitdir.
			if gitdir, ok := gitdirFromFile(gitPath); ok {
				bare, _ := adapter.IsBare(ctx, dir)
				return true, bare, gitdir, nil
			}
		}
		bare, err := adapter.IsBare(ctx, dir)
		if err != nil {
			return true, false, "", nil
		}
		return true, bare, "", nil
	}

	// Bare repo heuristic: HEAD file and objects dir.
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err == nil {
		if info, err := os.Stat(filepath.Join(dir, "objects")); err == nil && info.IsDir() {
			return true, true, "", nil
		}
	}

	ok, err := adapter.IsRepo(ctx, dir)
	if err != nil {
		return false, false, "", err
	}
	if ok {
		bare, _ := adapter.IsBare(ctx, dir)
		return true, bare, "", nil
	}
	return false, false, "", nil
}

func gitdirFromFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "gitdir:") {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(content, "gitdir:"))
	if raw == "" {
		return "", false
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), true
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), raw)), true
}

func buildResult(ctx context.Context, adapter vcs.Adapter, dir string, bare bool) (Result, error) {
	remotes, err := adapter.Remotes(ctx, dir)
	if err != nil {
		return Result{}, err
	}
	var remoteNames []string
	for _, r := range remotes {
		remoteNames = append(remoteNames, r.Name)
	}
	primary := adapter.PrimaryRemote(remoteNames)
	var remoteURL string
	for _, r := range remotes {
		if r.Name == primary {
			remoteURL = r.URL
			break
		}
	}
	return Result{
		Path:          dir,
		RepoID:        adapter.NormalizeURL(remoteURL),
		RemoteURL:     remoteURL,
		PrimaryRemote: primary,
		Remotes:       remotes,
		Bare:          bare,
	}, nil
}
