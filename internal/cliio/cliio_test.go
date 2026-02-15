package cliio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/cliio"
)

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
