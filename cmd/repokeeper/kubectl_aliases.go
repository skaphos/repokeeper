package repokeeper

import "github.com/spf13/cobra"

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Display one or many resources",
}

var getReposCmd = &cobra.Command{
	Use:     "repos",
	Aliases: []string{"repo"},
	Short:   "Report repo health for all registered repositories",
	RunE:    statusCmd.RunE,
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile local repositories with upstream state",
}

var reconcileReposCmd = &cobra.Command{
	Use:     "repos",
	Aliases: []string{"repo"},
	Short:   "Run safe fetch/prune on registered repositories",
	RunE:    syncCmd.RunE,
}

var repairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair repository metadata and tracking state",
}

var repairUpstreamAliasCmd = &cobra.Command{
	Use:   "upstream",
	Short: "Repair missing or mismatched upstream tracking for registered repos",
	RunE:  repairUpstreamCmd.RunE,
}

func init() {
	getReposCmd.Flags().String("roots", "", "additional roots to scan (optional)")
	getReposCmd.Flags().String("registry", "", "override registry file path")
	getReposCmd.Flags().String("format", "table", "output format: table or json")
	getReposCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing")
	getReposCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")
	getCmd.AddCommand(getReposCmd)

	reconcileReposCmd.Flags().String("only", "all", "filter: all, errors, dirty, clean, gone, diverged, remote-mismatch, missing")
	reconcileReposCmd.Flags().Int("concurrency", 0, "max concurrent repo operations (default: min(8, NumCPU))")
	reconcileReposCmd.Flags().Int("timeout", 60, "timeout in seconds per repo")
	reconcileReposCmd.Flags().Bool("continue-on-error", true, "continue syncing remaining repos after a per-repo failure")
	reconcileReposCmd.Flags().Bool("dry-run", false, "print intended operations without executing")
	reconcileReposCmd.Flags().Bool("yes", false, "accept sync plan and execute without confirmation")
	reconcileReposCmd.Flags().Bool("update-local", false, "after fetch, run pull --rebase only for clean branches tracking */main")
	reconcileReposCmd.Flags().Bool("push-local", false, "when used with --update-local, push branches that are ahead of upstream")
	reconcileReposCmd.Flags().Bool("rebase-dirty", false, "when used with --update-local, stash local changes before rebase and pop afterwards")
	reconcileReposCmd.Flags().Bool("force", false, "when used with --update-local, allow rebase even when branch tracking state is diverged")
	reconcileReposCmd.Flags().String("protected-branches", "main,master,release/*", "comma-separated branch patterns to protect from auto-rebase during --update-local")
	reconcileReposCmd.Flags().Bool("allow-protected-rebase", false, "when used with --update-local, allow rebase on branches matched by --protected-branches")
	reconcileReposCmd.Flags().Bool("checkout-missing", false, "clone missing repos from registry remote_url back to their registered paths")
	reconcileReposCmd.Flags().String("format", "table", "output format: table or json")
	reconcileReposCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")
	reconcileReposCmd.Flags().Bool("wrap", false, "allow table columns to wrap instead of truncating")
	reconcileCmd.AddCommand(reconcileReposCmd)

	repairUpstreamAliasCmd.Flags().String("registry", "", "override registry file path")
	repairUpstreamAliasCmd.Flags().Bool("dry-run", true, "preview upstream repairs without executing git changes")
	repairUpstreamAliasCmd.Flags().String("only", "all", "filter: all, missing, mismatch")
	repairUpstreamAliasCmd.Flags().String("format", "table", "output format: table or json")
	repairUpstreamAliasCmd.Flags().Bool("no-headers", false, "when using table format, do not print headers")
	repairCmd.AddCommand(repairUpstreamAliasCmd)

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(reconcileCmd)
	rootCmd.AddCommand(repairCmd)
}
