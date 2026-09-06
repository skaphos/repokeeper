// SPDX-License-Identifier: MIT
package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/v2/internal/contract"
)

// newToolError renders a failed tool call as an MCP error result.
//
// Every handler here reports failure as a *result* with a nil error return,
// which is the MCP convention: the call succeeded, its outcome was a failure.
// That makes this the only place an error becomes user-visible text, and
// therefore the only place the read-only-mount explanation can be attached.
// Translating in serializeTool alone would miss all of it -- that wrapper sees
// the Go error return, which these handlers do not use.
func newToolError(err error) *mcp.CallToolResult {
	if err == nil {
		return mcp.NewToolResultError("unknown error")
	}
	return mcp.NewToolResultError(explainReadOnly(err).Error())
}

// newToolErrorf adds context to err before rendering it. The verb must be %w so
// the cause stays unwrappable and the read-only translation can still recognise
// it through the wrapping.
func newToolErrorf(format string, err error) *mcp.CallToolResult {
	return newToolError(fmt.Errorf(format, err))
}

// newEnvelope builds the adapter contract envelope: the shared apiVersion plus
// exactly one named payload.
//
// MCP is an adapter-facing surface, so it carries the same contract version as
// the CLI, from the same constant. DESIGN.md §6.4 records that the CLI action
// commands and the MCP plan/execute tools deliberately expose identical fields
// so consumers can parse either; that parity is why the envelope had to arrive
// on both surfaces in one change rather than one after the other.
//
// The payload is keyed rather than inlined so a tool can add a sibling field
// later without restructuring its response.
func newEnvelope(key string, payload any) map[string]any {
	return map[string]any{
		"apiVersion": contract.APIVersion,
		key:          payload,
	}
}

// newStructuredResult renders an enveloped payload as an MCP result, with the
// text fallback carrying the same bytes as the structured content so the two
// cannot describe different things.
func newStructuredResult(key string, payload any) (*mcp.CallToolResult, error) {
	envelope := newEnvelope(key, payload)
	b, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal text fallback JSON for key %q: %w", key, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: mcp.ContentTypeText,
				Text: string(b),
			},
		},
		StructuredContent: envelope,
	}, nil
}

// newStructuredListResult envelopes a collection. A nil slice is normalised to
// an empty one: the contract requires an empty collection marshal as [] rather
// than null, so an adapter's parse path is the same whether or not there is
// anything to report.
func newStructuredListResult[T any](key string, entries []T) (*mcp.CallToolResult, error) {
	if entries == nil {
		entries = []T{}
	}
	return newStructuredResult(key, entries)
}
