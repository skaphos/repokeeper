// SPDX-License-Identifier: MIT

package buildinfo

import (
	"runtime/debug"
	"testing"
)

// bi builds a synthetic *debug.BuildInfo. The real debug.ReadBuildInfo reads
// the running test binary, so it reports whatever `go test` happened to stamp
// -- which is not the thing under test. Every case here drives the resolver
// through an injected fixture instead.
func bi(mainVersion string, settings map[string]string) *debug.BuildInfo {
	out := &debug.BuildInfo{}
	out.Main.Version = mainVersion
	for k, v := range settings {
		out.Settings = append(out.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return out
}

const (
	rev     = "9f2c1ab4e5d6789012345678901234567890abcd"
	ts      = "2026-07-26T14:22:31Z"
	release = "v0.8.0"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		ldVersion, ldCommit, ldDate string
		info                        *debug.BuildInfo
		ok                          bool

		wantVersion  string
		wantRevision string
		wantTime     string
		wantModified bool
		wantSource   Source
	}{
		{
			// Tier 1. A released binary must report exactly what the
			// pipeline stamped, regardless of what else is recorded.
			name:      "ldflags stamped wins over build metadata",
			ldVersion: release, ldCommit: rev, ldDate: ts,
			info: bi("v0.0.1", map[string]string{"vcs.revision": "other"}),
			ok:   true,

			wantVersion: release, wantRevision: rev, wantTime: ts,
			wantSource: SourceLDFlags,
		},
		{
			// The historical ldflags default is not a version. It must
			// fall through, or `go install` keeps reporting "dev".
			name:      "dev sentinel falls through to module version",
			ldVersion: "dev", ldCommit: "none", ldDate: "unknown",
			info: bi(release, nil),
			ok:   true,

			wantVersion: release,
			wantSource:  SourceBuildInfo,
		},
		{
			name:      "empty ldflags falls through to module version",
			ldVersion: "", ldCommit: "", ldDate: "",
			info: bi(release, nil),
			ok:   true,

			wantVersion: release,
			wantSource:  SourceBuildInfo,
		},
		{
			// A stamped version with unstamped commit/date: the
			// sentinels mean "not recorded" and must not be printed.
			name:      "commit and date sentinels map to empty",
			ldVersion: release, ldCommit: "none", ldDate: "unknown",
			info: bi("", nil),
			ok:   true,

			wantVersion: release, wantRevision: "", wantTime: "",
			wantSource: SourceLDFlags,
		},
		{
			// Tier 2 with full VCS stamps, as `go build` records.
			name: "module version with vcs settings",
			info: bi(release, map[string]string{
				"vcs.revision": rev,
				"vcs.time":     ts,
				"vcs.modified": "false",
			}),
			ok: true,

			wantVersion: release, wantRevision: rev, wantTime: ts,
			wantSource: SourceBuildInfo,
		},
		{
			// Tier 3. "(devel)" is a placeholder; surfacing it is the
			// same defect as surfacing "dev". Report the revision and
			// leave Version empty.
			name: "devel placeholder yields empty version with revision",
			info: bi("(devel)", map[string]string{
				"vcs.revision": rev,
				"vcs.time":     ts,
			}),
			ok: true,

			wantVersion: "", wantRevision: rev, wantTime: ts,
			wantSource: SourceBuildInfo,
		},
		{
			name: "modified working tree surfaces",
			info: bi("(devel)", map[string]string{
				"vcs.revision": rev,
				"vcs.modified": "true",
			}),
			ok: true,

			wantRevision: rev, wantModified: true,
			wantSource: SourceBuildInfo,
		},
		{
			name: "empty main version with revision reports the revision",
			info: bi("", map[string]string{"vcs.revision": rev}),
			ok:   true,

			wantRevision: rev,
			wantSource:   SourceBuildInfo,
		},
		{
			// Tier 4: nothing usable. Must not invent a value.
			name: "devel placeholder with no vcs data is unknown",
			info: bi("(devel)", nil),
			ok:   true,

			wantSource: SourceUnknown,
		},
		{
			name: "build info unavailable is unknown",
			info: nil,
			ok:   false,

			wantSource: SourceUnknown,
		},
		{
			name: "nil build info with ok=true is unknown",
			info: nil,
			ok:   true,

			wantSource: SourceUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolve(tc.ldVersion, tc.ldCommit, tc.ldDate, tc.info, tc.ok)

			if got.Version != tc.wantVersion {
				t.Errorf("Version: got %q want %q", got.Version, tc.wantVersion)
			}
			if got.Revision != tc.wantRevision {
				t.Errorf("Revision: got %q want %q", got.Revision, tc.wantRevision)
			}
			if got.Time != tc.wantTime {
				t.Errorf("Time: got %q want %q", got.Time, tc.wantTime)
			}
			if got.Modified != tc.wantModified {
				t.Errorf("Modified: got %v want %v", got.Modified, tc.wantModified)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source: got %v want %v", got.Source, tc.wantSource)
			}
		})
	}
}

// TestResolveNeverReportsAPlaceholder is the requirement stated directly: no
// input may produce a version a reader could mistake for a real release.
func TestResolveNeverReportsAPlaceholder(t *testing.T) {
	t.Parallel()

	forbidden := []string{"dev", "(devel)", "none", "unknown"}

	inputs := []struct {
		ldVersion, ldCommit, ldDate string
		info                        *debug.BuildInfo
		ok                          bool
	}{
		{"dev", "none", "unknown", bi("(devel)", nil), true},
		{"dev", "none", "unknown", nil, false},
		{"", "", "", bi("(devel)", map[string]string{"vcs.revision": rev}), true},
		{"dev", "none", "unknown", bi("", nil), true},
	}

	for _, in := range inputs {
		got := resolve(in.ldVersion, in.ldCommit, in.ldDate, in.info, in.ok)
		for _, bad := range forbidden {
			if got.Version == bad {
				t.Errorf("Version resolved to placeholder %q for input %+v", bad, in)
			}
			if got.Revision == bad {
				t.Errorf("Revision resolved to placeholder %q for input %+v", bad, in)
			}
			if got.Time == bad {
				t.Errorf("Time resolved to placeholder %q for input %+v", bad, in)
			}
		}
	}
}

func TestKnown(t *testing.T) {
	t.Parallel()

	if (Info{Version: release}).Known() != true {
		t.Error("Known(): a resolved version must report known")
	}
	if (Info{}).Known() != false {
		t.Error("Known(): an empty version must report unknown")
	}
}

func TestSourceString(t *testing.T) {
	t.Parallel()

	tests := map[Source]string{
		SourceLDFlags:   "release build",
		SourceBuildInfo: "build metadata",
		SourceUnknown:   "unavailable",
		Source(99):      "unavailable",
	}
	for src, want := range tests {
		if got := src.String(); got != want {
			t.Errorf("Source(%d).String(): got %q want %q", src, got, want)
		}
	}
}

// TestResolveReadsTheRunningBinary exercises the exported wrapper. It cannot
// assert specific values -- they depend on how the test binary was built -- but
// it must never panic and must never return a placeholder.
func TestResolveReadsTheRunningBinary(t *testing.T) {
	t.Parallel()

	got := Resolve("dev", "none", "unknown")
	if got.Version == "dev" || got.Version == "(devel)" {
		t.Errorf("Resolve leaked a placeholder version: %q", got.Version)
	}
}
