// SPDX-License-Identifier: MIT

package repokeeper

import (
	"encoding/json"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/contract"
)

// decodeTopLevel marshals v the way a command would and returns the decoded
// top-level JSON object, failing if the result is not an object.
//
// Asserting "is an object" matters independently of the field checks: the
// defect this contract exists to fix was surfaces emitting a top-level *array*,
// which has nowhere to carry a version.
func decodeTopLevel(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("top-level value is not a JSON object (%v): %s", err, data)
	}
	return decoded
}

func assertCarriesAPIVersion(t *testing.T, name string, v any) {
	t.Helper()

	decoded := decodeTopLevel(t, v)
	raw, present := decoded["apiVersion"]
	if !present {
		t.Fatalf("%s: envelope is missing apiVersion", name)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("%s: apiVersion is not a string: %v", name, err)
	}
	if got != contract.APIVersion {
		t.Errorf("%s: apiVersion = %q, want %q", name, got, contract.APIVersion)
	}
}

// TestEveryEnvelopeCarriesAPIVersion covers FR-001 across the envelope types
// the CLI surfaces use. Each adapter-facing command routes its payload through
// one of these, so an envelope that satisfies the contract here satisfies it at
// the command too.
func TestEveryEnvelopeCarriesAPIVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		envelope any
	}{
		{"results", newResultsEnvelope([]string{"a"})},
		{"repos", newReposEnvelope([]string{"a"})},
		{"repo", newRepoEnvelope("a")},
		{"labels", newLabelsEnvelope(map[string]string{"k": "v"})},
		{"version", newVersionEnvelope("v2.0.0")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertCarriesAPIVersion(t, tc.name, tc.envelope)
		})
	}
}

// TestCollectionEnvelopesEmitEmptyArrayNotNull covers FR-007.
//
// A nil Go slice marshals as `null`, which breaks an adapter parsing `[]`
// uniformly. Decoding cannot catch this -- `null` also unmarshals into a nil
// slice -- so the assertion is on the emitted bytes.
func TestCollectionEnvelopesEmitEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		key      string
		envelope any
	}{
		{"results", "results", newResultsEnvelope[string](nil)},
		{"repos", "repos", newReposEnvelope[string](nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decoded := decodeTopLevel(t, tc.envelope)
			raw, present := decoded[tc.key]
			if !present {
				t.Fatalf("%s: payload key %q is absent; an empty collection must still be reported", tc.name, tc.key)
			}
			if string(raw) != "[]" {
				t.Errorf("%s: payload %q = %s, want []", tc.name, tc.key, raw)
			}
		})
	}
}

// TestEnvelopesNameTheirPayload covers the "exactly one named payload" rule.
//
// An anonymous payload would force a breaking change the first time a surface
// needed to report anything alongside its results, so the shape is asserted
// rather than left to convention.
func TestEnvelopesNameTheirPayload(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key      string
		envelope any
	}{
		{"results", newResultsEnvelope([]string{"a"})},
		{"repos", newReposEnvelope([]string{"a"})},
		{"repo", newRepoEnvelope("a")},
		{"labels", newLabelsEnvelope("a")},
		{"version", newVersionEnvelope("a")},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			decoded := decodeTopLevel(t, tc.envelope)
			if _, present := decoded[tc.key]; !present {
				t.Fatalf("envelope does not name its payload %q; got keys %v", tc.key, keysOf(decoded))
			}
			// apiVersion plus exactly one payload. generated_at is omitted by
			// these constructors, so anything else is an unnamed extra.
			if len(decoded) != 2 {
				t.Errorf("expected apiVersion plus exactly one named payload, got keys %v", keysOf(decoded))
			}
		})
	}
}

// TestStatusEnvelopeEmitsEmptyReposForNilReport covers FR-007 on the one
// surface that was already enveloped. Before this contract the nil-report path
// emitted `"repos": null`.
func TestStatusEnvelopeEmitsEmptyReposForNilReport(t *testing.T) {
	t.Parallel()

	decoded := decodeTopLevel(t, buildStatusJSONOutput(nil, false))
	raw, present := decoded["repos"]
	if !present {
		t.Fatal("status envelope omitted repos for a nil report")
	}
	if string(raw) != "[]" {
		t.Errorf("repos = %s, want [] for a nil report", raw)
	}
	assertCarriesAPIVersion(t, "status", buildStatusJSONOutput(nil, false))
}

// TestStatusEnvelopeOmitsGeneratedAtForNilReport documents the deliberate
// choice not to stamp a zero time when there is no report to observe: a
// consumer cannot distinguish "not applicable" from "the epoch".
func TestStatusEnvelopeOmitsGeneratedAtForNilReport(t *testing.T) {
	t.Parallel()

	decoded := decodeTopLevel(t, buildStatusJSONOutput(nil, false))
	if _, present := decoded["generated_at"]; present {
		t.Error("generated_at must be omitted when there is no report to timestamp")
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
