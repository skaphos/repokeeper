package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
)

type benchAdapter struct{}

func (b *benchAdapter) Name() string { return "bench" }
func (b *benchAdapter) IsRepo(context.Context, string) (bool, error) {
	return true, nil
}
func (b *benchAdapter) IsBare(context.Context, string) (bool, error) { return false, nil }
func (b *benchAdapter) Remotes(context.Context, string) ([]model.Remote, error) {
	return []model.Remote{{Name: "origin", URL: "git@github.com:org/repo.git"}}, nil
}
func (b *benchAdapter) Head(context.Context, string) (model.Head, error) {
	return model.Head{Branch: "main"}, nil
}
func (b *benchAdapter) WorktreeStatus(context.Context, string) (*model.Worktree, error) {
	return &model.Worktree{Dirty: false}, nil
}
func (b *benchAdapter) TrackingStatus(context.Context, string) (model.Tracking, error) {
	return model.Tracking{Status: model.TrackingEqual, Upstream: "origin/main"}, nil
}
func (b *benchAdapter) HasSubmodules(context.Context, string) (bool, error) { return false, nil }
func (b *benchAdapter) Fetch(context.Context, string) error                 { return nil }
func (b *benchAdapter) PullRebase(context.Context, string) error            { return nil }
func (b *benchAdapter) Push(context.Context, string) error                  { return nil }
func (b *benchAdapter) SetUpstream(context.Context, string, string, string) error {
	return nil
}
func (b *benchAdapter) SetRemoteURL(context.Context, string, string, string) error { return nil }
func (b *benchAdapter) StashPush(context.Context, string, string) (bool, error) {
	return false, nil
}
func (b *benchAdapter) StashPop(context.Context, string) error                    { return nil }
func (b *benchAdapter) Clone(context.Context, string, string, string, bool) error { return nil }
func (b *benchAdapter) NormalizeURL(rawURL string) string                         { return rawURL }
func (b *benchAdapter) PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) == 0 {
		return ""
	}
	return remoteNames[0]
}

func benchmarkEngineWithRepos(repoCount int) *Engine {
	entries := make([]registry.Entry, 0, repoCount)
	for i := 0; i < repoCount; i++ {
		entries = append(entries, registry.Entry{
			RepoID:    fmt.Sprintf("repo-%d", i),
			Path:      fmt.Sprintf("/repos/repo-%d", i),
			RemoteURL: "git@github.com:org/repo.git",
			Status:    registry.StatusPresent,
		})
	}
	cfg := &config.Config{Defaults: config.Defaults{Concurrency: 8, TimeoutSeconds: 30}}
	reg := &registry.Registry{Entries: entries}
	return New(cfg, reg, &benchAdapter{})
}

func BenchmarkSyncDryRunPlan(b *testing.B) {
	eng := benchmarkEngineWithRepos(100)
	ctx := context.Background()
	opts := SyncOptions{DryRun: true, Concurrency: 8, Timeout: 30}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := eng.Sync(ctx, opts)
		if err != nil {
			b.Fatalf("sync failed: %v", err)
		}
		if len(results) != 100 {
			b.Fatalf("unexpected result count: got=%d want=100", len(results))
		}
	}
}

func BenchmarkStatusReport(b *testing.B) {
	eng := benchmarkEngineWithRepos(100)
	ctx := context.Background()
	opts := StatusOptions{Filter: FilterAll, Concurrency: 8, Timeout: 30}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := eng.Status(ctx, opts)
		if err != nil {
			b.Fatalf("status failed: %v", err)
		}
		if len(report.Repos) != 100 {
			b.Fatalf("unexpected repo count: got=%d want=100", len(report.Repos))
		}
	}
}
