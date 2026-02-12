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

// ClassifyError categorizes a git command error into a known error type.
type ErrorClass string

const (
	ErrAuth     ErrorClass = "auth"
	ErrNetwork  ErrorClass = "network"
	ErrNoRemote ErrorClass = "no_remote"
	ErrCorrupt  ErrorClass = "corrupt"
	ErrNotARepo ErrorClass = "not_a_repo"
	ErrTimeout  ErrorClass = "timeout"
	ErrUnknown  ErrorClass = "unknown"
)

// ClassifyGitError inspects the error output and returns a classification.
func ClassifyGitError(stderr string) ErrorClass {
	lower := strings.ToLower(stderr)

	switch {
	case strings.Contains(lower, "authentication failed"),
		strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "could not read from remote"),
		strings.Contains(lower, "invalid credentials"):
		return ErrAuth

	case strings.Contains(lower, "could not resolve host"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "network is unreachable"),
		strings.Contains(lower, "connection timed out"),
		strings.Contains(lower, "unable to access"),
		strings.Contains(lower, "unable to connect"):
		return ErrNetwork

	case strings.Contains(lower, "no remote repository"),
		strings.Contains(lower, "no such remote"):
		return ErrNoRemote

	case strings.Contains(lower, "object file is empty"),
		strings.Contains(lower, "loose object"),
		strings.Contains(lower, "corrupt"),
		strings.Contains(lower, "bad object"):
		return ErrCorrupt

	case strings.Contains(lower, "not a git repository"):
		return ErrNotARepo

	case strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "timed out"),
		strings.Contains(lower, "timeout"):
		return ErrTimeout

	default:
		return ErrUnknown
	}
}
