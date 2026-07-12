// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

func renderDetailView(m tuiModel) string {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return renderListView(m)
	}
	r := list[m.cursor]

	var b strings.Builder
	b.WriteString(titleStyle.Render("Repository: " + r.RepoID))
	b.WriteByte('\n')
	b.WriteString(renderDivider([]int{m.width - 1}))
	b.WriteByte('\n')

	field := func(label, value string) {
		fmt.Fprintf(&b, "  %-18s %s\n", label+":", value)
	}

	field("Path", r.Path)
	field("Type", repoType(r))
	field("Branch", colValueBranch(r))
	field("Status", colValueStatus(r))
	field("Delta", deltaOrDash(r))
	field("Dirty", dirtyDisplay(r))
	field("Error", errorDisplay(r))
	field("Upstream", upstreamDisplay(r))
	b.WriteByte('\n')

	if len(r.Remotes) > 0 {
		b.WriteString(headerStyle.Render("Remotes"))
		b.WriteByte('\n')
		for _, rem := range r.Remotes {
			fmt.Fprintf(&b, "  %-10s %s\n", rem.Name, rem.URL)
		}
		b.WriteByte('\n')
	}

	if r.RemoteTrackingRefs.StaleCount > 0 || r.RemoteTrackingRefs.InspectionError != "" {
		b.WriteString(headerStyle.Render("Remote-tracking refs"))
		b.WriteByte('\n')
		staleDisplay := fmt.Sprintf("%d", r.RemoteTrackingRefs.StaleCount)
		if r.RemoteTrackingRefs.InspectionError != "" {
			// Inspection failed, so StaleCount is an unreliable zero: render an
			// unknown marker to match the "?" the status table shows, so
			// operators don't read "couldn't inspect" as "no stale refs".
			staleDisplay = "?"
		}
		fmt.Fprintf(&b, "  Stale: %s\n", staleDisplay)
		for _, ref := range r.RemoteTrackingRefs.Stale {
			fmt.Fprintf(&b, "  - %s\n", sanitizeMetadataText(ref))
		}
		if r.RemoteTrackingRefs.InspectionError != "" {
			fmt.Fprintf(&b, "  Error: %s\n", sanitizeMetadataText(r.RemoteTrackingRefs.InspectionError))
		}
		b.WriteByte('\n')
	}

	if len(r.Labels) > 0 {
		b.WriteString(headerStyle.Render("Local Labels"))
		b.WriteByte('\n')
		for _, k := range sortedKeys(r.Labels) {
			v := r.Labels[k]
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
		b.WriteByte('\n')
	}

	if len(r.Annotations) > 0 {
		b.WriteString(headerStyle.Render("Annotations"))
		b.WriteByte('\n')
		for _, k := range sortedKeys(r.Annotations) {
			v := r.Annotations[k]
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
		b.WriteByte('\n')
	}

	if r.RepoMetadataFile != "" || r.RepoMetadata != nil || r.RepoMetadataError != "" {
		b.WriteString(headerStyle.Render("Repo Metadata"))
		b.WriteByte('\n')
		if r.RepoMetadataFile != "" {
			fmt.Fprintf(&b, "  File: %s\n", r.RepoMetadataFile)
		}
		if r.RepoMetadata != nil {
			if r.RepoMetadata.Name != "" {
				fmt.Fprintf(&b, "  Name: %s\n", sanitizeMetadataText(r.RepoMetadata.Name))
			}
			if r.RepoMetadata.RepoID != "" {
				fmt.Fprintf(&b, "  Repo ID: %s\n", sanitizeMetadataText(r.RepoMetadata.RepoID))
			}
			if len(r.RepoMetadata.Labels) > 0 {
				b.WriteString("  Shared Labels:\n")
				for _, k := range sortedKeys(r.RepoMetadata.Labels) {
					v := r.RepoMetadata.Labels[k]
					fmt.Fprintf(&b, "    %s=%s\n", sanitizeMetadataText(k), sanitizeMetadataText(v))
				}
			}
			if len(r.RepoMetadata.Entrypoints) > 0 {
				b.WriteString("  Entrypoints:\n")
				for _, k := range sortedKeys(r.RepoMetadata.Entrypoints) {
					v := r.RepoMetadata.Entrypoints[k]
					fmt.Fprintf(&b, "    %s=%s\n", sanitizeMetadataText(k), sanitizeMetadataText(v))
				}
			}
			if len(r.RepoMetadata.Paths.Authoritative) > 0 {
				fmt.Fprintf(&b, "  Authoritative: %s\n", strings.Join(sanitizeMetadataSlice(r.RepoMetadata.Paths.Authoritative), ", "))
			}
			if len(r.RepoMetadata.Paths.LowValue) > 0 {
				fmt.Fprintf(&b, "  Low value: %s\n", strings.Join(sanitizeMetadataSlice(r.RepoMetadata.Paths.LowValue), ", "))
			}
			if len(r.RepoMetadata.Provides) > 0 {
				fmt.Fprintf(&b, "  Provides: %s\n", strings.Join(sanitizeMetadataSlice(r.RepoMetadata.Provides), ", "))
			}
			if len(r.RepoMetadata.RelatedRepos) > 0 {
				b.WriteString("  Related repos:\n")
				for _, related := range r.RepoMetadata.RelatedRepos {
					repoID := sanitizeMetadataText(related.RepoID)
					relationship := sanitizeMetadataText(related.Relationship)
					if relationship == "" {
						fmt.Fprintf(&b, "    %s\n", repoID)
						continue
					}
					fmt.Fprintf(&b, "    %s (%s)\n", repoID, relationship)
				}
			}
		}
		if r.RepoMetadataError != "" {
			fmt.Fprintf(&b, "  Error: %s\n", sanitizeMetadataText(r.RepoMetadataError))
		}
		b.WriteByte('\n')
	}

	if r.LastSync != nil {
		b.WriteString(headerStyle.Render("Last Sync"))
		b.WriteByte('\n')
		ok := "✓"
		if !r.LastSync.OK {
			ok = "✗"
		}
		fmt.Fprintf(&b, "  %s  %s\n", ok, relativeTime(r.LastSync.At))
		if r.LastSync.Error != "" {
			fmt.Fprintf(&b, "  Error: %s\n", r.LastSync.Error)
		}
		b.WriteByte('\n')
	}

	b.WriteString(statusBarStyle.Render("esc/q: back  l: edit labels  i: repo metadata"))
	return b.String()
}

// sanitizeMetadataText strips control characters and whole ANSI/terminal
// escape sequences from repo-controlled metadata before it is written to
// the terminal. RepoMetadata is loaded from a file inside the cloned
// repository (repometa.Load), so a hostile or compromised upstream could
// otherwise smuggle escape sequences into the operator's terminal via Name,
// labels, entrypoints, and similar fields. Bare stripping of control runes
// is not enough: everything after the leading ESC byte in an escape
// sequence (e.g. "[31m") is ordinary printable text, so the sequence's
// effect survives unless the whole thing is dropped.
//
// This also covers the C1 controls (U+0080..U+009F). In UTF-8 those encode as
// 0xC2 followed by 0x80..0x9F, whose bytes are both >= 0x20 and so would
// otherwise pass straight through — yet many terminals honor them as control
// codes (U+009B is a single-byte CSI introducer, U+009D an OSC introducer), so
// they are just as dangerous as their ESC-prefixed C0 equivalents.
func sanitizeMetadataText(s string) string {
	if s == "" {
		return s
	}
	src := []byte(s)
	var out strings.Builder
	out.Grow(len(src))
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == 0x1b: // ESC: drop the entire escape sequence
			i = skipEscapeSequence(src, i)
		case c == 0xc2 && i+1 < len(src) && src[i+1] >= 0x80 && src[i+1] <= 0x9f:
			// UTF-8-encoded C1 control: drop it, and for the CSI/OSC introducers
			// the whole sequence they start.
			i = skipC1Sequence(src, i)
		case c < 0x20 || c == 0x7f: // bare C0 control byte / DEL
			continue
		default:
			out.WriteByte(c)
		}
	}
	return out.String()
}

// skipC1Sequence returns the index of the last byte belonging to a UTF-8
// encoded C1 control that starts at src[start] (0xC2, with the C1 byte at
// start+1). U+009B (CSI) and U+009D (OSC) introduce full control sequences,
// handled like their ESC '[' / ESC ']' equivalents (OSC may be terminated by a
// C1 ST, U+009C, as well as BEL or ESC '\'); any other C1 control is just the
// two-byte encoding itself.
func skipC1Sequence(src []byte, start int) int {
	switch src[start+1] {
	case 0x9b: // CSI
		j := start + 2
		for j < len(src) && src[j] >= 0x20 && src[j] <= 0x3f {
			j++
		}
		if j < len(src) {
			return j
		}
		return j - 1
	case 0x9d: // OSC
		j := start + 2
		for j < len(src) {
			if src[j] == 0x07 {
				return j
			}
			if src[j] == 0x1b && j+1 < len(src) && src[j+1] == '\\' {
				return j + 1
			}
			if src[j] == 0xc2 && j+1 < len(src) && src[j+1] == 0x9c { // C1 ST
				return j + 1
			}
			j++
		}
		return j - 1
	default:
		return start + 1
	}
}

// skipEscapeSequence returns the index of the last byte belonging to the
// escape sequence that starts with ESC at src[start], so the caller can
// resume scanning right after it. It recognizes CSI sequences (ESC '['
// parameter/intermediate bytes, final byte) and OSC sequences (ESC ']' ...
// terminated by BEL or ESC '\'); any other byte following ESC is treated as
// a minimal two-byte sequence.
func skipEscapeSequence(src []byte, start int) int {
	if start+1 >= len(src) {
		return start
	}
	switch src[start+1] {
	case '[':
		j := start + 2
		for j < len(src) && src[j] >= 0x20 && src[j] <= 0x3f {
			j++
		}
		if j < len(src) {
			return j
		}
		return j - 1
	case ']':
		j := start + 2
		for j < len(src) {
			if src[j] == 0x07 {
				return j
			}
			if src[j] == 0x1b && j+1 < len(src) && src[j+1] == '\\' {
				return j + 1
			}
			j++
		}
		return j - 1
	default:
		return start + 1
	}
}

func sanitizeMetadataSlice(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = sanitizeMetadataText(v)
	}
	return out
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func repoType(r model.RepoStatus) string {
	if r.Type == "" {
		return "checkout"
	}
	return r.Type
}

func deltaOrDash(r model.RepoStatus) string {
	v := colValueDelta(r)
	if v == "" {
		return "-"
	}
	return v
}

func dirtyDisplay(r model.RepoStatus) string {
	if r.Bare {
		return "bare (no worktree)"
	}
	if r.Worktree == nil {
		return "-"
	}
	if r.Worktree.Dirty {
		return fmt.Sprintf("yes (staged:%d unstaged:%d untracked:%d)",
			r.Worktree.Staged, r.Worktree.Unstaged, r.Worktree.Untracked)
	}
	return "clean"
}

func errorDisplay(r model.RepoStatus) string {
	if r.Error == "" {
		return "-"
	}
	if r.ErrorClass != "" {
		return fmt.Sprintf("[%s] %s", r.ErrorClass, r.Error)
	}
	return r.Error
}

func upstreamDisplay(r model.RepoStatus) string {
	if r.Tracking.Upstream == "" {
		return "-"
	}
	return r.Tracking.Upstream
}
