// SPDX-License-Identifier: MIT
package tui

import "charm.land/lipgloss/v2"

// Semantic color palette - mirrors termstyle semantics via lipgloss.
var (
	colorHealthy = lipgloss.Color("2")
	colorWarn    = lipgloss.Color("3")
	colorError   = lipgloss.Color("1")
	colorInfo    = lipgloss.Color("4")
	colorMuted   = lipgloss.Color("8")
)

var (
	titleStyle     = lipgloss.NewStyle().Bold(true)
	headerStyle    = lipgloss.NewStyle().Bold(true)
	cursorStyle    = lipgloss.NewStyle().Reverse(true)
	statusBarStyle = lipgloss.NewStyle().Foreground(colorMuted)
	errorTextStyle = lipgloss.NewStyle().Foreground(colorError)
	loadingStyle   = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	statusEqualStyle    = lipgloss.NewStyle().Foreground(colorHealthy)
	statusAheadStyle    = lipgloss.NewStyle().Foreground(colorInfo)
	statusBehindStyle   = lipgloss.NewStyle().Foreground(colorWarn)
	statusDivergedStyle = lipgloss.NewStyle().Foreground(colorError)
	statusGoneStyle     = lipgloss.NewStyle().Foreground(colorError)
	statusNoneStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	dirtyStyle          = lipgloss.NewStyle().Foreground(colorWarn)
	errorClassStyle     = lipgloss.NewStyle().Foreground(colorError)
	syncedOKStyle       = lipgloss.NewStyle().Foreground(colorHealthy)
	syncedFailStyle     = lipgloss.NewStyle().Foreground(colorError)
)
