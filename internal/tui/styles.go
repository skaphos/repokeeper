// SPDX-License-Identifier: MIT
package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/skaphos/repokeeper/internal/model"
)

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
	selectedStyle       = lipgloss.NewStyle().Bold(true)
)

func trackingRowStyle(r model.RepoStatus) lipgloss.Style {
	if r.Error != "" || r.ErrorClass != "" {
		return lipgloss.NewStyle().Foreground(colorError)
	}
	switch r.Tracking.Status {
	case model.TrackingBehind, model.TrackingDiverged, model.TrackingGone:
		return lipgloss.NewStyle().Foreground(colorError)
	case model.TrackingEqual:
		if r.Worktree != nil && r.Worktree.Dirty {
			return lipgloss.NewStyle().Foreground(colorWarn)
		}
		return lipgloss.NewStyle().Foreground(colorHealthy)
	default:
		if r.Worktree != nil && r.Worktree.Dirty {
			return lipgloss.NewStyle().Foreground(colorWarn)
		}
		return lipgloss.NewStyle()
	}
}
