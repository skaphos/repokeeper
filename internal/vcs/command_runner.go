// SPDX-License-Identifier: MIT
package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func runCommand(ctx context.Context, dir, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText != "" {
			return "", fmt.Errorf("%s %s: %s: %w", bin, strings.Join(args, " "), errText, err)
		}
		return "", fmt.Errorf("%s %s: %w", bin, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// rejectFlagLike rejects a positional argument (URL, path, or ref) that
// begins with "-". hg (like git) parses such values as an option rather
// than a literal positional argument, so an attacker-controlled remote
// URL, branch, or path beginning with "-" could otherwise smuggle
// arbitrary flags into the hg invocation.
func rejectFlagLike(field, value string) error {
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("vcs: %s must not start with '-': %q", field, value)
	}
	return nil
}
