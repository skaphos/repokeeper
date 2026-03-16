// SPDX-License-Identifier: MIT
// Package editor resolves the user's preferred editor command from the environment.
package editor

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/go-shellwords"
)

func ResolveEditorCommand() ([]string, error) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return nil, fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}
	parts, err := shellwords.Parse(editor)
	if err != nil {
		return nil, fmt.Errorf("invalid editor command %q: %w", editor, err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}
	return parts, nil
}
