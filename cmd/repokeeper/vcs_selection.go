package repokeeper

import (
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

func selectedAdapterForCommand(cmd *cobra.Command) (vcs.Adapter, error) {
	raw := getStringFlag(cmd, "vcs")
	return vcs.NewAdapterForSelection(raw)
}
