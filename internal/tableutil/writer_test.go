// SPDX-License-Identifier: MIT
package tableutil

import (
	"bytes"
	"testing"
)

func TestPrintHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := PrintHeaders(buf, true, "A\tB"); err != nil {
		t.Fatalf("unexpected header error: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output when disabled, got %q", buf.String())
	}

	if err := PrintHeaders(buf, false, "A\tB"); err != nil {
		t.Fatalf("unexpected header error: %v", err)
	}
	if got := buf.String(); got != "A\tB\n" {
		t.Fatalf("unexpected header output: %q", got)
	}
}

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New(buf, true)
	if _, err := w.Write([]byte("A\tB\n")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("unexpected flush error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected writer output")
	}
}
