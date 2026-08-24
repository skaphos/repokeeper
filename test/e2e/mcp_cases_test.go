// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

type MCPToolCase struct {
	Name       string
	Arguments  func(*MaterializedWorkspace) map[string]any
	Assert     func(*mcp.CallToolResult) error
	ReadOnly   bool
	SafetyCase bool
}

func orderedMCPToolCases() ([]MCPToolCase, error) {
	cases := append(readMCPToolCases(), mutationMCPToolCases()...)
	cases = append(cases, safetyMCPToolCases()...)
	seen := map[string]bool{}
	for _, toolCase := range cases {
		if strings.TrimSpace(toolCase.Name) == "" {
			return nil, fmt.Errorf("MCP tool case has empty name")
		}
		if seen[toolCase.Name] {
			return nil, fmt.Errorf("duplicate MCP tool case %q", toolCase.Name)
		}
		seen[toolCase.Name] = true
	}
	return cases, nil
}

func compareLiveToolCoverage(tools []mcp.Tool, cases []MCPToolCase) error {
	live := map[string]mcp.Tool{}
	for _, tool := range tools {
		if _, duplicate := live[tool.Name]; duplicate {
			return fmt.Errorf("duplicate live MCP tool %q", tool.Name)
		}
		live[tool.Name] = tool
	}
	declared := map[string]MCPToolCase{}
	for _, toolCase := range cases {
		declared[toolCase.Name] = toolCase
	}
	missing, unexpected := []string{}, []string{}
	for name := range live {
		if _, exists := declared[name]; !exists {
			missing = append(missing, name)
		}
	}
	for name := range declared {
		if _, exists := live[name]; !exists {
			unexpected = append(unexpected, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)
	if len(missing) > 0 || len(unexpected) > 0 {
		return fmt.Errorf("MCP tool coverage drift: missing cases=%v unexpected cases=%v", missing, unexpected)
	}
	for name, tool := range live {
		toolCase := declared[name]
		readOnly := tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint
		destructive := tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint
		if readOnly != toolCase.ReadOnly {
			return fmt.Errorf("tool %s readOnly annotation=%t, case=%t", name, readOnly, toolCase.ReadOnly)
		}
		if destructive && !toolCase.SafetyCase {
			return fmt.Errorf("destructive tool %s has no safety case", name)
		}
	}
	return nil
}

func findMCPToolCase(cases []MCPToolCase, name string) (MCPToolCase, error) {
	for _, toolCase := range cases {
		if toolCase.Name == name {
			return toolCase, nil
		}
	}
	return MCPToolCase{}, fmt.Errorf("MCP case %q not found", name)
}
