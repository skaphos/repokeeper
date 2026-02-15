package repokeeper

import "github.com/spf13/cobra"

const (
	repoFilterUsage = "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing"
	fieldSelectorUsage = "field selector (phase 1): tracking.status=diverged|gone, worktree.dirty=true|false, repo.error=true, repo.missing=true, remote.mismatch=true"
	upstreamRepairFilterUsage = "filter: all, missing, mismatch"
	noHeadersUsage = "when using table format, do not print headers"
)

func addFormatFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().StringP("format", "o", "table", usage)
}

func addNoHeadersFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("no-headers", false, noHeadersUsage)
}

func addRepoFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("only", "all", repoFilterUsage)
	cmd.Flags().String("field-selector", "", fieldSelectorUsage)
}

func addUpstreamRepairFilterFlag(cmd *cobra.Command) {
	cmd.Flags().String("only", "all", upstreamRepairFilterUsage)
}
