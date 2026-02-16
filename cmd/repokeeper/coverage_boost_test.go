package repokeeper

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestWriteImportCloneFailureSummary(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	failures := []engine.SyncResult{
		{RepoID: "github.com/org/repo-a", Path: "/tmp/repo-a", ErrorClass: "unknown", Error: "exit status 128"},
	}
	if err := writeImportCloneFailureSummary(cmd, failures, "/tmp"); err != nil {
		t.Fatalf("writeImportCloneFailureSummary returned error: %v", err)
	}
	out := errOut.String()
	if !strings.Contains(out, "Failed import clone operations:") || !strings.Contains(out, "ERROR_CLASS") {
		t.Fatalf("unexpected import failure summary output: %q", out)
	}
}

func TestWriteImportCloneFailureSummaryNoFailures(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)
	if err := writeImportCloneFailureSummary(cmd, nil, "/tmp"); err != nil {
		t.Fatalf("writeImportCloneFailureSummary returned error: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no output for empty failures, got: %q", errOut.String())
	}
}

func TestRemoveRegistryEntryByRepoID(t *testing.T) {
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "r1", Path: "/r1"},
			{RepoID: "r2", Path: "/r2"},
		},
	}
	removeRegistryEntryByRepoID(reg, "r1")
	if len(reg.Entries) != 1 || reg.Entries[0].RepoID != "r2" {
		t.Fatalf("unexpected registry after remove: %+v", reg.Entries)
	}

	// Nil registry should be a no-op.
	removeRegistryEntryByRepoID(nil, "r2")
}

func TestWriteRemoteMismatchPlan(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)
	plans := []remoteMismatchPlan{
		{
			RepoID:        "github.com/org/repo-a",
			Path:          "/tmp/repo-a",
			Action:        "update registry remote_url",
			PrimaryRemote: "origin",
			RepoRemoteURL: "git@github.com:org/repo-a.git",
			RegistryURL:   "git@github.com:other/repo-a.git",
		},
	}

	if err := writeRemoteMismatchPlan(cmd, plans, "/tmp", nil, true); err != nil {
		t.Fatalf("writeRemoteMismatchPlan (dry-run) returned error: %v", err)
	}
	if !strings.Contains(errOut.String(), "Remote mismatch reconcile (planned):") {
		t.Fatalf("unexpected dry-run plan output: %q", errOut.String())
	}

	errOut.Reset()
	if err := writeRemoteMismatchPlan(cmd, plans, "/tmp", nil, false); err != nil {
		t.Fatalf("writeRemoteMismatchPlan (apply) returned error: %v", err)
	}
	if !strings.Contains(errOut.String(), "Remote mismatch reconcile (applying):") {
		t.Fatalf("unexpected apply plan output: %q", errOut.String())
	}
}

func TestSyncProgressWriterPaths(t *testing.T) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	writer := newSyncProgressWriter(cmd, "/tmp", nil)
	res := engine.SyncResult{RepoID: "r1", Path: "/tmp/repo-a", Action: "git fetch --all --prune --prune-tags --no-recurse-submodules"}
	if err := writer.StartResult(res); err != nil {
		t.Fatalf("StartResult returned error: %v", err)
	}
	if err := writer.WriteResult(engine.SyncResult{RepoID: "r1", Path: "/tmp/repo-a", Action: res.Action, OK: true}); err != nil {
		t.Fatalf("WriteResult returned error: %v", err)
	}
	if !strings.Contains(out.String(), "repo-a .") || !strings.Contains(out.String(), "updated!") {
		t.Fatalf("unexpected progress output: %q", out.String())
	}

	out.Reset()
	s := &syncProgressState{displayPath: "repo-a", lastLen: len("repo-a ........")}
	writer.supportsInPlace = true
	if err := writer.writeProgressLine(s, ".", false); err != nil {
		t.Fatalf("writeProgressLine (in-place no newline) returned error: %v", err)
	}
	if err := writer.writeProgressLine(s, "updated!", true); err != nil {
		t.Fatalf("writeProgressLine (in-place newline) returned error: %v", err)
	}
	if !strings.Contains(out.String(), "\rrepo-a .") || !strings.Contains(out.String(), "\rrepo-a updated!") {
		t.Fatalf("expected carriage-return in-place output, got: %q", out.String())
	}
}
