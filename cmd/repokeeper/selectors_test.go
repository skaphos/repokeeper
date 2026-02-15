package repokeeper

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
)

func TestResolveRepoFilter(t *testing.T) {
	tests := []struct {
		name          string
		only          string
		fieldSelector string
		want          engine.FilterKind
		wantErr       bool
	}{
		{name: "only all default", only: "all", fieldSelector: "", want: engine.FilterAll},
		{name: "only dirty", only: "dirty", fieldSelector: "", want: engine.FilterDirty},
		{name: "field selector diverged", only: "all", fieldSelector: "tracking.status=diverged", want: engine.FilterDiverged},
		{name: "field selector missing", only: "", fieldSelector: "repo.missing=true", want: engine.FilterMissing},
		{name: "field selector dirty false", only: "", fieldSelector: "worktree.dirty=false", want: engine.FilterClean},
		{name: "field selector error", only: "", fieldSelector: "repo.error=true", want: engine.FilterErrors},
		{name: "field selector remote mismatch", only: "", fieldSelector: "remote.mismatch=true", want: engine.FilterRemoteMismatch},
		{name: "field selector gone", only: "", fieldSelector: "tracking.status=gone", want: engine.FilterGone},
		{name: "reject mixed only and selector", only: "dirty", fieldSelector: "tracking.status=gone", wantErr: true},
		{name: "reject unknown key", only: "all", fieldSelector: "repo.name=foo", wantErr: true},
		{name: "reject unsupported value", only: "all", fieldSelector: "tracking.status=equal", wantErr: true},
		{name: "reject multi selector", only: "all", fieldSelector: "tracking.status=gone,repo.error=true", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRepoFilter(tc.only, tc.fieldSelector)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got filter %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveRepoFilter() = %q, want %q", got, tc.want)
			}
		})
	}
}
