// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestRelatedReposString(t *testing.T) {
	t.Parallel()

	if got := relatedReposString(nil); got != "-" {
		t.Fatalf("expected dash for empty related repos, got %q", got)
	}

	got := relatedReposString([]model.RepoMetadataRelatedRepo{{RepoID: "z/repo", Relationship: "depends-on"}, {RepoID: "a/repo"}})
	if got != "a/repo,z/repo:depends-on" {
		t.Fatalf("expected sorted related repos string, got %q", got)
	}
}

// TestStatusJSONOutputIncludesAPIVersion locks the SKA-208 contract: the
// `get`/`status -o json` envelope carries a top-level apiVersion equal to the
// schema constant, in both the normal and --only diverged shapes, while
// preserving the existing generated_at/repos fields. Assertions unmarshal into
// a map so a *removed* field is detected (a struct field would silently default
// to its zero value).
func TestStatusJSONOutputIncludesAPIVersion(t *testing.T) {
	t.Parallel()

	generatedAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	report := &model.StatusReport{
		GeneratedAt: generatedAt,
		Repos: []model.RepoStatus{
			{RepoID: "github.com/org/healthy", Path: "/repos/healthy", Tracking: model.Tracking{Status: model.TrackingEqual}},
			{
				RepoID:   "github.com/org/diverged",
				Path:     "/repos/diverged",
				Tracking: model.Tracking{Status: model.TrackingDiverged, Upstream: "origin/main"},
			},
		},
	}

	decode := func(t *testing.T, v any) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal status json: %v", err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal status json: %v\n%s", err, raw)
		}
		return doc
	}

	assertString := func(t *testing.T, doc map[string]json.RawMessage, key string) string {
		t.Helper()
		raw, ok := doc[key]
		if !ok {
			t.Fatalf("expected key %q to be present in output (existing field must not be dropped)", key)
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("key %q is not a string: %v", key, err)
		}
		return s
	}

	assertArrayLen := func(t *testing.T, doc map[string]json.RawMessage, key string, want int) {
		t.Helper()
		raw, ok := doc[key]
		if !ok {
			t.Fatalf("expected key %q to be present in output", key)
		}
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Fatalf("key %q is not an array: %v", key, err)
		}
		if len(arr) != want {
			t.Errorf("%s length = %d, want %d", key, len(arr), want)
		}
	}

	t.Run("normal output", func(t *testing.T) {
		t.Parallel()
		doc := decode(t, buildStatusJSONOutput(report, false))

		if got := assertString(t, doc, "apiVersion"); got != statusJSONAPIVersion {
			t.Errorf("apiVersion = %q, want %q", got, statusJSONAPIVersion)
		}
		// generated_at must survive and round-trip, not just exist.
		if got := assertString(t, doc, "generated_at"); got != generatedAt.Format(time.RFC3339Nano) {
			t.Errorf("generated_at = %q, want %q", got, generatedAt.Format(time.RFC3339Nano))
		}
		assertArrayLen(t, doc, "repos", len(report.Repos))
		if _, ok := doc["diverged"]; ok {
			t.Errorf("normal output must not include a diverged array")
		}
	})

	t.Run("diverged output", func(t *testing.T) {
		t.Parallel()
		doc := decode(t, buildStatusJSONOutput(report, true))

		if got := assertString(t, doc, "apiVersion"); got != statusJSONAPIVersion {
			t.Errorf("diverged apiVersion = %q, want %q", got, statusJSONAPIVersion)
		}
		// The existing repos field must remain alongside the diverged advice.
		assertArrayLen(t, doc, "repos", len(report.Repos))
		assertArrayLen(t, doc, "diverged", 1)
	})
}

// TestStatusJSONOutputNilReportIsSafe guards the nil-report path: building the
// diverged envelope from a nil report must not panic and must still carry the
// apiVersion (regression for the buildDivergedAdvice nil-deref).
func TestStatusJSONOutputNilReportIsSafe(t *testing.T) {
	t.Parallel()
	for _, includeDiverged := range []bool{false, true} {
		raw, err := json.Marshal(buildStatusJSONOutput(nil, includeDiverged))
		if err != nil {
			t.Fatalf("marshal nil report (diverged=%t): %v", includeDiverged, err)
		}
		var doc struct {
			APIVersion string `json:"apiVersion"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal nil report (diverged=%t): %v", includeDiverged, err)
		}
		if doc.APIVersion != statusJSONAPIVersion {
			t.Errorf("nil report (diverged=%t) apiVersion = %q, want %q", includeDiverged, doc.APIVersion, statusJSONAPIVersion)
		}
	}
}

// TestDesignDocNamesStatusJSONAPIVersion is the drift guard backing DESIGN.md's
// claim that the documented schema version cannot silently diverge from the
// emitted constant. If statusJSONAPIVersion is bumped without updating §6.3,
// this fails.
func TestDesignDocNamesStatusJSONAPIVersion(t *testing.T) {
	t.Parallel()
	const docPath = "../../DESIGN.md"
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	content := string(data)

	// Scope the check to the §6.3 Status JSON schema section. The same
	// apiVersion value also appears elsewhere in the doc (e.g. the config
	// example), so a document-wide search would still pass if §6.3 went stale.
	const header = "### 6.3 Status JSON schema"
	start := strings.Index(content, header)
	if start < 0 {
		t.Fatalf("%s is missing the %q section header", docPath, header)
	}
	section := content[start:]
	// The section runs until the next top-level (## ) heading. "\n## " does not
	// match the "#### " subheadings within §6.3, so the policy text is included.
	if end := strings.Index(section[len(header):], "\n## "); end >= 0 {
		section = section[:len(header)+end]
	}
	if !strings.Contains(section, statusJSONAPIVersion) {
		t.Fatalf("%s §6.3 does not name the current statusJSONAPIVersion %q; update the Status JSON schema section", docPath, statusJSONAPIVersion)
	}
}
