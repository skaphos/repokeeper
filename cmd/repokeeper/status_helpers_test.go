// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/spf13/cobra"
)

// TestTruncateASCIIRuneBoundary guards against slicing multi-byte UTF-8
// values on a byte boundary, which produces invalid UTF-8 in table cells for
// non-ASCII paths/branches. Every case is asserted with utf8.ValidString in
// addition to checking the exact expected content.
func TestTruncateASCIIRuneBoundary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		max   int
		want  string
	}{
		{
			name:  "ascii value shorter than max is unchanged",
			value: "abc",
			max:   10,
			want:  "abc",
		},
		{
			name:  "ascii hard truncate at small max",
			value: "abcdef",
			max:   3,
			want:  "abc",
		},
		{
			name: "multi-byte value truncated with ellipsis stays valid utf-8",
			// each kanji is 3 bytes in UTF-8; old byte-slicing logic would cut
			// mid-rune for a max that isn't a multiple of the rune's byte width.
			value: "日本語テスト",
			max:   4,
			want:  "日...",
		},
		{
			name:  "multi-byte value hard-truncated at small max stays valid utf-8",
			value: "日本語",
			max:   2,
			want:  "日本",
		},
		{
			name:  "multi-byte value shorter than max is unchanged",
			value: "日本語",
			max:   10,
			want:  "日本語",
		},
		{
			name:  "zero max returns empty string without panicking",
			value: "abcdef",
			max:   0,
			want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := truncateASCII(tc.value, tc.max)
			if got != tc.want {
				t.Fatalf("truncateASCII(%q, %d) = %q, want %q", tc.value, tc.max, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("truncateASCII(%q, %d) = %q is not valid UTF-8", tc.value, tc.max, got)
			}
		})
	}
}

// TestSanitizeForDisplayStripsControlAndANSISequences guards writeStatusDetails
// against emitting raw ANSI/control sequences from untrusted label,
// annotation, error, and repo-metadata values (settable via the CLI or an
// imported repo-local metadata bundle) verbatim to the TTY.
func TestSanitizeForDisplayStripsControlAndANSISequences(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain value is unchanged", value: "prod-team", want: "prod-team"},
		{name: "empty value is unchanged", value: "", want: ""},
		{
			name:  "csi color sequence is stripped",
			value: "\x1b[31mdanger\x1b[0m",
			want:  "danger",
		},
		{
			name:  "osc terminal title injection is stripped",
			value: "safe\x1b]0;pwned\x07tail",
			want:  "safetail",
		},
		{
			name:  "embedded newline cannot forge an extra KEY: line",
			value: "team=platform\nADMIN: true",
			want:  "team=platformADMIN: true",
		},
		{
			name:  "carriage return and tab are stripped",
			value: "a\rb\tc",
			want:  "abc",
		},
		{
			name:  "del and other c0 control bytes are stripped",
			value: "a\x7fb\x00c",
			want:  "abc",
		},
		{
			name:  "unicode content outside the control range is preserved",
			value: "日本語-label",
			want:  "日本語-label",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeForDisplay(tc.value)
			if got != tc.want {
				t.Fatalf("sanitizeForDisplay(%q) = %q, want %q", tc.value, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("sanitizeForDisplay(%q) = %q is not valid UTF-8", tc.value, got)
			}
		})
	}
}

// TestWriteStatusDetailsSanitizesUntrustedValues is an end-to-end regression
// for the writeStatusDetails wiring: a control/ANSI sequence smuggled into a
// label, an annotation, the error string, or repo metadata must not reach
// cmd.OutOrStdout() verbatim.
func TestWriteStatusDetailsSanitizesUntrustedValues(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	repo := model.RepoStatus{
		Path:   "/repos/testrepo",
		Head:   model.Head{Branch: "main"},
		Labels: map[string]string{"team": "platform\x1b[31m"},
		Annotations: map[string]string{
			"owner": "sre\nADMIN: true",
		},
		Tracking:          model.Tracking{Status: model.TrackingNone},
		Error:             "boom\x1b]0;pwned\x07after",
		RepoMetadataError: "bad\x1b[2Jmetadata",
		RepoMetadata: &model.RepoMetadata{
			Name:   "evil\x1b[31mname",
			RepoID: "org/evil\x1b[0mid",
		},
	}

	if err := writeStatusDetails(cmd, repo, "/repos", nil); err != nil {
		t.Fatalf("writeStatusDetails returned error: %v", err)
	}
	got := out.String()

	if strings.ContainsRune(got, 0x1b) {
		t.Fatalf("expected no raw ESC bytes in output, got %q", got)
	}
	if strings.Contains(got, "\x07") {
		t.Fatalf("expected no raw BEL byte in output, got %q", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if line == "ADMIN: true" {
			t.Fatalf("embedded newline in annotation value forged a standalone ADMIN line: %q", got)
		}
	}
	if !strings.Contains(got, "LABELS: team=platform\n") {
		t.Fatalf("expected sanitized label line, got %q", got)
	}
	if !strings.Contains(got, "ERROR: boomafter\n") {
		t.Fatalf("expected sanitized error line, got %q", got)
	}
	if !strings.Contains(got, "REPO_METADATA_ERROR: badmetadata\n") {
		t.Fatalf("expected sanitized repo metadata error line, got %q", got)
	}
	if !strings.Contains(got, "REPO_METADATA_NAME: evilname\n") {
		t.Fatalf("expected sanitized repo metadata name line, got %q", got)
	}
	if !strings.Contains(got, "REPO_METADATA_REPO_ID: org/evilid\n") {
		t.Fatalf("expected sanitized repo metadata repo id line, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 output, got %q", got)
	}
}

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

func TestStatusJSONOutputFlagsGoneRepairSuggestion(t *testing.T) {
	t.Parallel()

	report := &model.StatusReport{
		GeneratedAt: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC),
		Repos: []model.RepoStatus{
			{RepoID: "github.com/org/gone", Path: "/repos/gone", Tracking: model.Tracking{Status: model.TrackingGone}, RemoteTrackingRefs: model.RemoteTrackingRefStatus{StaleCount: 1, Stale: []string{"origin/merged"}}},
			{RepoID: "github.com/org/clean", Path: "/repos/clean", Tracking: model.Tracking{Status: model.TrackingEqual}},
		},
	}

	raw, err := json.Marshal(buildStatusJSONOutput(report, false))
	if err != nil {
		t.Fatalf("marshal status json: %v", err)
	}
	var doc struct {
		Repos []struct {
			RepoID                   string                        `json:"repo_id"`
			RepairUpstreamSuggestion bool                          `json:"repair_upstream_suggestion"`
			RemoteTrackingRefs       model.RemoteTrackingRefStatus `json:"remote_tracking_refs"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal status json: %v\n%s", err, raw)
	}
	if len(doc.Repos) != 2 {
		t.Fatalf("repos length = %d, want 2", len(doc.Repos))
	}
	if !doc.Repos[0].RepairUpstreamSuggestion {
		t.Fatalf("expected gone repo to carry repair_upstream_suggestion: %s", raw)
	}
	if doc.Repos[0].RemoteTrackingRefs.StaleCount != 1 || len(doc.Repos[0].RemoteTrackingRefs.Stale) != 1 {
		t.Fatalf("expected machine-readable stale ref status: %s", raw)
	}
	if doc.Repos[1].RepairUpstreamSuggestion {
		t.Fatalf("did not expect clean repo to carry repair_upstream_suggestion: %s", raw)
	}
}

func TestWriteRepairUpstreamHint(t *testing.T) {
	t.Parallel()

	report := &model.StatusReport{Repos: []model.RepoStatus{
		{Tracking: model.Tracking{Status: model.TrackingGone}},
		{Tracking: model.Tracking{Status: model.TrackingEqual}},
		{Tracking: model.Tracking{Status: model.TrackingGone}},
	}}

	t.Run("writes hint for gone repos", func(t *testing.T) {
		t.Parallel()
		errOut := &bytes.Buffer{}
		cmd := &cobra.Command{}
		cmd.Flags().Bool("quiet", false, "")
		cmd.SetErr(errOut)

		if err := writeRepairUpstreamHint(cmd, report); err != nil {
			t.Fatalf("write hint: %v", err)
		}
		got := errOut.String()
		if !strings.Contains(got, "hint: 2 repo(s) have gone upstream") {
			t.Fatalf("expected gone hint count, got %q", got)
		}
		if !strings.Contains(got, "repokeeper repair upstream") {
			t.Fatalf("expected repair-upstream command hint, got %q", got)
		}
	})

	t.Run("omits hint without gone repos", func(t *testing.T) {
		t.Parallel()
		errOut := &bytes.Buffer{}
		cmd := &cobra.Command{}
		cmd.Flags().Bool("quiet", false, "")
		cmd.SetErr(errOut)

		cleanReport := &model.StatusReport{Repos: []model.RepoStatus{{Tracking: model.Tracking{Status: model.TrackingEqual}}}}
		if err := writeRepairUpstreamHint(cmd, cleanReport); err != nil {
			t.Fatalf("write hint: %v", err)
		}
		if got := errOut.String(); got != "" {
			t.Fatalf("expected no hint without gone repos, got %q", got)
		}
	})

	t.Run("quiet suppresses hint", func(t *testing.T) {
		t.Parallel()
		errOut := &bytes.Buffer{}
		cmd := &cobra.Command{}
		cmd.Flags().Bool("quiet", true, "")
		cmd.SetErr(errOut)

		if err := writeRepairUpstreamHint(cmd, report); err != nil {
			t.Fatalf("write hint: %v", err)
		}
		if got := errOut.String(); got != "" {
			t.Fatalf("expected quiet mode to suppress hint, got %q", got)
		}
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
