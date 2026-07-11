// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestGetReconcileRemoteMismatchFlagIsWired guards a regression where
// statusCmd's --reconcile-remote-mismatch and --dry-run flags were only ever
// registered on statusCmd itself; getCmd/getReposCmd share statusCmd's RunE
// but read those flags via cmd.Flags() (i.e. off of themselves), so on an
// unregistered flag GetString/GetBool silently returned the zero value and
// the documented `get --reconcile-remote-mismatch` feature was unreachable.
// This proves the flag is both present *and* actually consulted by the
// shared RunE, by asserting the invalid-value error from
// parseRemoteMismatchReconcileMode surfaces through get/get repos.
func TestGetReconcileRemoteMismatchFlagIsWired(t *testing.T) {
	cfgPath := writeEmptyConfig(t)
	cleanup := withConfigAndCWD(t, cfgPath)
	defer cleanup()

	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "get", cmd: getCmd},
		{name: "get repos", cmd: getReposCmd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			tc.cmd.SetOut(out)
			tc.cmd.SetErr(errOut)
			tc.cmd.SetContext(context.Background())
			defer tc.cmd.SetOut(os.Stdout)
			defer tc.cmd.SetErr(os.Stderr)

			if err := tc.cmd.Flags().Set("reconcile-remote-mismatch", "bogus"); err != nil {
				t.Fatalf("expected %s to expose --reconcile-remote-mismatch flag: %v", tc.name, err)
			}
			defer func() { _ = tc.cmd.Flags().Set("reconcile-remote-mismatch", "none") }()

			err := tc.cmd.RunE(tc.cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "unsupported --reconcile-remote-mismatch value") {
				t.Fatalf("expected reconcile mode validation error to surface through %s, got %v", tc.name, err)
			}
		})
	}
}

func TestKubectlAliasRunEParity(t *testing.T) {
	tests := []struct {
		name   string
		alias  any
		target any
	}{
		{name: "get", alias: getCmd.RunE, target: statusCmd.RunE},
		{name: "get repos", alias: getReposCmd.RunE, target: statusCmd.RunE},
		{name: "reconcile", alias: reconcileCmd.RunE, target: syncCmd.RunE},
		{name: "reconcile repos", alias: reconcileReposCmd.RunE, target: syncCmd.RunE},
		{name: "repair upstream", alias: repairUpstreamAliasCmd.RunE, target: repairUpstreamCmd.RunE},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.alias == nil || tc.target == nil {
				t.Fatalf("expected non-nil command handlers for %s", tc.name)
			}
			if reflect.ValueOf(tc.alias).Pointer() != reflect.ValueOf(tc.target).Pointer() {
				t.Fatalf("expected alias handler to match legacy handler for %s", tc.name)
			}
		})
	}
}

func TestKubectlAliasFlagParity(t *testing.T) {
	tests := []struct {
		name  string
		cmd   *cobra.Command
		flags []string
	}{
		{
			name: "get",
			cmd:  getCmd,
			flags: []string{
				"roots",
				"registry",
				"format",
				"only",
				"field-selector",
				"selector",
				"reconcile-remote-mismatch",
				"dry-run",
				"no-headers",
				"wrap",
				"vcs",
			},
		},
		{
			name: "get repos",
			cmd:  getReposCmd,
			flags: []string{
				"roots",
				"registry",
				"format",
				"only",
				"field-selector",
				"selector",
				"reconcile-remote-mismatch",
				"dry-run",
				"no-headers",
				"wrap",
				"vcs",
			},
		},
		{
			name: "reconcile",
			cmd:  reconcileCmd,
			flags: []string{
				"only",
				"field-selector",
				"concurrency",
				"timeout",
				"continue-on-error",
				"dry-run",
				"yes",
				"update-local",
				"push-local",
				"rebase-dirty",
				"force",
				"protected-branches",
				"allow-protected-rebase",
				"checkout-missing",
				"format",
				"no-headers",
				"wrap",
				"vcs",
			},
		},
		{
			name: "reconcile repos",
			cmd:  reconcileReposCmd,
			flags: []string{
				"only",
				"field-selector",
				"concurrency",
				"timeout",
				"continue-on-error",
				"dry-run",
				"yes",
				"update-local",
				"push-local",
				"rebase-dirty",
				"force",
				"protected-branches",
				"allow-protected-rebase",
				"checkout-missing",
				"format",
				"no-headers",
				"wrap",
				"vcs",
			},
		},
		{
			name: "repair upstream",
			cmd:  repairUpstreamAliasCmd,
			flags: []string{
				"registry",
				"dry-run",
				"only",
				"format",
				"no-headers",
				"wrap",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, flag := range tc.flags {
				if tc.cmd.Flags().Lookup(flag) == nil {
					t.Fatalf("missing flag %q on %s", flag, tc.name)
				}
			}
		})
	}
}
