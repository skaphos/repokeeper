package tableutil

import (
	"bytes"
	"testing"
)

func TestPrintHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	PrintHeaders(buf, true, "A\tB")
	if buf.Len() != 0 {
		t.Fatalf("expected no output when disabled, got %q", buf.String())
	}

	PrintHeaders(buf, false, "A\tB")
	if got := buf.String(); got != "A\tB\n" {
		t.Fatalf("unexpected header output: %q", got)
	}
}

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New(buf, true)
	_, _ = w.Write([]byte("A\tB\n"))
	_ = w.Flush()
	if buf.Len() == 0 {
		t.Fatal("expected writer output")
	}
}
