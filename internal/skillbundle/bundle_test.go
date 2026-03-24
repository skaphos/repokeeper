// SPDX-License-Identifier: MIT
package skillbundle

import (
	"strings"
	"testing"
)

func TestRepoKeeperSkillContainsEmbeddedContent(t *testing.T) {
	t.Parallel()
	skill := RepoKeeperSkill()
	for _, want := range []string{"name: repokeeper", "# RepoKeeper", "compatibility: opencode", "## Core rules"} {
		if !strings.Contains(skill, want) {
			t.Fatalf("expected embedded skill to contain %q", want)
		}
	}
}
