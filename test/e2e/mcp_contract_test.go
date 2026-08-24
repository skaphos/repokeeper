// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func decodeStructured[T any](result *mcp.CallToolResult) (T, error) {
	var target T
	if result == nil {
		return target, fmt.Errorf("nil MCP tool result")
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return target, err
	}
	if err := json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("decode structuredContent: %w", err)
	}
	return target, nil
}

func requireObjectFields(result *mcp.CallToolResult, fields ...string) error {
	object, err := decodeStructured[map[string]any](result)
	if err != nil {
		return err
	}
	for _, field := range fields {
		if _, exists := object[field]; !exists {
			return fmt.Errorf("structuredContent missing required field %q", field)
		}
	}
	return nil
}

func requireListField(result *mcp.CallToolResult, field string) ([]any, error) {
	object, err := decodeStructured[map[string]any](result)
	if err != nil {
		return nil, err
	}
	value, exists := object[field]
	if !exists {
		return nil, fmt.Errorf("structuredContent missing list field %q", field)
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("structuredContent field %q is %T, want list", field, value)
	}
	return items, nil
}

func requireToolSuccess(result *mcp.CallToolResult) error {
	if result == nil {
		return fmt.Errorf("nil MCP result")
	}
	if result.IsError {
		return fmt.Errorf("MCP tool returned isError=true: %v", result.Content)
	}
	if result.StructuredContent == nil {
		return fmt.Errorf("MCP tool returned no structuredContent")
	}
	return nil
}
