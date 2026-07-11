// SPDX-License-Identifier: MIT
package selector

import (
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/strutil"
)

// knownOnlyFilterKinds is the closed set of values accepted by --only. Keep in
// sync with repoFilterUsage in cmd/repokeeper/flags.go and the engine.FilterKind
// constants: an unrecognized value must be rejected rather than silently
// falling through to "match everything".
var knownOnlyFilterKinds = map[engine.FilterKind]struct{}{
	engine.FilterAll:            {},
	engine.FilterErrors:         {},
	engine.FilterDirty:          {},
	engine.FilterClean:          {},
	engine.FilterGone:           {},
	engine.FilterDiverged:       {},
	engine.FilterBehind:         {},
	engine.FilterAhead:          {},
	engine.FilterEqual:          {},
	engine.FilterRemoteMismatch: {},
	engine.FilterMissing:        {},
}

// ResolveRepoFilter combines --only and --field-selector into a single FilterKind.
// If fieldSelector is non-empty, only must be "all" (or empty).
func ResolveRepoFilter(only, fieldSelector string) (engine.FilterKind, error) {
	onlyTrimmed := strings.ToLower(strings.TrimSpace(only))
	if onlyTrimmed == "" {
		onlyTrimmed = string(engine.FilterAll)
	}

	selectorTrimmed := strings.TrimSpace(fieldSelector)
	if selectorTrimmed == "" {
		if fieldSelector != "" {
			return "", fmt.Errorf("--field-selector cannot be blank")
		}
		kind := engine.FilterKind(onlyTrimmed)
		if _, ok := knownOnlyFilterKinds[kind]; !ok {
			return "", fmt.Errorf("unsupported --only value %q (expected one of: all, errors, dirty, clean, gone, diverged, behind, ahead, equal, remote-mismatch, missing)", only)
		}
		return kind, nil
	}
	if len(strutil.SplitCSV(fieldSelector)) == 0 {
		return "", fmt.Errorf("--field-selector cannot be blank")
	}
	if onlyTrimmed != string(engine.FilterAll) {
		return "", fmt.Errorf("--field-selector cannot be combined with --only=%q", onlyTrimmed)
	}
	return ParseFieldSelectorFilter(selectorTrimmed)
}

// ParseFieldSelectorFilter parses a single field selector expression into a FilterKind.
// Only one expression is currently supported.
func ParseFieldSelectorFilter(fieldSelector string) (engine.FilterKind, error) {
	if strings.TrimSpace(fieldSelector) == "" {
		return "", fmt.Errorf("--field-selector cannot be blank")
	}
	parts := strutil.SplitCSV(fieldSelector)
	if len(parts) == 0 {
		return "", fmt.Errorf("--field-selector cannot be blank")
	}
	if len(parts) != 1 {
		return "", fmt.Errorf("only a single field selector is currently supported")
	}
	expr := strings.TrimSpace(parts[0])
	tokens := strings.SplitN(expr, "=", 2)
	if len(tokens) != 2 {
		return "", fmt.Errorf("invalid --field-selector expression %q (expected key=value)", expr)
	}
	key := strings.ToLower(strings.TrimSpace(tokens[0]))
	value := strings.ToLower(strings.TrimSpace(tokens[1]))

	switch key {
	case "tracking.status":
		switch value {
		case "all":
			return engine.FilterAll, nil
		case "gone":
			return engine.FilterGone, nil
		case "diverged":
			return engine.FilterDiverged, nil
		case "behind":
			return engine.FilterBehind, nil
		case "ahead":
			return engine.FilterAhead, nil
		case "equal":
			return engine.FilterEqual, nil
		default:
			return "", fmt.Errorf("unsupported tracking.status value %q (expected one of: all, gone, diverged, behind, ahead, equal)", value)
		}
	case "worktree.dirty":
		switch value {
		case "true":
			return engine.FilterDirty, nil
		case "false":
			return engine.FilterClean, nil
		default:
			return "", fmt.Errorf("unsupported worktree.dirty value %q", value)
		}
	case "repo.error":
		if value != "true" {
			return "", fmt.Errorf("unsupported repo.error value %q", value)
		}
		return engine.FilterErrors, nil
	case "repo.missing":
		if value != "true" {
			return "", fmt.Errorf("unsupported repo.missing value %q", value)
		}
		return engine.FilterMissing, nil
	case "remote.mismatch":
		if value != "true" {
			return "", fmt.Errorf("unsupported remote.mismatch value %q", value)
		}
		return engine.FilterRemoteMismatch, nil
	default:
		return "", fmt.Errorf("unsupported --field-selector key %q", key)
	}
}
