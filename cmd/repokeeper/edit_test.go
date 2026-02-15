package repokeeper

import "testing"

func TestTrackingBranchFromUpstream(t *testing.T) {
	if got := trackingBranchFromUpstream("origin/main"); got != "main" {
		t.Fatalf("expected main, got %q", got)
	}
	if got := trackingBranchFromUpstream("upstream/release/v1"); got != "v1" {
		t.Fatalf("expected v1, got %q", got)
	}
	if got := trackingBranchFromUpstream(""); got != "" {
		t.Fatalf("expected empty for empty upstream, got %q", got)
	}
}
