// SPDX-License-Identifier: MIT
package repokeeper

import (
	"github.com/spf13/cobra"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/registry"
)

// cloneImportedRepos is a test-only helper that collapses the plan/execute
// two-step into a single call. Production code calls planImportedEntries and
// executeImportClonePlanWithProgress separately so the user can confirm the
// plan before any clones run.
func cloneImportedRepos(cmd *cobra.Command, cfg *config.Config, bundle exportBundle, cwd string, dangerouslyDeleteExisting bool) error {
	_, err := cloneImportedReposWithProgress(cmd, cfg, bundle, cwd, dangerouslyDeleteExisting, nil)
	return err
}

func cloneImportedReposWithProgress(cmd *cobra.Command, cfg *config.Config, bundle exportBundle, cwd string, dangerouslyDeleteExisting bool, progress *syncProgressWriter) ([]engine.SyncResult, error) {
	return cloneImportedEntriesWithProgress(cmd, cfg, bundle, cwd, dangerouslyDeleteExisting, nil, progress)
}

func cloneImportedEntriesWithProgress(
	cmd *cobra.Command,
	cfg *config.Config,
	bundle exportBundle,
	cwd string,
	dangerouslyDeleteExisting bool,
	entries []registry.Entry,
	progress *syncProgressWriter,
) ([]engine.SyncResult, error) {
	plan, err := planImportedEntries(cfg, bundle, cwd, dangerouslyDeleteExisting, entries)
	if err != nil {
		return nil, err
	}
	return executeImportClonePlanWithProgress(cmd, cfg, plan, progress)
}
