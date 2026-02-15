package repokeeper

import (
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/strutil"
)

func resolveRepoFilter(only, fieldSelector string) (engine.FilterKind, error) {
	onlyTrimmed := strings.ToLower(strings.TrimSpace(only))
	if onlyTrimmed == "" {
		onlyTrimmed = string(engine.FilterAll)
	}

	selectorTrimmed := strings.TrimSpace(fieldSelector)
	if selectorTrimmed == "" {
		return engine.FilterKind(onlyTrimmed), nil
	}
	if onlyTrimmed != string(engine.FilterAll) {
		return "", fmt.Errorf("--field-selector cannot be combined with --only=%q", onlyTrimmed)
	}
	return parseFieldSelectorFilter(selectorTrimmed)
}

func parseFieldSelectorFilter(fieldSelector string) (engine.FilterKind, error) {
	parts := strutil.SplitCSV(fieldSelector)
	if len(parts) != 1 {
		return "", fmt.Errorf("only a single field selector is currently supported")
	}
	expr := strings.TrimSpace(parts[0])
	if expr == "" {
		return engine.FilterAll, nil
	}
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
		default:
			return "", fmt.Errorf("unsupported tracking.status value %q", value)
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
