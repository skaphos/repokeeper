package repokeeper

import "github.com/spf13/cobra"

// logOutputWriteFailure records non-fatal output write/flush failures.
// CLI consumers frequently pipe to tools that close early (for example `head`),
// so we log and continue instead of treating these as command failures.
func logOutputWriteFailure(cmd *cobra.Command, context string, err error) {
	if err == nil {
		return
	}
	debugf(cmd, "ignored output write failure (%s): %v", context, err)
}
