// SPDX-License-Identifier: MIT
package gitx

import (
	"strconv"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

// ParsePorcelainStatus parses the output of `git status --porcelain=v1`
// into a Worktree struct.
func ParsePorcelainStatus(output string) *model.Worktree {
	wt := &model.Worktree{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]

		if x == '?' && y == '?' {
			wt.Untracked++
			continue
		}
		if x != ' ' && x != '?' {
			wt.Staged++
		}
		if y != ' ' && y != '?' {
			wt.Unstaged++
		}
	}
	wt.Dirty = wt.Staged > 0 || wt.Unstaged > 0 || wt.Untracked > 0
	return wt
}

// ForEachRefEntry represents a single line from git for-each-ref output.
type ForEachRefEntry struct {
	Branch     string
	Upstream   string
	Track      string // e.g. "[ahead 2]", "[behind 1]", "[gone]", ""
	TrackShort string // e.g. ">", "<", "<>", "="
}

// ParseForEachRef parses the pipe-delimited output of:
//
//	git for-each-ref refs/heads --format="%(refname:short)|%(upstream:short)|%(upstream:track)|%(upstream:trackshort)"
//
// KNOWN LIMITATION (deferred): "|" is technically a
// legal character in a git ref name (git-check-ref-format does not
// disallow it), so a branch such as "feat|x" can be misparsed here. The
// correct fix — switching the --format string to a NUL ("%00") delimiter,
// which cannot appear in a ref name — was prototyped but reverted because
// it changes the exact argv git is invoked with, which breaks
// internal/engine's mock-runner test fixtures that hardcode the current
// pipe-delimited format string. internal/engine is outside this change's
// file scope, so that fix needs a coordinated follow-up that updates both
// packages together.
func ParseForEachRef(output string) []ForEachRefEntry {
	if output == "" {
		return nil
	}
	var entries []ForEachRefEntry
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		entry := ForEachRefEntry{}
		if len(parts) > 0 {
			entry.Branch = parts[0]
		}
		if len(parts) > 1 {
			entry.Upstream = parts[1]
		}
		if len(parts) > 2 {
			entry.Track = parts[2]
		}
		if len(parts) > 3 {
			entry.TrackShort = parts[3]
		}
		entries = append(entries, entry)
	}
	return entries
}

// ParseRevListCount parses the output of:
//
//	git rev-list --left-right --count <branch>...@{upstream}
//
// Returns (ahead, behind).
func ParseRevListCount(output string) (int, int) {
	output = strings.TrimSpace(output)
	if output == "" {
		return 0, 0
	}
	parts := strings.SplitN(output, "\t", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	ahead, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	behind, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return ahead, behind
}
