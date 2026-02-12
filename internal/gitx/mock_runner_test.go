package gitx_test

import (
	"context"
	"fmt"
	"strings"
)

// MockRunner implements gitx.Runner for testing.
type MockRunner struct {
	// Responses maps "dir:args" keys to (output, error) pairs.
	Responses map[string]MockResponse
}

type MockResponse struct {
	Output string
	Err    error
}

func (m *MockRunner) Run(_ context.Context, dir string, args ...string) (string, error) {
	key := dir + ":" + strings.Join(args, " ")
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Err
	}
	// Also try without dir for convenience
	keyNoDir := ":" + strings.Join(args, " ")
	if resp, ok := m.Responses[keyNoDir]; ok {
		return resp.Output, resp.Err
	}
	return "", fmt.Errorf("unexpected call: dir=%q args=%v", dir, args)
}
