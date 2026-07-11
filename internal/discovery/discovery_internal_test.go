// SPDX-License-Identifier: MIT
package discovery

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

type stubAdapter struct {
	isRepoFn        func(context.Context, string) (bool, error)
	isBareFn        func(context.Context, string) (bool, error)
	remotesFn       func(context.Context, string) ([]model.Remote, error)
	normalizeURLFn  func(string) string
	primaryRemoteFn func([]string) string
}

func (s *stubAdapter) Name() string { return "stub" }
func (s *stubAdapter) IsRepo(ctx context.Context, dir string) (bool, error) {
	if s.isRepoFn == nil {
		return false, nil
	}
	return s.isRepoFn(ctx, dir)
}
func (s *stubAdapter) IsBare(ctx context.Context, dir string) (bool, error) {
	if s.isBareFn == nil {
		return false, nil
	}
	return s.isBareFn(ctx, dir)
}
func (s *stubAdapter) Remotes(ctx context.Context, dir string) ([]model.Remote, error) {
	if s.remotesFn == nil {
		return nil, nil
	}
	return s.remotesFn(ctx, dir)
}
func (s *stubAdapter) Head(context.Context, string) (model.Head, error) {
	return model.Head{}, nil
}
func (s *stubAdapter) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return nil, nil
}
func (s *stubAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{}, nil
}
func (s *stubAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (s *stubAdapter) Fetch(context.Context, string) error                 { return nil }
func (s *stubAdapter) PullRebase(context.Context, string) error            { return nil }
func (s *stubAdapter) Push(context.Context, string) error                  { return nil }
func (s *stubAdapter) SetUpstream(context.Context, string, string, string) error {
	return nil
}
func (s *stubAdapter) SetRemoteURL(context.Context, string, string, string) error { return nil }
func (s *stubAdapter) StashPush(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *stubAdapter) StashPop(context.Context, string) error                    { return nil }
func (s *stubAdapter) ResetHard(context.Context, string) error                   { return nil }
func (s *stubAdapter) CleanFD(context.Context, string) error                     { return nil }
func (s *stubAdapter) Clone(context.Context, string, string, string, bool) error { return nil }
func (s *stubAdapter) NormalizeURL(rawURL string) string {
	if s.normalizeURLFn == nil {
		return rawURL
	}
	return s.normalizeURLFn(rawURL)
}
func (s *stubAdapter) PrimaryRemote(remoteNames []string) string {
	if s.primaryRemoteFn == nil {
		return ""
	}
	return s.primaryRemoteFn(remoteNames)
}

func TestGitdirFromFile(t *testing.T) {
	tmp := t.TempDir()
	if _, ok := gitdirFromFile(filepath.Join(tmp, "missing")); ok {
		t.Fatal("expected missing file to return false")
	}

	invalid := filepath.Join(tmp, ".git.invalid")
	if err := os.WriteFile(invalid, []byte("not-gitdir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := gitdirFromFile(invalid); ok {
		t.Fatal("expected invalid content to return false")
	}

	empty := filepath.Join(tmp, ".git.empty")
	if err := os.WriteFile(empty, []byte("gitdir:   "), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := gitdirFromFile(empty); ok {
		t.Fatal("expected empty gitdir to return false")
	}

	relative := filepath.Join(tmp, ".git.rel")
	if err := os.WriteFile(relative, []byte("gitdir: ../actual.git"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := gitdirFromFile(relative)
	if !ok {
		t.Fatal("expected relative gitdir to parse")
	}
	want := filepath.Clean(filepath.Join(filepath.Dir(relative), "../actual.git"))
	if got != want {
		t.Fatalf("unexpected relative gitdir: got %q want %q", got, want)
	}
}

func TestDetectRepoBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("bare-heuristic-head-and-objects", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "objects"), 0o755); err != nil {
			t.Fatal(err)
		}
		ok, bare, gitdir, err := detectRepo(ctx, &stubAdapter{}, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || !bare || gitdir != "" {
			t.Fatalf("unexpected detect result: ok=%v bare=%v gitdir=%q", ok, bare, gitdir)
		}
	})

	t.Run("dotgit-dir-isbare-error-still-repo", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		ok, bare, gitdir, err := detectRepo(ctx, &stubAdapter{
			isBareFn: func(context.Context, string) (bool, error) {
				return false, errors.New("isbare failed")
			},
		}, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || bare || gitdir != "" {
			t.Fatalf("unexpected detect result: ok=%v bare=%v gitdir=%q", ok, bare, gitdir)
		}
	})

	t.Run("dotgit-dir-bare-success", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		ok, bare, gitdir, err := detectRepo(ctx, &stubAdapter{
			isBareFn: func(context.Context, string) (bool, error) {
				return true, nil
			},
		}, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || !bare || gitdir != "" {
			t.Fatalf("unexpected detect result: ok=%v bare=%v gitdir=%q", ok, bare, gitdir)
		}
	})

	t.Run("dotgit-file-linked-worktree", func(t *testing.T) {
		dir := t.TempDir()
		linked := filepath.Join(dir, "linked.git")
		if err := os.Mkdir(linked, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: "+linked), 0o644); err != nil {
			t.Fatal(err)
		}
		ok, bare, gitdir, err := detectRepo(ctx, &stubAdapter{
			isBareFn: func(context.Context, string) (bool, error) { return false, nil },
		}, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || bare || gitdir != linked {
			t.Fatalf("unexpected detect result: ok=%v bare=%v gitdir=%q", ok, bare, gitdir)
		}
	})

	t.Run("adapter-isrepo-error", func(t *testing.T) {
		dir := t.TempDir()
		_, _, _, err := detectRepo(ctx, &stubAdapter{
			isRepoFn: func(context.Context, string) (bool, error) { return false, errors.New("boom") },
		}, dir)
		if err == nil {
			t.Fatal("expected adapter error")
		}
	})
}

func TestBuildResultBranches(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	_, err := buildResult(ctx, &stubAdapter{
		remotesFn: func(context.Context, string) ([]model.Remote, error) {
			return nil, errors.New("remote fail")
		},
	}, dir, false)
	if err == nil {
		t.Fatal("expected remote error")
	}

	res, err := buildResult(ctx, &stubAdapter{
		remotesFn: func(context.Context, string) ([]model.Remote, error) {
			return []model.Remote{
				{Name: "origin", URL: "git@github.com:Org/Repo.git"},
				{Name: "upstream", URL: "git@github.com:Up/Repo.git"},
			}, nil
		},
		primaryRemoteFn: func(names []string) string { return "origin" },
		normalizeURLFn:  func(raw string) string { return "norm:" + raw },
	}, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.PrimaryRemote != "origin" || res.RemoteURL != "git@github.com:Org/Repo.git" || res.RepoID != "norm:git@github.com:Org/Repo.git" || !res.Bare {
		t.Fatalf("unexpected result: %+v", res)
	}

	res, err = buildResult(ctx, &stubAdapter{
		remotesFn: func(context.Context, string) ([]model.Remote, error) {
			return []model.Remote{{Name: "upstream", URL: "git@github.com:Up/Repo.git"}}, nil
		},
		primaryRemoteFn: func([]string) string { return "origin" },
		normalizeURLFn:  func(raw string) string { return "norm:" + raw },
	}, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.RemoteURL != "" || res.RepoID != "norm:" {
		t.Fatalf("unexpected missing-primary behavior: %+v", res)
	}
}

func TestScanDefaultsAndEmptyRoots(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	cmd := exec.Command("git", "init", repo)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v %s", err, string(out))
	}

	results, err := Scan(context.Background(), Options{
		Roots: []string{"", root},
		// Adapter intentionally nil to cover default adapter path.
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != repo {
		t.Fatalf("unexpected scan results: %+v", results)
	}
}

func TestRootCovered(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		accepted []string
		want     bool
	}{
		{"exact-match", "/A", []string{"/A"}, true},
		{"nested-under-parent", filepath.Join("/A", "sub"), []string{"/A"}, true},
		{"deeply-nested-under-parent", filepath.Join("/A", "sub", "deeper"), []string{"/A"}, true},
		{"sibling-with-shared-prefix-not-nested", "/AB", []string{"/A"}, false},
		{"unrelated-path-not-covered", "/B", []string{"/A"}, false},
		{"no-accepted-roots", "/A", nil, false},
		{"covered-by-second-of-several", "/C/sub", []string{"/A", "/C"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rootCovered(tc.path, tc.accepted); got != tc.want {
				t.Fatalf("rootCovered(%q, %v) = %v, want %v", tc.path, tc.accepted, got, tc.want)
			}
		})
	}
}

func TestWarnInvalidExcludePatterns(t *testing.T) {
	cases := []struct {
		name           string
		patterns       []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "invalid pattern warns with the offending pattern",
			patterns:     []string{"[invalid"},
			wantContains: []string{"invalid exclude pattern", "[invalid"},
		},
		{
			name:           "valid patterns do not warn",
			patterns:       []string{"**/vendor/**", "*.go"},
			wantNotContain: []string{"invalid exclude pattern"},
		},
		{
			name:           "no patterns do not warn",
			patterns:       nil,
			wantNotContain: []string{"invalid exclude pattern"},
		},
		{
			name:           "mixed patterns warn only for the invalid one",
			patterns:       []string{"**/vendor/**", "[bad"},
			wantContains:   []string{"[bad"},
			wantNotContain: []string{"vendor"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			warnInvalidExcludePatterns(tc.patterns)

			out := buf.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(out, want) {
					t.Fatalf("expected warning output to contain %q, got: %q", want, out)
				}
			}
			for _, notWant := range tc.wantNotContain {
				if strings.Contains(out, notWant) {
					t.Fatalf("expected warning output to NOT contain %q, got: %q", notWant, out)
				}
			}
		})
	}
}

// TestScanToleratesDirectoryRemovedMidScan covers the fix for a WalkDir
// error handler that previously aborted the entire Scan on any error other
// than fs.ErrPermission. A directory that disappears mid-walk (e.g. a build
// tool cleaning a temp dir concurrently) must not stop discovery of the
// remaining roots.
func TestScanToleratesDirectoryRemovedMidScan(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	keepDir := filepath.Join(root, "a-keep")      // processed before removeDir (lexical order)
	removeDir := filepath.Join(root, "b-removed") // removed while a-keep is processed
	afterRepo := filepath.Join(root, "c-after", "repo")

	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(removeDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(afterRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	adapter := &stubAdapter{
		isRepoFn: func(_ context.Context, dir string) (bool, error) {
			if dir == keepDir {
				if err := os.RemoveAll(removeDir); err != nil {
					t.Fatal(err)
				}
			}
			return false, nil
		},
	}

	results, err := Scan(ctx, Options{
		Roots:   []string{root},
		Adapter: adapter,
	})
	if err != nil {
		t.Fatalf("expected Scan to tolerate a directory removed mid-scan, got error: %v", err)
	}
	if len(results) != 1 || filepath.Clean(results[0].Path) != filepath.Clean(afterRepo) {
		t.Fatalf("unexpected scan results: %+v", results)
	}
}

func TestMatchesExcludeWithInvalidPattern(t *testing.T) {
	// Test that MatchesExclude gracefully handles invalid glob patterns
	// by continuing to the next pattern instead of failing.
	// An unclosed bracket is an invalid pattern that causes doublestar.Match to error.
	path := "/some/path/to/file"
	patterns := []string{"[invalid", "*.go"}

	// Should not panic or error; should return false since no valid pattern matches
	result := MatchesExclude(path, patterns)
	if result {
		t.Fatalf("expected MatchesExclude to return false for non-matching patterns")
	}

	// Test with a pattern that matches after an invalid one
	patterns = []string{"[invalid", "**/path/**"}
	result = MatchesExclude(path, patterns)
	if !result {
		t.Fatalf("expected MatchesExclude to return true when a valid pattern matches")
	}
}
