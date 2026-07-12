// SPDX-License-Identifier: MIT
package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/obs"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// stubLBAdapter embeds vcs.Adapter (nil) and implements only the local-branch
// inspection capability, which is all inspectLocalBranches calls.
type stubLBAdapter struct {
	vcs.Adapter
	signals   []vcs.LocalBranchSignal
	err       error
	baseSeen  *string // when non-nil, records the base ref the inspector was called with
	patchSeen *bool   // when non-nil, records the patchEquivalence flag passed
}

func (s stubLBAdapter) InspectLocalBranches(_ context.Context, _, base string, patchEquivalence bool) ([]vcs.LocalBranchSignal, error) {
	if s.baseSeen != nil {
		*s.baseSeen = base
	}
	if s.patchSeen != nil {
		*s.patchSeen = patchEquivalence
	}
	return s.signals, s.err
}

// plainAdapter embeds vcs.Adapter but does NOT implement LocalBranchInspector.
type plainAdapter struct{ vcs.Adapter }

func newEngineWith(cfg config.Config, adapter vcs.Adapter) *Engine {
	return &Engine{cfg: &cfg, registry: &registry.Registry{}, adapter: adapter, logger: obs.NopLogger()}
}

func TestInspectLocalBranchesClassifies(t *testing.T) {
	tru, fls := true, false
	adapter := stubLBAdapter{signals: []vcs.LocalBranchSignal{
		{Name: "main", Upstream: "origin/main", TrackShort: "=", MergedIntoBase: &tru},
		{Name: "feature/done", Upstream: "origin/feature/done", TrackShort: "=", MergedIntoBase: &tru},
		{Name: "feature/wip", Upstream: "origin/feature/wip", TrackShort: ">", MergedIntoBase: &fls, PatchEquivalentToBase: &fls},
	}}
	e := newEngineWith(config.DefaultConfig(), adapter)

	got := e.inspectLocalBranches(context.Background(), "/repo", "origin", "repoID",
		model.Head{Branch: "main"}, model.Tracking{Upstream: "origin/main"}, false)

	if got.InspectionError != "" {
		t.Fatalf("unexpected inspection error: %s", got.InspectionError)
	}
	byName := map[string]model.LocalBranch{}
	for _, b := range got.Branches {
		byName[b.Name] = b
	}
	if b := byName["main"]; b.Category != model.PruneKeep || b.Reasons[0] != model.ReasonCurrentBranch {
		t.Errorf("main = %s %v, want keep/current_branch", b.Category, b.Reasons)
	}
	if b := byName["feature/done"]; b.Category != model.PruneSafeToPrune {
		t.Errorf("feature/done = %s, want safe_to_prune", b.Category)
	}
	if b := byName["feature/wip"]; b.Category != model.PruneKeep || b.Reasons[0] != model.ReasonActiveUnmerged {
		t.Errorf("feature/wip = %s %v, want keep/active_unmerged", b.Category, b.Reasons)
	}
}

func TestInspectLocalBranchesBaseUnresolved(t *testing.T) {
	tru := true
	cfg := config.DefaultConfig()
	cfg.Defaults.MainBranch = "" // remove the last-resort fallback so no base resolves
	adapter := stubLBAdapter{signals: []vcs.LocalBranchSignal{{Name: "feature/x", MergedIntoBase: &tru}}}
	e := newEngineWith(cfg, adapter)

	// A non-current branch with no resolvable base must be needs_review/base_unresolved.
	got := e.inspectLocalBranches(context.Background(), "/repo", "", "repoID",
		model.Head{Branch: "other"}, model.Tracking{}, false)
	if b := got.Branches[0]; b.Category != model.PruneNeedsReview || b.Reasons[0] != model.ReasonBaseUnresolved {
		t.Errorf("unresolved base = %s %v, want needs_review/base_unresolved", b.Category, b.Reasons)
	}
}

func TestInspectLocalBranchesRemoteQualifiedBaseOverride(t *testing.T) {
	tru := true
	var seen string
	cfg := config.DefaultConfig()
	cfg.BranchPolicy.BaseBranch = "origin/main" // remote-qualified override
	adapter := stubLBAdapter{
		baseSeen: &seen,
		signals: []vcs.LocalBranchSignal{
			{Name: "main", MergedIntoBase: &tru},      // the base branch itself
			{Name: "feature/x", MergedIntoBase: &tru}, // a merged topic branch
		},
	}
	e := newEngineWith(cfg, adapter)

	got := e.inspectLocalBranches(context.Background(), "/repo", "origin", "id",
		model.Head{Branch: "other"}, model.Tracking{}, false)

	// The git query base must be the qualified ref, not double-prefixed.
	if seen != "origin/main" {
		t.Errorf("query base = %q, want origin/main (no double prefix)", seen)
	}
	byName := map[string]model.LocalBranch{}
	for _, b := range got.Branches {
		byName[b.Name] = b
	}
	// The base branch must be recognized despite the qualified override.
	if b := byName["main"]; b.Category != model.PruneKeep || b.Reasons[0] != model.ReasonBaseBranch {
		t.Errorf("main = %s %v, want keep/base_branch", b.Category, b.Reasons)
	}
	if b := byName["feature/x"]; b.Category != model.PruneSafeToPrune {
		t.Errorf("feature/x = %s, want safe_to_prune", b.Category)
	}
}

func TestInspectLocalBranchesPatchEquivalenceGatedByRequireMerged(t *testing.T) {
	tru := true
	sig := []vcs.LocalBranchSignal{{Name: "feature/x", MergedIntoBase: &tru}}

	// Default (require_merged: true) must NOT request patch-equivalence.
	var patchDefault bool
	def := stubLBAdapter{signals: sig, patchSeen: &patchDefault}
	newEngineWith(config.DefaultConfig(), def).inspectLocalBranches(
		context.Background(), "/repo", "origin", "id", model.Head{Branch: "main"}, model.Tracking{}, false)
	if patchDefault {
		t.Errorf("default (require_merged=true) should skip patch-equivalence")
	}

	// require_merged: false must request patch-equivalence.
	var patchOptIn bool
	cfg := config.DefaultConfig()
	cfg.BranchPolicy.RequireMerged = false
	optIn := stubLBAdapter{signals: sig, patchSeen: &patchOptIn}
	newEngineWith(cfg, optIn).inspectLocalBranches(
		context.Background(), "/repo", "origin", "id", model.Head{Branch: "main"}, model.Tracking{}, false)
	if !patchOptIn {
		t.Errorf("require_merged=false should request patch-equivalence")
	}
}

func TestInspectLocalBranchesUnsupportedAndBareAndError(t *testing.T) {
	e := newEngineWith(config.DefaultConfig(), plainAdapter{})
	if got := e.inspectLocalBranches(context.Background(), "/repo", "origin", "id", model.Head{}, model.Tracking{}, false); len(got.Branches) != 0 || got.InspectionError != "" {
		t.Errorf("unsupported adapter should yield empty result, got %+v", got)
	}

	bareEng := newEngineWith(config.DefaultConfig(), stubLBAdapter{})
	if got := bareEng.inspectLocalBranches(context.Background(), "/repo", "origin", "id", model.Head{}, model.Tracking{}, true); len(got.Branches) != 0 {
		t.Errorf("bare repo should yield empty result, got %+v", got)
	}

	errEng := newEngineWith(config.DefaultConfig(), stubLBAdapter{err: errors.New("boom")})
	if got := errEng.inspectLocalBranches(context.Background(), "/repo", "origin", "id", model.Head{Branch: "x"}, model.Tracking{Upstream: "origin/main"}, false); got.InspectionError == "" {
		t.Errorf("inspection error should be surfaced")
	}
}

func TestResolveBaseBranchNamePrecedence(t *testing.T) {
	upstream := model.Tracking{Upstream: "origin/develop"}

	// 1. explicit config override wins.
	cfg := config.DefaultConfig()
	cfg.BranchPolicy.BaseBranch = "trunk"
	e := newEngineWith(cfg, plainAdapter{})
	if got := e.resolveBaseBranchName("id", "/repo", upstream); got != "trunk" {
		t.Errorf("override: got %q, want trunk", got)
	}

	// 2. registry entry branch.
	cfg2 := config.DefaultConfig()
	e2 := newEngineWith(cfg2, plainAdapter{})
	e2.registry = &registry.Registry{Entries: []registry.Entry{{RepoID: "id", Path: "/repo", Branch: "master"}}}
	if got := e2.resolveBaseBranchName("id", "/repo", upstream); got != "master" {
		t.Errorf("registry: got %q, want master", got)
	}

	// 3. upstream-derived (no override, no registry branch).
	e3 := newEngineWith(config.DefaultConfig(), plainAdapter{})
	if got := e3.resolveBaseBranchName("id", "/repo", upstream); got != "develop" {
		t.Errorf("upstream: got %q, want develop", got)
	}

	// 4. workspace default.
	e4 := newEngineWith(config.DefaultConfig(), plainAdapter{})
	if got := e4.resolveBaseBranchName("id", "/repo", model.Tracking{}); got != "main" {
		t.Errorf("default: got %q, want main", got)
	}

	// 5. nothing resolves.
	cfg5 := config.DefaultConfig()
	cfg5.Defaults.MainBranch = ""
	e5 := newEngineWith(cfg5, plainAdapter{})
	if got := e5.resolveBaseBranchName("id", "/repo", model.Tracking{}); got != "" {
		t.Errorf("none: got %q, want empty", got)
	}
}

func TestUpstreamStatusFromSignal(t *testing.T) {
	tests := []struct {
		sig  vcs.LocalBranchSignal
		want model.TrackingStatus
	}{
		{vcs.LocalBranchSignal{Upstream: ""}, model.TrackingNone},
		{vcs.LocalBranchSignal{Upstream: "origin/x", Track: "[gone]"}, model.TrackingGone},
		{vcs.LocalBranchSignal{Upstream: "origin/x", TrackShort: ">"}, model.TrackingAhead},
		{vcs.LocalBranchSignal{Upstream: "origin/x", TrackShort: "<"}, model.TrackingBehind},
		{vcs.LocalBranchSignal{Upstream: "origin/x", TrackShort: "<>"}, model.TrackingDiverged},
		{vcs.LocalBranchSignal{Upstream: "origin/x", TrackShort: "="}, model.TrackingEqual},
	}
	for _, tc := range tests {
		if got := upstreamStatusFromSignal(tc.sig); got != tc.want {
			t.Errorf("upstreamStatusFromSignal(%+v) = %q, want %q", tc.sig, got, tc.want)
		}
	}
}
