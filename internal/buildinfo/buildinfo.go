// SPDX-License-Identifier: MIT

// Package buildinfo resolves what a running binary knows about itself.
//
// Three sources, in precedence order: values stamped at release time by
// GoReleaser's ldflags, then the metadata the Go toolchain records in the
// binary, then nothing. The middle tier is what makes `go install` produce a
// version-stamped binary; before it existed, any build that was not
// GoReleaser's reported "dev".
//
// The tiers are complementary rather than redundant. A module-proxy install
// (`go install ...@latest`) records a real module version but no VCS
// information; a local `go build` records the reverse. Neither is a superset of
// the other, so both are consulted.
//
// Two placeholder values must never reach a user: "dev", the historical ldflags
// default, and "(devel)", what the module system records for a build that is
// not from a released module version. Surfacing either is the defect this
// package exists to fix, so both are mapped back to "not recorded".
package buildinfo

import "runtime/debug"

// Source records where the resolved information came from, so callers can tell
// an authoritative version from an inferred one. Reporting a version without
// saying whether the release pipeline asserted it or the module proxy implied
// it would collapse two different levels of confidence.
type Source int

const (
	// SourceUnknown means nothing usable was recorded. Callers must not
	// substitute a placeholder that could be mistaken for a real version.
	SourceUnknown Source = iota
	// SourceBuildInfo means the values came from the Go toolchain's own
	// build metadata.
	SourceBuildInfo
	// SourceLDFlags means the values were stamped in at release time.
	SourceLDFlags
)

// String renders the source for human-readable output.
func (s Source) String() string {
	switch s {
	case SourceLDFlags:
		return "release build"
	case SourceBuildInfo:
		return "build metadata"
	default:
		return "unavailable"
	}
}

// Info is what a binary knows about itself. Every string field is best-effort:
// an empty string means "not recorded", never "zero".
type Info struct {
	Version  string // e.g. "v0.8.0"; empty when unknown
	Revision string // full VCS revision; empty when unknown
	Time     string // RFC3339 build time; empty when unknown
	Modified bool   // built from a working tree with uncommitted changes
	Source   Source
}

// Known reports whether a usable version string was resolved.
func (i Info) Known() bool { return i.Version != "" }

// devSentinel is the historical default of the ldflags Version variable. A
// binary carrying it was not stamped, so it must fall through to build
// metadata rather than reporting "dev" as though it were a version.
const devSentinel = "dev"

// develVersion is what the module system records for a build that is not from
// a released module version. It is a placeholder, not a version.
const develVersion = "(devel)"

// commitSentinel and dateSentinel are the historical defaults of the ldflags
// Commit and Date variables. They mean "not recorded".
const (
	commitSentinel = "none"
	dateSentinel   = "unknown"
)

// Resolve determines the running binary's identity. The ld* arguments are the
// release-time stamped values; empty or sentinel values mean "not stamped".
func Resolve(ldVersion, ldCommit, ldDate string) Info {
	bi, ok := debug.ReadBuildInfo()
	return resolve(ldVersion, ldCommit, ldDate, bi, ok)
}

// resolve is the testable core. It is separated from Resolve because
// debug.ReadBuildInfo reads the running binary and cannot be faked.
func resolve(ldVersion, ldCommit, ldDate string, bi *debug.BuildInfo, ok bool) Info {
	// Tier 1: release-time stamping wins outright, so released binaries
	// report exactly what the pipeline put in them.
	if stamped(ldVersion) {
		return Info{
			Version:  ldVersion,
			Revision: cleanSentinel(ldCommit, commitSentinel),
			Time:     cleanSentinel(ldDate, dateSentinel),
			Source:   SourceLDFlags,
		}
	}

	if !ok || bi == nil {
		return Info{Source: SourceUnknown}
	}

	revision, buildTime, modified := vcsSettings(bi)

	// Tier 2: a real module version, as recorded for `go install module@version`.
	if v := bi.Main.Version; v != "" && v != develVersion {
		return Info{
			Version:  v,
			Revision: revision,
			Time:     buildTime,
			Modified: modified,
			Source:   SourceBuildInfo,
		}
	}

	// Tier 3: no module version, but the build was stamped from a VCS tree.
	// Report the revision honestly and leave Version empty -- "(devel)" is a
	// placeholder and surfacing it is the same defect as surfacing "dev".
	if revision != "" {
		return Info{
			Revision: revision,
			Time:     buildTime,
			Modified: modified,
			Source:   SourceBuildInfo,
		}
	}

	// Tier 4: nothing usable.
	return Info{Source: SourceUnknown}
}

// stamped reports whether an ldflags value carries real information.
func stamped(v string) bool { return v != "" && v != devSentinel }

// cleanSentinel maps a historical placeholder default back to "not recorded".
func cleanSentinel(v, sentinel string) string {
	if v == sentinel {
		return ""
	}
	return v
}

// vcsSettings extracts the VCS stamps the toolchain records when building from
// a source tree. All three are absent for module-proxy builds.
func vcsSettings(bi *debug.BuildInfo) (revision, buildTime string, modified bool) {
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			buildTime = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return revision, buildTime, modified
}
