// SPDX-License-Identifier: MIT
package sortutil

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSortutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sortutil Suite")
}
