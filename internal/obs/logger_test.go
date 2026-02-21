// SPDX-License-Identifier: MIT
package obs

import "testing"

func TestNopLoggerDoesNotPanic(t *testing.T) {
	l := NopLogger()
	l.Infof("info %s", "msg")
	l.Debugf("debug %d", 42)
	l.Warnf("warn %v", true)
}

func TestNopLoggerSatisfiesInterface(t *testing.T) {
	var _ = NopLogger()
}
