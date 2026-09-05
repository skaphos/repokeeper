// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestCheckoutSelectorPrecedenceAndAmbiguity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := registry.Entry{RepoID: "github.com/example/proj", CheckoutID: "proj", Path: filepath.Join(root, "a", "proj")}
	second := registry.Entry{RepoID: first.RepoID, CheckoutID: "proj-other", Path: filepath.Join(root, "b", "proj")}
	for _, tc := range []struct {
		name, selector string
		extra          []registry.Entry
		want           string
		ambiguous      bool
	}{
		{name: "bare checkout ID", selector: second.CheckoutID, want: second.Path},
		{name: "qualified ID", selector: first.RepoID + "@" + second.CheckoutID, want: second.Path},
		{name: "absolute missing path", selector: second.Path, want: second.Path},
		{name: "relative path", selector: filepath.Join("b", "proj"), want: second.Path},
		{name: "checkout ID precedes repo ID", selector: second.CheckoutID, extra: []registry.Entry{{RepoID: second.CheckoutID, CheckoutID: "other", Path: filepath.Join(root, "other")}}, want: second.Path},
		{name: "absolute path precedes checkout ID", selector: second.Path, extra: []registry.Entry{{RepoID: "other", CheckoutID: second.Path, Path: filepath.Join(root, "other")}}, want: second.Path},
		{name: "repo ambiguity", selector: first.RepoID, ambiguous: true},
		{name: "checkout ambiguity across repos", selector: second.CheckoutID, extra: []registry.Entry{{RepoID: "other", CheckoutID: second.CheckoutID, Path: filepath.Join(root, "other")}}, ambiguous: true},
		{name: "qualified ambiguity", selector: first.RepoID + "@" + second.CheckoutID, extra: []registry.Entry{{RepoID: first.RepoID, CheckoutID: second.CheckoutID, Path: filepath.Join(root, "other")}}, ambiguous: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entries := append([]registry.Entry{first, second}, tc.extra...)
			got, err := selectRegistryEntryForDescribe(entries, tc.selector, root, nil)
			if tc.ambiguous {
				if err == nil {
					t.Fatal("expected ambiguity")
				}
				for _, hint := range []string{"ambiguous", second.CheckoutID, second.Path, "absolute path"} {
					if !strings.Contains(err.Error(), hint) {
						t.Errorf("missing hint %q: %v", hint, err)
					}
				}
				return
			}
			if err != nil || got.Path != tc.want {
				t.Fatalf("got %q, %v; want %q", got.Path, err, tc.want)
			}
		})
	}
}

func TestCheckoutSelectorSymlinkAndAtSignPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := filepath.Join(root, "real@checkout")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	entry := registry.Entry{RepoID: "github.com/example/proj", CheckoutID: "custom", Path: target}
	for _, selector := range []string{target, alias, "alias", "real@checkout"} {
		got, err := selectRegistryEntryForDescribe([]registry.Entry{entry}, selector, root, nil)
		if err != nil || got.Path != target {
			t.Errorf("selector %q: got %q, %v", selector, got.Path, err)
		}
	}
	// Also resolve a real path when the registry stores the symlink spelling.
	entry.Path = alias
	got, err := selectRegistryEntryForDescribe([]registry.Entry{entry}, target, root, nil)
	if err != nil || got.Path != alias {
		t.Fatalf("stored alias: got %q, %v", got.Path, err)
	}
}

func TestGetReposReportsErrors(t *testing.T) {
	// Each command has its own flags/context; no global CLI state is modified.
	t.Parallel()
	for _, tc := range []struct {
		name, format, localSelector, fieldSelector string
		quiet                                      bool
		count                                      int
	}{
		{name: "table", format: "table", count: 2},
		{name: "wide", format: "wide", count: 2},
		{name: "json", format: "json", count: 2},
		{name: "custom columns", format: "custom-columns=PATH:.path", count: 2},
		{name: "quiet json", format: "json", quiet: true, count: 2},
		{name: "filtered", format: "table", localSelector: "env=prod", fieldSelector: "repo.missing=true", count: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			cfg := config.DefaultConfig()
			cfg.Registry = &registry.Registry{Entries: []registry.Entry{
				{RepoID: "github.com/example/proj", CheckoutID: "prod", Path: filepath.Join(root, "prod"), Status: registry.StatusMissing, Labels: map[string]string{"env": "prod"}},
				{RepoID: "github.com/example/proj", CheckoutID: "dev", Path: filepath.Join(root, "dev"), Status: registry.StatusMissing, Labels: map[string]string{"env": "dev"}},
			}}
			cfgPath := filepath.Join(root, "config.yaml")
			if err := config.Save(&cfg, cfgPath); err != nil {
				t.Fatal(err)
			}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			var out, errOut bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)
			cmd.Flags().String("config", cfgPath, "")
			cmd.Flags().String("format", tc.format, "")
			cmd.Flags().String("local-selector", tc.localSelector, "")
			cmd.Flags().String("field-selector", tc.fieldSelector, "")
			cmd.Flags().Bool("quiet", tc.quiet, "")
			addVCSFlag(cmd)
			if err := getReposCmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if code := runtimeStateFor(cmd).exitCode; code != 2 {
				t.Errorf("exit code=%d, want 2", code)
			}
			diagnostic := errOut.String()
			if !strings.Contains(diagnostic, "error: prod (missing: path missing)") {
				t.Errorf("missing diagnostic: %q", diagnostic)
			}
			if strings.Contains(diagnostic, "error: dev") != (tc.count == 2) {
				t.Errorf("wrong filtered diagnostics: %q", diagnostic)
			}
			if tc.quiet {
				if strings.Contains(diagnostic, "status completed") {
					t.Errorf("quiet summary: %q", diagnostic)
				}
			} else {
				suffix := "errors"
				if tc.count == 1 {
					suffix = "error"
				}
				want := fmt.Sprintf("status completed: %d repos (%d %s)", tc.count, tc.count, suffix)
				if !strings.Contains(diagnostic, want) {
					t.Errorf("missing summary %q: %q", want, diagnostic)
				}
			}
			switch tc.format {
			case "table", "wide":
				if !strings.Contains(out.String(), "ERROR_CLASS") || !strings.Contains(out.String(), "missing") {
					t.Errorf("hidden error column: %q", out.String())
				}
			case "json":
				var report statusJSONReport
				if err := json.Unmarshal(out.Bytes(), &report); err != nil {
					t.Fatal(err)
				}
				if len(report.Repos) != tc.count || report.Repos[0].ErrorClass != "missing" {
					t.Errorf("unexpected JSON: %s", out.String())
				}
			}
		})
	}
}

func TestStatusTableErrorColumn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		repo model.RepoStatus
		want string
	}{
		{name: "healthy", repo: model.RepoStatus{Path: "healthy"}},
		{name: "unclassified error", repo: model.RepoStatus{Path: "broken", Error: "failure"}, want: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)
			err := writeStatusTable(cmd, &model.StatusReport{Repos: []model.RepoStatus{tc.repo}}, "", nil, false, false)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "ERROR_CLASS") != (tc.want != "") {
				t.Errorf("wrong columns: %q", out.String())
			}
			if tc.want != "" && !strings.Contains(out.String(), tc.want) {
				t.Errorf("missing error class: %q", out.String())
			}
		})
	}
}
