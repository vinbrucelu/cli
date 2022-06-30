package ignitecmd

import (
	"github.com/spf13/cobra"

	"github.com/ignite/cli/ignite/services/scaffolder"
)

// NewScaffoldType returns a new command to scaffold a type.
func NewScaffoldType() *cobra.Command {
	c := &cobra.Command{
		Use:   "type NAME [field]...",
		Short: "Scaffold only a type definition",
		Args:  cobra.MinimumNArgs(1),
		RunE:  scaffoldTypeHandler,
	}

	flagSetPath(c)
	flagSetClearCache(c)
	c.Flags().AddFlagSet(flagSetScaffoldType())

	return c
}

func scaffoldTypeHandler(cmd *cobra.Command, args []string) error {
	return scaffoldType(cmd, args, scaffolder.DryType())
}
