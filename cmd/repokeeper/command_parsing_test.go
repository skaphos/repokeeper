// SPDX-License-Identifier: MIT
package repokeeper

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/spf13/cobra"
)

func TestParseRemoteMismatchReconcileModeTable(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    remoteMismatchReconcileMode
		wantErr bool
	}{
		{name: "empty defaults none", in: "", want: remoteMismatchReconcileNone},
		{name: "explicit none", in: "none", want: remoteMismatchReconcileNone},
		{name: "registry", in: "registry", want: remoteMismatchReconcileRegistry},
		{name: "git", in: "git", want: remoteMismatchReconcileGit},
		{name: "case-insensitive", in: "GIT", want: remoteMismatchReconcileGit},
		{name: "invalid", in: "oops", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRemoteMismatchReconcileMode(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("parseRemoteMismatchReconcileMode(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSyncResultNeedsConfirmationTable(t *testing.T) {
	tests := []struct {
		name string
		res  engine.SyncResult
		want bool
	}{
		{
			name: "fetch only",
			res:  engine.SyncResult{Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"},
			want: false,
		},
		{
			name: "pull rebase",
			res:  engine.SyncResult{Action: "git pull --rebase --no-recurse-submodules"},
			want: true,
		},
		{
			name: "stash and rebase",
			res:  engine.SyncResult{Action: "git stash push -u -m \"repokeeper: pre-rebase stash\" && git pull --rebase --no-recurse-submodules && git stash pop"},
			want: true,
		},
		{
			name: "checkout missing clone",
			res:  engine.SyncResult{Action: "git clone --branch main --single-branch git@github.com:org/repo.git /tmp/repo"},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := syncResultNeedsConfirmation(tc.res)
			if got != tc.want {
				t.Fatalf("syncResultNeedsConfirmation(%q) = %v, want %v", tc.res.Action, got, tc.want)
			}
		})
	}
}

func TestRepairUpstreamMatchesFilterTable(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		filter  string
		want    bool
	}{
		{name: "all default", current: "origin/main", target: "origin/main", filter: "", want: true},
		{name: "all explicit", current: "origin/main", target: "origin/main", filter: "all", want: true},
		{name: "missing matches empty current", current: "", target: "origin/main", filter: "missing", want: true},
		{name: "missing rejects set current", current: "origin/main", target: "origin/main", filter: "missing", want: false},
		{name: "mismatch matches different", current: "origin/main", target: "upstream/main", filter: "mismatch", want: true},
		{name: "mismatch rejects equal", current: "origin/main", target: "origin/main", filter: "mismatch", want: false},
		{name: "unknown filter defaults true", current: "origin/main", target: "origin/main", filter: "unknown", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := repairUpstreamMatchesFilter(tc.current, tc.target, tc.filter)
			if got != tc.want {
				t.Fatalf("repairUpstreamMatchesFilter(%q, %q, %q) = %v, want %v", tc.current, tc.target, tc.filter, got, tc.want)
			}
		})
	}
}

func TestShouldStreamSyncResults(t *testing.T) {
	reconcile := &cobra.Command{Use: "reconcile"}
	repos := &cobra.Command{Use: "repos"}
	reconcile.AddCommand(repos)

	if !shouldStreamSyncResults(repos, false, outputKindTable) {
		t.Fatal("expected reconcile repos table output to stream")
	}
	if !shouldStreamSyncResults(repos, false, outputKindWide) {
		t.Fatal("expected reconcile repos wide output to stream")
	}
	if shouldStreamSyncResults(repos, true, outputKindTable) {
		t.Fatal("did not expect dry-run to stream")
	}
	if shouldStreamSyncResults(repos, false, outputKindJSON) {
		t.Fatal("did not expect json output to stream")
	}

	syncOnly := &cobra.Command{Use: "sync"}
	if shouldStreamSyncResults(syncOnly, false, outputKindTable) {
		t.Fatal("did not expect sync command table output to stream by default")
	}
}

func TestSyncProgressMessageKinds(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	runtimeStateFor(cmd).colorOutputEnabled = false

	if got := syncProgressMessage(cmd, engine.SyncResult{
		Outcome: engine.SyncOutcomeFetched,
		OK:      true,
		Action:  "git fetch --all --prune --prune-tags --no-recurse-submodules",
	}); got != "updated! (fetch)" {
		t.Fatalf("unexpected updated progress message: %q", got)
	}

	if got := syncProgressMessage(cmd, engine.SyncResult{
		OK:    true,
		Error: engine.SyncErrorSkippedNoUpstream,
	}); got != "skip no upstream" {
		t.Fatalf("unexpected skipped progress message: %q", got)
	}

	if got := syncProgressMessage(cmd, engine.SyncResult{
		OK:         false,
		ErrorClass: "network",
		Error:      engine.SyncErrorFetchNetwork,
	}); got != "failed (network)" {
		t.Fatalf("unexpected failed progress message: %q", got)
	}
}
