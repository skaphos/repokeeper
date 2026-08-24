// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var portableInheritedEnvironment = []string{"PATH"}
var windowsInheritedEnvironment = []string{"SystemRoot", "ComSpec", "PATHEXT", "WINDIR"}

// buildChildEnvironment returns a complete allowlisted environment. It never
// mutates the process environment, so concurrent scenarios cannot leak state.
func buildChildEnvironment(root, configPath string) ([]string, error) {
	paths := map[string]string{
		"home":   filepath.Join(root, "home"),
		"cache":  filepath.Join(root, "cache"),
		"tmp":    filepath.Join(root, "tmp"),
		"config": filepath.Join(root, "config"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return nil, fmt.Errorf("create isolated environment path %s: %w", path, err)
		}
	}

	gitConfig := filepath.Join(paths["config"], "gitconfig")
	gitConfigBody := []byte("[user]\n\tname = RepoKeeper E2E\n\temail = e2e@invalid.example\n[commit]\n\tgpgSign = false\n[tag]\n\tgpgSign = false\n[credential]\n\thelper =\n[init]\n\tdefaultBranch = main\n[protocol \"file\"]\n\tallow = always\n")
	if err := os.WriteFile(gitConfig, gitConfigBody, 0o600); err != nil {
		return nil, fmt.Errorf("write isolated git config: %w", err)
	}

	env := make(map[string]string)
	for _, key := range portableInheritedEnvironment {
		copyHostEnvironment(env, key)
	}
	if runtime.GOOS == "windows" {
		for _, key := range windowsInheritedEnvironment {
			copyHostEnvironment(env, key)
		}
	}

	values := map[string]string{
		"HOME":                paths["home"],
		"USERPROFILE":         paths["home"],
		"XDG_CONFIG_HOME":     paths["config"],
		"XDG_CACHE_HOME":      paths["cache"],
		"APPDATA":             paths["config"],
		"LOCALAPPDATA":        paths["cache"],
		"TMPDIR":              paths["tmp"],
		"TMP":                 paths["tmp"],
		"TEMP":                paths["tmp"],
		"REPOKEEPER_CONFIG":   configPath,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_GLOBAL":   gitConfig,
		"GIT_TERMINAL_PROMPT": "0",
		"GIT_ASKPASS":         executableNullDevice(),
		"GIT_SSH_COMMAND":     "ssh -oBatchMode=yes",
		"GIT_ALLOW_PROTOCOL":  "file",
		"GIT_AUTHOR_NAME":     "RepoKeeper E2E",
		"GIT_AUTHOR_EMAIL":    "e2e@invalid.example",
		"GIT_COMMITTER_NAME":  "RepoKeeper E2E",
		"GIT_COMMITTER_EMAIL": "e2e@invalid.example",
		"GIT_AUTHOR_DATE":     "2024-01-02T03:04:05Z",
		"GIT_COMMITTER_DATE":  "2024-01-02T03:04:05Z",
		"TZ":                  "UTC",
		"LC_ALL":              "C",
		"LANG":                "C",
		"NO_COLOR":            "1",
		"CLICOLOR":            "0",
		"GCM_INTERACTIVE":     "Never",
		"GIT_CONFIG_COUNT":    "2",
		"GIT_CONFIG_KEY_0":    "credential.helper",
		"GIT_CONFIG_VALUE_0":  "",
		"GIT_CONFIG_KEY_1":    "commit.gpgSign",
		"GIT_CONFIG_VALUE_1":  "false",
	}
	for key, value := range values {
		setEnvironmentValue(env, key, value)
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return strings.ToUpper(keys[i]) < strings.ToUpper(keys[j]) })
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result, nil
}

func copyHostEnvironment(target map[string]string, key string) {
	if value, ok := lookupEnvironmentFold(os.Environ(), key); ok {
		setEnvironmentValue(target, key, value)
	}
}

func setEnvironmentValue(target map[string]string, key, value string) {
	for existing := range target {
		if strings.EqualFold(existing, key) {
			delete(target, existing)
		}
	}
	target[key] = value
}

func lookupEnvironmentFold(environment []string, key string) (string, bool) {
	for _, item := range environment {
		name, value, found := strings.Cut(item, "=")
		if found && strings.EqualFold(name, key) {
			return value, true
		}
	}
	return "", false
}

func executableNullDevice() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/bin/false"
}

func environmentMap(environment []string) map[string]string {
	result := make(map[string]string, len(environment))
	for _, item := range environment {
		key, value, found := strings.Cut(item, "=")
		if found {
			result[strings.ToUpper(key)] = value
		}
	}
	return result
}
