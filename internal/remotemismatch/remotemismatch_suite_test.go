// SPDX-License-Identifier: MIT
package remotemismatch

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRemoteMismatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RemoteMismatch Suite")
}
