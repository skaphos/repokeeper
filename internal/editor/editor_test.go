// SPDX-License-Identifier: MIT
package editor

import (
	"testing"
)

func TestResolveEditorCommand(t *testing.T) {
	tests := []struct {
		name      string
		visual    string
		editor    string
		wantParts []string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "VISUAL env set",
			visual:    "vim",
			editor:    "",
			wantParts: []string{"vim"},
			wantErr:   false,
		},
		{
			name:      "EDITOR env set when VISUAL empty",
			visual:    "",
			editor:    "nano",
			wantParts: []string{"nano"},
			wantErr:   false,
		},
		{
			name:      "VISUAL takes precedence over EDITOR",
			visual:    "vim",
			editor:    "nano",
			wantParts: []string{"vim"},
			wantErr:   false,
		},
		{
			name:      "Neither VISUAL nor EDITOR set",
			visual:    "",
			editor:    "",
			wantParts: nil,
			wantErr:   true,
			errMsg:    "no editor configured",
		},
		{
			name:      "Editor with arguments",
			visual:    "vim -u NONE",
			editor:    "",
			wantParts: []string{"vim", "-u", "NONE"},
			wantErr:   false,
		},
		{
			name:      "Editor with multiple arguments",
			visual:    "code --wait --new-window",
			editor:    "",
			wantParts: []string{"code", "--wait", "--new-window"},
			wantErr:   false,
		},
		{
			name:      "Editor with quoted arguments",
			visual:    `emacs "file with spaces.txt"`,
			editor:    "",
			wantParts: []string{"emacs", "file with spaces.txt"},
			wantErr:   false,
		},
		{
			name:      "Invalid shell quoting",
			visual:    `vim "unclosed quote`,
			editor:    "",
			wantParts: nil,
			wantErr:   true,
			errMsg:    "invalid editor command",
		},
		{
			name:      "VISUAL with leading/trailing whitespace",
			visual:    "  vim  ",
			editor:    "",
			wantParts: []string{"vim"},
			wantErr:   false,
		},
		{
			name:      "EDITOR with leading/trailing whitespace",
			visual:    "",
			editor:    "  nano  ",
			wantParts: []string{"nano"},
			wantErr:   false,
		},
		{
			name:      "Empty string after trimming VISUAL",
			visual:    "   ",
			editor:    "nano",
			wantParts: []string{"nano"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			parts, err := ResolveEditorCommand()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(parts) != len(tt.wantParts) {
				t.Errorf("expected %d parts, got %d: %v", len(tt.wantParts), len(parts), parts)
				return
			}

			for i, want := range tt.wantParts {
				if parts[i] != want {
					t.Errorf("part[%d]: expected %q, got %q", i, want, parts[i])
				}
			}
		})
	}
}

// contains checks if substr is in str
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 || (len(substr) > 0 && len(str) > 0 && str[0:len(substr)] == substr) || (len(substr) > 0 && len(str) > 0 && findSubstring(str, substr)))
}

// findSubstring is a simple substring search
func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
