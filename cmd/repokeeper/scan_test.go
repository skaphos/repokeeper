// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/contract"
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

	// scan is an adapter-facing surface, so the top level is the contract
	// envelope rather than a bare array: an array has nowhere to carry
	// apiVersion. The empty-collection guarantee still applies, now to the
	// named payload field.
	var decoded struct {
		APIVersion string            `json:"apiVersion"`
		Repos      []json.RawMessage `json:"repos"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal scan json: %v", err)
	}
	if decoded.APIVersion != contract.APIVersion {
		t.Fatalf("expected apiVersion %q, got %q", contract.APIVersion, decoded.APIVersion)
	}
	if decoded.Repos == nil {
		t.Fatal("expected repos to be a non-nil empty slice (i.e. [] and not JSON null)")
	}
	if len(decoded.Repos) != 0 {
		t.Fatalf("expected zero repos, got %d", len(decoded.Repos))
	}
	// Guard the null-vs-[] distinction at the byte level too: decoding alone
	// cannot tell them apart, since JSON null also unmarshals into a nil slice.
	if !strings.Contains(out.String(), `"repos": []`) {
		t.Fatalf("expected an explicit empty array for repos, got %q", strings.TrimSpace(out.String()))
	}
}
