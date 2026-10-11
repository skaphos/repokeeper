// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/v2/internal/registry"
)

// hangingFetchAdapter blocks Fetch until its context is done, like a git fetch
// stuck on an unresponsive remote.
type hangingFetchAdapter struct {
	*planAdapter
}

func (a *hangingFetchAdapter) Fetch(ctx context.Context, _ string) error {
	<-ctx.Done()
	return ctx.Err()
}

// The per-repository --timeout must bound plan execution in both executors:
// sequential (--continue-on-error=false) and concurrent (#363).
func TestExecuteSyncPlanHonorsPerRepoTimeout(t *testing.T) {
	for _, continueOnError := range []bool{false, true} {
		name := "sequential"
		if continueOnError {
			name = "concurrent"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			eng := newPlanExecEngine(&hangingFetchAdapter{planAdapter: &planAdapter{}})
			eng.registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "repo", Path: "/repo", Status: registry.StatusPresent}}}
			plan := []SyncResult{{
				RepoID:  "repo",
				Path:    "/repo",
				Outcome: SyncOutcomePlannedFetch,
				OK:      true,
				Planned: true,
				steps:   []syncStep{syncStepFetch},
			}}

			done := make(chan []SyncResult, 1)
			go func() {
				results, err := eng.ExecuteSyncPlanWithCallbacks(context.Background(), plan, SyncOptions{Timeout: 1, ContinueOnError: continueOnError}, nil, nil)
				if err != nil {
					t.Errorf("execute plan: %v", err)
				}
				done <- results
			}()

			select {
			case results := <-done:
				if len(results) != 1 || results[0].OK || results[0].Outcome != SyncOutcomeFailedFetch {
					t.Fatalf("expected a failed fetch after the timeout, got %+v", results)
				}
			case <-time.After(15 * time.Second):
				t.Fatal("plan execution ignored the 1s per-repository timeout")
			}
		})
	}
}
