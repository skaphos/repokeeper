// SPDX-License-Identifier: MIT
package gitx

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/model"
)

func TestTrackingFromShort(t *testing.T) {
	cases := []struct {
		short string
		want  model.TrackingStatus
	}{
		{short: ">", want: model.TrackingAhead},
		{short: "<", want: model.TrackingBehind},
		{short: "<>", want: model.TrackingDiverged},
		{short: "=", want: model.TrackingEqual},
		{short: "?", want: model.TrackingNone},
	}
	for _, tc := range cases {
		got := trackingFromShort(ForEachRefEntry{Upstream: "origin/main", TrackShort: tc.short})
		if got.Status != tc.want {
			t.Fatalf("short=%q got=%q want=%q", tc.short, got.Status, tc.want)
		}
	}
}
