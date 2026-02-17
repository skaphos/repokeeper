// SPDX-License-Identifier: MIT
package termstyle

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	if got := Colorize(false, "up", Green); got != "up" {
		t.Fatalf("expected plain output when disabled, got %q", got)
	}
	if got := Colorize(true, "", Green); got != "" {
		t.Fatalf("expected empty value passthrough, got %q", got)
	}
	if got := Colorize(true, "up", ""); got != "up" {
		t.Fatalf("expected empty color passthrough, got %q", got)
	}
	colored := Colorize(true, "up", Green)
	if !strings.Contains(colored, Green) || !strings.Contains(colored, Reset) {
		t.Fatalf("expected ANSI wrapped output, got %q", colored)
	}
}
