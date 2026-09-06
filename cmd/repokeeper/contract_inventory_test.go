// SPDX-License-Identifier: MIT
package repokeeper

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/v2/internal/mcpserver"
	"github.com/spf13/cobra"
)

// Compare the published inventory with registrations, not a second hand-written
// list of commands. New JSON commands must be documented or explicitly excluded.
func TestPublishedAdapterInventoryMatchesRegistrations(t *testing.T) {
	document, err := os.ReadFile("../../docs/adapter-contract.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(document)
	t.Run("CLI", func(t *testing.T) {
		var actual []string
		var walk func(*cobra.Command)
		walk = func(cmd *cobra.Command) {
			if cmd.Runnable() {
				flag := ""
				if f := cmd.Flags().Lookup("format"); f != nil && strings.Contains(f.Usage, "json") {
					flag = " -o json"
				}
				if cmd.Flags().Lookup("json") != nil {
					flag = " --json"
				}
				if flag != "" {
					path := strings.TrimPrefix(cmd.CommandPath(), "repokeeper ")
					actual = append(actual, path+flag)
					for _, alias := range cmd.Aliases {
						actual = append(actual, strings.TrimSuffix(path, cmd.Name())+alias+flag)
					}
				}
			}
			for _, child := range cmd.Commands() {
				walk(child)
			}
		}
		walk(rootCmd)
		section := inventorySection(t, doc, "### CLI", "### MCP tools")
		var documented []string
		for _, match := range regexp.MustCompile("`([^`]+(?: -o json| --json))`").FindAllStringSubmatch(section, -1) {
			documented = append(documented, match[1])
		}
		assertInventory(t, actual, documented)
	})
	server := mcpserver.New(nil, "", "test", nil).Inner()
	t.Run("MCP tools", func(t *testing.T) {
		var actual []string
		for name := range server.ListTools() {
			actual = append(actual, name)
		}
		assertInventory(t, actual, inventoryRows(inventorySection(t, doc, "### MCP tools", "### MCP resources")))
	})
	t.Run("MCP resources", func(t *testing.T) {
		var actual []string
		for uri := range server.ListResources() {
			actual = append(actual, uri)
		}
		assertInventory(t, actual, inventoryRows(inventorySection(t, doc, "### MCP resources", "### MCP resource templates")))
	})
	t.Run("MCP resource templates", func(t *testing.T) {
		response := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"resources/templates/list"}`))
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Result struct {
				Templates []struct {
					URI string `json:"uriTemplate"`
				} `json:"resourceTemplates"`
			} `json:"result"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		var actual []string
		for _, template := range decoded.Result.Templates {
			actual = append(actual, template.URI)
		}
		if len(actual) == 0 {
			t.Fatalf("no resource templates returned: %s", data)
		}
		assertInventory(t, actual, inventoryRows(inventorySection(t, doc, "### MCP resource templates", "### Resource payload fields")))
	})
}

func inventorySection(t *testing.T, doc, start, end string) string {
	t.Helper()
	_, section, ok := strings.Cut(doc, start)
	if !ok {
		t.Fatalf("missing inventory heading %s", start)
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		t.Fatalf("missing inventory heading %s", end)
	}
	return section
}

func inventoryRows(section string) []string {
	var names []string
	for _, match := range regexp.MustCompile("(?m)^\\| `([^`]+)` \\|").FindAllStringSubmatch(section, -1) {
		names = append(names, match[1])
	}
	return names
}

func assertInventory(t *testing.T, actual, documented []string) {
	t.Helper()
	slices.Sort(actual)
	slices.Sort(documented)
	if !slices.Equal(actual, documented) {
		t.Fatalf("registered surfaces: %v\ndocumented surfaces: %v", actual, documented)
	}
}
