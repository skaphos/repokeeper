// SPDX-License-Identifier: MIT
package vcs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVcs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VCS Suite")
}
