package config

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type ConfigStoreCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigStoreCmd) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "store",
	}
}

func (cmd *ConfigStoreCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("no action specified")
	}
	cmd.args = rest
	return nil
}

func (cmd *ConfigStoreCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	err := dispatchSubcommand(ctx, "store", cmd.args[0], cmd.args[1:])
	if err != nil {
		return 1, err
	}
	return 0, nil
}
