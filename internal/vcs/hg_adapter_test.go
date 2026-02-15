package vcs

import (
	"context"
	"testing"
)

func TestHgAdapterSyncCapabilityMetadata(t *testing.T) {
	adapter := NewHgAdapter()

	supported, reason, err := adapter.SupportsLocalUpdate(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("SupportsLocalUpdate returned error: %v", err)
	}
	if supported {
		t.Fatal("expected local updates to be unsupported for hg")
	}
	if reason == "" {
		t.Fatal("expected non-empty skip reason")
	}

	action, err := adapter.FetchAction(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("FetchAction returned error: %v", err)
	}
	if action != "hg pull" {
		t.Fatalf("unexpected fetch action: %q", action)
	}
}
