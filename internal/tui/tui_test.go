// SPDX-License-Identifier: MIT
package tui_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/tui"
)

var _ = Describe("Run", func() {
	It("returns no error when context is cancelled immediately", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := config.DefaultConfig()
		reg := &registry.Registry{}
		err := tui.Run(ctx, &cfg, reg, "")
		Expect(err).To(Or(
			BeNil(),
			MatchError(context.Canceled),
			MatchError(ContainSubstring("bubbletea: could not create cancelable reader")),
			MatchError(ContainSubstring("bubbletea: error opening TTY")),
		))
	})
})

func TestRunWithNilConfig(t *testing.T) {
	ctx := context.Background()
	err := tui.Run(ctx, nil, nil, "")
	if err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}
