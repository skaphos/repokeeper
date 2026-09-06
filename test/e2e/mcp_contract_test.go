// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/skaphos/repokeeper/v2/internal/contract"
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

// mcpEnvelopePayload returns the tool's payload from a decoded envelope. When
// the payload is an object its fields are returned directly; when it is a
// collection there are no object fields to check, so the envelope is returned
// unchanged and the caller's field assertions fail loudly rather than silently
// passing against the wrong level.
func mcpEnvelopePayload(envelope map[string]any) map[string]any {
	for key, value := range envelope {
		if key == "apiVersion" {
			continue
		}
		if payload, ok := value.(map[string]any); ok {
			return payload
		}
	}
	return envelope
}

func requireObjectFields(result *mcp.CallToolResult, fields ...string) error {
	object, err := decodeStructured[map[string]any](result)
	if err != nil {
		return err
	}
	// Every adapter-facing MCP result is the contract envelope: apiVersion plus
	// one named payload. Assert the version end-to-end against the real stdio
	// process, then look for the tool's own fields inside the payload.
	version, versioned := object["apiVersion"]
	if !versioned {
		return fmt.Errorf("structuredContent is missing the contract apiVersion")
	}
	if version != contract.APIVersion {
		return fmt.Errorf("structuredContent apiVersion = %v, want %q", version, contract.APIVersion)
	}
	object = mcpEnvelopePayload(object)

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
