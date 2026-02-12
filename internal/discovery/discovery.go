// Package discovery walks configured root directories to find git repositories.
package discovery

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
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

	visited := make(map[string]struct{})
	var results []Result
	skipDirs := make(map[string]struct{})

	for _, root := range opts.Roots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if err := walkRoot(ctx, absRoot, opts, visited, skipDirs, &results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// MatchesExclude checks whether a path matches any of the given exclude
// glob patterns.
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
	if _, ok := visited[realRoot]; ok {
		return nil
	}
	visited[realRoot] = struct{}{}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSymlink != 0 && d.IsDir() && !opts.FollowSymlinks {
			return fs.SkipDir
		}

		if d.IsDir() {
			if _, ok := skipDirs[path]; ok {
				return fs.SkipDir
			}
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			if MatchesExclude(path, opts.Exclude) {
				return fs.SkipDir
			}
		} else {
			return nil
		}

		isRepoRoot, bare, gitdir, err := detectRepo(ctx, opts.Adapter, path)
		if err != nil {
			return err
		}
		if isRepoRoot {
			if gitdir != "" {
				skipDirs[gitdir] = struct{}{}
			}
			result, err := buildResult(ctx, opts.Adapter, path, bare)
			if err != nil {
				return err
			}
			*results = append(*results, result)
			return fs.SkipDir
		}

		if d.Type()&os.ModeSymlink != 0 && d.IsDir() && opts.FollowSymlinks {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			info, err := os.Stat(target)
			if err != nil || !info.IsDir() {
				return nil
			}
			if err := walkRoot(ctx, target, opts, visited, skipDirs, results); err != nil {
				return err
			}
			return fs.SkipDir
		}

		return nil
	})
}

func detectRepo(ctx context.Context, adapter vcs.Adapter, dir string) (bool, bool, string, error) {
	gitPath := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitPath); err == nil {
		if info.Mode().IsRegular() {
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
