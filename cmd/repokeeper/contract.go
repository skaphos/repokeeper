// SPDX-License-Identifier: MIT

package repokeeper

import "github.com/skaphos/repokeeper/v2/internal/contract"

// Envelope types for the adapter-facing CLI surfaces.
//
// Every adapter-facing response wraps its payload in one of these so the
// top-level object always carries `apiVersion` (see
// specs/004-adapter-contract/contracts/envelope.md). Before this, the action
// commands emitted bare top-level arrays, which have nowhere to put a version
// and so left adapters unable to detect a breaking change on any surface except
// `get`/`status`.
//
// The payload field is named per surface rather than anonymous. Naming it means
// a surface can add a sibling field later as an additive, non-breaking change;
// an anonymous payload would force a breaking change the first time a surface
// needed to report anything alongside its results.
//
// The constructors below exist to centralise one easily-forgotten guarantee: a
// nil Go slice marshals as `null`, and the contract requires `[]`. Going
// through a constructor makes that automatic instead of per-call-site
// vigilance.

// resultsEnvelope wraps the per-repo record collections emitted by the action
// commands (`reconcile`/`sync`, `repair upstream`).
type resultsEnvelope[T any] struct {
	contract.Header
	Results []T `json:"results"`
}

func newResultsEnvelope[T any](results []T) resultsEnvelope[T] {
	if results == nil {
		results = []T{}
	}
	return resultsEnvelope[T]{Header: contract.NewHeader(), Results: results}
}

// reposEnvelope wraps repository collections (`scan`).
type reposEnvelope[T any] struct {
	contract.Header
	Repos []T `json:"repos"`
}

func newReposEnvelope[T any](repos []T) reposEnvelope[T] {
	if repos == nil {
		repos = []T{}
	}
	return reposEnvelope[T]{Header: contract.NewHeader(), Repos: repos}
}

// repoEnvelope wraps a single repository record (`describe`).
type repoEnvelope[T any] struct {
	contract.Header
	Repo T `json:"repo"`
}

func newRepoEnvelope[T any](repo T) repoEnvelope[T] {
	return repoEnvelope[T]{Header: contract.NewHeader(), Repo: repo}
}

// labelsEnvelope wraps the label view of a repository (`label`).
type labelsEnvelope[T any] struct {
	contract.Header
	Labels T `json:"labels"`
}

func newLabelsEnvelope[T any](labels T) labelsEnvelope[T] {
	return labelsEnvelope[T]{Header: contract.NewHeader(), Labels: labels}
}

// versionEnvelope wraps build information (`version`).
type versionEnvelope[T any] struct {
	contract.Header
	Version T `json:"version"`
}

func newVersionEnvelope[T any](version T) versionEnvelope[T] {
	return versionEnvelope[T]{Header: contract.NewHeader(), Version: version}
}
