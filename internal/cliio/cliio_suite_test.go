// SPDX-License-Identifier: MIT
package cliio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCliio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cliio Suite")
}
