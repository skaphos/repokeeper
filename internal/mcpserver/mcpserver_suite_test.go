// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMCPServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCPServer Suite")
}
