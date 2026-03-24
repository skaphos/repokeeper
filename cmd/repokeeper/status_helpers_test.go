// SPDX-License-Identifier: MIT
package repokeeper

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestRelatedReposString(t *testing.T) {
	t.Parallel()

	if got := relatedReposString(nil); got != "-" {
		t.Fatalf("expected dash for empty related repos, got %q", got)
	}

	got := relatedReposString([]model.RepoMetadataRelatedRepo{{RepoID: "z/repo", Relationship: "depends-on"}, {RepoID: "a/repo"}})
	if got != "a/repo,z/repo:depends-on" {
		t.Fatalf("expected sorted related repos string, got %q", got)
	}
}
