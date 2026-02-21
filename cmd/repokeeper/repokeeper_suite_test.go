// SPDX-License-Identifier: MIT
package repokeeper

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRepokeeperSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repokeeper Suite")
}
