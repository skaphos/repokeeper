// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

type invalidJSONMarshaler struct{}

func (invalidJSONMarshaler) MarshalJSON() ([]byte, error) {
	return []byte("{"), nil
}

type alwaysErrWriter struct{}

func (alwaysErrWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

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

func TestCommonPathRoot(t *testing.T) {
	t.Parallel()

	sep := string(os.PathSeparator)
	tests := []struct {
		name  string
		left  string
		right string
		want  string
	}{
		{name: "empty left", left: "", right: sep + "a", want: ""},
		{name: "empty right", left: sep + "a", right: "", want: ""},
		{name: "identical", left: sep + "a" + sep + "b", right: sep + "a" + sep + "b", want: sep + "a" + sep + "b"},
		{name: "common prefix", left: sep + "a" + sep + "b" + sep + "c", right: sep + "a" + sep + "b" + sep + "d", want: sep + "a" + sep + "b"},
		{name: "no common prefix", left: sep + "a", right: sep + "b", want: sep},
		{name: "single component identical", left: sep + "a", right: sep + "a", want: sep + "a"},
		{name: "single component different", left: "a", right: "b", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := commonPathRoot(tc.left, tc.right); got != tc.want {
				t.Fatalf("commonPathRoot(%q, %q) = %q, want %q", tc.left, tc.right, got, tc.want)
			}
		})
	}
}

func TestLogOutputWriteFailureNilError(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	logOutputWriteFailure(cmd, "sync table", nil)

	if errOut.Len() != 0 {
		t.Fatalf("expected no output for nil error, got %q", errOut.String())
	}
}

func TestRowsForCustomColumnsFallbackPaths(t *testing.T) {
	t.Parallel()

	t.Run("map without known list keys", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{"repo_id": "r1"}
		got := rowsForCustomColumns(row)
		if len(got) != 1 {
			t.Fatalf("rows len = %d, want 1", len(got))
		}
		if !reflect.DeepEqual(got[0], row) {
			t.Fatalf("rows[0] = %#v, want original row %#v", got[0], row)
		}
	})

	t.Run("map with repos key", func(t *testing.T) {
		t.Parallel()

		rows := []any{map[string]any{"repo_id": "r1"}}
		got := rowsForCustomColumns(map[string]any{"repos": rows})
		if !reflect.DeepEqual(got, rows) {
			t.Fatalf("rows = %#v, want %#v", got, rows)
		}
	})

	t.Run("map with results key", func(t *testing.T) {
		t.Parallel()

		rows := []any{map[string]any{"repo_id": "r2"}}
		got := rowsForCustomColumns(map[string]any{"results": rows})
		if !reflect.DeepEqual(got, rows) {
			t.Fatalf("rows = %#v, want %#v", got, rows)
		}
	})

	t.Run("map with items key", func(t *testing.T) {
		t.Parallel()

		rows := []any{map[string]any{"repo_id": "r3"}}
		got := rowsForCustomColumns(map[string]any{"items": rows})
		if !reflect.DeepEqual(got, rows) {
			t.Fatalf("rows = %#v, want %#v", got, rows)
		}
	})

	t.Run("non map non slice", func(t *testing.T) {
		t.Parallel()

		got := rowsForCustomColumns("raw")
		if len(got) != 1 || got[0] != "raw" {
			t.Fatalf("unexpected rows: %#v", got)
		}
	})

	t.Run("path with empty intermediate segment", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"tracking": map[string]any{"status": "equal"}}, ".tracking..status")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "equal" {
			t.Fatalf("value = %q, want equal", got)
		}
	})
}

func TestResolveCustomColumnValueEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"x": "y"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("value = %q, want empty", got)
		}
	})

	t.Run("dot path", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"x": "y"}, ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("value = %q, want empty", got)
		}
	})

	t.Run("non map intermediate", func(t *testing.T) {
		t.Parallel()

		_, err := resolveCustomColumnValue(map[string]any{"tracking": "equal"}, ".tracking.status")
		if err == nil {
			t.Fatal("expected non-map intermediate error")
		}
		if !strings.Contains(err.Error(), "not a map") {
			t.Fatalf("expected map error, got %v", err)
		}
	})

	t.Run("final bool true", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"worktree": map[string]any{"dirty": true}}, ".worktree.dirty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "true" {
			t.Fatalf("value = %q, want true", got)
		}
	})

	t.Run("final bool false", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"worktree": map[string]any{"dirty": false}}, ".worktree.dirty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "false" {
			t.Fatalf("value = %q, want false", got)
		}
	})

	t.Run("final nil", func(t *testing.T) {
		t.Parallel()

		got, err := resolveCustomColumnValue(map[string]any{"meta": map[string]any{"value": nil}}, ".meta.value")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("value = %q, want empty", got)
		}
	})
}

func TestMarshalToGenericUnmarshalErrorPath(t *testing.T) {
	t.Parallel()

	_, err := marshalToGeneric(invalidJSONMarshaler{})
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestMarshalToGenericMarshalErrorPath(t *testing.T) {
	t.Parallel()

	_, err := marshalToGeneric(make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestShouldStreamSyncResultsBranches(t *testing.T) {
	t.Parallel()

	reconcile := &cobra.Command{Use: "reconcile"}
	repos := &cobra.Command{Use: "repos"}
	reconcile.AddCommand(repos)
	other := &cobra.Command{Use: "status"}

	tests := []struct {
		name   string
		cmd    *cobra.Command
		dryRun bool
		kind   outputKind
		want   bool
	}{
		{name: "dry run always false", cmd: reconcile, dryRun: true, kind: outputKindTable, want: false},
		{name: "json never streams", cmd: reconcile, dryRun: false, kind: outputKindJSON, want: false},
		{name: "nil command", cmd: nil, dryRun: false, kind: outputKindTable, want: false},
		{name: "reconcile table streams", cmd: reconcile, dryRun: false, kind: outputKindTable, want: true},
		{name: "reconcile wide streams", cmd: reconcile, dryRun: false, kind: outputKindWide, want: true},
		{name: "reconcile repos subcommand streams", cmd: repos, dryRun: false, kind: outputKindTable, want: true},
		{name: "other command does not stream", cmd: other, dryRun: false, kind: outputKindTable, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldStreamSyncResults(tc.cmd, tc.dryRun, tc.kind); got != tc.want {
				t.Fatalf("shouldStreamSyncResults() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewSyncProgressWriter(t *testing.T) {
	t.Parallel()

	t.Run("non file output disables in place", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})

		writer := newSyncProgressWriter(cmd, "/tmp", []string{"/tmp"})
		if writer == nil {
			t.Fatal("expected writer")
		}
		if writer.supportsInPlace {
			t.Fatal("expected non-file output to disable in-place updates")
		}
		if writer.cmd != cmd {
			t.Fatal("expected command to be stored on writer")
		}
		if writer.cwd != "/tmp" {
			t.Fatalf("cwd = %q, want /tmp", writer.cwd)
		}
		if !reflect.DeepEqual(writer.roots, []string{"/tmp"}) {
			t.Fatalf("roots = %#v, want [/tmp]", writer.roots)
		}
		if writer.running == nil {
			t.Fatal("expected running map to be initialized")
		}
	})

	t.Run("terminal file output enables in place", func(t *testing.T) {
		t.Parallel()
		commandTestStateMu.Lock()
		defer commandTestStateMu.Unlock()

		prevIsTerminalFD := isTerminalFD
		defer func() { isTerminalFD = prevIsTerminalFD }()
		isTerminalFD = func(_ int) bool { return true }

		tmp, err := os.CreateTemp("", "repokeeper-sync-progress-*")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer func() {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}()

		cmd := &cobra.Command{}
		cmd.SetOut(tmp)

		writer := newSyncProgressWriter(cmd, "/tmp", nil)
		if writer == nil {
			t.Fatal("expected writer")
		}
		if !writer.supportsInPlace {
			t.Fatal("expected terminal file output to enable in-place updates")
		}
	})
}

func TestLogOutputWriteFailureLogsError(t *testing.T) {
	t.Parallel()
	commandTestStateMu.Lock()
	defer commandTestStateMu.Unlock()

	prevQuiet, _ := rootCmd.PersistentFlags().GetBool("quiet")
	defer func() { _ = rootCmd.PersistentFlags().Set("quiet", boolToFlag(prevQuiet)) }()
	_ = rootCmd.PersistentFlags().Set("quiet", "false")

	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	logOutputWriteFailure(cmd, "sync table", errors.New("boom"))

	if !strings.Contains(errOut.String(), "ignored output write failure (sync table): boom") {
		t.Fatalf("expected info log for output failure, got %q", errOut.String())
	}
}

func TestRootRunEHelpFallbackForNonTerminalOutput(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "repokeeper"}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := rootRunE(cmd, nil); err != nil {
		t.Fatalf("rootRunE returned error: %v", err)
	}
}

func TestFlagGettersBranchCoverage(t *testing.T) {
	t.Parallel()
	commandTestStateMu.Lock()
	defer commandTestStateMu.Unlock()

	child := &cobra.Command{Use: "child"}
	child.Flags().Bool("quiet", true, "")
	child.Flags().Count("verbose", "")
	child.Flags().String("config", " /tmp/config.yaml ", "")
	if got := getBoolFlag(child, "quiet"); !got {
		t.Fatal("expected command-local bool flag")
	}
	if got := getCountFlag(child, "verbose"); got != 0 {
		t.Fatalf("expected default count 0, got %d", got)
	}
	if got := configOverride(child); got != "/tmp/config.yaml" {
		t.Fatalf("expected trimmed config override, got %q", got)
	}

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("yes", true, "")
	root.PersistentFlags().Bool("no-color", true, "")
	root.PersistentFlags().Count("verbose", "")
	root.PersistentFlags().String("config", " /root-config.yaml ", "")
	root.AddCommand(child)
	if got := assumeYes(child); !got {
		t.Fatal("expected root persistent bool lookup")
	}
	if got := isNoColor(child); !got {
		t.Fatal("expected root persistent no-color lookup")
	}
	if got := getStringFlag(child, "config"); got != " /tmp/config.yaml " {
		t.Fatalf("expected command-local string flag to win, got %q", got)
	}

	if got := getBoolFlag(nil, "quiet"); got {
		t.Fatal("expected nil command lookup to use root persistent quiet default false")
	}
}

func TestRootRunEInteractiveMissingRegistry(t *testing.T) {
	commandTestStateMu.Lock()
	defer commandTestStateMu.Unlock()

	prevIsTerminalFD := isTerminalFD
	prevConfig, _ := rootCmd.PersistentFlags().GetString("config")
	defer func() {
		isTerminalFD = prevIsTerminalFD
		_ = rootCmd.PersistentFlags().Set("config", prevConfig)
	}()

	isTerminalFD = func(_ int) bool { return true }

	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/.repokeeper.yaml"
	cfg := config.DefaultConfig()
	cfg.Registry = nil
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := rootCmd.PersistentFlags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config: %v", err)
	}

	outFile, err := os.CreateTemp("", "repokeeper-rootrun-*")
	if err != nil {
		t.Fatalf("create temp output file: %v", err)
	}
	defer func() {
		_ = outFile.Close()
		_ = os.Remove(outFile.Name())
	}()

	cmd := &cobra.Command{}
	cmd.SetOut(outFile)

	err = rootRunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "registry not found") {
		t.Fatalf("expected missing registry error, got %v", err)
	}
}

func TestSyncProgressWriterAdditionalBranches(t *testing.T) {
	t.Parallel()

	t.Run("start result duplicate path is noop", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		cmd.SetOut(&bytes.Buffer{})
		writer := &syncProgressWriter{cmd: cmd, running: make(map[string]*syncProgressState)}
		res := engine.SyncResult{Path: "/tmp/repo-a"}

		if err := writer.StartResult(res); err != nil {
			t.Fatalf("first StartResult error: %v", err)
		}
		if err := writer.StartResult(res); err != nil {
			t.Fatalf("second StartResult error: %v", err)
		}
		if len(writer.running) != 1 {
			t.Fatalf("expected one running state, got %d", len(writer.running))
		}
	})

	t.Run("start result write failure cleans state", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{}
		cmd.SetOut(alwaysErrWriter{})
		writer := &syncProgressWriter{cmd: cmd, running: make(map[string]*syncProgressState)}
		res := engine.SyncResult{Path: "/tmp/repo-a"}

		err := writer.StartResult(res)
		if err == nil {
			t.Fatal("expected StartResult to fail when output writer fails")
		}
		if len(writer.running) != 0 {
			t.Fatalf("expected running state cleanup on write error, got %d", len(writer.running))
		}
	})

	t.Run("run dots returns when stop closes", func(t *testing.T) {
		t.Parallel()

		writer := &syncProgressWriter{running: make(map[string]*syncProgressState)}
		state := &syncProgressState{stop: make(chan struct{}), done: make(chan struct{})}
		close(state.stop)

		writer.runDots("/tmp/repo-a", state)

		select {
		case <-state.done:
		default:
			t.Fatal("expected runDots to close done channel")
		}
	})
}
