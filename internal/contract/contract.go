// SPDX-License-Identifier: MIT

// Package contract defines the adapter-facing output contract: the version
// identifier every machine-readable RepoKeeper response carries, and the
// envelope header that carries it.
//
// It exists as its own package for one reason: the contract must be identical
// across the CLI and the MCP server, and the only way to make that mechanically
// true rather than a convention reviewers police is to give both a single
// constant to import. `cmd/repokeeper` cannot host it -- `internal/mcpserver`
// would have to import the command package to reach it, and the command package
// already imports the server.
//
// See specs/004-adapter-contract/contracts/envelope.md for the contract this
// implements, and ADR-0006 for the policy it implements.
package contract

import "time"

// APIVersion identifies the schema of every adapter-facing machine-readable
// response, CLI and MCP alike.
//
// This is versioned INDEPENDENTLY of two other apiVersion values in this
// codebase, and the distinction is load-bearing:
//
//   - config.ConfigAPIVersion identifies the on-disk `.repokeeper.yaml` schema.
//     It is written into user config files and validated on load, so changing
//     it invalidates every existing user's configuration.
//   - repometa's apiVersion identifies the repo-local metadata file schema.
//
// All three shared the value "skaphos.io/repokeeper/v1beta1" until the output
// contract was promoted to v1 for the 2.0.0 release. They are free to diverge
// and must never be coupled: a bump to one does not imply a bump to another.
// TestContractVersionIsIndependentOfConfig guards this.
const APIVersion = "skaphos.io/repokeeper/v1"

// Header is the envelope prefix shared by every adapter-facing response. It is
// embedded, not nested, so its fields marshal as siblings of the payload:
//
//	{"apiVersion": "...", "generated_at": "...", "repos": [...]}
//
// Each surface embeds this alongside its own single named payload field. The
// payload is named rather than anonymous so a surface can later add a sibling
// field additively, without the breaking change an anonymous payload would
// force.
type Header struct {
	APIVersion string `json:"apiVersion"`

	// GeneratedAt is present only where the response reports observed state at
	// a point in time. A pure configuration echo has nothing meaningful to
	// timestamp, and emitting a zero time there would be worse than omitting
	// it -- a consumer cannot distinguish "not applicable" from "the epoch".
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
}

// NewHeader returns a header with no timestamp, for responses that do not
// report point-in-time observed state.
func NewHeader() Header {
	return Header{APIVersion: APIVersion}
}

// NewHeaderAt returns a header stamped with an observation time.
func NewHeaderAt(observed time.Time) Header {
	return Header{APIVersion: APIVersion, GeneratedAt: &observed}
}
