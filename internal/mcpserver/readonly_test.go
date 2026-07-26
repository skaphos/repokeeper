// SPDX-License-Identifier: MIT

package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestExplainReadOnlyPassesThroughUnrelatedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"plain error", errors.New("registry entry not found")},
		{"permission denied", fmt.Errorf("writing config: %w", fs.ErrPermission)},
		{"not exist", fmt.Errorf("reading config: %w", fs.ErrNotExist)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := explainReadOnly(tc.err)
			if !errors.Is(got, tc.err) {
				t.Errorf("error identity not preserved: got %v want %v", got, tc.err)
			}
			if tc.err != nil && strings.Contains(got.Error(), "read-only") {
				t.Errorf("unrelated error was misdiagnosed as read-only: %v", got)
			}
		})
	}
}

// TestExplainReadOnlyNamesCauseAndRemedy is the requirement: a refusal that does
// not say why, and what would change it, is a defect.
func TestExplainReadOnlyNamesCauseAndRemedy(t *testing.T) {
	t.Parallel()

	underlying := readOnlyError()
	if underlying == nil {
		t.Skip("read-only filesystem errors are not detectable on this platform")
	}

	got := explainReadOnly(fmt.Errorf("writing registry: %w", underlying))
	if got == nil {
		t.Fatal("explainReadOnly returned nil for a read-only failure")
	}
	msg := got.Error()

	// Names the cause.
	if !strings.Contains(msg, "read-only") {
		t.Errorf("refusal does not name the read-only mount as the cause: %q", msg)
	}
	// States the remedy, in the terms the user configured.
	for _, want := range []string{"remount", ":ro"} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal does not state the remedy (missing %q): %q", want, msg)
		}
	}
	// Says inspection still works -- degradation is read-only, not blind.
	if !strings.Contains(msg, "inspection tools are unaffected") {
		t.Errorf("refusal does not say inspection is unaffected: %q", msg)
	}
	// The original cause must remain unwrappable for callers and tests.
	if !errors.Is(got, underlying) {
		t.Errorf("original error was not wrapped: %v", got)
	}
}

// TestSerializeToolTranslatesReadOnlyFailures proves the translation is actually
// reached by the shared handler wrapper, rather than merely existing.
func TestSerializeToolTranslatesReadOnlyFailures(t *testing.T) {
	t.Parallel()

	underlying := readOnlyError()
	if underlying == nil {
		t.Skip("read-only filesystem errors are not detectable on this platform")
	}

	s := New(nil, "", "", nil)
	handler := s.serializeTool(func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, fmt.Errorf("updating registry: %w", underlying)
	})

	_, err := handler(context.Background(), mcp.CallToolRequest{})
	if err == nil {
		t.Fatal("expected an error from the wrapped handler")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("serializeTool did not translate the read-only failure: %v", err)
	}
}

// TestSerializeToolLeavesSuccessfulCallsAlone: the wrapper must be inert for the
// read-only inspection tools, which is the whole point of translating centrally.
func TestSerializeToolLeavesSuccessfulCallsAlone(t *testing.T) {
	t.Parallel()

	s := New(nil, "", "", nil)
	want := &mcp.CallToolResult{}
	handler := s.serializeTool(func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return want, nil
	})

	got, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Error("result was not passed through unchanged")
	}
}

// TestReadOnlyToolsRemainAdvertised guards the decision not to hide mutating
// tools under a read-only mount. A silently reduced surface is the failure mode
// this refusal exists to avoid.
func TestMutatingToolsRemainAdvertised(t *testing.T) {
	t.Parallel()

	s := New(nil, "", "", nil)
	all := s.inner.ListTools()
	readOnly := ReadOnlyToolNames()

	if len(all) <= len(readOnly) {
		t.Fatalf("expected mutating tools to be registered: %d total, %d read-only", len(all), len(readOnly))
	}

	// Every mutating tool must still be present in the advertised surface.
	for _, name := range []string{
		"add_repository", "remove_repository", "execute_sync", "set_labels", "scan_workspace",
	} {
		if _, ok := all[name]; !ok {
			t.Errorf("mutating tool %q is not advertised; it must be present and refuse with a reason", name)
		}
	}
}
