// SPDX-License-Identifier: MIT
package repokeeper

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
	"github.com/spf13/cobra"
)

func TestParseOutputMode(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		kind   outputKind
		expr   string
		hasErr bool
	}{
		{name: "table", input: "table", kind: outputKindTable},
		{name: "wide", input: "wide", kind: outputKindWide},
		{name: "json", input: "json", kind: outputKindJSON},
		{name: "custom columns", input: "custom-columns=REPO:.repo_id,TRACKING:.tracking.status", kind: outputKindCustomColumns, expr: "REPO:.repo_id,TRACKING:.tracking.status"},
		{name: "invalid", input: "yaml", hasErr: true},
		{name: "custom missing expr", input: "custom-columns=", hasErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := parseOutputMode(tc.input)
			if tc.hasErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if mode.kind != tc.kind {
				t.Fatalf("kind = %q, want %q", mode.kind, tc.kind)
			}
			if mode.expr != tc.expr {
				t.Fatalf("expr = %q, want %q", mode.expr, tc.expr)
			}
		})
	}
}

func TestWriteCustomColumnsOutput(t *testing.T) {
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{RepoID: "github.com/org/repo-a", Tracking: model.Tracking{Status: model.TrackingEqual}},
		},
	}
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	if err := writeCustomColumnsOutput(cmd, report, "REPO:.repo_id,TRACKING:.tracking.status", false); err != nil {
		t.Fatalf("writeCustomColumnsOutput returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "REPO") || !strings.Contains(got, "TRACKING") {
		t.Fatalf("expected custom column headers, got: %q", got)
	}
	if !strings.Contains(got, "github.com/org/repo-a") || !strings.Contains(got, "equal") {
		t.Fatalf("unexpected custom-column row output: %q", got)
	}
}

func TestParseCustomColumnsSpecValidation(t *testing.T) {
	if _, err := parseCustomColumnsSpec("BROKEN"); err == nil {
		t.Fatal("expected invalid custom-columns segment error")
	}
	if _, err := parseCustomColumnsSpec(" , "); err == nil {
		t.Fatal("expected empty custom-columns error")
	}
}

func TestResolveCustomColumnValue(t *testing.T) {
	row := map[string]any{
		"repo_id": "github.com/org/repo-a",
		"tracking": map[string]any{
			"status": "equal",
		},
	}
	value, err := resolveCustomColumnValue(row, ".tracking.status")
	if err != nil {
		t.Fatalf("resolveCustomColumnValue returned error: %v", err)
	}
	if value != "equal" {
		t.Fatalf("value = %q, want equal", value)
	}
}

func TestCustomColumnsDeterministicAcrossTerminalWidths(t *testing.T) {
	report := &model.StatusReport{
		Repos: []model.RepoStatus{
			{RepoID: "github.com/org/repo-a", Tracking: model.Tracking{Status: model.TrackingEqual}},
		},
	}
	render := func(width int) (string, error) {
		prevIsTerminalFD := isTerminalFD
		prevGetTerminalSize := getTerminalSize
		defer func() {
			isTerminalFD = prevIsTerminalFD
			getTerminalSize = prevGetTerminalSize
		}()
		isTerminalFD = func(int) bool { return true }
		getTerminalSize = func(int) (int, int, error) { return width, 24, nil }

		out := &bytes.Buffer{}
		cmd := &cobra.Command{}
		cmd.SetOut(out)
		err := writeCustomColumnsOutput(cmd, report, "REPO:.repo_id,TRACKING:.tracking.status", false)
		return out.String(), err
	}

	narrow, err := render(80)
	if err != nil {
		t.Fatalf("narrow render failed: %v", err)
	}
	wide, err := render(160)
	if err != nil {
		t.Fatalf("wide render failed: %v", err)
	}
	if narrow != wide {
		t.Fatalf("expected deterministic custom-column output across widths, narrow=%q wide=%q", narrow, wide)
	}
}
