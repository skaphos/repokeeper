package repokeeper

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/spf13/cobra"
)

func TestNeedsUpstreamRepair(t *testing.T) {
	repo := model.RepoStatus{
		Tracking: model.Tracking{
			Upstream: "origin/main",
			Status:   model.TrackingEqual,
		},
	}
	if needsUpstreamRepair(repo, "origin/main") {
		t.Fatal("expected matching upstream to not need repair")
	}
	if !needsUpstreamRepair(repo, "upstream/main") {
		t.Fatal("expected mismatched upstream to need repair")
	}
	repo.Tracking = model.Tracking{Status: model.TrackingNone}
	if !needsUpstreamRepair(repo, "origin/main") {
		t.Fatal("expected missing tracking to need repair")
	}
	if needsUpstreamRepair(repo, "  ") {
		t.Fatal("expected blank target upstream to skip repair")
	}
}

func TestRepairUpstreamMatchesFilter(t *testing.T) {
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "") {
		t.Fatal("expected empty filter to match")
	}
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "all") {
		t.Fatal("expected all filter to match")
	}
	if !repairUpstreamMatchesFilter("", "origin/main", "missing") {
		t.Fatal("expected missing filter match for empty upstream")
	}
	if repairUpstreamMatchesFilter("origin/main", "origin/main", "missing") {
		t.Fatal("did not expect missing filter match for non-empty upstream")
	}
	if !repairUpstreamMatchesFilter("origin/main", "upstream/main", "mismatch") {
		t.Fatal("expected mismatch filter match")
	}
	if repairUpstreamMatchesFilter("origin/main", "origin/main", "mismatch") {
		t.Fatal("did not expect mismatch filter match for equal upstream")
	}
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "unknown") {
		t.Fatal("expected unknown filters to default to match")
	}
}

func TestWriteRepairUpstreamTable(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeRepairUpstreamTable(cmd, []repairUpstreamResult{
		{
			RepoID:          "github.com/org/repo",
			Path:            "/tmp/work/repo",
			LocalBranch:     "main",
			CurrentUpstream: "",
			TargetUpstream:  "origin/main",
			Action:          "would repair",
			OK:              true,
		},
	}, "/tmp/work", nil, false)

	got := out.String()
	if !strings.Contains(got, "PATH") || !strings.Contains(got, "TARGET") || !strings.Contains(got, "REPO") {
		t.Fatalf("expected repair table header, got: %q", got)
	}
	if !strings.Contains(got, "repo") || !strings.Contains(got, "would repair") || !strings.Contains(got, "origin/main") {
		t.Fatalf("expected repair row content, got: %q", got)
	}
}

func TestWriteRepairUpstreamTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeRepairUpstreamTable(cmd, []repairUpstreamResult{
		{
			RepoID:          "github.com/org/repo",
			Path:            "/tmp/work/repo",
			LocalBranch:     "main",
			CurrentUpstream: "origin/main",
			TargetUpstream:  "origin/main",
			Action:          "unchanged",
			OK:              true,
		},
	}, "/tmp/work", nil, true)

	if strings.Contains(out.String(), "ACTION") {
		t.Fatalf("expected no table headers, got: %q", out.String())
	}
}
