// SPDX-License-Identifier: MIT
package engine

import (
	"testing"
)

func TestHintForErrorClass_KnownClasses(t *testing.T) {
	known := []string{"auth", "network", "timeout", "corrupt", "missing_remote"}
	for _, class := range known {
		hint := hintForErrorClass(class)
		if hint == "" {
			t.Errorf("hintForErrorClass(%q): expected non-empty hint, got empty string", class)
		}
	}
}

func TestHintForErrorClass_UnknownClasses(t *testing.T) {
	unknown := []string{"unknown", "", "invalid", "skipped", "missing", "bogus"}
	for _, class := range unknown {
		hint := hintForErrorClass(class)
		if hint != "" {
			t.Errorf("hintForErrorClass(%q): expected empty string, got %q", class, hint)
		}
	}
}

func TestHintForErrorClass_Deterministic(t *testing.T) {
	for i := 0; i < 20; i++ {
		got := hintForErrorClass("auth")
		want := "check SSH keys or credentials for this remote"
		if got != want {
			t.Errorf("iteration %d: hintForErrorClass(\"auth\") = %q, want %q", i, got, want)
		}
	}
}
