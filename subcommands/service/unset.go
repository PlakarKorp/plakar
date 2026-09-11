package services

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type ServiceUnset struct {
	subcommands.SubcommandBase

	Service string
	Keys    []string
}

func (cmd *ServiceUnset) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service unset",
	}
}

func (cmd *ServiceUnset) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		return fmt.Errorf("no service specified")
	}

	cmd.Service = rest[0]
	cmd.Keys = rest[1:]

	return nil
}

func (cmd *ServiceUnset) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}

	if len(cmd.Keys) == 0 {
		return 0, nil
	}

	config, err := sc.GetServiceConfiguration(cmd.Service)
	if err != nil {
		return 1, err
	}

	for _, key := range cmd.Keys {
		delete(config, key)
	}

	if err := sc.SetServiceConfiguration(cmd.Service, config); err != nil {
		return 1, err
	}

	return 0, nil
}
