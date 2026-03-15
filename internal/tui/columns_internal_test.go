// SPDX-License-Identifier: MIT
package tui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestRenderHeader(t *testing.T) {
	t.Parallel()

	cols := defaultColumns()
	widths := distributeWidths(cols, 100)
	header := renderHeader(cols, widths)

	if !strings.Contains(header, "│") {
		t.Fatalf("expected header to contain column separators: %q", header)
	}
	for _, title := range []string{"REPO", "BRANCH", "STATUS", "±", "DIRTY", "ERROR", "SYNCED"} {
		if !strings.Contains(header, title) {
			t.Fatalf("expected header to contain %q: %q", title, header)
		}
	}
}

func TestRenderDivider(t *testing.T) {
	t.Parallel()

	divider := renderDivider(distributeWidths(defaultColumns(), 100))
	if got := strings.Count(divider, "┼"); got != 6 {
		t.Fatalf("expected 6 crossing separators, got %d (%q)", got, divider)
	}
	if !strings.Contains(divider, "─") {
		t.Fatalf("expected divider to contain horizontal glyphs: %q", divider)
	}
	if strings.Contains(divider, "│") {
		t.Fatalf("did not expect vertical separators in divider: %q", divider)
	}
}

func TestRenderRow(t *testing.T) {
	t.Parallel()

	repo := model.RepoStatus{RepoID: "acme/service"}
	row := renderRow(defaultColumns(), distributeWidths(defaultColumns(), 100), repo)
	if !strings.Contains(row, "│") {
		t.Fatalf("expected row to contain column separators: %q", row)
	}
	if !strings.Contains(row, "acme/service") {
		t.Fatalf("expected row to contain repo id: %q", row)
	}
}

func TestRenderRowNilWorktree(t *testing.T) {
	t.Parallel()

	repo := model.RepoStatus{RepoID: "r1", Worktree: nil}
	row := renderRow(defaultColumns(), distributeWidths(defaultColumns(), 100), repo)
	parts := strings.Split(row, " │ ")
	if len(parts) != 7 {
		t.Fatalf("expected 7 columns, got %d", len(parts))
	}
	if got := strings.TrimSpace(parts[4]); got != "-" {
		t.Fatalf("expected DIRTY column to be '-', got %q", got)
	}
}

func TestRenderRowMirror(t *testing.T) {
	t.Parallel()

	repo := model.RepoStatus{RepoID: "r1", Type: "mirror", Head: model.Head{Branch: "main"}}
	row := renderRow(defaultColumns(), distributeWidths(defaultColumns(), 100), repo)
	parts := strings.Split(row, " │ ")
	if got := strings.TrimSpace(parts[1]); got != "-" {
		t.Fatalf("expected BRANCH column to be '-', got %q", got)
	}
}

func TestRenderRowDetachedHead(t *testing.T) {
	t.Parallel()

	repo := model.RepoStatus{RepoID: "r1", Head: model.Head{Detached: true}}
	row := renderRow(defaultColumns(), distributeWidths(defaultColumns(), 100), repo)
	parts := strings.Split(row, " │ ")
	if got := strings.TrimSpace(parts[1]); got != "detached" {
		t.Fatalf("expected BRANCH column to be 'detached', got %q", got)
	}
}

func TestColValueRepo(t *testing.T) {
	t.Parallel()

	if got := colValueRepo(model.RepoStatus{RepoID: "a/b"}); got != "a/b" {
		t.Fatalf("expected repo id, got %q", got)
	}
}

func TestColValueBranch(t *testing.T) {
	t.Parallel()

	if got := colValueBranch(model.RepoStatus{Head: model.Head{Branch: "main"}}); got != "main" {
		t.Fatalf("expected branch, got %q", got)
	}
	if got := colValueBranch(model.RepoStatus{Head: model.Head{Detached: true}}); got != "detached" {
		t.Fatalf("expected detached, got %q", got)
	}
	if got := colValueBranch(model.RepoStatus{Type: "mirror", Head: model.Head{Branch: "main"}}); got != "-" {
		t.Fatalf("expected '-', got %q", got)
	}
}

func TestColValueStatus(t *testing.T) {
	t.Parallel()

	if got := colValueStatus(model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingEqual}}); got != "up to date" {
		t.Fatalf("expected 'up to date', got %q", got)
	}
	if got := colValueStatus(model.RepoStatus{Tracking: model.Tracking{Status: model.TrackingGone}}); got != "gone" {
		t.Fatalf("expected 'gone', got %q", got)
	}
}

func TestColValueDelta(t *testing.T) {
	t.Parallel()

	ahead := 3
	if got := colValueDelta(model.RepoStatus{Tracking: model.Tracking{Ahead: &ahead}}); got != "+3" {
		t.Fatalf("expected '+3', got %q", got)
	}
	behind := 2
	if got := colValueDelta(model.RepoStatus{Tracking: model.Tracking{Behind: &behind}}); got != "-2" {
		t.Fatalf("expected '-2', got %q", got)
	}
	if got := colValueDelta(model.RepoStatus{Tracking: model.Tracking{Ahead: &ahead, Behind: &behind}}); got != "+3/-2" {
		t.Fatalf("expected '+3/-2', got %q", got)
	}
	zero := 0
	if got := colValueDelta(model.RepoStatus{Tracking: model.Tracking{Ahead: &zero, Behind: &zero}}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestColValueDirty(t *testing.T) {
	t.Parallel()

	if got := colValueDirty(model.RepoStatus{Worktree: &model.Worktree{Dirty: true}}); got != "✗" {
		t.Fatalf("expected dirty marker, got %q", got)
	}
	if got := colValueDirty(model.RepoStatus{Worktree: &model.Worktree{Dirty: false}}); got != "" {
		t.Fatalf("expected clean marker, got %q", got)
	}
	if got := colValueDirty(model.RepoStatus{Worktree: nil}); got != "-" {
		t.Fatalf("expected '-' for nil worktree, got %q", got)
	}
}

func TestColValueError(t *testing.T) {
	t.Parallel()

	if got := colValueError(model.RepoStatus{ErrorClass: "missing"}); got != "missing" {
		t.Fatalf("expected error class, got %q", got)
	}
}

func TestColValueSynced(t *testing.T) {
	t.Parallel()

	if got := colValueSynced(model.RepoStatus{}); got != "" {
		t.Fatalf("expected empty synced value, got %q", got)
	}
	if got := colValueSynced(model.RepoStatus{LastSync: &model.SyncResult{At: time.Now().Add(-2 * time.Hour)}}); got == "" {
		t.Fatal("expected non-empty synced value")
	}
}

func TestDistributeWidths(t *testing.T) {
	t.Parallel()

	cols := defaultColumns()
	total := 120
	widths := distributeWidths(cols, total)
	if len(widths) != len(cols) {
		t.Fatalf("expected %d widths, got %d", len(cols), len(widths))
	}
	used := 0
	for i, w := range widths {
		if w < cols[i].minWidth {
			t.Fatalf("column %d width %d below min %d", i, w, cols[i].minWidth)
		}
		used += w
	}
	used += (len(cols) - 1) * 3
	if used > total {
		t.Fatalf("expected used width <= total, got used=%d total=%d", used, total)
	}
	if used < total-10 {
		t.Fatalf("expected used width close to total, got used=%d total=%d", used, total)
	}
}

func TestTruncPad(t *testing.T) {
	t.Parallel()

	if got := truncPad("hello", 5); utf8.RuneCountInString(got) != 5 {
		t.Fatalf("expected length 5, got %d (%q)", utf8.RuneCountInString(got), got)
	}
	truncated := truncPad("helloworld", 5)
	if utf8.RuneCountInString(truncated) != 5 {
		t.Fatalf("expected length 5, got %d (%q)", utf8.RuneCountInString(truncated), truncated)
	}
	if !strings.HasSuffix(truncated, "…") {
		t.Fatalf("expected ellipsis truncation, got %q", truncated)
	}
	padded := truncPad("hi", 5)
	if utf8.RuneCountInString(padded) != 5 {
		t.Fatalf("expected length 5, got %d (%q)", utf8.RuneCountInString(padded), padded)
	}
	if !strings.HasPrefix(padded, "hi") {
		t.Fatalf("expected padded string to start with original text, got %q", padded)
	}
}
