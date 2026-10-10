// SPDX-License-Identifier: MIT
package mcpinstall

import "testing"

// setTestHome points os.UserHomeDir at dir on every platform: it reads HOME on
// Unix and USERPROFILE on Windows.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}
