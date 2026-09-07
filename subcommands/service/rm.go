package services

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type ServiceRm struct {
	subcommands.SubcommandBase

	Service string
}

func (cmd *ServiceRm) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service rm",
	}
}

func (cmd *ServiceRm) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		return fmt.Errorf("no service specified")
	}

	if len(rest) > 1 {
		return fmt.Errorf("invalid argument %q", rest[1])
	}

	cmd.Service = rest[0]

	return nil
}

func (cmd *ServiceRm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}
	if err := sc.SetServiceStatus(cmd.Service, false); err != nil {
		return 1, err
	}
	if err := sc.SetServiceConfiguration(cmd.Service, make(map[string]string)); err != nil {
		return 1, err
	}

	return 0, nil
}
