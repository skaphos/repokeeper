// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/skaphos/repokeeper/internal/buildinfo"
	"github.com/spf13/cobra"
)

// Set via ldflags at build time.
//
// These names are referenced by full path in .goreleaser.yaml's -X flags
// (github.com/skaphos/repokeeper/cmd/repokeeper.Version and friends). Renaming
// or moving them silently breaks release stamping: the build still succeeds and
// the binary falls back to build metadata, so no test catches it.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// unavailable is what every unknown field renders as. An omitted line invites
// the reader to assume the field was not applicable; naming it as unavailable
// says the information was looked for and not found.
const unavailable = "unavailable"

// resolvedBuildInfo returns the running binary's identity. Resolution happens
// here and nowhere else so the version command and the MCP server handshake
// cannot disagree about the same binary.
func resolvedBuildInfo() buildinfo.Info {
	return buildinfo.Resolve(Version, Commit, Date)
}

// versionJSON is the machine-readable contract. Unknown string fields are
// empty rather than omitted or spelled "unavailable": a consumer testing for
// emptiness gets one answer, where matching a literal would mean parsing prose.
type versionJSON struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Built    string `json:"built"`
	Modified bool   `json:"modified"`
	Source   string `json:"source"`
	Go       string `json:"go"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	Long: "Reports the running binary's version, source revision and build time.\n\n" +
		"Release builds report the values stamped in by the release pipeline. Builds\n" +
		"installed from the module proxy or compiled locally report what the Go\n" +
		"toolchain recorded. When nothing was recorded the report says so rather than\n" +
		"substituting a placeholder.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		info := resolvedBuildInfo()

		format, _ := cmd.Flags().GetString("format")
		mode, err := parseOutputMode(format)
		if err != nil {
			return err
		}

		switch mode.kind {
		case outputKindJSON:
			writeVersionJSON(cmd, info)
			return nil
		case outputKindTable, outputKindWide:
			writeVersionText(cmd, info)
			return nil
		default:
			return fmt.Errorf("unsupported output format %q for version: use table, wide or json", format)
		}
	},
}

func writeVersionJSON(cmd *cobra.Command, info buildinfo.Info) {
	payload := versionJSON{
		Version:  info.Version,
		Revision: info.Revision,
		Built:    info.Time,
		Modified: info.Modified,
		Source:   info.Source.String(),
		Go:       runtime.Version(),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		logOutputWriteFailure(cmd, "version json", err)
	}
}

func writeVersionText(cmd *cobra.Command, info buildinfo.Info) {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "repokeeper %s\n", versionHeadline(info)); err != nil {
		logOutputWriteFailure(cmd, "version headline", err)
		return
	}

	fields := [][2]string{
		{"commit", revisionField(info)},
		{"built", orUnavailable(info.Time)},
		{"go", runtime.Version()},
		{"os/arch", runtime.GOOS + "/" + runtime.GOARCH},
	}
	for _, f := range fields {
		if _, err := fmt.Fprintf(out, "  %-8s %s\n", f[0]+":", f[1]); err != nil {
			logOutputWriteFailure(cmd, "version field", err)
			return
		}
	}
}

// versionHeadline renders the first line. A release build reads exactly as it
// did before this command learned about build metadata.
func versionHeadline(info buildinfo.Info) string {
	if info.Source == buildinfo.SourceLDFlags {
		return info.Version
	}

	if !info.Known() {
		if info.Revision == "" {
			return "— version information " + unavailable
		}
		headline := "(devel) — no released version"
		if info.Modified {
			headline += ", modified working tree"
		}
		return headline + " (" + info.Source.String() + ")"
	}

	headline := info.Version
	if info.Modified {
		headline += " — modified working tree"
	}
	return headline + " (" + info.Source.String() + ")"
}

func revisionField(info buildinfo.Info) string {
	if info.Revision == "" {
		return unavailable
	}
	if info.Modified {
		return info.Revision + " (dirty)"
	}
	return info.Revision
}

func orUnavailable(v string) string {
	if v == "" {
		return unavailable
	}
	return v
}

// advertisedVersion is what the MCP server reports during protocol handshake.
// It resolves from the same buildinfo.Info the version command uses, so the two
// can never disagree about the same binary. When no version was recorded it
// degrades to the revision, and then to a plain marker -- never to a value a
// client could mistake for a release.
func advertisedVersion() string {
	info := resolvedBuildInfo()
	switch {
	case info.Known():
		return info.Version
	case info.Revision != "":
		const shortRevLen = 12
		short := info.Revision
		if len(short) > shortRevLen {
			short = short[:shortRevLen]
		}
		if info.Modified {
			return short + "+dirty"
		}
		return short
	default:
		return unavailable
	}
}

func init() {
	addFormatFlag(versionCmd, "output format: table, wide or json (wide renders as table)")
	rootCmd.AddCommand(versionCmd)
}
