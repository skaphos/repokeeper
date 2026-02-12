package gitx

import (
	"net/url"
	"sort"
	"strings"
)

// NormalizeURL converts a git remote URL into a canonical repo_id.
//
// Rules:
//   - Strip protocol (https://, git://, ssh://) and user (git@)
//   - Convert git@host:path to host/path
//   - Lowercase the host portion
//   - Strip trailing ".git"
//   - Strip trailing slashes
//
// Examples:
//
//	git@github.com:Org/Repo.git  → github.com/Org/Repo
//	https://github.com/Org/Repo.git → github.com/Org/Repo
func NormalizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	var host, path string

	// Handle SSH shorthand: git@host:path
	if i := strings.Index(rawURL, "@"); i >= 0 && !strings.Contains(rawURL[:i], "://") {
		// SSH shorthand like git@github.com:Org/Repo.git
		rest := rawURL[i+1:]
		if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
			host = rest[:colonIdx]
			path = rest[colonIdx+1:]
		}
	} else {
		// URL with protocol
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return rawURL
		}
		host = parsed.Hostname()
		path = strings.TrimPrefix(parsed.Path, "/")
	}

	host = strings.ToLower(host)
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimRight(path, "/")

	if host == "" {
		return path
	}
	return host + "/" + path
}

// PrimaryRemote selects the preferred remote from a list.
// Prefers "origin", falls back to first alphabetically.
func PrimaryRemote(remoteNames []string) string {
	if len(remoteNames) == 0 {
		return ""
	}
	for _, name := range remoteNames {
		if name == "origin" {
			return "origin"
		}
	}
	sorted := make([]string, len(remoteNames))
	copy(sorted, remoteNames)
	sort.Strings(sorted)
	return sorted[0]
}
