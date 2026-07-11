// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestScanJSONOutputEmptyResultSetIsEmptyArray guards a divergence from the
// get/status -o json contract: those commands always marshal a non-nil
// (possibly zero-length) slice, so an empty result prints "[]"; scan used to
// marshal the raw (possibly nil) []model.RepoStatus straight from eng.Scan,
// which json.MarshalIndent renders as the literal "null" for a nil slice.
// Scanning an empty root must still print "[]" like get/status do.
func TestScanJSONOutputEmptyResultSetIsEmptyArray(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	scanCmd.SetOut(out)
	scanCmd.SetContext(context.Background())
	defer scanCmd.SetOut(os.Stdout)

	// An empty, freshly created temp dir (the config root) has no repos to
	// discover, so eng.Scan returns a nil/empty []model.RepoStatus.
	_ = scanCmd.Flags().Set("roots", "")
	_ = scanCmd.Flags().Set("exclude", "")
	_ = scanCmd.Flags().Set("follow-symlinks", "false")
	_ = scanCmd.Flags().Set("write-registry", "false")
	_ = scanCmd.Flags().Set("prune-stale", "false")
	_ = scanCmd.Flags().Set("format", "json")
	_ = scanCmd.Flags().Set("no-headers", "false")

	if err := scanCmd.RunE(scanCmd, nil); err != nil {
		t.Fatalf("scan json failed: %v", err)
	}

	got := strings.TrimSpace(out.String())
	if got != "[]" {
		t.Fatalf("expected empty scan result to print exactly %q, got %q", "[]", got)
	}

	// Also confirm it round-trips as a non-nil empty slice, not a JSON null.
	var decoded []json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal scan json: %v", err)
	}
	if decoded == nil {
		t.Fatal("expected decoded slice to be non-nil (i.e. not JSON null)")
	}
	if len(decoded) != 0 {
		t.Fatalf("expected zero elements, got %d", len(decoded))
	}
}
