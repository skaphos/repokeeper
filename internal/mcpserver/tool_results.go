// SPDX-License-Identifier: MIT
package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
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

func newStructuredListResult[T any](key string, entries []T) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal text fallback JSON for list key %q: %w", key, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: mcp.ContentTypeText,
				Text: string(b),
			},
		},
		StructuredContent: map[string]any{key: entries},
	}, nil
}
