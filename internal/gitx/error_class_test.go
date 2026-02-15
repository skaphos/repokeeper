package gitx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/skaphos/repokeeper/internal/gitx"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "timeout", err: context.DeadlineExceeded, want: "timeout"},
		{name: "auth sentinel", err: fmt.Errorf("wrapped: %w", gitx.ErrAuthFailure), want: "auth"},
		{name: "network sentinel", err: fmt.Errorf("wrapped: %w", gitx.ErrNetworkFailure), want: "network"},
		{name: "corrupt sentinel", err: fmt.Errorf("wrapped: %w", gitx.ErrCorruptRepo), want: "corrupt"},
		{name: "missing remote sentinel", err: fmt.Errorf("wrapped: %w", gitx.ErrMissingRemoteRef), want: "missing_remote"},
		{name: "auth", err: errors.New("permission denied (publickey)"), want: "auth"},
		{name: "network", err: errors.New("Could not resolve host: github.com"), want: "network"},
		{name: "corrupt", err: errors.New("fatal: not a git repository"), want: "corrupt"},
		{name: "missing remote", err: errors.New("fatal: couldn't find remote ref main"), want: "missing_remote"},
		{name: "unknown", err: errors.New("something odd"), want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gitx.ClassifyError(tc.err); got != tc.want {
				t.Fatalf("unexpected class: got %q want %q", got, tc.want)
			}
		})
	}
}
