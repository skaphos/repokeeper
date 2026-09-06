// SPDX-License-Identifier: MIT

package contract_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/skaphos/repokeeper/v2/internal/config"
	"github.com/skaphos/repokeeper/v2/internal/contract"
	"github.com/skaphos/repokeeper/v2/internal/repometa"
)

func TestHeaderCarriesAPIVersion(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(contract.NewHeader())
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if got := decoded["apiVersion"]; got != contract.APIVersion {
		t.Errorf("apiVersion = %v, want %q", got, contract.APIVersion)
	}
}

// TestHeaderOmitsUnsetGeneratedAt guards the distinction between "this response
// is not a point-in-time observation" and "this response was generated at the
// epoch". Emitting a zero time would make those indistinguishable to a consumer.
func TestHeaderOmitsUnsetGeneratedAt(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(contract.NewHeader())
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if _, present := decoded["generated_at"]; present {
		t.Errorf("generated_at must be omitted when unset, got %s", data)
	}
}

func TestHeaderIncludesSetGeneratedAt(t *testing.T) {
	t.Parallel()

	observed := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	data, err := json.Marshal(contract.NewHeaderAt(observed))
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	var decoded struct {
		GeneratedAt *time.Time `json:"generated_at"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if decoded.GeneratedAt == nil || !decoded.GeneratedAt.Equal(observed) {
		t.Errorf("generated_at = %v, want %v", decoded.GeneratedAt, observed)
	}
}

// TestContractVersionIsIndependentOfOtherSchemas guards FR-004: the
// adapter-facing output contract is versioned independently of the on-disk
// config schema and the repo-local metadata schema.
//
// The output contract and the config schema shared a single string before the
// output contract was promoted to v1, which makes accidental coupling --
// defining one in terms of another -- easy to introduce and invisible in
// review. (Repo metadata has always used its own unprefixed scheme and never
// shared the value; it is pinned here so it stays that way.)
//
// Note what this test does NOT do. Asserting the three values merely differ
// would be a weak guard: it passes for aliased constants that happen to hold
// different strings today, and it would fail spuriously if two schemas were
// legitimately bumped to the same version later. Instead each is pinned to its
// own literal. If any of them is ever defined in terms of another, bumping one
// moves both and one of these pins fails, which is exactly the coupling FR-004
// forbids.
//
// Changing a pin here is a deliberate act: it means that schema's version
// really did change, and the corresponding documentation must move with it.
func TestContractVersionIsIndependentOfOtherSchemas(t *testing.T) {
	t.Parallel()

	if contract.APIVersion != "skaphos.io/repokeeper/v1" {
		t.Errorf("output contract apiVersion = %q; update DESIGN.md §6.3 and this pin together", contract.APIVersion)
	}
	if config.ConfigAPIVersion != "skaphos.io/repokeeper/v1beta1" {
		t.Errorf("config apiVersion = %q; it is written into user .repokeeper.yaml files and validated on load, so changing it invalidates existing configs", config.ConfigAPIVersion)
	}
	if repometa.APIVersion != "repokeeper/v1" {
		t.Errorf("repo metadata apiVersion = %q; changing it invalidates existing .repokeeper-repo.yaml files", repometa.APIVersion)
	}
}
