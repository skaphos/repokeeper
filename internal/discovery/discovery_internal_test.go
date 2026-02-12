package discovery

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mfacenet/repokeeper/internal/model"
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
