// SPDX-License-Identifier: MIT
package termstyle

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTermstyle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Termstyle Suite")
}
