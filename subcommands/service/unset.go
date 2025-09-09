package services

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ServiceUnset struct {
	subcommands.SubcommandBase

	Service string
	Keys    []string
}

func (cmd *ServiceUnset) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("service unset", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s <name> <key>...\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("no service specified")
	}

	cmd.Service = flags.Arg(0)
	cmd.Keys = flags.Args()[1:]

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
