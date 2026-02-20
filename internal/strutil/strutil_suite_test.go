// SPDX-License-Identifier: MIT
package strutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStrutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Strutil Suite")
}
