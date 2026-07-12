// SPDX-License-Identifier: MIT
package gitx_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/gitx"
)

// stubRunner returns a fixed output/error regardless of args and records the
// args of the last call, so wrapper functions can be exercised without pinning
// the exact NUL-delimited format string.
type stubRunner struct {
	out  string
	err  error
	args []string
}

func (s *stubRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	s.args = append([]string(nil), args...)
	return s.out, s.err
}

func TestParseLocalBranches(t *testing.T) {
	// A branch name containing "|" must parse correctly: this is why the format
	// is NUL-delimited rather than pipe-delimited.
	line := strings.Join([]string{
		"feat|x", "origin/feat|x", "[ahead 1]", ">", "2026-07-11T23:30:59-05:00", "/tmp/wt",
	}, "\x00")
	got := gitx.ParseLocalBranches(line + "\n")
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	b := got[0]
	if b.Name != "feat|x" {
		t.Errorf("Name = %q, want %q", b.Name, "feat|x")
	}
	if b.Upstream != "origin/feat|x" {
		t.Errorf("Upstream = %q, want %q", b.Upstream, "origin/feat|x")
	}
	if b.Track != "[ahead 1]" || b.TrackShort != ">" {
		t.Errorf("Track/TrackShort = %q/%q", b.Track, b.TrackShort)
	}
	if !b.HasLastCommit || b.LastCommit.Year() != 2026 {
		t.Errorf("LastCommit = %v (has=%v)", b.LastCommit, b.HasLastCommit)
	}
	if b.WorktreePath != "/tmp/wt" {
		t.Errorf("WorktreePath = %q, want /tmp/wt", b.WorktreePath)
	}
}

func TestParseLocalBranchesEmptyAndNoDate(t *testing.T) {
	if got := gitx.ParseLocalBranches(""); got != nil {
		t.Errorf("empty output = %v, want nil", got)
	}
	// No committer date and no worktree path (an unborn/edge ref).
	got := gitx.ParseLocalBranches("main\x00\x00\x00\x00\x00\n")
	if len(got) != 1 || got[0].Name != "main" {
		t.Fatalf("got %v", got)
	}
	if got[0].HasLastCommit {
		t.Errorf("HasLastCommit = true, want false for empty date")
	}
}

func TestParseCherryEquivalent(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"all equivalent", "- abc123\n- def456\n", true},
		{"has non-equivalent", "- abc123\n+ def456\n", false},
		{"empty (no unique commits)", "", true},
		{"only non-equivalent", "+ abc123\n", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := gitx.ParseCherryEquivalent(tc.output); got != tc.want {
				t.Errorf("ParseCherryEquivalent(%q) = %v, want %v", tc.output, got, tc.want)
			}
		})
	}
}

func TestLocalBranchesRunner(t *testing.T) {
	s := &stubRunner{out: "main\x00origin/main\x00\x00=\x002026-07-11T00:00:00Z\x00\n"}
	got, err := gitx.LocalBranches(context.Background(), s, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "main" {
		t.Fatalf("got %v", got)
	}
	if s.args[0] != "for-each-ref" || s.args[len(s.args)-1] != "refs/heads" {
		t.Errorf("args = %v", s.args)
	}
	if !strings.Contains(s.args[1], "%00") {
		t.Errorf("format arg does not use the %%00 NUL placeholder: %q", s.args[1])
	}

	s.err = errors.New("boom")
	if _, err := gitx.LocalBranches(context.Background(), s, "/repo"); err == nil {
		t.Errorf("expected error when runner fails")
	}
}

func TestMergedBranches(t *testing.T) {
	s := &stubRunner{out: "main\nfeature/done\n\n"}
	got, err := gitx.MergedBranches(context.Background(), s, "/repo", "origin/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got["feature/done"] || !got["main"] || len(got) != 2 {
		t.Errorf("merged set = %v", got)
	}
	if s.args[1] != "--merged=origin/main" {
		t.Errorf("args = %v", s.args)
	}

	if _, err := gitx.MergedBranches(context.Background(), s, "/repo", "  "); err == nil {
		t.Errorf("expected error on empty base")
	}
	s.err = errors.New("fatal: malformed object name")
	if _, err := gitx.MergedBranches(context.Background(), s, "/repo", "nope"); err == nil {
		t.Errorf("expected error when git errors")
	}
}

func TestPatchEquivalentToBase(t *testing.T) {
	s := &stubRunner{out: "- abc123\n"}
	got, err := gitx.PatchEquivalentToBase(context.Background(), s, "/repo", "origin/main", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Errorf("expected patch-equivalent")
	}
	if s.args[0] != "cherry" || s.args[1] != "origin/main" || s.args[2] != "feature" {
		t.Errorf("args = %v", s.args)
	}

	if _, err := gitx.PatchEquivalentToBase(context.Background(), s, "/repo", "", "feature"); err == nil {
		t.Errorf("expected error on empty base")
	}
	s.err = errors.New("boom")
	if _, err := gitx.PatchEquivalentToBase(context.Background(), s, "/repo", "origin/main", "feature"); err == nil {
		t.Errorf("expected error when git errors")
	}
}
