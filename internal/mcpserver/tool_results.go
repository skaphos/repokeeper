// SPDX-License-Identifier: MIT
package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func newStructuredListResult[T any](key string, entries []T) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal JSON: %w", err)
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
