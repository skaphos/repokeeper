package cliio_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/cliio"
)

type errorWriter struct{}

func (e *errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestPromptYesNo(t *testing.T) {
	out := &bytes.Buffer{}
	ok, err := cliio.PromptYesNo(out, strings.NewReader("yes\n"), "Proceed? [y/N]: ")
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if !ok {
		t.Fatal("expected yes response")
	}
	if got := out.String(); got != "Proceed? [y/N]: " {
		t.Fatalf("unexpected prompt output: %q", got)
	}
}

func TestPromptYesNoNoAndEOF(t *testing.T) {
	out := &bytes.Buffer{}
	ok, err := cliio.PromptYesNo(out, strings.NewReader("n"), "Proceed? [y/N]: ")
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}
	if ok {
		t.Fatal("expected no response to be false")
	}
}

func TestWriteTable(t *testing.T) {
	out := &bytes.Buffer{}
	err := cliio.WriteTable(out, false, false, []string{"A", "B"}, [][]string{{"1", "2"}})
	if err != nil {
		t.Fatalf("unexpected write table error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "A") || !strings.Contains(got, "1") {
		t.Fatalf("unexpected table output: %q", got)
	}
}

func TestWriteTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	err := cliio.WriteTable(out, false, true, []string{"A", "B"}, [][]string{{"1", "2"}})
	if err != nil {
		t.Fatalf("unexpected write table error: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "A") {
		t.Fatalf("expected header omission, got %q", got)
	}
	if !strings.Contains(got, "1") {
		t.Fatalf("expected row output, got %q", got)
	}
}

func TestPromptYesNoWriteError(t *testing.T) {
	if _, err := cliio.PromptYesNo(&errorWriter{}, strings.NewReader("y\n"), "Proceed? "); err == nil {
		t.Fatal("expected prompt writer error")
	}
}

func TestWriteTableWriteError(t *testing.T) {
	err := cliio.WriteTable(&errorWriter{}, false, false, []string{"A"}, [][]string{{"1"}})
	if err == nil {
		t.Fatal("expected table writer error")
	}
}
