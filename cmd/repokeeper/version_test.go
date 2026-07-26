// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/buildinfo"
)

// withLDFlags swaps the package-level ldflags variables for one test and
// restores them afterwards, so cases can drive each resolution tier.
func withLDFlags(t *testing.T, version, commit, date string) {
	t.Helper()
	oldV, oldC, oldD := Version, Commit, Date
	Version, Commit, Date = version, commit, date
	t.Cleanup(func() { Version, Commit, Date = oldV, oldC, oldD })
}

// runVersion executes the version command with the given format and returns
// stdout.
func runVersion(t *testing.T, format string) string {
	t.Helper()
	var buf bytes.Buffer
	cmd := versionCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	t.Cleanup(func() { cmd.SetOut(nil); cmd.SetErr(nil) })

	if err := cmd.Flags().Set("format", format); err != nil {
		t.Fatalf("setting format flag: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("format", "table") })

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("version command: %v", err)
	}
	return buf.String()
}

// TestVersionReleaseBuildOutputUnchanged pins the release-build rendering. A
// released binary must report exactly what it reported before this command
// learned about build metadata -- the change has to be invisible to anyone
// installing from a release.
func TestVersionReleaseBuildOutputUnchanged(t *testing.T) {
	withLDFlags(t, "v0.8.0", "9f2c1ab", "2026-07-26T14:22:31Z")

	out := runVersion(t, "table")

	if !strings.HasPrefix(out, "repokeeper v0.8.0\n") {
		t.Errorf("release headline must be bare version, got:\n%s", out)
	}
	for _, want := range []string{
		"  commit:  9f2c1ab\n",
		"  built:   2026-07-26T14:22:31Z\n",
		"  go:      ",
		"  os/arch: ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// A release build must not be annotated with its source; that would
	// change output every existing user already sees.
	if strings.Contains(out, "release build") {
		t.Errorf("release output must not carry a source annotation:\n%s", out)
	}
}

// TestVersionNeverPrintsAPlaceholder is the requirement stated directly at the
// command boundary, across every tier reachable by varying the ldflags.
func TestVersionNeverPrintsAPlaceholder(t *testing.T) {
	cases := []struct{ name, v, c, d string }{
		{"unstamped sentinels", "dev", "none", "unknown"},
		{"empty ldflags", "", "", ""},
		{"stamped version, unstamped commit and date", "v0.8.0", "none", "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLDFlags(t, tc.v, tc.c, tc.d)
			out := runVersion(t, "table")

			// "repokeeper dev" is the exact defect this feature fixes.
			if strings.Contains(out, "repokeeper dev\n") {
				t.Errorf("printed the dev placeholder:\n%s", out)
			}
			for _, bad := range []string{"none", "unknown\n"} {
				if strings.Contains(out, "commit:  "+bad) || strings.Contains(out, "built:   "+bad) {
					t.Errorf("printed sentinel %q literally:\n%s", bad, out)
				}
			}
		})
	}
}

// TestVersionUnstampedReportsUnavailableNotOmitted: an absent line invites the
// reader to assume the field did not apply. Unknown fields must say so.
func TestVersionUnstampedFieldsReadUnavailable(t *testing.T) {
	withLDFlags(t, "v0.8.0", "none", "unknown")

	out := runVersion(t, "table")

	if !strings.Contains(out, "commit:  unavailable") {
		t.Errorf("unknown commit must render as unavailable:\n%s", out)
	}
	if !strings.Contains(out, "built:   unavailable") {
		t.Errorf("unknown build time must render as unavailable:\n%s", out)
	}
}

func TestVersionJSONShape(t *testing.T) {
	withLDFlags(t, "v0.8.0", "9f2c1ab", "2026-07-26T14:22:31Z")

	var got versionJSON
	if err := json.Unmarshal([]byte(runVersion(t, "json")), &got); err != nil {
		t.Fatalf("version --format json did not emit valid JSON: %v", err)
	}

	if got.Version != "v0.8.0" {
		t.Errorf("version: got %q want %q", got.Version, "v0.8.0")
	}
	if got.Revision != "9f2c1ab" {
		t.Errorf("revision: got %q", got.Revision)
	}
	if got.Source != "release build" {
		t.Errorf("source: got %q want %q", got.Source, "release build")
	}
	if got.Go == "" || got.OS == "" || got.Arch == "" {
		t.Errorf("go/os/arch must always be populated: %+v", got)
	}
}

// TestVersionJSONUnknownFieldsAreEmptyNotProse: a consumer testing for
// emptiness gets one answer; matching the literal "unavailable" would mean
// parsing prose out of a machine-readable field.
func TestVersionJSONUnknownFieldsAreEmpty(t *testing.T) {
	withLDFlags(t, "v0.8.0", "none", "unknown")

	raw := runVersion(t, "json")

	var got versionJSON
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.Revision != "" {
		t.Errorf("unknown revision must be empty in JSON, got %q", got.Revision)
	}
	if got.Built != "" {
		t.Errorf("unknown build time must be empty in JSON, got %q", got.Built)
	}
	if strings.Contains(raw, `"unavailable"`) {
		t.Errorf("JSON must not carry the human-readable unavailable marker:\n%s", raw)
	}

	// Every key is present even when empty -- omitempty would make a
	// consumer unable to distinguish "absent" from "not recorded".
	for _, key := range []string{`"version"`, `"revision"`, `"built"`, `"modified"`, `"source"`} {
		if !strings.Contains(raw, key) {
			t.Errorf("JSON missing required key %s:\n%s", key, raw)
		}
	}
}

func TestVersionRejectsUnsupportedFormat(t *testing.T) {
	withLDFlags(t, "v0.8.0", "9f2c1ab", "2026-07-26T14:22:31Z")

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	t.Cleanup(func() { versionCmd.SetOut(nil) })

	if err := versionCmd.Flags().Set("format", "wide"); err != nil {
		t.Fatalf("setting format: %v", err)
	}
	t.Cleanup(func() { _ = versionCmd.Flags().Set("format", "table") })

	// wide is accepted (rendered as table); custom-columns is not.
	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Errorf("wide format should render as table, got error: %v", err)
	}

	if err := versionCmd.Flags().Set("format", "custom-columns=A:.b"); err != nil {
		t.Fatalf("setting format: %v", err)
	}
	if err := versionCmd.RunE(versionCmd, nil); err == nil {
		t.Error("custom-columns must be rejected for version")
	}
}

func TestVersionHeadline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info buildinfo.Info
		want string
	}{
		{
			name: "release build is a bare version",
			info: buildinfo.Info{Version: "v0.8.0", Source: buildinfo.SourceLDFlags},
			want: "v0.8.0",
		},
		{
			name: "module install is annotated with its source",
			info: buildinfo.Info{Version: "v0.8.0", Source: buildinfo.SourceBuildInfo},
			want: "v0.8.0 (build metadata)",
		},
		{
			name: "local clean build reports no released version",
			info: buildinfo.Info{Revision: "abc123", Source: buildinfo.SourceBuildInfo},
			want: "(devel) — no released version (build metadata)",
		},
		{
			name: "local dirty build says so",
			info: buildinfo.Info{Revision: "abc123", Modified: true, Source: buildinfo.SourceBuildInfo},
			want: "(devel) — no released version, modified working tree (build metadata)",
		},
		{
			name: "nothing recorded",
			info: buildinfo.Info{Source: buildinfo.SourceUnknown},
			want: "— version information unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := versionHeadline(tc.info); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

// TestAdvertisedVersionMatchesResolution guards FR-005: the MCP handshake and
// the version command must resolve from the same identity.
func TestAdvertisedVersion(t *testing.T) {
	t.Run("released version is advertised verbatim", func(t *testing.T) {
		withLDFlags(t, "v0.8.0", "9f2c1ab", "2026-07-26T14:22:31Z")
		if got := advertisedVersion(); got != "v0.8.0" {
			t.Errorf("got %q want %q", got, "v0.8.0")
		}
		if got := advertisedVersion(); got != resolvedBuildInfo().Version {
			t.Errorf("advertised version diverged from resolved identity: %q", got)
		}
	})

	t.Run("never advertises a placeholder", func(t *testing.T) {
		withLDFlags(t, "dev", "none", "unknown")
		got := advertisedVersion()
		for _, bad := range []string{"dev", "(devel)", "none"} {
			if got == bad {
				t.Errorf("advertised placeholder %q", bad)
			}
		}
	})
}

func TestRevisionField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info buildinfo.Info
		want string
	}{
		{"absent", buildinfo.Info{}, "unavailable"},
		{"clean", buildinfo.Info{Revision: "abc"}, "abc"},
		{"dirty", buildinfo.Info{Revision: "abc", Modified: true}, "abc (dirty)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := revisionField(tc.info); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}
