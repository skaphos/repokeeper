// SPDX-License-Identifier: MIT
package repokeeper

import (
	"path/filepath"
	"strings"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
)

type exportBundle struct {
	Version    int                `yaml:"version"`
	ExportedAt string             `yaml:"exported_at"`
	Root       string             `yaml:"root,omitempty"`
	Config     config.Config      `yaml:"config"`
	Registry   *registry.Registry `yaml:"registry,omitempty"`
}

const currentExportBundleVersion = 2

func cloneRegistry(reg *registry.Registry) *registry.Registry {
	if reg == nil {
		return nil
	}
	clone := *reg
	clone.Entries = append([]registry.Entry(nil), reg.Entries...)
	return &clone
}

func normalizePathLikeInput(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, "\\", string(filepath.Separator))
}

func cleanRelativePath(path string) (string, bool) {
	raw := normalizePathLikeInput(path)
	cleaned := filepath.Clean(raw)
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) || isAbsoluteLikePath(raw, cleaned) {
		return "", false
	}
	rel := filepath.ToSlash(cleaned)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func isAbsoluteLikePath(raw, cleaned string) bool {
	if filepath.IsAbs(cleaned) {
		return true
	}
	if strings.HasPrefix(cleaned, string(filepath.Separator)) {
		return true
	}
	trimmedRaw := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmedRaw, `/`) || strings.HasPrefix(trimmedRaw, `\`) {
		return true
	}
	if len(trimmedRaw) >= 3 && trimmedRaw[1] == ':' {
		drive := trimmedRaw[0]
		if (drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z') {
			return trimmedRaw[2] == '/' || trimmedRaw[2] == '\\'
		}
	}
	return false
}
