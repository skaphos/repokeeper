package gitx_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGitx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gitx Suite")
}
