// SPDX-License-Identifier: MIT
package vcs_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/vcs"
)

// localFmtArgs pins the exact argv of the enumeration call. If the NUL-delimited
// format changes, these tests fail loudly rather than silently missing the call.
const localFmtArgs = "for-each-ref --format=%(refname:short)%00%(upstream:short)%00%(upstream:track)%00%(upstream:trackshort)%00%(committerdate:iso-strict)%00%(worktreepath) refs/heads"

type stubResp = struct {
	out string
	err error
}

func enumLine(fields ...string) string { return strings.Join(fields, "\x00") }

func TestGitAdapterInspectLocalBranches(t *testing.T) {
	enum := enumLine("main", "origin/main", "", "=", "2026-07-11T00:00:00Z", "") + "\n" +
		enumLine("feature", "origin/feature", "[gone]", "", "2026-07-10T00:00:00Z", "") + "\n"

	r := &runnerStub{responses: map[string]stubResp{
		"/repo:" + localFmtArgs: {out: enum},
		"/repo:for-each-ref --merged=origin/main --format=%(refname:short) refs/heads": {out: "main\n"},
		"/repo:cherry origin/main feature":                                             {out: "- abc123\n"},
	}}

	sigs, err := vcs.NewGitAdapter(r).InspectLocalBranches(context.Background(), "/repo", "origin/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sigs) != 2 {
		t.Fatalf("got %d signals, want 2", len(sigs))
	}
	byName := map[string]vcs.LocalBranchSignal{}
	for _, s := range sigs {
		byName[s.Name] = s
	}

	if m := byName["main"]; m.MergedIntoBase == nil || !*m.MergedIntoBase || m.PatchEquivalentToBase != nil {
		t.Errorf("main: merged=%v patch=%v; want merged=true, patch=nil (skipped)", m.MergedIntoBase, m.PatchEquivalentToBase)
	}
	feat := byName["feature"]
	if feat.MergedIntoBase == nil || *feat.MergedIntoBase {
		t.Errorf("feature MergedIntoBase = %v, want false", feat.MergedIntoBase)
	}
	if feat.PatchEquivalentToBase == nil || !*feat.PatchEquivalentToBase {
		t.Errorf("feature PatchEquivalentToBase = %v, want true", feat.PatchEquivalentToBase)
	}
	if feat.Track != "[gone]" {
		t.Errorf("feature Track = %q, want [gone]", feat.Track)
	}
}

func TestGitAdapterInspectLocalBranchesNoBase(t *testing.T) {
	enum := enumLine("main", "", "", "", "2026-07-11T00:00:00Z", "") + "\n"
	r := &runnerStub{responses: map[string]stubResp{"/repo:" + localFmtArgs: {out: enum}}}

	sigs, err := vcs.NewGitAdapter(r).InspectLocalBranches(context.Background(), "/repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sigs) != 1 || sigs[0].MergedIntoBase != nil || sigs[0].PatchEquivalentToBase != nil {
		t.Errorf("empty base should leave integration signals nil: %+v", sigs)
	}
}

func TestGitAdapterInspectLocalBranchesMergedCheckFails(t *testing.T) {
	enum := enumLine("feature", "origin/feature", "", ">", "2026-07-11T00:00:00Z", "") + "\n"
	r := &runnerStub{responses: map[string]stubResp{
		"/repo:" + localFmtArgs: {out: enum},
		"/repo:for-each-ref --merged=origin/main --format=%(refname:short) refs/heads": {err: errors.New("fatal: malformed object name")},
		"/repo:cherry origin/main feature":                                             {out: "+ abc123\n"},
	}}

	sigs, err := vcs.NewGitAdapter(r).InspectLocalBranches(context.Background(), "/repo", "origin/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sigs[0].MergedIntoBase != nil {
		t.Errorf("MergedIntoBase should be nil when merged check fails, got %v", sigs[0].MergedIntoBase)
	}
	if sigs[0].PatchEquivalentToBase == nil || *sigs[0].PatchEquivalentToBase {
		t.Errorf("PatchEquivalentToBase should be false, got %v", sigs[0].PatchEquivalentToBase)
	}
}

func TestGitAdapterInspectLocalBranchesEnumError(t *testing.T) {
	r := &runnerStub{responses: map[string]stubResp{"/repo:" + localFmtArgs: {err: errors.New("boom")}}}
	if _, err := vcs.NewGitAdapter(r).InspectLocalBranches(context.Background(), "/repo", "origin/main"); err == nil {
		t.Errorf("expected error when enumeration fails")
	}
}
