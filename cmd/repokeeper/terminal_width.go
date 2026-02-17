// SPDX-License-Identifier: MIT
package repokeeper

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	narrowTableWidth = 100
	tinyTableWidth   = 80
)

var getTerminalSize = term.GetSize

func tableWidth(cmd *cobra.Command) (int, bool) {
	if cmd == nil {
		return 0, false
	}
	file, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return 0, false
	}
	fd := int(file.Fd())
	if !isTerminalFD(fd) {
		return 0, false
	}
	width, _, err := getTerminalSize(fd)
	if err != nil || width <= 0 {
		return 0, false
	}
	return width, true
}

func adaptiveCellLimit(cmd *cobra.Command, normal, narrow, tiny int) int {
	width, ok := tableWidth(cmd)
	if !ok {
		return normal
	}
	return adaptiveCellLimitForWidth(width, normal, narrow, tiny)
}

func adaptiveCellLimitForWidth(width, normal, narrow, tiny int) int {
	switch {
	case width > 0 && width < tinyTableWidth && tiny > 0:
		return tiny
	case width > 0 && width < narrowTableWidth && narrow > 0:
		return narrow
	default:
		return normal
	}
}
