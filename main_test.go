// SPDX-License-Identifier: MIT
package main

import "testing"

func TestMainInvokesExecute(t *testing.T) {
	prev := execute
	called := false
	execute = func() { called = true }
	defer func() { execute = prev }()

	main()

	if !called {
		t.Fatal("expected main to invoke execute")
	}
}
