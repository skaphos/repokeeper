// SPDX-License-Identifier: MIT
package termstyle

import "github.com/liggitt/tabwriter"

const (
	Reset = "\x1b[0m"
	Green = "\x1b[32m"
	Brown = "\x1b[33m"
	Red   = "\x1b[31m"
	Blue  = "\x1b[34m"

	// Semantic aliases used by table/status output.
	Healthy = Green
	Warn    = Brown
	Error   = Red
	Info    = Blue
)

// Colorize wraps a value in ANSI escapes when color output is enabled.
func Colorize(enabled bool, value, color string) string {
	if !enabled || value == "" || color == "" {
		return value
	}
	// Hide ANSI sequences from tabwriter width calculations so columns align.
	esc := string([]byte{tabwriter.Escape})
	return esc + color + esc + value + esc + Reset + esc
}
