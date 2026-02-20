// SPDX-License-Identifier: MIT
package tableutil

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTableutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tableutil Suite")
}
