// SPDX-License-Identifier: MIT
package repokeeper

import "github.com/spf13/cobra"

const (
	repoSelectorUsage         = "Select a checkout by absolute path (symlinks are resolved), checkout_id, or repo_id.\nUse repo_id@checkout_id to qualify an ID shared by different repositories.\nRelative paths are resolved from the current directory or config root.\nAmbiguous selectors list the matching checkouts."
	repoFilterUsage           = "filter: all, errors, dirty, clean, gone, diverged, behind, ahead, equal, remote-mismatch, missing"
	fieldSelectorUsage        = "field selector (phase 1): tracking.status=all|gone|diverged|behind|ahead|equal, worktree.dirty=true|false, repo.error=true, repo.missing=true, remote.mismatch=true"
	labelSelectorUsage        = "label selector: key or key=value (comma-separated AND)"
	upstreamRepairFilterUsage = "filter: all, missing, mismatch"
	noHeadersUsage            = "when using table format, do not print headers"
	vcsUsage                  = "comma-separated vcs backends: git,hg (default: git)"
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

func addLabelSelectorFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("selector", "l", "", labelSelectorUsage)
}

func addUpstreamRepairFilterFlag(cmd *cobra.Command) {
	cmd.Flags().String("only", "all", upstreamRepairFilterUsage)
}

func addVCSFlag(cmd *cobra.Command) {
	cmd.Flags().String("vcs", "git", vcsUsage)
}
