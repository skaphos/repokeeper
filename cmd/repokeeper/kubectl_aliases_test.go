package repokeeper

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

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
