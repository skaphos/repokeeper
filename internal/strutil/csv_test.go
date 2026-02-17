// SPDX-License-Identifier: MIT
package strutil_test

import (
	"testing"

	"github.com/skaphos/repokeeper/internal/strutil"
)

func TestSplitCSV(t *testing.T) {
	got := strutil.SplitCSV(" a, ,b,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}
